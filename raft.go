package jsn_raft

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/rpc"
	"sync"
	"time"
)

const (
	follower int32 = iota
	candidate
	leader
)

var (
	hbTimeout         = time.Second * 3
	electionTimoutMin = time.Second * 5
	electionTimoutMax = time.Second * 10

	rpcTimeout = time.Millisecond * 50
)

var (
	globalWg      = sync.WaitGroup{}
	clusterNum    = 5
	AppendEntries = "RaftNew.AppendEntries"
	Vote          = "RaftNew.Vote"
)

type Raft struct {
	who  []byte
	addr string

	persistentState
	volatileState
	leaderVolatileState

	goroutineWaitGroup sync.WaitGroup

	log JLogger

	serverState int32

	timeout *time.Timer

	others     map[string]*rpc.Client
	otherIndex map[string]int

	loopIn  chan struct{}
	loopOut chan struct{}
}

func NewRaft(addr string, who string, cluster []string) *Raft {
	r := new(Raft)
	r.who = []byte(who)
	r.addr = addr
	r.log = new(defaultLogger)

	r.loopIn = make(chan struct{})
	r.loopOut = make(chan struct{})

	ls, err := net.Listen("tcp", addr)
	if nil != err {
		panic(err)
	} else {
		r.log.Info("handlerRpc listen in %v", addr)
	}
	svr := rpc.NewServer()
	if err := svr.RegisterName("AppendEntries", r); nil != err {
		r.log.Panic(err.Error())
	}
	if err := svr.RegisterName("Vote", r); nil != err {
		r.log.Panic(err.Error())
	}

	go svr.Accept(ls)
	cnt := 0
	r.otherIndex = map[string]int{}
	r.others = map[string]*rpc.Client{}
	for _, v := range cluster {
		v := v
		if v == addr {
			continue
		}
		r.otherIndex[addr] = cnt
		cnt++
		for {
			conn, err := net.Dial("tcp", v)
			if nil != err {
				time.Sleep(time.Second)
			} else {
				r.others[v] = rpc.NewClient(conn)
				break
			}
		}
	}
	r.matchIndex = nil
	r.nextIndex = nil

	globalWg.Done()
	globalWg.Wait()

	r.safeGo("main loop", r.loop)
	return r
}

// term uint64, leaderId []byte, prevLogIndex uint64, prevLogTerm uint64, entries []JLog, leaderCommit uint64
type AppendEntriesRequest struct {
	Term         uint64
	LeaderId     []byte
	PrevLogIndex uint64
	PrevLogTerm  uint64
	Entries      []JLog
	LeaderCommit uint64
}

type AppendEntriesResponse struct {
	CurrentTerm uint64
	Success     bool
}

func (r *Raft) AppendEntries(args *AppendEntriesRequest, reply *AppendEntriesResponse) error {
	r.loopIn <- struct{}{}
	reply.CurrentTerm, reply.Success = r.appendEntries(args.Term, args.LeaderId, args.PrevLogIndex, args.PrevLogTerm, args.Entries, args.LeaderCommit)
	rpcInfo(args, reply, r.log, fmt.Sprintf("who:%v handlerRpc AppendEntries ", r.addr))
	<-r.loopOut
	return nil
}

// term uint64, candidateId []byte, lastLogIndex uint64, lastLogTerm uint64
type VoteRequest struct {
	Term         uint64 `json:"term,omitempty"`
	CandidateId  []byte `json:"candidate_id,omitempty"`
	LastLogIndex uint64 `json:"last_log_index,omitempty"`
	LastLogTerm  uint64 `json:"last_log_term,omitempty"`
}
type VoteResponse struct {
	CurrentTerm uint64 `json:"current_term,omitempty"`
	VoteGranted bool   `json:"vote_granted,omitempty"`
}

func (r *Raft) Vote(args *VoteRequest, reply *VoteResponse) error {
	r.loopIn <- struct{}{}
	reply.CurrentTerm, reply.VoteGranted = r.vote(args.Term, args.CandidateId, args.LastLogIndex, args.LastLogTerm)
	rpcInfo(args, reply, r.log, fmt.Sprintf("who:%v handlerRpc Vote", r.addr))
	<-r.loopOut
	return nil
}

func rpcInfo[A, B any](a *A, b *B, log JLogger, before string) {
	sa, _ := json.Marshal(a)
	sb, _ := json.Marshal(b)
	log.Info("%v request:%v response:%v", before, sa, sb)
}

func (r *Raft) loop() {
	r.timeout = time.NewTimer(hbTimeout)
	for {
		select {

		case <-r.loopIn:

			r.loopOut <- struct{}{}
			r.timeout.Reset(hbTimeout)

		case <-r.timeout.C:
			r.log.Info("who[%v] state %v", r.addr, r.serverState)
			switch r.serverState {
			case follower:
				r.serverState = candidate
				r.timeout.Reset(0)
			case candidate:
				r.currentTerm++
				r.votedFor = r.who
				votedCnt := 1
				r.timeout.Reset(time.Duration(rand.Int()%int(electionTimoutMax-electionTimoutMin)) + electionTimoutMin)
				lastLogIndex, lastLogTerm := r.persistentState.lastLog()
				vote := &VoteRequest{
					Term:         r.currentTerm,
					CandidateId:  r.who,
					LastLogIndex: lastLogIndex,
					LastLogTerm:  lastLogTerm,
				}
				respCh := make(chan *VoteResponse, len(r.others))
				for _, v := range r.others {
					v := v
					r.safeGo("vote handlerRpc", func() {
						resp := new(VoteResponse)
						select {
						case err := <-v.Go(Vote, vote, resp, nil).Done:
							if nil == err {
								respCh <- resp
								return
							}
						case <-time.After(rpcTimeout):
							r.log.Info("who:%v VoteResponse timeout")
						}
						respCh <- nil
					})
				}
				var recv = len(r.others)
				for recv != 0 {
					select {
					case resp := <-respCh:
						if nil != resp {
							r.follow(resp.CurrentTerm)
							if r.serverState == candidate && resp.VoteGranted {
								votedCnt++
							}
						}
					}
					recv--
				}
				if r.serverState == candidate && (votedCnt<<1) > len(r.others)+1 {
					r.serverState = leader
					r.log.Info("%v leader", r.addr)
					r.timeout.Reset(0)
					lastLogIndex, lastLogTerm = r.persistentState.lastLog()
					r.nextIndex = nil
					r.matchIndex = nil
					for range r.others {
						r.nextIndex = append(r.nextIndex, lastLogIndex+1)
						r.matchIndex = append(r.matchIndex, 0)
					}
				}
			case leader:
				r.timeout.Reset(hbTimeout)
				req := &AppendEntriesRequest{
					Term:         r.currentTerm,
					LeaderId:     r.who,
					PrevLogIndex: 0,
					PrevLogTerm:  0,
					Entries:      nil,
					LeaderCommit: 0,
				}
				respCh := make(chan *AppendEntriesResponse, len(r.others))
				for _, v := range r.others {
					v := v
					r.safeGo("vote handlerRpc", func() {
						resp := new(AppendEntriesResponse)
						select {
						case err := <-v.Go(AppendEntries, req, resp, nil).Done:
							if nil == err {
								respCh <- resp
								return
							}
						case <-time.After(rpcTimeout):
						}
						respCh <- nil
					})
				}
				var recv = len(r.others)
				for recv != 0 {
					select {
					case resp := <-respCh:
						if nil != resp {
							r.follow(resp.CurrentTerm)
						}
					}
					recv--
				}
			}
		}
	}
}

func (r *Raft) safeGo(name string, f func()) {
	r.goroutineWaitGroup.Add(1)
	go func() {
		defer func() {
			if err := recover(); nil != err {
				r.log.Error("safe go panic %v recover %v", name, err)
			}
			r.goroutineWaitGroup.Done()
		}()
		f()
	}()
}

func (r *Raft) fsmRun() {
	switch r.serverState {
	case follower:

	case candidate:
		r.persistentState.currentTerm++
		r.persistentState.votedFor = r.who
	case leader:

	}
}

func (r *Raft) askVote() {}

func (r *Raft) rpcHandler() {

}

func (r *Raft) appendEntries(term uint64, leaderId []byte, prevLogIndex uint64, prevLogTerm uint64, entries []JLog, leaderCommit uint64) (currentTerm uint64, success bool) {

	r.follow(term)
	currentTerm = r.currentTerm

	if term < r.currentTerm {
		return
	}
	if r.persistentState.matchLog(prevLogIndex, prevLogTerm) {
		return
	}
	lastLogIndex, _ := r.persistentState.lastLog()
	var newEntries []JLog
	for i, entry := range entries {
		if entry.Index() > lastLogIndex {
			newEntries = entries[i:]
			break
		}
		jlog := r.persistentState.getLog(entry.Index())
		if nil == jlog {
			return
		}
		if entry.Term() != jlog.Term() {
			r.persistentState.deleteLog(entry.Index(), lastLogIndex)
			newEntries = entries[i:]
			break
		}
	}
	if length := len(newEntries); 0 != length {
		r.persistentState.storeLog(newEntries)
	}

	if leaderCommit > 0 && leaderCommit > r.commitIndex {
		lastLogIndex, _ = r.persistentState.lastLog()

		if leaderCommit > lastLogIndex {
			r.commitIndex = lastLogIndex
		} else {
			r.commitIndex = leaderCommit
		}
	}

	if r.commitIndex > r.lastApplied {
		// todo
		// 如果commitIndex > lastApplied，则 lastApplied 递增，并将log[lastApplied]应用到状态机中
		r.lastApplied = r.commitIndex
	}

	success = true
	return
}

func (r *Raft) vote(term uint64, candidateId []byte, lastLogIndex uint64, lastLogTerm uint64) (currentTerm uint64, voteGranted bool) {

	r.follow(term)

	if term < r.persistentState.currentTerm {
		return r.persistentState.currentTerm, false
	}
	votedFor := r.persistentState.votedFor
	currentLastLogIndex, currentLastLogTerm := r.persistentState.lastLog()
	if (0 == len(votedFor) || checkSame(votedFor, candidateId)) &&
		(lastLogTerm > currentLastLogTerm) ||
		(lastLogTerm == currentLastLogTerm && lastLogIndex >= currentLastLogIndex) {

		r.persistentState.votedFor = candidateId

		return r.persistentState.currentTerm, true
	}
	return r.persistentState.currentTerm, false
}

func (r *Raft) follow(term uint64) bool {
	if term > r.persistentState.currentTerm {
		r.persistentState.currentTerm = term
		r.serverState = follower
		r.persistentState.votedFor = nil
		return true
	}
	return false
}

func checkSame[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
