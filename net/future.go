package net

import (
	"context"
	"sync"
	"thunder/internal"
	"thunder/protocol"
)

type ResponseFuture struct {
	Response     *protocol.Packet
	Err          error
	PacketId     int32
	callback     func(*ResponseFuture)
	Done         chan bool
	callbackOnce sync.Once
	ctx          context.Context
}

func NewResponseFuture(ctx context.Context, opaque int32, callback func(*ResponseFuture)) *ResponseFuture {
	return &ResponseFuture{
		PacketId: opaque,
		Done:     make(chan bool),
		callback: callback,
		ctx:      ctx,
	}
}

func (r *ResponseFuture) executeInvokeCallback() {
	r.callbackOnce.Do(func() {
		if r.callback != nil {
			r.callback(r)
		}
	})
}

func (r *ResponseFuture) waitResponse() (*protocol.Packet, error) {
	var (
		pkt *protocol.Packet
		err error
	)
	select {
	case <-r.Done:
		pkt, err = r.Response, r.Err
	case <-r.ctx.Done():
		err = internal.ErrRequestTimeout
		r.Err = err
	}
	return pkt, err
}

