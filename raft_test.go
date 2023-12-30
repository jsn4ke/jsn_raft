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

		rafts[v.Who] = r
	}
	for _, v := range rs {
		v.Go()
	}
	tk1 := time.NewTicker(time.Second)
	tk2 := time.NewTicker(time.Second)
	transfer := time.NewTicker(randomTimeout(time.Millisecond * 300))
	var (
		idx  int64
		same = map[int32]string{}
	)
	go func() {
		for in := range logcheck {
			if s, ok := same[in.idx]; ok {
				if s == in.body {
					continue
				} else if s == "output" {
					fmt.Println(in.body)
				} else {
					fmt.Println(s)
					fmt.Println(in.body)
					same[in.idx] = "output"
				}
			} else {
				same[in.idx] = in.body
			}
		}
	}()

	for {
		select {
		case <-transfer.C:
			for _, v := range rs {
				if v.getServerState() == leader {
					fmt.Println("trigger leader to follower")
					select {
					case v.leaderTransfer <- struct{}{}:
					default:
					}

				}
			}
		case <-tk1.C:
			idx++
			for _, v := range rs {
				v.outputLog <- idx
			}
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
