package jsn_raft

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	jsn_rpc "github.com/jsn4ke/jsn_net/rpc"
)

const (
	AppendEntries = "RaftNew.AppendEntries"
	Vote          = "RaftNew.Vote"
)

var (
	_ jsn_rpc.RpcUnit = (*VoteRequest)(nil)
	_ jsn_rpc.RpcUnit = (*VoteResponse)(nil)
	_ jsn_rpc.RpcUnit = (*AppendEntriesRequest)(nil)
	_ jsn_rpc.RpcUnit = (*AppendEntriesResponse)(nil)
)

// term uint64, candidateId []byte, lastLogIndex uint64, lastLogTerm uint64
type VoteRequest struct {
	Term         uint64 `json:"term,omitempty"`
	CandidateId  []byte `json:"candidate_id,omitempty"`
	LastLogIndex uint64 `json:"last_log_index,omitempty"`
	LastLogTerm  uint64 `json:"last_log_term,omitempty"`
}

// CmdId implements jsn_rpc.RpcUnit.
func (*VoteRequest) CmdId() uint32 {
	return 1
}

// Marshal implements jsn_rpc.RpcUnit.
func (r *VoteRequest) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

// New implements jsn_rpc.RpcUnit.
func (*VoteRequest) New() jsn_rpc.RpcUnit {
	return new(VoteRequest)
}

// Unmarshal implements jsn_rpc.RpcUnit.
func (r *VoteRequest) Unmarshal(in []byte) error {
	return json.Unmarshal(in, r)
}

type VoteResponse struct {
	CurrentTerm uint64 `json:"current_term,omitempty"`
	VoteGranted bool   `json:"vote_granted,omitempty"`
}

// CmdId implements jsn_rpc.RpcUnit.
func (*VoteResponse) CmdId() uint32 {
	return 2
}

// Marshal implements jsn_rpc.RpcUnit.
func (r *VoteResponse) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

// New implements jsn_rpc.RpcUnit.
func (*VoteResponse) New() jsn_rpc.RpcUnit {
	return new(VoteResponse)
}

// Unmarshal implements jsn_rpc.RpcUnit.
func (r *VoteResponse) Unmarshal(in []byte) error {
	return json.Unmarshal(in, r)
}

// term uint64, leaderId []byte, prevLogIndex uint64, prevLogTerm uint64, entries []*jsnLog, leaderCommit uint64
type AppendEntriesRequest struct {
	Term         uint64    `json:"term,omitempty"`
	LeaderId     []byte    `json:"leader_id,omitempty"`
	PrevLogIndex uint64    `json:"prev_log_index,omitempty"`
	PrevLogTerm  uint64    `json:"prev_log_term,omitempty"`
	Entries      []*JsnLog `json:"entries,omitempty"`
	LeaderCommit uint64    `json:"leader_commit,omitempty"`
	Heartbeat    bool      `json:"heartbeat,omitempty"`
}

// CmdId implements jsn_rpc.RpcUnit.
func (*AppendEntriesRequest) CmdId() uint32 {
	return 3
}

// Marshal implements jsn_rpc.RpcUnit.
func (r *AppendEntriesRequest) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

// New implements jsn_rpc.RpcUnit.
func (*AppendEntriesRequest) New() jsn_rpc.RpcUnit {
	return new(AppendEntriesRequest)
}

// Unmarshal implements jsn_rpc.RpcUnit.
func (r *AppendEntriesRequest) Unmarshal(in []byte) error {
	return json.Unmarshal(in, r)
}

type AppendEntriesResponse struct {
	CurrentTerm uint64 `json:"current_term,omitempty"`
	Success     bool   `json:"success,omitempty"`
}

// CmdId implements jsn_rpc.RpcUnit.
func (*AppendEntriesResponse) CmdId() uint32 {
	return 4
}

// Marshal implements jsn_rpc.RpcUnit.
func (r *AppendEntriesResponse) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

// New implements jsn_rpc.RpcUnit.
func (*AppendEntriesResponse) New() jsn_rpc.RpcUnit {
	return new(AppendEntriesResponse)
}

// Unmarshal implements jsn_rpc.RpcUnit.
func (r *AppendEntriesResponse) Unmarshal(in []byte) error {
	return json.Unmarshal(in, r)
}

func (r *RaftNew) registerRpc() {
	r.rpcServer.RegisterExecutor(new(VoteRequest), r.rpcVote)
	r.rpcServer.RegisterExecutor(new(AppendEntriesRequest), r.rpcAppendEntries)
}

func (r *RaftNew) rpcVote(in jsn_rpc.RpcUnit) (jsn_rpc.RpcUnit, error) {

	args, _ := in.(*VoteRequest)
	if nil == args {
		return nil, errors.New("invalid request")
	}
	now := time.Now()
	defer func() {
		diff := time.Since(now)
		if diff*5 > r.rpcTimeout() {
			r.logger.Info("[%v] rpcVote from %s  cost %v",
				r.who, args.CandidateId, diff)
		}

	}()
	reply := new(VoteResponse)
	err := r.Vote(args, reply)
	return reply, err
}

func (r *RaftNew) rpcAppendEntries(in jsn_rpc.RpcUnit) (jsn_rpc.RpcUnit, error) {
	args, _ := in.(*AppendEntriesRequest)
	if nil == args {
		return nil, errors.New("invalid request")
	}
	now := time.Now()
	defer func() {
		diff := time.Since(now)
		if diff*5 > r.rpcTimeout() {
			r.logger.Info("[%v] rpcAppendEntries from %s  cost %v",
				r.who, args.LeaderId, diff)
		}

	}()
	reply := new(AppendEntriesResponse)
	err := r.AppendEntries(args, reply)
	return reply, err
}

func (r *RaftNew) Vote(args *VoteRequest, reply *VoteResponse) error {
	wrap := &rpcWrap{
		In:   args,
		Resp: reply,
		Done: make(chan error, 1),
	}
	r.rpcChannel <- wrap
	err := <-wrap.Done
	return err
}

func (r *RaftNew) AppendEntries(args *AppendEntriesRequest, reply *AppendEntriesResponse) error {
	wrap := &rpcWrap{
		In:   args,
		Resp: reply,
		Done: make(chan error, 1),
	}
	r.rpcChannel <- wrap
	err := <-wrap.Done
	return err
}

type rpcWrap struct {
	In   any
	Resp any
	Done chan error
}

func (r *RaftNew) handlerRpc(wrap *rpcWrap) {
	var err error
	switch tp := wrap.In.(type) {
	case *VoteRequest:
		err = r.vote(tp, wrap.Resp.(*VoteResponse))
	case *AppendEntriesRequest:
		err = r.appendEntries(tp, wrap.Resp.(*AppendEntriesResponse))
	}
	wrap.Done <- err
}

var (
	clis = sync.Map{}
)

func (r *RaftNew) rpcCall(who string, args jsn_rpc.RpcUnit, reply jsn_rpc.RpcUnit, done <-chan struct{}, timeout time.Duration) error {
	err := r.rpcClients[who].Call(args, reply, done, timeout)
	// err := rpcCall(who, args, reply, done, timeout)
	return err
}

func (r *RaftNew) vote(args *VoteRequest, reply *VoteResponse) error {
	reply.CurrentTerm = r.getCurrentTerm()
	reply.VoteGranted = false

	if args.Term < r.getCurrentTerm() {
		return nil
	} else if args.Term > r.getCurrentTerm() {

		r.setCurrentTerm(args.Term)
		r.setServerState(follower)

		reply.CurrentTerm = r.getCurrentTerm()
	}

	if r.voteForTerm == args.Term {
		if 0 != len(r.voteFor) && r.voteFor != string(args.CandidateId) {
			return nil
		}
	} else if r.voteForTerm > args.Term {
		return nil
	}

	lastLogIndex, lastLogTerm := r.lastLog()
	if lastLogTerm > args.LastLogTerm {
		return nil
	}
	if lastLogTerm == args.LastLogTerm && lastLogIndex > args.LastLogIndex {
		return nil
	}
	reply.VoteGranted = true

	r.setVoteFor(string(args.CandidateId))

	r.logger.Debug("[%v] vote for %v term %v",
		r.who, string(args.CandidateId), r.getCurrentTerm())
	return nil

}

func (r *RaftNew) appendEntries(args *AppendEntriesRequest, reply *AppendEntriesResponse) error {

	reply.CurrentTerm = r.getCurrentTerm()
	reply.Success = false

	defer func() {
		if !args.Heartbeat {

			r.logger.Debug("[%v] append entries from %s commit %v res %v",
				r.who, args.LeaderId, r.getCommitIndex(), reply.Success)
		}

	}()

	if args.Term < r.getCurrentTerm() {
		return nil
	} else if args.Term > r.getCurrentTerm() {

		r.setServerState(follower)
		r.setCurrentTerm(args.Term)

		reply.CurrentTerm = r.getCurrentTerm()
	}
	if candidate == r.getServerState() {
		r.setServerState(follower)
	}

	if args.Heartbeat {
		return nil
	}

	if 0 != args.PrevLogIndex {
		if !r.matchLog(args.PrevLogIndex, args.PrevLogTerm) {
			return nil
		}
	}

	lastLogIndex, _ := r.lastLog()

	var entries []*JsnLog
	for i, entry := range args.Entries {
		if entry.Index() > lastLogIndex {
			entries = args.Entries[i:]
			break
		}
		jlog := r.getLog(entry.Index())
		if nil == jlog {
			return nil
		}
		if jlog.Term() != entry.Term() {
			r.logDeleteFrom(entry.Index())
			entries = args.Entries[i:]
		}
	}

	if 0 != len(entries) {
		r.logStore(entries)
	}
	if args.LeaderCommit > r.getCommitIndex() {
		lastLogIndex, _ = r.lastLog()
		if lastLogIndex < args.LeaderCommit {
			r.setCommitIndex(lastLogIndex)
		} else {
			r.setCommitIndex(args.LeaderCommit)
		}
	}
	reply.Success = true

	return nil
}
