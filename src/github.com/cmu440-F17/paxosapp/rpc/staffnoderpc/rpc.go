package staffnoderpc

// Staff nodes used in a ring with student nodes to force a particular
// sequence of messages. Staff nodes must be able to be wrapped both as
// a RemoteStaffNode and as a RemotePaxosNode (it must implement the
// union of the functions in these interfaces).
type RemoteStaffNode interface {
	SetProposalNums(args *SetNumsArgs, reply *SetNumsReply) error
	RunPaxosRound(args *RunPaxosArgs, reply *RunPaxosReply) error
	SetDropType(args *SetDropArgs, reply *SetDropReply) error
	RecvPhaseDone(args *PhaseDoneArgs, reply *PhaseDoneReply) error
}

type StaffNode struct {
	RemoteStaffNode
}

// Wrap wraps t in a type-safe wrapper struct to ensure that
// only the desired methods are exported to receive RPCs.
func Wrap(t RemoteStaffNode) RemoteStaffNode {
	return &StaffNode{t}
}
