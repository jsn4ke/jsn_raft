package jsn_raft

import (
	"fmt"
	"time"

	"github.com/jsn4ke/jsn_net"
)

func (r *RaftNew) runFollower() {
	r.logger.Info("[%v] run follower", r.who)

	timeout := time.NewTimer(randomTimeout(r.heartbeatTimeout()))
	// if r.firstFollower {
	// 	timeout.Reset(0)
	// 	r.firstFollower = false
	// }
	var (
		last = time.Now()
	)

	for r.getServerState() == follower {
		select {
		case idx := <-r.outputLog:
			s := fmt.Sprintf("CheckLog[%v] logs:", idx)
			var st = jsn_net.Clip(len(r.logs)-20, 0, len(r.logs))
			for _, v := range r.logs[st:] {
				s += fmt.Sprintf("%v-%v-%s|", v.Index(), v.Term(), v.JData)
			}
			logcheck <- struct {
				idx  int32
				body string
			}{int32(idx), s}
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
