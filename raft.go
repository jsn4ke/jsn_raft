package jsn_raft

import (
	"sync"
	"sync/atomic"

	jsn_rpc "github.com/jsn4ke/jsn_net/rpc"
)

type RaftNew struct {
	firstFollower bool

	who string

	currentTerm uint64

	serverState int32

	commitIndex uint64

	logs []*JsnLog

	logger JLogger

	voteFor     string
	voteForTerm uint64

	config ServerConfig

	goroutineWaitGroup sync.WaitGroup

	rpcChannel chan *rpcWrap

	// rpc
	rpcServer  *jsn_rpc.Server
	rpcClients map[string]*jsn_rpc.Client

	// debug
	logModify      chan *JsnLog
	leaderTransfer chan struct{}
	outputLog      chan int64
}

func NewRaftNew(who string, config ServerConfig) *RaftNew {
	r := new(RaftNew)

	r.firstFollower = true

	r.who = who
	r.config = config
	r.logger = new(defaultLogger)
	r.rpcChannel = make(chan *rpcWrap, 256)

	// debug//////
	r.logModify = make(chan *JsnLog, 256)
	r.leaderTransfer = make(chan struct{})
	r.outputLog = make(chan int64, 1)
	//////////////

	//// rpc //////
	r.rpcServer = jsn_rpc.NewServer(r.who, 128, 4)
	r.registerRpc()
	r.rpcServer.Start()
	r.rpcClients = make(map[string]*jsn_rpc.Client)
	for _, v := range config.List {
		if r.who == v.Who {
			continue
		}
		cli := jsn_rpc.NewClient(v.Who, 128, 4)
		r.rpcClients[v.Who] = cli
	}
	///////////////

	//////////
	return r
}

func (r *RaftNew) Go() {
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

func (r *RaftNew) safeGo(name string, f func()) {
	r.goroutineWaitGroup.Add(1)
	go func() {
		defer func() {
			// if err := recover(); nil != err {
			// 	r.logger.Error("safe go panic %v recover %v", name, err)
			// }
			r.goroutineWaitGroup.Done()
		}()
		f()
	}()
}

func (r *RaftNew) getServerState() int32 {
	return atomic.LoadInt32(&r.serverState)
}

func (r *RaftNew) getCurrentTerm() uint64 {
	return r.currentTerm
}

func (r *RaftNew) setServerState(state int32) {
	old := atomic.SwapInt32(&r.serverState, state)
	r.logger.Info("[%v] state from %v to %v ",
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

func (r *RaftNew) setVoteFor(who string) {
	r.voteFor = who
	r.voteForTerm = r.getCurrentTerm()
}
