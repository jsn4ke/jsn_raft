package jsn_raft

import "time"

func (r *RaftNew) runFollower() {

	timeout := time.NewTimer(randomTimeout(r.heartbeatTimeout()))
	if r.firstFollower {
		timeout.Reset(0)
		r.firstFollower = false
	}

	for r.getServerState() == follower {
		select {
		case wrap := <-r.rpcChannel:
			r.handlerRpc(wrap)
			timeout.Reset(randomTimeout(r.heartbeatTimeout()))
		case <-timeout.C:
			r.setServerState(candidate)
			return
		}
	}
}
