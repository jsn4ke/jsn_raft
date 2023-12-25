package jsn_raft

type followerProcess struct {
	who string

	//nextIndex  []uint64
	//matchIndex []uint64
	nextIndex  uint64
	matchIndex uint64

	notify    <-chan struct{}
	heartbeat <-chan struct{}
	stop      <-chan struct{}
	usurper   chan<- struct{}
}

func (r *RaftNew) heartbeat(p *followerProcess) (done bool) {
	req := &AppendEntriesRequest{
		Term:         r.getCurrentTerm(),
		LeaderId:     []byte(r.who),
		PrevLogIndex: 0,
		PrevLogTerm:  0,
		Entries:      nil,
		LeaderCommit: 0,
	}

	resp := new(AppendEntriesResponse)
	err := r.rpcCall(p.who, AppendEntries, req, resp)
	if nil != err {
		r.logger.Error("[%v] heartbeat to %v in append entries err %v",
			r.who, p.who, err.Error())
		return
	}
	if resp.CurrentTerm > r.getCurrentTerm() {
		r.setServerState(follower)
		r.setCurrentTerm(resp.CurrentTerm)
		r.notifyChan(p.usurper)
		return true
	}
	return
}

func (r *RaftNew) replicate(p *followerProcess, nextLogIndex uint64) (done bool) {
	req := &AppendEntriesRequest{
		Term:         r.getCurrentTerm(),
		LeaderId:     []byte(r.who),
		PrevLogIndex: 0,
		PrevLogTerm:  0,
		Entries:      nil,
		LeaderCommit: 0,
	}
	jlog := r.getLog(nextLogIndex)
	if nil == jlog {
		return
	}

	req.PrevLogIndex = jlog.Index()
	req.PrevLogTerm = jlog.Term()

	lastLogIndex, _ := r.lastLog()

	req.Entries = r.logEntries(nextLogIndex, lastLogIndex)

	resp := new(AppendEntriesResponse)

	err := r.rpcCall(p.who, AppendEntries, req, resp)
	if nil != err {
		r.logger.Error("[%v] replicate to %v err %v",
			r.who, p.who, err.Error())
		return
	}

	if resp.CurrentTerm > r.getCurrentTerm() {
		r.setServerState(follower)
		r.setCurrentTerm(resp.CurrentTerm)
		r.notifyChan(p.usurper)
		return true
	}
	if resp.Success {
		p.nextIndex = lastLogIndex + 1
	}
	return
}

func (r *RaftNew) runReplicate(p *followerProcess) {
	var done bool
	for !done {
		select {

		case <-p.heartbeat:
			done = r.heartbeat(p)
		case <-p.notify:
			lastLogIndex, _ := r.lastLog()
			done = r.replicate(p, lastLogIndex)
		case <-p.stop:
			return
		}
	}
}
