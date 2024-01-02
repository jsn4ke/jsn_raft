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
	tk2 := time.NewTicker(time.Second / 10000)
	transfer := time.NewTicker(randomTimeout(time.Millisecond * 3000))

	hb := time.NewTimer(time.Second)
	for {
		select {
		case <-hb.C:
			for _, v := range rs {
				idx, term := v.lastLog()
				fmt.Printf("====[%v][%v][%v,%v,%v] \n",
					v.getServerState(), v.who,
					idx, term, v.getCommitIndex())
			}
			fmt.Println("====")
			hb.Reset(time.Second)
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
		case <-tk2.C:
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
