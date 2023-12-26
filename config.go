package jsn_raft

import "time"

type ServerConfig struct {
	List []struct {
		Who  string `yaml:"who"`
		Addr string `yaml:"addr"`
	} `yaml:"list"`
}

func (r *RaftNew) heartbeatTimeout() time.Duration {
	return time.Microsecond * 2000
}

func (r *RaftNew) electionTimeout() time.Duration {
	return time.Millisecond * 2000
}

func (r *RaftNew) rpcTimeout() time.Duration {
	return time.Millisecond * 1000
}

func (r *RaftNew) orphanTimeout() time.Duration {
	return time.Second * 20
}
