package jsn_raft

import (
	"crypto/md5"
	"sort"
	"sync"

	"github.com/jsn4ke/jsn_raft/v2/pb"
	"google.golang.org/protobuf/proto"
)

type (
	Fsm struct {
		rw sync.RWMutex
		pb.KVStore
	}
	LogOp struct {
		Op pb.LogOperation
	}
)

func (f *Fsm) Md5() [md5.Size]byte {
	f.rw.RLock()
	defer f.rw.RUnlock()
	arr := new(pb.HelpArray)
	for k := range f.Data {
		arr.Data = append(arr.Data, k)
	}
	sort.Slice(arr.Data, func(i, j int) bool {
		return arr.Data[i] < arr.Data[j]
	})
	body1, _ := proto.Marshal(arr)
	for i, v := range arr.Data {
		arr.Data[i] = f.GetData()[v]
	}
	body2, _ := proto.Marshal(arr)
	return md5.Sum(append(body1, body2...))
}

func (f *Fsm) Apply(jlog *pb.JsnLog) {
	f.rw.Lock()
	defer f.rw.Unlock()
	msg := new(pb.LogCmd)
	proto.Unmarshal(jlog.GetCmd(), msg)
	switch msg.GetOp() {
	case pb.LogOperation_LogOperation_Fsm:
		switch tp := msg.GetBehavior().(type) {
		case *pb.LogCmd_Update:
			update := tp.Update
			if nil == f.Data {
				f.Data = map[uint64]uint64{}
			}
			f.Data[update.GetKey()] = update.GetValue()
		case *pb.LogCmd_Delete:
			del := tp.Delete
			delete(f.Data, del.GetKey())
		}
	}
}

func (r *Raft) applyLog(fromIndex, toIndex int64) {
	for idx := fromIndex + 1; idx <= toIndex; idx++ {
		jlog := r.getLog(idx)
		if nil == jlog {
			panic("no log")
		}
		r.fsm.Apply(jlog)
	}
}
