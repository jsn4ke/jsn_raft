package jsn_raft

import (
	_ "embed"
	"fmt"
	"net/http"
	"testing"
	"time"

	_ "net/http/pprof"

	"gopkg.in/yaml.v3"
)

//go:embed test_server_list.yml
var serverConfig []byte

func TestNewRaftNew(t *testing.T) {
	go func() {
		http.ListenAndServe("0.0.0.0:7777", nil)
	}()
	config := new(ServerConfig)
	err := yaml.Unmarshal(serverConfig, config)
	if nil != err {
		panic(err)
	}
	var rs []*RaftNew
	for _, v := range config.List {
		r := NewRaftNew(v.Who, *config)
		rs = append(rs, r)
	}
	tk1 := time.NewTicker(time.Second)
	tk2 := time.NewTicker(time.Millisecond * 600)
	transfer := time.NewTimer(randomTimeout(time.Second * 10))
	go func() {
		for in := range logcheck {
			fmt.Println(in)
		}
	}()
	var idx int64
	for {
		select {
		case <-transfer.C:
			// for _, v := range rs {
			// 	if v.getServerState() == leader {
			// 		fmt.Println("trigger leader to follower")
			// 		select {
			// 		case v.leaderTransfer <- struct{}{}:
			// 		default:
			// 		}

			// 	}
			// }
			transfer.Reset(randomTimeout(time.Second * 10))
		case <-tk1.C:
			idx++
			// for _, v := range rs {
			// 	v.outputLog <- idx
			// }
		case <-tk2.C:
			for _, v := range rs {
				if v.getServerState() == leader {
					v.logModify <- &JsnLog{
						JData: []byte(fmt.Sprintf("%v", time.Now().Unix())),
					}
				}
			}
		}

	}
}
