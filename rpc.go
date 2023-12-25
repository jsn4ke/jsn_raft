package jsn_raft

import (
	"errors"
	"net"
	"net/rpc"
	"time"
)

func (r *RaftNew) Vote(args *VoteRequest, reply *VoteResponse) error {
	wrap := &rpcWrap{
		In:   args,
		Resp: reply,
		Done: make(chan error),
	}
	r.rpcChannel <- wrap
	err := <-wrap.Done
	return err
}

func (r *RaftNew) AppendEntries(args *AppendEntriesRequest, reply *AppendEntriesResponse) error {
	wrap := &rpcWrap{
		In:   args,
		Resp: reply,
		Done: make(chan error),
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

func (r *RaftNew) rpcCall(who string, service string, in, out any) error {
	// todo
	dial, err := net.Dial("tcp", who)
	if nil != err {
		r.logger.Error("[%v] get handlerRpc error from %v err %v",
			r.who, who, err)
		return nil
	}
	cli := rpc.NewClient(dial)
	if nil == cli {
		return errors.New("handlerRpc connect fail")
	}
	defer cli.Close()
	select {
	case <-time.After(r.rpcTimeout()):
		return errors.New("handlerRpc timeout")
	case call := <-cli.Go(service, in, out, nil).Done:
		return call.Error
	}
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

	if 0 != len(r.voteFor) && r.voteFor == string(args.CandidateId) {
		reply.VoteGranted = true
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
	r.voteFor = string(args.CandidateId)
	r.logger.Debug("[%v] vote for %v term %v",
		r.who, string(args.CandidateId), r.getCurrentTerm())
	return nil

}

func (r *RaftNew) appendEntries(args *AppendEntriesRequest, reply *AppendEntriesResponse) error {

	reply.CurrentTerm = r.getCurrentTerm()
	reply.Success = false

	if args.Term < r.getCurrentTerm() {
		return nil
	} else if args.Term > r.getCurrentTerm() {

		r.setServerState(follower)
		r.setCurrentTerm(args.Term)

		reply.CurrentTerm = r.getCurrentTerm()
	}

	if !r.matchLog(args.PrevLogIndex, args.PrevLogTerm) {
		return nil
	}

	lastLogIndex, _ := r.lastLog()

	var entries []JLog
	for i, entry := range args.Entries {
		if entry.Index() > lastLogIndex {
			continue
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
