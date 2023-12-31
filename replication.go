package jsn_raft

import (
	"time"

	"github.com/jsn4ke/jsn_raft/v2/pb"
)

func newReplication(raft *Raft, commit *commitment, who string,
	done <-chan struct{}, usurper chan<- uint64, fetch <-chan struct{}) *replication {
	r := new(replication)

	r.who = who
	r.raft = raft
	r.commitment = commit

	r.done = done
	r.usurper = usurper
	r.fetch = fetch

	r.retry = make(chan struct{})
	return r
}

type replication struct {
	who string

	raft       *Raft
	commitment *commitment

	fetch   <-chan struct{}
	done    <-chan struct{}
	usurper chan<- uint64

	retry chan struct{}
}

func (r *replication) heartbeat() {
	req := &pb.AppendEntriesRequest{
		Term:              r.raft.getCurrentTerm(),
		LeaderId:          []byte(r.raft.who),
		PrevLogIndex:      0,
		PrevLogTerm:       0,
		Entries:           []*pb.JsnLog{},
		LeaderCommitIndex: r.raft.getCommitIndex(),
		Heartbeat:         true,
	}
	resp := new(pb.AppendEntriesResponse)
	// err := r.raft.rpcClients[r.who].Call(req, resp, r.done, r.raft.rpcTimeout())
	err := r.raft.rpcCall(r.who, req, resp, r.done, r.raft.rpcTimeout())
	if nil != err {
		r.raft.logger.Error("[%v] heartbeat to %v error %v",
			r.raft.who, r.who, err.Error())
		return
	}
	if resp.CurrentTerm > r.raft.getCurrentTerm() {
		select {
		case r.usurper <- resp.CurrentTerm:
		default:
		}
		return
	}
}

func (r *replication) replicateTo() {
	req := &pb.AppendEntriesRequest{
		Term:              r.raft.getCurrentTerm(),
		LeaderId:          []byte(r.raft.who),
		PrevLogIndex:      0,
		PrevLogTerm:       0,
		Entries:           []*pb.JsnLog{},
		LeaderCommitIndex: 0,
	}

	nextIndex := r.commitment.getNextIndex(r.who)

	if 1 < nextIndex {
		jlog := r.raft.getLog(nextIndex - 1)
		if nil == jlog {
			r.raft.logger.Error("[%v] replicate to %v no log %v in raft",
				r.raft.who, r.who, nextIndex-1)
			return
		}
		req.PrevLogIndex = jlog.Index
		req.PrevLogTerm = jlog.Term
	}

	req.LeaderCommitIndex = r.raft.getCommitIndex()

	lastLogIndex, _ := r.raft.lastLog()

	req.Entries = r.raft.logEntries(nextIndex, lastLogIndex)

	resp := new(pb.AppendEntriesResponse)
	// err := r.raft.rpcClients[r.who].Call(req, resp, r.done, r.raft.rpcTimeout())
	err := r.raft.rpcCall(r.who, req, resp, r.done, r.raft.rpcTimeout())

	if nil != err {
		notifyChan(r.retry)
		return
	}
	if resp.CurrentTerm > r.raft.getCurrentTerm() {
		select {
		case r.usurper <- resp.CurrentTerm:
		default:
		}
		return
	}
	if resp.Success {
		if 0 == len(req.Entries) {
			return
		}
		index := req.Entries[len(req.Entries)-1].Index
		r.commitment.updateIndex(r.who, index+1, index)

		r.raft.logger.Info("[%v] replicate to %v success next index modify %v",
			r.raft.who, r.who, r.commitment.getNextIndex(r.who))
	} else {
		r.commitment.stepMinusNextIndex(r.who)
		r.raft.logger.Info("[%v] replicate to %v failure next index modify %v",
			r.raft.who, r.who, r.commitment.getNextIndex(r.who))
		notifyChan(r.retry)
		return
	}
}

func (r *replication) run() {
	tk := time.NewTimer(randomTimeout(r.raft.heartbeatTimeout() / 10))
	for {
		select {
		case <-r.done:
			return
		case <-tk.C:
			r.heartbeat()
			tk.Reset(randomTimeout(r.raft.heartbeatTimeout() / 10))
		case <-r.fetch:
			r.replicateTo()
		case <-r.retry:
			time.Sleep(time.Millisecond * 1)
			r.replicateTo()
		}
	}
}
