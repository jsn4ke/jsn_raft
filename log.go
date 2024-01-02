package jsn_raft

import (
	"fmt"
	"sync"

	"github.com/jsn4ke/jsn_raft/v2/pb"
)

func (r *Raft) lastLog() (lastLogIndex int64, lastLogTerm uint64) {
	return r.logStore.Last()
}

func (r *Raft) lastLogIndex() int64 {
	index, _ := r.lastLog()
	return index
}

func (r *Raft) matchLog(logIndex int64, logTerm uint64) bool {
	return r.logStore.Match(logIndex, logTerm)
}

func (r *Raft) getLog(logIndex int64) *pb.JsnLog {
	return r.logStore.Entry(logIndex)
}

func (r *Raft) logEntries(fromIndex, toIndex int64) []*pb.JsnLog {
	var res []*pb.JsnLog
	for i := fromIndex; i <= toIndex; i++ {
		jlog := r.logStore.Entry(i)
		if nil == jlog {
			r.logger.Error("[%v] miss log %v",
				r.who, i)
			return nil
		}
		res = append(res, jlog)
	}
	return res
}

func (r *Raft) logTruncate(logIndex int64) {
	r.logStore.Truncate(logIndex)
}

func (r *Raft) appendLog(log *pb.JsnLog) {
	lastIndex, _ := r.lastLog()
	log.Index = lastIndex + 1
	log.Term = r.getCurrentTerm()
	r.logStore.AppendEntries([]*pb.JsnLog{log})
}

func (r *Raft) appendLogs(entries []*pb.JsnLog) {
	r.logStore.AppendEntries(entries)
}

type memoryStore struct {
	startIndex int64

	lastIndex int64
	lastTerm  uint64

	rw sync.RWMutex

	logs map[int64]*pb.JsnLog
}

func (s *memoryStore) StartIndex() int64 {
	s.rw.RLock()
	defer s.rw.RUnlock()
	return s.startIndex
}

func (s *memoryStore) Last() (index int64, term uint64) {
	s.rw.RLock()
	defer s.rw.RUnlock()
	return s.lastIndex, s.lastTerm
}

func (s *memoryStore) Match(index int64, term uint64) bool {
	s.rw.RLock()
	defer s.rw.RUnlock()

	log, ok := s.logs[index]
	if !ok {
		return false
	}
	return log.Term == term
}

func (s *memoryStore) Entry(index int64) *pb.JsnLog {
	s.rw.RLock()
	defer s.rw.RUnlock()

	return s.logs[index]
}

func (s *memoryStore) AppendEntries(entries []*pb.JsnLog) {
	length := len(entries)
	if 0 == length {
		return
	}

	s.rw.Lock()
	defer s.rw.Unlock()

	if nil == s.logs {
		s.logs = map[int64]*pb.JsnLog{}
	}

	var (
		index int64
	)
	for _, v := range entries {
		index = v.Index
		s.logs[index] = v
		if 0 == s.startIndex {
			s.startIndex = index
		}

	}

	if index > s.lastIndex {
		s.lastIndex = index
		s.lastTerm = entries[length-1].Term
	}
}

func (s *memoryStore) Truncate(min int64) error {
	s.rw.Lock()
	defer s.rw.Unlock()

	max := s.lastIndex
	for i := min; i <= max; i++ {
		delete(s.logs, i)
	}

	if min <= s.startIndex {
		s.startIndex = max + 1
	}

	if max >= s.lastIndex {
		s.lastIndex = min - 1
	}

	for s.lastIndex >= s.startIndex {
		lastLog := s.logs[s.lastIndex]
		if nil == lastLog {
			s.lastIndex--
			fmt.Printf("==========miss index %v\n", s.lastIndex)
		} else {
			s.lastTerm = lastLog.Term
			break
		}
	}
	if s.lastIndex < s.startIndex {
		s.startIndex = 0
		s.lastIndex = 0
		s.lastTerm = 0
	}

	return nil
}
