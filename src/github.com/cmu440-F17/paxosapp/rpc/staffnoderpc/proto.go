// DO NOT MODIFY!

package staffnoderpc

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
	Key        string
	Value      interface{}
	Msg        MsgName
	NodeID     int
	Targets    []int // Do not set this field, testing with specific targets doesn't work
	IsResponse bool
}

type RunPaxosArgs struct {
	Events []Event
}

type RunPaxosReply struct {
	Err string
}

type SetNumsArgs struct {
	IDToPNum map[int]int // Map from node ID to proposal number
}

type SetNumsReply struct {
	Err string
}

type SetDropArgs struct {
	Drop int
}

type SetDropReply struct {
	Err string
}

type PhaseDoneArgs struct {
}

type PhaseDoneReply struct {
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
	Key string
	N   int
}

type PrepareReply struct {
	Status Status
	N_a    int         // Highest proposal number accepted
	V_a    interface{} // Corresponding value
}

type AcceptArgs struct {
	Key string
	N   int
	V   interface{}
}

type AcceptReply struct {
	Status Status
}

type CommitArgs struct {
	Key string
	V   interface{}
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
