package jsn_raft

import "time"

func (r *RaftNew) runCandidate() {
	r.logger.Info("[%v] in candidate",
		r.who)
	r.addCurrentTerm()

	r.voteFor = r.who
	lastLogIndex, lastLogTerm := r.lastLog()
	req := &VoteRequest{
		Term:         r.getCurrentTerm(),
		CandidateId:  []byte(r.who),
		LastLogIndex: lastLogIndex,
		LastLogTerm:  lastLogTerm,
	}
	var (
		needVotes    = len(r.config.List)/2 + 1
		grantedVotes = 0
	)

	voteResponseChannel := make(chan *VoteResponse, len(r.config.List)-1)
	for _, v := range r.config.List {
		who := v.Who
		if who == r.who {
			grantedVotes++
			r.voteFor = r.who
		} else {
			r.safeGo("peer vote request", func() {
				resp := &VoteResponse{}
				err := r.rpcCall(who, Vote, req, resp)
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
			}
		case <-electionTimer:
			// 重新开始投票
			r.logger.Debug("[%v] current votes %v need %v timeout",
				r.who, grantedVotes, needVotes)
			return
		}
	}

}
