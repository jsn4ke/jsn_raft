package pb

import (
	jsn_rpc "github.com/jsn4ke/jsn_net/rpc"
	"google.golang.org/protobuf/proto"
)

var (
	_ jsn_rpc.RpcUnit = (*VoteRequest)(nil)
	_ jsn_rpc.RpcUnit = (*VoteResponse)(nil)
	_ jsn_rpc.RpcUnit = (*AppendEntriesRequest)(nil)
	_ jsn_rpc.RpcUnit = (*AppendEntriesResponse)(nil)
)

func (*VoteRequest) CmdId() uint32 {
	return uint32(Cmd_Cmd_VoteRequest)
}
func (x *VoteRequest) Marshal() ([]byte, error) {
	return proto.Marshal(x)
}
func (x *VoteRequest) Unmarshal(in []byte) error {
	return proto.Unmarshal(in, x)
}
func (*VoteRequest) New() jsn_rpc.RpcUnit {
	return new(VoteRequest)
}
func (*VoteResponse) CmdId() uint32 {
	return uint32(Cmd_Cmd_VoteResponse)
}
func (x *VoteResponse) Marshal() ([]byte, error) {
	return proto.Marshal(x)
}
func (x *VoteResponse) Unmarshal(in []byte) error {
	return proto.Unmarshal(in, x)
}
func (*VoteResponse) New() jsn_rpc.RpcUnit {
	return new(VoteResponse)
}
func (*AppendEntriesRequest) CmdId() uint32 {
	return uint32(Cmd_Cmd_AppendEntriesRequest)
}
func (x *AppendEntriesRequest) Marshal() ([]byte, error) {
	return proto.Marshal(x)
}
func (x *AppendEntriesRequest) Unmarshal(in []byte) error {
	return proto.Unmarshal(in, x)
}
func (*AppendEntriesRequest) New() jsn_rpc.RpcUnit {
	return new(AppendEntriesRequest)
}
func (*AppendEntriesResponse) CmdId() uint32 {
	return uint32(Cmd_Cmd_AppendEntriesResponse)
}
func (x *AppendEntriesResponse) Marshal() ([]byte, error) {
	return proto.Marshal(x)
}
func (x *AppendEntriesResponse) Unmarshal(in []byte) error {
	return proto.Unmarshal(in, x)
}
func (*AppendEntriesResponse) New() jsn_rpc.RpcUnit {
	return new(AppendEntriesResponse)
}
