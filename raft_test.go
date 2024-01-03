package jsn_raft

import (
	_ "embed"
	"fmt"
	"math/rand"
	"net/http"
	"testing"
	"time"
	"unsafe"

	_ "net/http/pprof"

	"github.com/jsn4ke/jsn_raft/v2/pb"
	"google.golang.org/protobuf/proto"
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
				fmt.Printf("====[%v][%v][%v,%v,%v] [%X] \n",
					v.getServerState(), v.who,
					idx, term, v.getCommitIndex(), v.fsm.Md5())
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
					cmd := &pb.LogCmd{
						Op:       pb.LogOperation(1 + rand.Int31()%2),
						Behavior: nil,
					}
				re:
					if rand.Int31()%2 == 1 {
						cmd.Behavior = &pb.LogCmd_Update{
							Update: &pb.LogCmdKV{
								Key:   uint64(rand.Int31()),
								Value: uint64(rand.Int31()),
							},
						}
					} else {
						v.fsm.rw.RLock()
						for k := range v.fsm.KVStore.Data {
							cmd.Behavior = &pb.LogCmd_Delete{
								Delete: &pb.LogCmdDelete{
									Key: k,
								},
							}
							break
						}
						v.fsm.rw.RUnlock()
					}
					if nil == cmd.Behavior {
						goto re
					}
					body, err := proto.Marshal(cmd)
					if nil != err {
						panic(err)
					}
					v.logModify <- &pb.JsnLog{
						Cmd: body,
					}
				}
			}
		}

	}
}

func TestRange(t *testing.T) {
	type a struct {
		val int
	}
	var b = []a{
		{
			val: 1,
		},
		{
			val: 2,
		},
	}
	for _, v := range b {
		fmt.Println(unsafe.Pointer(&v))
	}
}

func TestRange2(t *testing.T) {
	type a struct {
		val int
	}
	var b = []*a{
		{
			val: 1,
		},
		{
			val: 2,
		},
	}
	var nw []*a
	for _, v := range b {
		fmt.Println(unsafe.Pointer(&v))
		go func() {
			time.Sleep(time.Second)
			fmt.Println(v)
		}()
	}
	for _, v := range nw {
		fmt.Println(v)
	}
	time.Sleep(time.Second * 3)
}
