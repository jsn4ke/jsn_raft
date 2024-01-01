package jsn_raft

import (
	"errors"
	"sync"
	"time"

	jsn_rpc "github.com/jsn4ke/jsn_net/rpc"
	"github.com/jsn4ke/jsn_raft/v2/pb"
)

const (
	AppendEntries = "RaftNew.AppendEntries"
	Vote          = "RaftNew.Vote"
)

func (r *Raft) registerRpc() {
	r.rpcServer.RegisterExecutor(new(pb.VoteRequest), r.rpcVote)
	r.rpcServer.RegisterExecutor(new(pb.AppendEntriesRequest), r.rpcAppendEntries)
}

func (r *Raft) rpcVote(in jsn_rpc.RpcUnit) (jsn_rpc.RpcUnit, error) {

	args, _ := in.(*pb.VoteRequest)
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
	reply := new(pb.VoteResponse)
	err := r.Vote(args, reply)
	return reply, err
}

func (r *Raft) rpcAppendEntries(in jsn_rpc.RpcUnit) (jsn_rpc.RpcUnit, error) {
	args, _ := in.(*pb.AppendEntriesRequest)
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
	reply := new(pb.AppendEntriesResponse)
	err := r.AppendEntries(args, reply)
	return reply, err
}

func (r *Raft) Vote(args *pb.VoteRequest, reply *pb.VoteResponse) error {
	wrap := &rpcWrap{
		In:   args,
		Resp: reply,
		Done: make(chan error, 1),
	}
	r.rpcChannel <- wrap
	err := <-wrap.Done
	return err
}

func (r *Raft) AppendEntries(args *pb.AppendEntriesRequest, reply *pb.AppendEntriesResponse) error {
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

func (r *Raft) handlerRpc(wrap *rpcWrap) {
	var err error
	switch tp := wrap.In.(type) {
	case *pb.VoteRequest:
		err = r.vote(tp, wrap.Resp.(*pb.VoteResponse))
	case *pb.AppendEntriesRequest:
		err = r.appendEntries(tp, wrap.Resp.(*pb.AppendEntriesResponse))
	}
	wrap.Done <- err
}

var (
	clis = sync.Map{}
)

func (r *Raft) rpcCall(who string, args jsn_rpc.RpcUnit, reply jsn_rpc.RpcUnit, done <-chan struct{}, timeout time.Duration) error {
	err := r.rpcClients[who].Call(args, reply, done, timeout)
	// err := rpcCall(who, args, reply, done, timeout)
	return err
}

func (r *Raft) vote(args *pb.VoteRequest, reply *pb.VoteResponse) error {
	reply.CurrentTerm = r.getCurrentTerm()
	reply.VoteGranted = false

	if args.Term < r.getCurrentTerm() {
		return nil
	} else if args.Term > r.getCurrentTerm() {

		r.setCurrentTerm(args.Term)
		r.setServerState(follower)

		reply.CurrentTerm = r.getCurrentTerm()
	}

	// 一个服务器最多会对一个任期号投出一张选票，按照先来先服务的原则
	if r.voteForTerm == args.Term {
		if 0 != len(r.voteFor) && r.voteFor != string(args.CandidateId) {
			return nil
		}
	} else if r.voteForTerm > args.Term {
		return nil
	}

	lastLogIndex, lastLogTerm := r.lastLog()
	if lastLogTerm > args.LastLogTerm {
		r.logger.Info("[%v] vote candiate last log term %v mine %v",
			r.who, args.LastLogTerm, lastLogTerm)
		return nil
	}
	if lastLogTerm == args.LastLogTerm && lastLogIndex > args.LastLogIndex {
		r.logger.Info("[%v] vote candiate last log index,term [%v,%v] mine [%v,%v]",
			r.who, args.LastLogTerm, args.LastLogIndex, args.LastLogTerm, lastLogIndex, lastLogTerm)
		return nil
	}
	reply.VoteGranted = true

	r.setVoteFor(string(args.CandidateId))

	r.logger.Debug("[%v] vote for %v term %v",
		r.who, string(args.CandidateId), r.getCurrentTerm())
	r.lastFollowerCheck = time.Now()
	return nil

}

func (r *Raft) appendEntries(args *pb.AppendEntriesRequest, reply *pb.AppendEntriesResponse) error {

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

	// 如果这个领导人的任期号（包含在此次的 RPC中）不小于候选人当前的任期号，那么候选人会承认领导人合法并回到跟随者状态
	if candidate == r.getServerState() {
		r.setServerState(follower)
	}

	if args.Heartbeat {
		r.lastFollowerCheck = time.Now()
		return nil
	}

	if 0 != args.PrevLogIndex {
		if !r.matchLog(args.PrevLogIndex, args.PrevLogTerm) {
			return nil
		}
	}

	lastLogIndex, _ := r.lastLog()

	var entries []*pb.JsnLog
	for i, entry := range args.Entries {
		if entry.Index > lastLogIndex {
			entries = args.Entries[i:]
			break
		}
		jlog := r.getLog(entry.Index)
		if nil == jlog {
			return nil
		}
		if jlog.Term != entry.Term {
			r.logTruncate(entry.Index)
			entries = args.Entries[i:]
			break
		}
	}

	if 0 != len(entries) {
		r.appendLogs(entries)
	}
	if args.LeaderCommitIndex > r.getCommitIndex() {
		lastLogIndex, _ = r.lastLog()
		if lastLogIndex < args.LeaderCommitIndex {
			r.setCommitIndex(lastLogIndex)
		} else {
			r.setCommitIndex(args.LeaderCommitIndex)
		}
	}
	reply.Success = true
	r.lastFollowerCheck = time.Now()
	return nil
}
