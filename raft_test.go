package jsn_raft

import (
	_ "embed"
	"fmt"
	"net/http"
	"testing"
	"time"

	_ "net/http/pprof"

	"github.com/jsn4ke/jsn_raft/v2/pb"
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
	var rs []*Raft
	for _, v := range config.List {
		r := NewRaftNew(v.Who, *config)
		rs = append(rs, r)

		rafts[v.Who] = r
	}
	for _, v := range rs {
		v.Go()
	}
	// tk2 := time.NewTicker(time.Second / 100000)
	transfer := time.NewTicker(randomTimeout(time.Millisecond * 3000))

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
		default:
			for _, v := range rs {
				if v.getServerState() == leader {
					v.logModify <- &pb.JsnLog{
						Cmd: []byte(fmt.Sprintf("%v", time.Now().Unix())),
					}
				}
			}
		}

	}
}
