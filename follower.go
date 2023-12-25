package jsn_raft

import "time"

func (r *RaftNew) runFollower() {

	timeout := time.After(randomTimeout(r.heartbeatTimeout()))

	for r.getServerState() == follower {
		select {
		case wrap := <-r.rpcChannel:
			r.handlerRpc(wrap)
			timeout = time.After(randomTimeout(r.heartbeatTimeout()))
		case <-timeout:
			r.setServerState(candidate)
			return
		}
	}
}
