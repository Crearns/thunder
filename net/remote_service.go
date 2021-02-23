package net

import (
	"context"
	"github.com/panjf2000/gnet"
	"net"
	"sync"
	"thunder/internal/logging"
	"thunder/protocol"
	"time"
)

type Server interface {
	InvokeSync(ctx context.Context, conn gnet.Conn, packet *protocol.Packet, timeout time.Duration) (*protocol.Packet, error)
	InvokeAsync(ctx context.Context, conn gnet.Conn, packet *protocol.Packet, callback func(future *ResponseFuture), timeout time.Duration) error
	InvokeOneway(ctx context.Context, conn gnet.Conn, packet *protocol.Packet, timeout time.Duration) error
	RegisterProcessor(code int16, processFunc processFunc)
	ShutDown()
}

type Client interface {
	InvokeSync(ctx context.Context, addr net.Addr, packet *protocol.Packet, timeout time.Duration) (*protocol.Packet, error)
	InvokeAsync(ctx context.Context, addr net.Addr, packet *protocol.Packet, callback func(future *ResponseFuture), timeout time.Duration) error
	InvokeOneway(ctx context.Context, addr net.Addr, packet *protocol.Packet, timeout time.Duration) error
	RegisterProcessor(code int16, processFunc processFunc)
	ShutDown()
}

type RemoteService struct {
	logger logging.Logger
	packetProcessors map[int16]processFunc
	responseTable    sync.Map
}

func (r *RemoteService) InvokeSync(ctx context.Context, conn gnet.Conn, packet *protocol.Packet) (*protocol.Packet, error){
	resp := NewResponseFuture(ctx, packet.PacketId, nil)
	r.responseTable.Store(packet.PacketId, resp)
	defer r.responseTable.Delete(packet.PacketId)
	data, err := protocol.Encode(packet)
	if err != nil {
		return nil, err
	}
	err = conn.AsyncWrite(data)
	return resp.waitResponse()
}

func (r *RemoteService) InvokeAsync(ctx context.Context, conn gnet.Conn, packet *protocol.Packet, callback func(future *ResponseFuture)) error {
	resp := NewResponseFuture(ctx, packet.PacketId, nil)
	r.responseTable.Store(packet.PacketId, resp)
	data, err := protocol.Encode(packet)
	if err != nil {
		return err
	}
	err = conn.AsyncWrite(data)
	if err != nil {
		return err
	}
	go func() {
		if err := recover(); err != nil {
			r.logger.Errorf("receive message async error, err: %v", err)
		}
		r.receiveAsync(resp)
	}()
	return nil
}

func (r *RemoteService) InvokeOneway(ctx context.Context, conn gnet.Conn, packet *protocol.Packet) error {
	data, err := protocol.Encode(packet)
	if err != nil {
		return err
	}
	err = conn.AsyncWrite(data)
	if err != nil {
		return err
	}
	return nil
}

func (r *RemoteService) receiveAsync(f *ResponseFuture) {
	_, err := f.waitResponse()
	if err != nil {
		f.executeInvokeCallback()
	}
}