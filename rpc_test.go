package jsn_raft

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestChanClose(t *testing.T) {
	var (
		wg    = sync.WaitGroup{}
		count = 5
	)

	wg.Add(count)
	ch := make(chan struct{})
	fn := func(in <-chan struct{}) {
		defer wg.Done()
		select {
		case <-in:
			return
		}
	}

	for i := 0; i < count; i++ {
		go fn(ch)
	}
	<-time.After(time.Second * 3)
	fmt.Println("sleep over start close")
	close(ch)
	wg.Wait()
	fmt.Println("success")
}

func TestChanDoubleClose(t *testing.T) {
	var (
		count = 5
	)

	ch := make(chan struct{})
	fn := func(in <-chan struct{}) {
		for {

			select {
			case <-in:
				fmt.Println("receive")
			}
		}
	}

	for i := 0; i < count; i++ {
		go fn(ch)
	}
	<-time.After(time.Second * 3)
	fmt.Println("sleep over start close")
	close(ch)
	ch = make(chan struct{})
	<-time.After(time.Second * 3)
	fmt.Println("sleep over start close 2")
	close(ch)
	<-time.After(time.Second * 3)
}
