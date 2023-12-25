package jsn_raft

import (
	"sort"
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
		match    = make(chan struct {
			who   string
			index uint64
		}, len(r.config.List))
	)
	r.nexted = map[string]uint64{}

	r.matched = map[string]uint64{}

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
		p.match = match

		notifies = append(notifies, ch1)
		hbs = append(hbs, ch2)
		stop = append(stop, ch3)

		p.nextIndex = lastLogIndex
		p.matchIndex = 0

		r.matched[p.who] = 0

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

		case elem := <-match:

			r.matched[elem.who] = elem.index
			var matches []uint64
			for _, v := range r.matched {
				matches = append(matches, v)
			}
			sort.Slice(matches, func(i, j int) bool {
				return matches[i] < matches[j]
			})
			index := matches[(len(matches)-1)/2]

			if index > r.getCommitIndex() {
				r.setCommitIndex(index)
			}

		case <-time.After(r.heartbeatTimeout() / 5):
			for _, hb := range hbs {
				r.notifyChan(hb)
			}

		case jlog := <-r.logModify:
			r.appendLog(jlog)
			for _, nty := range notifies {
				r.notifyChan(nty)
			}

		case <-usurper:
			return
		}
	}
}
