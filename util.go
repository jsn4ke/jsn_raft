package jsn_raft

import (
	"math/rand"
	"time"
)

func randomTimeout(interval time.Duration) time.Duration {
	return interval + time.Duration(rand.Int63())%interval
}

func notifyChan(ch chan<- struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}
