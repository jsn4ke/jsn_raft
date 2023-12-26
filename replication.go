package jsn_raft

import "time"

func newReplication(raft *RaftNew, commit *commitment, who string,
	done chan struct{}, usurper <-chan struct{}, fetch chan<- struct{}) *replication {
	r := new(replication)
	r.raft = raft
	r.who = who
	r.done = done
	r.commitment = commit
	r.retry = make(chan struct{})
	return r
}

type replication struct {
	who string

	raft       *RaftNew
	commitment *commitment

	fetch   <-chan struct{}
	done    <-chan struct{}
	usurper chan<- struct{}

	retry chan struct{}
}

func (r *replication) heartbeat() {
	req := &AppendEntriesRequest{
		Term:         r.raft.getCurrentTerm(),
		LeaderId:     []byte(r.raft.who),
		PrevLogIndex: 0,
		PrevLogTerm:  0,
		Entries:      []*JsnLog{},
		LeaderCommit: 0,
	}
	resp := new(AppendEntriesResponse)
	err := r.raft.rpcCallWithDone(r.who, AppendEntries, req, resp, r.done)
	if nil != err {
		r.raft.logger.Error("[%v] heartbeat to %v error %v",
			r.raft.who, r.who, err.Error())
		return
	}
	if resp.CurrentTerm > r.raft.getCurrentTerm() {
		r.raft.setServerState(follower)
		r.raft.setCurrentTerm(resp.CurrentTerm)
		notifyChan(r.usurper)
		return
	}
}

func (r *replication) replicateTo() {
	req := &AppendEntriesRequest{
		Term:         r.raft.getCurrentTerm(),
		LeaderId:     []byte(r.raft.who),
		PrevLogIndex: 0,
		PrevLogTerm:  0,
		Entries:      []*JsnLog{},
		LeaderCommit: 0,
	}

	nextIndex := r.commitment.getNextIndex(r.who)

	jlog := r.raft.getLog(nextIndex)
	if nil == jlog {
		return
	}
	req.PrevLogIndex = jlog.Index()
	req.PrevLogTerm = jlog.Term()

	req.LeaderCommit = r.raft.getCommitIndex()

	lastLogIndex, _ := r.raft.lastLog()

	req.Entries = r.raft.logEntries(nextIndex, lastLogIndex)

	resp := new(AppendEntriesResponse)
	err := r.raft.rpcCallWithDone(r.who, AppendEntries, req, resp, r.done)
	if nil != err {
		notifyChan(r.retry)
		return
	}
	if resp.CurrentTerm > r.raft.getCurrentTerm() {
		r.raft.setServerState(follower)
		r.raft.setCurrentTerm(resp.CurrentTerm)
		notifyChan(r.usurper)
		return
	}
	if resp.Success {
		if 0 == len(req.Entries) {
			return
		}
		index := req.Entries[len(req.Entries)-1].Index()
		r.commitment.updateIndex(r.who, index+1, index)
	} else {
		notifyChan(r.retry)
		return
	}
}

func (r *replication) run() {
	tk := time.NewTimer(randomTimeout(r.raft.heartbeatTimeout() / 5))
	for {
		select {
		case <-r.done:
			return
		case <-tk.C:
			r.heartbeat()
			tk.Reset(randomTimeout(r.raft.heartbeatTimeout() / 5))
		case <-r.fetch:

		}
	}
}
