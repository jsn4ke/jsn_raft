package jsn_raft

type JLog interface {
	SetTerm(term uint64)
	SetIndex(index uint64)
	Term() uint64
	Index() uint64
	Cmd() []byte
}

func (r *RaftNew) lastLog() (lastLogIndex, lastLogTerm uint64) {
	if 0 == len(r.logs) {
		return
	}
	log := r.logs[len(r.logs)-1]
	return log.Index(), log.Term()
}

func (r *RaftNew) matchLog(lastLogIndex, lastLogTerm uint64) bool {
	length := len(r.logs)
	if 0 == length {
		return false
	}
	for 0 != length {
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

func (r *RaftNew) getLog(logIndex uint64) JLog {
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

func (r *RaftNew) logEntries(fromIndex, toIndex uint64) []JLog {
	var ret []JLog
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

func (r *RaftNew) appendLog(log JLog) {
	lastIndex, _ := r.lastLog()
	log.SetIndex(lastIndex + 1)
	log.SetTerm(r.getCurrentTerm())
	r.logs = append(r.logs, log)
}

func (r *RaftNew) logStore(entries []JLog) {
	r.logs = append(r.logs, entries...)
}
