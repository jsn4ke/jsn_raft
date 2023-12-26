package jsn_raft

import (
	"sort"
	"sync"
)

// 只有leader可以处理这块数据
// 可以设置为仅在leader时非空
type commitment struct {
	rw sync.RWMutex

	notify chan<- struct{}

	who map[string]int

	nextIndex  []uint64
	matchIndex []uint64

	commitIndex uint64

	// 一个任期一个 term
	leaderStartIndex uint64
}

func (c *commitment) getNextIndex(who string) uint64 {
	c.rw.RLock()
	defer c.rw.RUnlock()

	slot, ok := c.who[who]
	if !ok {
		return 0
	}
	return c.nextIndex[slot]
}

func (c *commitment) getCommitmentIndex() uint64 {
	c.rw.RLock()
	defer c.rw.RUnlock()

	return c.commitIndex
}

func (c *commitment) updateIndex(who string, nextIndex uint64, matchIndex uint64) {
	c.rw.Lock()
	slot, ok := c.who[who]
	if !ok {
		c.rw.Unlock()
		return
	}
	c.nextIndex[slot] = nextIndex
	c.matchIndex[slot] = matchIndex
	c.rw.Unlock()

	c.adapt()
}

func (c *commitment) stepMinusNextIndex(who string) {
	c.rw.Lock()
	defer c.rw.Unlock()

	slot, ok := c.who[who]
	if !ok {
		return
	}
	index := c.nextIndex[slot]
	if 1 >= index {
		return
	}
	c.nextIndex[slot]--
}

func (c *commitment) adapt() {
	c.rw.RLock()
	defer c.rw.RUnlock()

	matches := make([]uint64, len(c.matchIndex))
	copy(matches, c.matchIndex)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i] < matches[j]
	})

	need := len(matches)/2 + 1

	newCommitentIndex := matches[need-1]
	if newCommitentIndex > c.commitIndex && newCommitentIndex > c.leaderStartIndex {
		c.commitIndex = newCommitentIndex
		notifyChan(c.notify)
	}
}
