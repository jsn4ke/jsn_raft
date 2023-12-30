package jsn_raft

import "time"

type ServerConfig struct {
	List []struct {
		Who  string `yaml:"who"`
		Addr string `yaml:"addr"`
	} `yaml:"list"`
}

// 广播时间（broadcastTime） << 选举超时时间（electionTimeout） << 平均故障间隔时间（MTBF）

func (r *RaftNew) heartbeatTimeout() time.Duration {
	return time.Millisecond * 500
}

func (r *RaftNew) electionTimeout() time.Duration {
	return time.Millisecond * 150
}

func (r *RaftNew) rpcTimeout() time.Duration {
	return time.Millisecond * 20
}

func (r *RaftNew) orphanTimeout() time.Duration {
	return time.Second * 20
}
