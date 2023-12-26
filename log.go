package jsn_raft

type JsnLog struct {
	JIndex uint64
	JTerm  uint64
	JData  []byte
}

func (l *JsnLog) Index() uint64 {
	return l.JIndex
}

func (l *JsnLog) Term() uint64 {
	return l.JTerm
}

func (r *RaftNew) lastLog() (lastLogIndex, lastLogTerm uint64) {
	if 0 == len(r.logs) {
		return
	}
	log := r.logs[len(r.logs)-1]
	return log.Index(), log.Term()
}

func (r *RaftNew) lastLogIndex() uint64 {
	index, _ := r.lastLog()
	return index
}

func (r *RaftNew) matchLog(lastLogIndex, lastLogTerm uint64) bool {
	length := len(r.logs)
	if 0 == length {
		return false
	}
	for ; 0 != length; length-- {
		log := r.logs[length-1]
		if log.Index() > lastLogIndex {
			continue
		} else if log.Index() < lastLogIndex {
			break
		}
		return lastLogTerm == log.Term()
	}
	return false
}

func (r *RaftNew) getLog(logIndex uint64) *JsnLog {
	for _, v := range r.logs {
		if v.Index() < logIndex {
			continue
		} else if v.Index() > logIndex {
			break
		}
		return v
	}
	return nil
}

func (r *RaftNew) logEntries(fromIndex, toIndex uint64) []*JsnLog {
	var ret []*JsnLog
	for _, v := range r.logs {
		if v.Index() >= fromIndex && v.Index() <= toIndex {
			ret = append(ret, v)
		}
	}
	return ret
}

func (r *RaftNew) logDeleteFrom(logIndex uint64) {
	for i, v := range r.logs {
		if v.Index() < logIndex {
			continue
		} else if v.Index() > logIndex {
			break
		}
		r.logs = r.logs[:i+1]
		break
	}
}

func (r *RaftNew) appendLog(log *JsnLog) {
	lastIndex, _ := r.lastLog()
	log.JIndex = lastIndex + 1
	log.JTerm = r.getCurrentTerm()
	r.logs = append(r.logs, log)
}

func (r *RaftNew) logStore(entries []*JsnLog) {
	r.logs = append(r.logs, entries...)
}
