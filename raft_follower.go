package jsn_raft

import (
	"math/rand"
	"net"
	"net/rpc"
	"sync"
	"sync/atomic"
	"time"
)

type RaftNew struct {
	who string

	currentTerm uint64

	serverState int32

	commitIndex uint64

	logs []JLog

	logger JLogger

	voteFor string

	config ServerConfig

	goroutineWaitGroup sync.WaitGroup

	rpcChannel chan *rpcWrap

	logModify chan struct{}
}

func NewRaftNew(who string, config ServerConfig) {
	r := new(RaftNew)
	r.who = who
	r.config = config
	r.logger = new(defaultLogger)
	r.rpcChannel = make(chan *rpcWrap, 256)

	svr := rpc.NewServer()
	if err := svr.RegisterName("RaftNew", r); nil != err {
		panic(err)
	}
	if ls, err := net.Listen("tcp", who); nil != err {
		panic(err)
	} else {
		r.safeGo("accept", func() {
			svr.Accept(ls)
		})
	}
	r.safeGo("fsm", r.fsm)
}

func (r *RaftNew) fsm() {
	for {
		switch r.getServerState() {
		case follower:
			r.runFollower()
		case candidate:
			r.runCandidate()
		case leader:
			r.runLeader()

		}
	}
}

func (r *RaftNew) notifyChan(ch chan<- struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}

func (r *RaftNew) safeGo(name string, f func()) {
	r.goroutineWaitGroup.Add(1)
	go func() {
		defer func() {
			if err := recover(); nil != err {
				r.logger.Error("safe go panic %v recover %v", name, err)
			}
			r.goroutineWaitGroup.Done()
		}()
		f()
	}()
}

func randomTimeout(interval time.Duration) time.Duration {
	return interval + time.Duration(rand.Int63())%interval
}

func (r *RaftNew) getServerState() int32 {
	return atomic.LoadInt32(&r.serverState)
}

func (r *RaftNew) getCurrentTerm() uint64 {
	return r.currentTerm
}

func (r *RaftNew) setServerState(state int32) {
	old := atomic.SwapInt32(&r.serverState, state)
	r.logger.Info("[%v] state from %v to %v",
		r.who, old, state)
}

func (r *RaftNew) setCurrentTerm(term uint64) {
	old := atomic.SwapUint64(&r.currentTerm, term)
	r.logger.Debug("[%v] term from %v to %v",
		r.who, old, term)
}

func (r *RaftNew) addCurrentTerm() uint64 {
	ret := atomic.AddUint64(&r.currentTerm, 1)
	r.logger.Debug("[%v] term from %v to %v",
		r.who, ret-1, ret)
	return ret
}

func (r *RaftNew) getCommitIndex() uint64 {
	val := atomic.LoadUint64(&r.commitIndex)
	return val
}

func (r *RaftNew) setCommitIndex(index uint64) {
	old := atomic.SwapUint64(&r.commitIndex, index)
	r.logger.Debug("[%v] commit update from %v to %v",
		r.who, old, index)
}
