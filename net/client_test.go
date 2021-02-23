package net

import (
	"context"
	"net"
	"testing"
	"thunder/config"
	"thunder/protocol"
	"time"
)

func TestInvokeSync(t *testing.T) {
	addr, err := net.ResolveTCPAddr("", "127.0.0.1:9003")
	if err != nil {
		panic(err)
	}
	c := NewRPCClient(config.NewClientConfig())
	err = c.InvokeAsync(context.Background(), addr, protocol.NewPacket(1, []byte("Creams"), nil), func(future *ResponseFuture) {
		c.logger.Infof("%+v", future.Response)
	}, time.Second * 3000)
	if err != nil {
		panic(err)
	}

	ctx, _ := context.WithTimeout(context.Background(), 4*time.Second)
	select {
	case <-ctx.Done():
	}
}
