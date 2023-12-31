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

// CmdId implements jsn_rpc.RpcUnit.
func (*VoteRequest) CmdId() uint32 {
	return uint32(Cmd_Cmd_VoterRequest)
}

// Marshal implements jsn_rpc.RpcUnit.
func (x *VoteRequest) Marshal() ([]byte, error) {
	return proto.Marshal(x)
}

// New implements jsn_rpc.RpcUnit.
func (*VoteRequest) New() jsn_rpc.RpcUnit {
	return new(VoteRequest)
}

// Unmarshal implements jsn_rpc.RpcUnit.
func (x *VoteRequest) Unmarshal(in []byte) error {
	return proto.Unmarshal(in, x)
}

// CmdId implements jsn_rpc.RpcUnit.
func (*VoteResponse) CmdId() uint32 {
	return uint32(Cmd_Cmd_VoteResponse)
}

// Marshal implements jsn_rpc.RpcUnit.
func (x *VoteResponse) Marshal() ([]byte, error) {
	return proto.Marshal(x)
}

// New implements jsn_rpc.RpcUnit.
func (*VoteResponse) New() jsn_rpc.RpcUnit {
	return new(VoteResponse)
}

// Unmarshal implements jsn_rpc.RpcUnit.
func (x *VoteResponse) Unmarshal(in []byte) error {
	return proto.Unmarshal(in, x)
}

// CmdId implements jsn_rpc.RpcUnit.
func (*AppendEntriesRequest) CmdId() uint32 {
	return uint32(Cmd_Cmd_AppendEntriesRequest)
}

// Marshal implements jsn_rpc.RpcUnit.
func (x *AppendEntriesRequest) Marshal() ([]byte, error) {
	return proto.Marshal(x)
}

// New implements jsn_rpc.RpcUnit.
func (*AppendEntriesRequest) New() jsn_rpc.RpcUnit {
	return new(AppendEntriesRequest)
}

// Unmarshal implements jsn_rpc.RpcUnit.
func (x *AppendEntriesRequest) Unmarshal(in []byte) error {
	return proto.Unmarshal(in, x)
}

// CmdId implements jsn_rpc.RpcUnit.
func (*AppendEntriesResponse) CmdId() uint32 {
	return uint32(Cmd_Cmd_AppendEntriesResponse)
}

// Marshal implements jsn_rpc.RpcUnit.
func (x *AppendEntriesResponse) Marshal() ([]byte, error) {
	return proto.Marshal(x)
}

// New implements jsn_rpc.RpcUnit.
func (*AppendEntriesResponse) New() jsn_rpc.RpcUnit {
	return new(AppendEntriesResponse)
}

// Unmarshal implements jsn_rpc.RpcUnit.
func (x *AppendEntriesResponse) Unmarshal(in []byte) error {
	return proto.Unmarshal(in, x)
}
