package jsn_raft

import (
	"time"

	jsn_rpc "github.com/jsn4ke/jsn_net/rpc"
)

var (
	logcheck = make(chan struct {
		idx  int32
		body string
	}, 128)
)
var (
	rafts = map[string]*RaftNew{}
)

func rpcCall(who string, args jsn_rpc.RpcUnit, reply jsn_rpc.RpcUnit, done <-chan struct{}, timeout time.Duration) error {
	r := rafts[who]
	wrap := &rpcWrap{
		In:   args,
		Resp: reply,
		Done: make(chan error, 1),
	}
	r.rpcChannel <- wrap
	select {
	case <-time.After(timeout):
		return jsn_rpc.TimeoutError
	case <-done:
		return jsn_rpc.MaualError
	case err := <-wrap.Done:
		return err
	}
}
