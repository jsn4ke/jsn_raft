package jsn_raft

import (
	_ "embed"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

//go:embed test_server_list.yml
var serverConfig []byte

type testLog struct {
	term  uint64
	index uint64
	tm    int64
}

func (t *testLog) SetTerm(term uint64) {
	t.term = term
}

func (t *testLog) SetIndex(index uint64) {
	t.index = index
}

func (t *testLog) Term() uint64 {
	return t.term
}

func (t *testLog) Index() uint64 {
	return t.index
}

func (t *testLog) Cmd() []byte {
	return nil
}

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
					v.logModify <- &testLog{
						term:  0,
						index: 0,
						tm:    time.Now().Unix(),
					}
				}
			}
		}

	}
}
