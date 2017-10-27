package paxos

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cmu440-F17/paxosapp/rpc/paxosrpc"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"sync"
	"time"
)

type acceptedValue struct {
	nk    int
	value interface{}
}

type paxosNode struct {
	// private info
	myHostPort string
	srvId      int
	numNodes   int
	majority   int

	// nodesInfo
	nodes        []*rpc.Client
	oneOtherNode *rpc.Client

	// stored info
	npMap              map[string]int
	nextProposalNumMap map[string]int
	acceptedMap        map[string]*acceptedValue
	storage            map[string]interface{}

	// locks
	npMapLock              *sync.Mutex
	nextProposalNumMapLock *sync.Mutex
	storageLock            *sync.Mutex
}

var PROPOSE_TIMEOUT = 15 * time.Second

/**
*   Helper function
 */
// set up connections with all nodes
func (pn *paxosNode) setUpAllConnections(numRetries int, hostMap map[int]string) error {
	idx := 0
	for id, hostPort := range hostMap {
		attemptNum := 0
		for ; attemptNum < numRetries; attemptNum++ {
			node, err := rpc.DialHTTP("tcp", hostPort)
			if err != nil {
				time.Sleep(1 * time.Second)
			} else {
				if id != pn.srvId {
					pn.oneOtherNode = node
				}
				pn.nodes[idx] = node
				break
			}
		}
		if attemptNum == numRetries {
			return errors.New("Can't connect to node " + hostPort)
		}
		idx++
	}
	return nil
}

func (pn *paxosNode) sendPrepare(key string, n int, node *rpc.Client, stage1Chan chan *paxosrpc.PrepareReply) {
	args := &paxosrpc.PrepareArgs{
		Key:         key,
		N:           n,
		RequesterId: pn.srvId,
	}
	var reply paxosrpc.PrepareReply

	err := node.Call("PaxosNode.RecvPrepare", args, &reply)
	if err != nil {
		stage1Chan <- nil
	} else {
		stage1Chan <- &reply
	}
}

func (pn *paxosNode) sendPleaseAccept(key string, n int, v interface{}, node *rpc.Client, stage2Chan chan *paxosrpc.AcceptReply) {
	args := &paxosrpc.AcceptArgs{
		Key:         key,
		N:           n,
		V:           v,
		RequesterId: pn.srvId,
	}
	var reply paxosrpc.AcceptReply
	err := node.Call("PaxosNode.RecvAccept", args, &reply)
	if err != nil {
		stage2Chan <- nil
	} else {
		stage2Chan <- &reply
	}
}

func (pn *paxosNode) sendCommit(key string, v interface{}, node *rpc.Client, stage3Chan chan *paxosrpc.CommitReply) {
	args := &paxosrpc.CommitArgs{
		Key:         key,
		V:           v,
		RequesterId: pn.srvId,
	}
	var reply paxosrpc.CommitReply
	err := node.Call("PaxosNode.RecvCommit", args, &reply)
	if err != nil {
		stage3Chan <- nil
	} else {
		stage3Chan <- &reply
	}
}

func (pn *paxosNode) doPropse(key string, n int, v interface{},
	reply *paxosrpc.ProposeReply,
	done chan error) {

	//fmt.Printf("node %d: begin propose: key=%s, n=%d, v=%v\n", pn.srvId, key, n, v)
	// 1. prepare stage
	stage1Chan := make(chan *paxosrpc.PrepareReply)
	for _, node := range pn.nodes {
		go pn.sendPrepare(key, n, node, stage1Chan)
	}

	promiseCount := 0
	curHighestN := -1
	for {
		res := <-stage1Chan
		if res == nil {
			done <- errors.New("prepare rpc error")
			return
		} else if res.Status == paxosrpc.Reject {
			done <- errors.New("stage1 reject")
			return
		} else {
			promiseCount++
			if res.N_a > curHighestN && res.V_a != nil {
				v = res.V_a
				curHighestN = res.N_a
			}

			if promiseCount >= pn.majority {
				break
			}
		}
	}

	//fmt.Printf("node %d: begin 2nd stage: key=%s, n=%d, v=%v\n", pn.srvId, key, n, v)
	// 2. please accept stage
	stage2Chan := make(chan *paxosrpc.AcceptReply)
	for _, node := range pn.nodes {
		go pn.sendPleaseAccept(key, n, v, node, stage2Chan)
	}

	acceptCount := 0
	for {
		res := <-stage2Chan
		if res == nil {
			done <- errors.New("PleaseAccept rpc error")
			return
		} else if res.Status == paxosrpc.Reject {
			done <- errors.New("stage2 reject")
			return
		} else {
			acceptCount++
			if acceptCount >= pn.majority {
				break
			}
		}
	}

	//fmt.Printf("node %d: begin commit stage: key=%s, v=%v\n", pn.srvId, key, v)
	// 3. commit stage
	stage3Chan := make(chan *paxosrpc.CommitReply)
	for _, node := range pn.nodes {
		go pn.sendCommit(key, v, node, stage3Chan)
	}

	commitCount := 0
	for {
		res := <-stage3Chan
		if res == nil {
			done <- errors.New("Commit rpc error")
			return
		} else {
			commitCount++
			if commitCount >= pn.numNodes {
				reply.V = v
				//fmt.Printf("node %d: finish commit key=%s, v=%v\n", pn.srvId, key, v)
				done <- nil
				return
			}
		}
	}
}

// Desc:
// NewPaxosNode creates a new PaxosNode. This function should return only when
// all nodes have joined the ring, and should return a non-nil error if the node
// could not be started in spite of dialing any other nodes numRetries times.
//
// Params:
// myHostPort: the hostport string of this new node. We use tcp in this project
// hostMap: a map from all node IDs to their hostports
// numNodes: the number of nodes in the ring
// numRetries: if we can't connect with some nodes in hostMap after numRetries attempts, an error should be returned
// replace: a flag which indicates whether this node is a replacement for a node which failed.
func NewPaxosNode(myHostPort string, hostMap map[int]string, numNodes, srvId, numRetries int, replace bool) (PaxosNode, error) {
	pn := &paxosNode{
		// private info
		myHostPort: myHostPort,
		srvId:      srvId,
		numNodes:   numNodes,
		majority:   numNodes/2 + 1,

		// nodesInfo
		nodes:        make([]*rpc.Client, numNodes),
		oneOtherNode: nil,

		// stored info
		npMap:              make(map[string]int),
		nextProposalNumMap: make(map[string]int),
		acceptedMap:        make(map[string]*acceptedValue),
		storage:            make(map[string]interface{}),

		// locks
		npMapLock:              &sync.Mutex{},
		nextProposalNumMapLock: &sync.Mutex{},
		storageLock:            &sync.Mutex{},
	}

	l, err := net.Listen("tcp", myHostPort)
	if err != nil {
		log.Fatal("listen error:", err)
		return nil, err
	}

	rpc.RegisterName("PaxosNode", paxosrpc.Wrap(pn))
	rpc.HandleHTTP()
	go http.Serve(l, nil)

	if err := pn.setUpAllConnections(numRetries, hostMap); err != nil {
		return nil, err
	}

	// do some recovery work
	if replace {
		// 1. notify all other nodes
		for _, node := range pn.nodes {
			args := &paxosrpc.ReplaceServerArgs{
				SrvID:    srvId,
				Hostport: myHostPort,
			}
			var reply paxosrpc.ReplaceServerReply
			err := node.Call("PaxosNode.RecvReplaceServer", args, &reply)
			if err != nil {
				return nil, err
			}
		}

		// 2. recover storage from oneOtherNode
		args := &paxosrpc.ReplaceCatchupArgs{}
		var reply paxosrpc.ReplaceCatchupReply
		err := pn.oneOtherNode.Call("PaxosNode.RecvReplaceCatchup", args, &reply)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(reply.Data, &pn.storage)
		if err != nil {
			return nil, err
		}
	}

	fmt.Printf("Paxos node %d listens on %s\n", srvId, myHostPort)

	return pn, nil
}

// Desc:
// GetNextProposalNumber generates a proposal number which will be passed to
// Propose. Proposal numbers should not repeat for a key, and for a particular
// <node, key> pair, they should be strictly increasing.
//
// Params:
// args: the key to propose
// reply: the next proposal number for the given key
func (pn *paxosNode) GetNextProposalNumber(args *paxosrpc.ProposalNumberArgs, reply *paxosrpc.ProposalNumberReply) error {
	key := args.Key
	pn.nextProposalNumMapLock.Lock()
	defer pn.nextProposalNumMapLock.Unlock()

	if _, exist := pn.nextProposalNumMap[key]; !exist {
		pn.nextProposalNumMap[key] = pn.srvId
	} else {
		pn.nextProposalNumMap[key] += pn.numNodes
	}

	reply.N = pn.nextProposalNumMap[key]

	fmt.Printf("node %d: key = %s, next prop num = %d\n", pn.srvId, key, reply.N)
	return nil
}

// Desc:
// Propose initializes proposing a value for a key, and replies with the
// value that was committed for that key. Propose should not return until
// a value has been committed, or an error occurs, or 15 seconds have passed.
//
// Params:
// args: the key, value pair to propose together with the proposal number returned by GetNextProposalNumber
// reply: value that was actually committed for the given key
func (pn *paxosNode) Propose(args *paxosrpc.ProposeArgs, reply *paxosrpc.ProposeReply) error {
	key := args.Key
	n := args.N
	v := args.V

	c := make(chan error, 1)
	go pn.doPropse(key, n, v, reply, c)
	select {
	case err := <-c:
		//fmt.Printf("node %d: finish propose err=%s\n", pn.srvId, err)
		return err
	case <-time.After(PROPOSE_TIMEOUT):
		return errors.New("Propose Times out")
	}

	return nil
}

// Desc:
// GetValue looks up the value for a key, and replies with the value or with
// the Status KeyNotFound.
//
// Params:
// args: the key to check
// reply: the value and status for this lookup of the given key
func (pn *paxosNode) GetValue(args *paxosrpc.GetValueArgs, reply *paxosrpc.GetValueReply) error {
	pn.storageLock.Lock()
	defer pn.storageLock.Unlock()

	v, exist := pn.storage[args.Key]
	if exist {
		reply.Status = paxosrpc.KeyFound
		reply.V = v
	} else {
		reply.Status = paxosrpc.KeyNotFound
	}

	return nil
}

// Desc:
// Receive a Prepare message from another Paxos Node. The message contains
// the key whose value is being proposed by the node sending the prepare
// message. This function should respond with Status OK if the prepare is
// accepted and Reject otherwise.
//
// Params:
// args: the Prepare Message
// reply: the Prepare Reply Message
func (pn *paxosNode) RecvPrepare(args *paxosrpc.PrepareArgs, reply *paxosrpc.PrepareReply) error {
	key := args.Key
	n := args.N
	pn.npMapLock.Lock()
	defer pn.npMapLock.Unlock()

	if np, exist := pn.npMap[key]; !exist || n > np {
		pn.npMap[key] = n
		reply.Status = paxosrpc.OK

		if v, exist2 := pn.acceptedMap[key]; exist2 {
			reply.N_a = v.nk
			reply.V_a = v.value
		} else {
			reply.N_a = -1
			reply.V_a = nil
		}
	} else {
		reply.Status = paxosrpc.Reject
	}

	return nil
}

// Desc:
// Receive an Accept message from another Paxos Node. The message contains
// the key whose value is being proposed by the node sending the accept
// message. This function should respond with Status OK if the prepare is
// accepted and Reject otherwise.
//
// Params:
// args: the Please Accept Message
// reply: the Accept Reply Message
func (pn *paxosNode) RecvAccept(args *paxosrpc.AcceptArgs, reply *paxosrpc.AcceptReply) error {
	key := args.Key
	n := args.N
	v := args.V
	pn.npMapLock.Lock()
	defer pn.npMapLock.Unlock()

	if np, exist := pn.npMap[key]; !exist || n >= np {
		pn.acceptedMap[key] = &acceptedValue{
			nk:    n,
			value: v,
		}
		reply.Status = paxosrpc.OK
	} else {
		reply.Status = paxosrpc.Reject
	}

	return nil
}

// Desc:
// Receive a Commit message from another Paxos Node. The message contains
// the key whose value was proposed by the node sending the commit
// message.
//
// Params:
// args: the Commit Message
// reply: the Commit Reply Message
func (pn *paxosNode) RecvCommit(args *paxosrpc.CommitArgs, reply *paxosrpc.CommitReply) error {
	key := args.Key
	v := args.V
	pn.storageLock.Lock()
	defer pn.storageLock.Unlock()

	pn.storage[key] = v

	// reset accepted <n, value> pair, but leave np as it is!
	pn.npMapLock.Lock()
	delete(pn.acceptedMap, key)
	pn.npMapLock.Unlock()
	return nil
}

// Desc:
// Notify another node of a replacement server which has started up. The
// message contains the Server ID of the node being replaced, and the
// hostport of the replacement node
//
// Params:
// args: the id and the hostport of the server being replaced
// reply: no use
func (pn *paxosNode) RecvReplaceServer(args *paxosrpc.ReplaceServerArgs, reply *paxosrpc.ReplaceServerReply) error {
	srvId := args.SrvID
	replaceHostPort := args.Hostport

	attemptNum := 0
	for ; attemptNum < 5; attemptNum++ {
		node, err := rpc.DialHTTP("tcp", replaceHostPort)
		if err != nil {
			time.Sleep(1 * time.Second)
		} else {
			pn.nodes[srvId] = node
			break
		}
	}
	if attemptNum == 5 {
		return errors.New("Can't connect to node " + replaceHostPort)
	}
	return nil
}

// Desc:
// Request the value that was agreed upon for a particular round. A node
// receiving this message should reply with the data (as an array of bytes)
// needed to make the replacement server aware of the keys and values
// committed so far.
//
// Params:
// args: no use
// reply: a byte array containing necessary data used by replacement server to recover
func (pn *paxosNode) RecvReplaceCatchup(args *paxosrpc.ReplaceCatchupArgs, reply *paxosrpc.ReplaceCatchupReply) error {
	pn.storageLock.Lock()
	defer pn.storageLock.Unlock()

	b, err := json.Marshal(pn.storage)
	if err != nil {
		return err
	}
	reply.Data = b

	return nil
}
