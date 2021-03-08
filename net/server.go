package net

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/panjf2000/gnet"
	"github.com/panjf2000/gnet/pool/goroutine"
	"net"
	"sync"
	"thunder/config"
	"thunder/internal"
	"thunder/internal/logging"
	"thunder/protocol"
	"time"
)

type processFunc func(p *protocol.Packet, addr net.Addr) *protocol.Packet

type RPCServer struct {
	gnet.EventServer
	logger           logging.Logger
	packetProcessors map[int16]processFunc
	responseTable    sync.Map

	serverConfig *config.ServerConfig

	codec      gnet.ICodec
	workerPool *goroutine.Pool
}

func NewRPCServer(serverConfig *config.ServerConfig) *RPCServer {
	server := &RPCServer{
		packetProcessors: make(map[int16]processFunc),
		logger: serverConfig.Logger,
	}

	encoderConfig := gnet.EncoderConfig{
		ByteOrder:                       binary.BigEndian,
		LengthFieldLength:               4,
		LengthAdjustment:                0,
		LengthIncludesLengthFieldLength: false,
	}
	decoderConfig := gnet.DecoderConfig{
		ByteOrder:           binary.BigEndian,
		LengthFieldOffset:   0,
		LengthFieldLength:   4,
		LengthAdjustment:    0,
		InitialBytesToStrip: 4,
	}

	server.codec = gnet.NewLengthFieldBasedFrameCodec(encoderConfig, decoderConfig)
	server.serverConfig = serverConfig
	server.workerPool = goroutine.Default()

	return server
}

func (r *RPCServer) InvokeSync(ctx context.Context, conn gnet.Conn, packet *protocol.Packet, timeout time.Duration) (*protocol.Packet, error) {
	ctx, _ = context.WithTimeout(ctx, timeout)
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

func (r *RPCServer) InvokeAsync(ctx context.Context, conn gnet.Conn, packet *protocol.Packet, callback func(future *ResponseFuture), timeout time.Duration) error {
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

func (r *RPCServer) InvokeOneway(ctx context.Context, conn gnet.Conn, packet *protocol.Packet, timeout time.Duration) error {
	ctx, _ = context.WithTimeout(ctx, timeout)
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

func (r *RPCServer) RegisterProcessor(code int16, processFunc processFunc) {
	r.packetProcessors[code] = processFunc
}

func (r *RPCServer) ShutDown() {
	panic("implement me")
}

func (r *RPCServer) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	p, err := protocol.Decode(frame)
	if err != nil {
		return
	}
	r.logger.Debugf("receive packet: %+v", p)
	r.processPacket(p, c)
	return
}

func (r *RPCServer) OnInitComplete(srv gnet.Server) (action gnet.Action) {
	r.logger.Infof(internal.BannerString())
	r.logger.Infof("[THUNDER] Thunder server is listening on %s (multi-cores: %t, loops: %d)\n",
		srv.Addr.String(), srv.Multicore, srv.NumEventLoop)
	return
}

func (r *RPCServer) Start() {
	err := gnet.Serve(r, r.serverConfig.Addr, func(opts *gnet.Options) {
		opts.Logger = r.serverConfig.Logger
		opts.Codec = r.codec
		opts.Multicore = r.serverConfig.Multicore
		opts.TCPKeepAlive = r.serverConfig.TcpKeepAlive
		opts.LB = r.serverConfig.LoadBalance
	})

	if err != nil {
		panic(err)
	}
}

func (r *RPCServer) processPacket(packet *protocol.Packet, conn gnet.Conn) {
	if packet.IsResponseType() {
		resp, exist := r.responseTable.Load(packet.PacketId)
		if exist {
			r.responseTable.Delete(packet.PacketId)
			responseFuture := resp.(*ResponseFuture)
			err := r.workerPool.Submit(func() {
				defer func() {
					if err := recover(); err != nil {
						r.logger.Errorf("executeCallback error: %v", err)
					}
				}()
				responseFuture.Response = packet
				responseFuture.executeInvokeCallback()
				if responseFuture.Done != nil {
					responseFuture.Done <- true
				}

			})

			if err != nil {
				r.logger.Warnf("submit func to workerpool error, err: %v", err)
			}
		}
	} else {
		f := r.packetProcessors[packet.Code]
		if f != nil {
			err := r.workerPool.Submit(func() {
				defer func() {
					if err := recover(); err != nil {
						r.logger.Errorf("execute process func error: %v", err)
					}
				}()
				res := f(packet, conn.RemoteAddr())
				if res != nil && !packet.IsOneway() {
					res.PacketId = packet.PacketId
					res.MarkResponseType()
					data, err := protocol.Encode(res)
					if err != nil {
						r.logger.Errorf("encode response packet error, err: %v", err)
					}
					err = conn.AsyncWrite(data)
					if err != nil {
						r.logger.Warnf("send response packet error, response: %+v, err: %+v", res, err)
					}
				}
			})

			if err != nil {
				r.logger.Warnf("submit func to workerpool error, err: %v", err)
			}
		} else {
			p := protocol.NewPacket(internal.NotSupport, nil, nil)
			p.PacketId = packet.PacketId
			p.MarkResponseType()
			p.Message = fmt.Sprintf("there is no process func registered with code: %d", packet.Code)
			data, err := protocol.Encode(p)
			if err != nil {
				r.logger.Errorf("encode response packet error, err: %v", err)
			}
			err = conn.AsyncWrite(data)
			if err != nil {
				r.logger.Warnf("send response packet error, response: %+v, err: %+v", p, err)
			}
		}
	}
}

func (r *RPCServer) receiveAsync(f *ResponseFuture) {
	_, err := f.waitResponse()
	if err != nil {
		f.executeInvokeCallback()
	}
}
