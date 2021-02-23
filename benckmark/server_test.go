package benckmark

import (
	"context"
	net2 "net"
	"testing"
	"thunder/config"
	"thunder/net"
	"thunder/protocol"
	"time"
)

func BenchmarkServer(b *testing.B) {
	c := net.NewRPCClient(config.NewClientConfig())
	for i := 0; i < b.N; i++ {
		p := protocol.NewPacket(1, nil, nil)
		addr, _ := net2.ResolveTCPAddr("", "127.0.0.1:9003")
		_, _ = c.InvokeSync(context.Background(), addr, p, 3 * time.Second)
	}
}