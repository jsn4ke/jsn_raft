package jsn_raft

import (
	"time"
)

func (r *RaftNew) runLeader() {
	r.logger.Info("[%v] in leader",
		r.who)

	orphanTimeout := time.After(randomTimeout(r.orphanTimeout()))

	lastLogIndex, _ := r.lastLog()
	var (
		notifies []chan<- struct{}
		stop     []chan<- struct{}
		hbs      []chan<- struct{}
		usurper  = make(chan struct{})
	)
	for _, v := range r.config.List {
		if v.Who == r.who {
			continue
		}
		p := new(followerProcess)
		p.who = v.Who
		ch1 := make(chan struct{})
		ch2 := make(chan struct{})
		ch3 := make(chan struct{})
		p.notify = ch1
		p.heartbeat = ch2
		p.stop = ch3
		p.usurper = usurper

		notifies = append(notifies, ch1)
		hbs = append(hbs, ch2)
		stop = append(stop, ch3)

		p.nextIndex = lastLogIndex
		p.matchIndex = 0
		//p.nextIndex = append(p.nextIndex, lastLogIndex)
		//p.matchIndex = append(p.matchIndex, 0)

		r.safeGo("follower sync process", func() {
			r.runReplicate(p)
		})
	}

	defer func() {
		for _, v := range stop {
			r.notifyChan(v)
		}
	}()

	for r.getServerState() == leader {
		select {
		case wrap := <-r.rpcChannel:
			r.handlerRpc(wrap)

		case <-orphanTimeout:

		case <-time.After(r.heartbeatTimeout() / 5):
			for _, hb := range hbs {
				r.notifyChan(hb)
			}

		case <-r.logModify:
			for _, nty := range notifies {
				r.notifyChan(nty)
			}

		case <-usurper:
			return
		}
	}
}
