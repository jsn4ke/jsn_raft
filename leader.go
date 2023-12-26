package jsn_raft

import (
	"fmt"
	"sync"
)

func (r *RaftNew) runLeader() {
	r.logger.Info("[%v] in leader",
		r.who)

	var (
		// orphanTimeout = time.After(randomTimeout(r.orphanTimeout()))
		lastLogIndex = r.lastLogIndex()
		// 通知所有replication结束
		replicationDone = make(chan struct{})
		// 通知leader任期结束
		usurper = make(chan uint64, 1)
		// 通知leader更新自身的commit index
		commitmentNotify = make(chan struct{}, 1)
		// 通知所有的replication有新的数据拉取
		fetchList []chan struct{}
	)

	// 生成leader 易失性数据 next 和 match
	commitment := &commitment{
		rw:               sync.RWMutex{},
		notify:           commitmentNotify,
		who:              make(map[string]int),
		nextIndex:        []uint64{},
		matchIndex:       []uint64{},
		commitIndex:      0,
		leaderStartIndex: lastLogIndex,
	}

	// 初始化 commitment
	for _, v := range r.config.List {
		commitment.who[v.Who] = len(commitment.nextIndex)
		commitment.nextIndex = append(commitment.nextIndex, lastLogIndex+1)
		commitment.matchIndex = append(commitment.matchIndex, 0)
	}
	// 提交leader的首条日志
	r.appendLog(&JsnLog{
		JData: []byte(fmt.Sprintf("leader commit log term %v", r.getCurrentTerm())),
	})

	lastLogIndex = r.lastLogIndex()
	commitment.updateIndex(r.who, lastLogIndex+1, lastLogIndex)

	// 生成同步逻辑
	for _, v := range r.config.List {
		if v.Who == r.who {
			continue
		}
		fetch := make(chan struct{}, 1)
		fetchList = append(fetchList, fetch)
		rpl := newReplication(r, commitment, v.Who, replicationDone, usurper, fetch)
		r.safeGo("replication", func() {
			rpl.run()
		})
	}
	// 创建完毕，执行第一次同步
	for _, v := range fetchList {
		notifyChan(v)
	}

	defer func() {
		// 通知所有replication结束
		close(replicationDone)
	}()

	// 清空下
	select {
	case <-r.leaderTransfer:
	default:
	}

	// 开始leader的逻辑
	for leader == r.getServerState() {
		select {
		case term := <-usurper:
			if term > r.getCurrentTerm() {
				r.setServerState(follower)
				r.setCurrentTerm(term)
			}
		case <-commitmentNotify:

			lastCommitIndex := r.getCommitIndex()

			newCommitIndex := commitment.getCommitmentIndex()
			r.setCommitIndex(newCommitIndex)

			if newCommitIndex > lastCommitIndex {
				// todo 日志落地

			}

		case jlog := <-r.logModify:
			r.appendLog(jlog)
			for _, v := range fetchList {
				notifyChan(v)
			}

		case <-r.leaderTransfer:
			r.setServerState(follower)
		}
	}
}
