// DO NOT MODIFY!

package proxyrpc

import (
	"fmt"
	"strings"
)

// Status represents the status of a RPC's reply
type Status int
type Lookup int
type MsgName int

const (
	OK     Status = iota + 1 // Paxos replied OK
	Reject                   // Paxos rejected the message
)

const (
	KeyFound    Lookup = iota + 1 // GetValue key found
	KeyNotFound                   // GetValue key not found
)

const (
	Prepare MsgName = iota + 1
	Accept
	Commit
	PhaseDone
)

type Event struct {
	Msg          MsgName
	SrcNodeID    int
	TargetNodeID int
	N            int
	Status       Status
	Key          string
	Value        interface{}
	IsResponse   bool
}

func getRealKey(key string) string {
	return strings.Split(key, ":")[0]
}

func (e Event) ToString() string {
	var res = ""
	if e.IsResponse {
		res += fmt.Sprintf("node %d response node %d: ", e.SrcNodeID, e.TargetNodeID)
	} else {
		res += fmt.Sprintf("node %d received from ", e.TargetNodeID)
	}
	switch e.Msg {
	case Prepare:
		if e.IsResponse {
			if e.Status == OK {
				res += fmt.Sprintf("Promise with <nk=%d, ak=%v>", e.N, e.Value)
			} else {
				res += "Reject Prepare"
			}
		} else {
			res += fmt.Sprintf("node %d: Prepare <key=%s, n=%d>", e.SrcNodeID, getRealKey(e.Key), e.N)
		}
	case Accept:
		if e.IsResponse {
			if e.Status == OK {
				res += "Accept_OK"
			} else {
				res += "Reject Accept"
			}
		} else {
			res += fmt.Sprintf("node %d: Accept <key=%s, v=%v, n=%d>", e.SrcNodeID, getRealKey(e.Key), e.Value, e.N)
		}
	case Commit:
		if e.IsResponse {
			res += "Commit_OK"
		} else {
			res += fmt.Sprintf("node %d: Commit <key=%s, v=%v>", e.SrcNodeID, getRealKey(e.Key), e.Value)
		}
	}
	return res
}

type ProposalNumberArgs struct {
	Key string
}

type ProposalNumberReply struct {
	N int
}

type ProposeArgs struct {
	N   int // Proposal number
	Key string
	V   interface{} // Value for the Key
}

type ProposeReply struct {
	V interface{} // Value that was actually committed for that key
}

type GetValueArgs struct {
	Key string
}

type GetValueReply struct {
	V      interface{}
	Status Lookup
}

type PrepareArgs struct {
	Key         string
	N           int
	RequesterId int
}

type PrepareReply struct {
	Status Status
	N_a    int         // Highest proposal number accepted
	V_a    interface{} // Corresponding value
}

type AcceptArgs struct {
	Key         string
	N           int
	V           interface{}
	RequesterId int
}

type AcceptReply struct {
	Status Status
}

type CommitArgs struct {
	Key         string
	V           interface{}
	RequesterId int
}

type CommitReply struct {
	// No content, no reply necessary
}

type ReplaceServerArgs struct {
	SrvID    int // Server being replaced
	Hostport string
}

type ReplaceServerReply struct {
	// No content necessary
}

type ReplaceCatchupArgs struct {
	// No content necessary
}

type ReplaceCatchupReply struct {
	Data []byte
}

type CheckEventsArgs struct {
	TestKey        string
	ExpectedEvents []Event
}

type CheckEventsReply struct {
	Success bool
	ErrMsg  string
}
