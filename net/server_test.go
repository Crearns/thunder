package net

import (
	"net"
	"testing"
	"thunder/config"
	"thunder/protocol"
)

func TestStart(t *testing.T) {
	s := NewRPCServer(config.NewDefaultServerConfig(9003))
	s.RegisterProcessor(1, func(p *protocol.Packet, addr net.Addr) *protocol.Packet {
		resp := protocol.NewPacket(1, nil, nil)
		resp.Message = "test test"
		return resp
	})
	s.Start()
}

