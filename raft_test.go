package jsn_raft

import (
	_ "embed"
	"fmt"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

//go:embed test_server_list.yml
var serverConfig []byte

func TestNewRaftNew(t *testing.T) {
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
	tk := time.NewTicker(time.Second)
	for {
		select {
		case <-tk.C:
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
