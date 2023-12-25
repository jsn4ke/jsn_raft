package jsn_raft

type (
	// 所有服务器上的持久性状态 (在响应 RPC 请求之前，已经更新到了稳定的存储设备)
	persistentState struct {
		// 服务器已知最新的任期（在服务器首次启动时初始化为0，单调递增）
		currentTerm uint64
		// 当前任期内收到选票的 candidateId，如果没有投给任何候选人 则为空
		votedFor []byte
		// 日志条目；每个条目包含了用于状态机的命令，以及领导人接收到该条目时的任期（初始索引为1）
		logs []JLog
	}
	// 所有服务器上的易失性状态
	volatileState struct {
		// 已知已提交的最高的日志条目的索引（初始值为0，单调递增）
		commitIndex uint64
		// 已经被应用到状态机的最高的日志条目的索引（初始值为0，单调递增）
		lastApplied uint64
	}
	// 领导人（服务器）上的易失性状态 (选举后已经重新初始化)
	leaderVolatileState struct {
		// 对于每一台服务器，发送到该服务器的下一个日志条目的索引（初始值为领导人最后的日志条目的索引+1）
		nextIndex []uint64
		// 对于每一台服务器，已知的已经复制到该服务器的最高日志条目的索引（初始值为0，单调递增）
		matchIndex []uint64
	}
)

func (s *persistentState) storeLog([]JLog) {

}

func (s *persistentState) getLog(index uint64) JLog {
	for _, v := range s.logs {
		if v.Index() == index {
			return v
		}
	}
	return nil
}

func (s *persistentState) deleteLog(fromIndex, lasIndex uint64) {}

func (s *persistentState) matchLog(logIndex uint64, logTerm uint64) bool {
	for i := len(s.logs) - 1; i >= 0; i-- {
		if s.logs[i].Index() > logIndex {
			continue
		} else if s.logs[i].Index() < logIndex {
			break
		}
		return s.logs[i].Term() == logTerm
	}
	return false
}

func (s *persistentState) lastLog() (logIndex, logTerm uint64) {
	length := len(s.logs)
	if 0 == length {
		return 0, 0
	}
	return s.logs[length-1].Index(), s.logs[length-1].Term()
}
