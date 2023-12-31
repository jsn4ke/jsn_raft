package jsn_raft

import (
	"time"

	"github.com/jsn4ke/jsn_raft/v2/pb"
)

func (r *Raft) runCandidate() {
	r.logger.Info("[%v] run candidate",
		r.who)
	r.addCurrentTerm()

	lastLogIndex, lastLogTerm := r.lastLog()
	req := &pb.VoteRequest{
		Term:         r.getCurrentTerm(),
		CandidateId:  []byte(r.who),
		LastLogIndex: lastLogIndex,
		LastLogTerm:  lastLogTerm,
	}
	var (
		needVotes    = len(r.config.List)/2 + 1
		grantedVotes = 0
	)

	voteResponseChannel := make(chan *pb.VoteResponse, len(r.config.List)-1)
	for _, v := range r.config.List {
		who := v.Who
		if who == r.who {
			grantedVotes++
			r.setVoteFor(who)
		} else {
			r.safeGo("peer vote request", func() {
				resp := &pb.VoteResponse{}
				// err := r.rpcClients[who].Call(req, resp, nil, r.rpcTimeout())
				err := r.rpcCall(who, req, resp, nil, r.rpcTimeout())
				if nil != err {
					r.logger.Error("`[%v] vote to %v err %v",
						r.who, who, err.Error())
				}
				voteResponseChannel <- resp
			})
		}
	}

	electionTimer := time.After(randomTimeout(r.electionTimeout()))

	for r.getServerState() == candidate {
		select {
		case wrap := <-r.rpcChannel:
			r.handlerRpc(wrap)
		case resp := <-voteResponseChannel:
			if resp.CurrentTerm > r.getCurrentTerm() {
				r.setServerState(follower)
				r.setCurrentTerm(resp.CurrentTerm)
				return
			}
			if resp.VoteGranted {
				grantedVotes++
				r.logger.Debug("[%v] receive vote, current votes %v need %v",
					r.who, grantedVotes, needVotes)
			}
			if grantedVotes >= needVotes {
				r.setServerState(leader)
				r.voteFor = r.who
			}
		case <-electionTimer:
			// 重新开始投票
			r.logger.Debug("[%v] current votes %v need %v timeout",
				r.who, grantedVotes, needVotes)
			return
		}
	}

}
