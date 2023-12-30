package jsn_raft

import (
	"fmt"
	"time"
)

func (r *RaftNew) runFollower() {

	timeout := time.NewTimer(randomTimeout(r.heartbeatTimeout()))
	if r.firstFollower {
		timeout.Reset(0)
		r.firstFollower = false
	}
	var (
		last = time.Now()
	)

	for r.getServerState() == follower {
		select {
		case idx := <-r.outputLog:
			s := fmt.Sprintf("CheckLog[%v] %v logs:", idx, r.who)
			for _, v := range r.logs {
				s += fmt.Sprintf("%v-%v-%s|", v.Index(), v.Term(), v.JData)
			}
			logcheck <- s
		case wrap := <-r.rpcChannel:
			r.handlerRpc(wrap)
			timeout.Reset(randomTimeout(r.heartbeatTimeout()))
			last = time.Now()
		case <-timeout.C:
			r.logger.Debug("[%v] follower timeout %v", r.who, time.Since(last))

			r.setServerState(candidate)
			return
		}
	}
}
