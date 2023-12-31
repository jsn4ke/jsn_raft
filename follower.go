package jsn_raft

import (
	"time"
)

func (r *Raft) runFollower() {
	r.logger.Info("[%v] run follower", r.who)

	timeout := time.NewTimer(randomTimeout(r.heartbeatTimeout()))
	r.lastFollowerCheck = time.Now()

	for r.getServerState() == follower {
		select {

		case wrap := <-r.rpcChannel:
			r.handlerRpc(wrap)
		case <-timeout.C:
			diff := time.Since(r.lastFollowerCheck)

			if diff < r.heartbeatTimeout() {
				timeout.Reset(randomTimeout(r.heartbeatTimeout()))
			} else {
				r.logger.Debug("[%v] follower timeout %v exit follower", r.who, diff)
				r.setServerState(candidate)
				return
			}
		}
	}
}
