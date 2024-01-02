package jsn_raft

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	jsn_rpc "github.com/jsn4ke/jsn_net/rpc"
	"github.com/jsn4ke/jsn_raft/v2/pb"
)

const (
	follower int32 = iota
	candidate
	leader
)

type Raft struct {
	firstFollower bool

	who string

	currentTerm uint64

	serverState int32

	commitIndex int64

	lastFollowerCheck time.Time

	logStore *memoryStore

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
	logModify      chan *pb.JsnLog
	leaderTransfer chan struct{}
}

func NewRaftNew(who string, config ServerConfig) *Raft {
	r := new(Raft)

	r.firstFollower = true

	r.who = who
	r.config = config
	r.logger = new(defaultLogger)
	r.rpcChannel = make(chan *rpcWrap, 256)
	r.logStore = new(memoryStore)

	// debug//////
	r.logModify = make(chan *pb.JsnLog, 256)
	r.leaderTransfer = make(chan struct{})
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

func (r *Raft) Go() {
	r.safeGo("fsm", r.fsm)
	r.safeGo("reply", func() {
		dir, err := os.Getwd()
		if nil != err {
			panic(err)
		}
		fmt.Println(dir)
		filename := path.Join(dir, r.who)
		fd, err := os.Create(filename)
		if nil != err {
			panic(err)
		}
		defer fd.Close()
		commited := int64(0)
		tk := time.NewTicker(time.Millisecond * 200)
		for range tk.C {
			for i := 0; i < 200; i++ {
				if commited < r.getCommitIndex() {
					commited++
					jlog := r.getLog(commited)
					if nil == jlog {
						panic("no log")
					}
					fd.WriteString(fmt.Sprintf("index:%v term:%v data:%s\n", jlog.Index, jlog.Term, jlog.Cmd))
				} else {
					break
				}
				runtime.Gosched()
			}
		}
	})
}

func (r *Raft) fsm() {
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

func (r *Raft) safeGo(name string, f func()) {
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

func (r *Raft) getServerState() int32 {
	return atomic.LoadInt32(&r.serverState)
}

func (r *Raft) getCurrentTerm() uint64 {
	return r.currentTerm
}

func (r *Raft) setServerState(state int32) {
	old := atomic.SwapInt32(&r.serverState, state)
	r.logger.Info("[%v] state from %v to %v ",
		r.who, old, state)
}

func (r *Raft) setCurrentTerm(term uint64) {
	old := atomic.SwapUint64(&r.currentTerm, term)
	r.logger.Debug("[%v] term from %v to %v",
		r.who, old, term)
}

func (r *Raft) addCurrentTerm() uint64 {
	ret := atomic.AddUint64(&r.currentTerm, 1)
	r.logger.Debug("[%v] term from %v to %v",
		r.who, ret-1, ret)
	return ret
}

func (r *Raft) getCommitIndex() int64 {
	val := atomic.LoadInt64(&r.commitIndex)
	return val
}

func (r *Raft) setCommitIndex(index int64) {
	old := atomic.SwapInt64(&r.commitIndex, index)
	r.logger.Debug("[%v] commit update from %v to %v",
		r.who, old, index)
}

func (r *Raft) setVoteFor(who string) {
	r.voteFor = who
	r.voteForTerm = r.getCurrentTerm()
}
