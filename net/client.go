package net

import (
	"context"
	"encoding/binary"
	"github.com/panjf2000/gnet/pool/goroutine"
	"github.com/smallnest/goframe"
	"io"
	"net"
	"sync"
	"thunder/config"
	"thunder/internal/logging"
	"thunder/protocol"
	"time"
)

type RPCClient struct {
	logger           logging.Logger
	packetProcessors map[int16]processFunc
	responseTable    sync.Map

	connectionTable  sync.Map
	connectionLocker sync.Mutex

	workerPool *goroutine.Pool
}

func NewRPCClient(config *config.ClientConfig) *RPCClient {
	return &RPCClient{
		logger: config.Logger,
		packetProcessors: make(map[int16]processFunc),
		workerPool: goroutine.Default(),
	}
}



type connWrapper struct {
	conn goframe.FrameConn
	addr net.Addr
}

func (R *RPCClient) InvokeSync(ctx context.Context, addr net.Addr, packet *protocol.Packet, timeout time.Duration) (*protocol.Packet, error) {
	cw, err := R.connect(addr)
	if err != nil {
		return nil, err
	}
	ctx, _ = context.WithTimeout(ctx, timeout)
	resp := NewResponseFuture(ctx, packet.PacketId, nil)
	R.responseTable.Store(packet.PacketId, resp)
	defer R.responseTable.Delete(packet.PacketId)
	conn := cw.conn
	data, err := protocol.Encode(packet)
	if err != nil {
		return nil, err
	}
	err = conn.WriteFrame(data)
	if err != nil {
		return nil, err
	}
	return resp.waitResponse()
}

func (R *RPCClient) InvokeAsync(ctx context.Context, addr net.Addr, packet *protocol.Packet, callback func(future *ResponseFuture), timeout time.Duration) error {
	cw, err := R.connect(addr)
	if err != nil {
		return err
	}
	ctx, _ = context.WithTimeout(ctx, timeout)
	conn := cw.conn
	resp := NewResponseFuture(ctx, packet.PacketId, callback)
	R.responseTable.Store(packet.PacketId, resp)
	data, err := protocol.Encode(packet)
	if err != nil {
		return err
	}
	err = conn.WriteFrame(data)
	if err != nil {
		return err
	}
	go func() {
		if err := recover(); err != nil {
			R.logger.Errorf("receive message async error, err: %v", err)
		}
		R.receiveAsync(resp)
	}()
	return nil
}

func (R *RPCClient) InvokeOneway(ctx context.Context, addr net.Addr, packet *protocol.Packet, timeout time.Duration) error {
	cw, err := R.connect(addr)
	if err != nil {
		return err
	}
	conn := cw.conn
	ctx, _ = context.WithTimeout(ctx, timeout)
	data, err := protocol.Encode(packet)
	if err != nil {
		return err
	}
	err = conn.WriteFrame(data)
	if err != nil {
		return err
	}
	return nil
}

func (R *RPCClient) RegisterProcessor(code int16, processFunc processFunc) {
	R.packetProcessors[code] = processFunc
}

func (R *RPCClient) connect(addr net.Addr) (*connWrapper, error) {
	R.connectionLocker.Lock()
	defer R.connectionLocker.Unlock()
	conn, ok := R.connectionTable.Load(addr.String())
	if ok {
		return conn.(*connWrapper), nil
	}

	cw, err := createGoFrameConn(addr)
	if err != nil {
		return nil, err
	}

	R.connectionTable.Store(addr.String(), cw)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				R.logger.Errorf("receive response packet error, addr: %d, err: %v", addr.String(), err)
			}
		}()
		R.receivePacket(cw)
	}()
	return cw, nil
}

func createGoFrameConn(addr net.Addr) (*connWrapper, error) {
	conn, err := net.Dial("tcp", addr.String())
	if err != nil {
		return nil, err
	}

	encoderConfig := goframe.EncoderConfig{
		ByteOrder:                       binary.BigEndian,
		LengthFieldLength:               4,
		LengthAdjustment:                0,
		LengthIncludesLengthFieldLength: false,
	}

	decoderConfig := goframe.DecoderConfig{
		ByteOrder:           binary.BigEndian,
		LengthFieldOffset:   0,
		LengthFieldLength:   4,
		LengthAdjustment:    0,
		InitialBytesToStrip: 4,
	}

	fc := goframe.NewLengthFieldBasedFrameConn(encoderConfig, decoderConfig, conn)
	cw := &connWrapper{
		conn: fc,
		addr: addr,
	}

	return cw, nil
}

func (R *RPCClient) ShutDown() {
	panic("implement me")
}

func (R *RPCClient) receiveAsync(f *ResponseFuture) {
	_, err := f.waitResponse()
	if err != nil {
		f.executeInvokeCallback()
	}
}

func (R *RPCClient) receivePacket(cw *connWrapper) {
	var err error
	for {
		conn := cw.conn
		if err != nil {
			if err == io.EOF {
				R.logger.Errorf("conn error, close connection, addr: %s, err: %v", cw.addr.String(), err)
			}
			cw.conn.Close()
			break
		}
		data, tmpErr := conn.ReadFrame()
		if tmpErr != nil {
			err = tmpErr
			continue
		}

		pkt, tmpErr := protocol.Decode(data)
		if tmpErr != nil {
			err = tmpErr
			R.logger.Errorf("decode packet error, err: %v", err)
		}
		R.processPacket(pkt, cw)
	}
}

func (R *RPCClient) processPacket(packet *protocol.Packet, cw *connWrapper) {
	if packet.IsResponseType() {
		resp, exist := R.responseTable.Load(packet.PacketId)
		if exist {
			R.responseTable.Delete(packet.PacketId)
			responseFuture := resp.(*ResponseFuture)
			err := R.workerPool.Submit(func() {
				defer func() {
					if err := recover(); err != nil {
						R.logger.Errorf("executeCallback error: %v", err)
					}
				}()
				responseFuture.Response = packet
				responseFuture.executeInvokeCallback()
				if responseFuture.Done != nil {
					responseFuture.Done <- true
				}

			})

			if err != nil {
				R.logger.Warnf("submit func to workerpool error, err: %v", err)
			}
		}
	} else {
		f := R.packetProcessors[packet.Code]
		if f != nil {
			err := R.workerPool.Submit(func() {
				res := f(packet, cw.addr)
				if res != nil && !packet.IsOneway()  {
					res.PacketId = packet.PacketId
					res.MarkResponseType()
					data, err := protocol.Encode(res)
					if err != nil {
						R.logger.Errorf("encode response packet error, err: %v", err)
					}
					err = cw.conn.WriteFrame(data)
					if err != nil {
						R.logger.Warnf("send response packet error, response: %+v, err: %+v", res, err)
					}
				}
			})

			if err != nil {
				R.logger.Warnf("submit func to workerpool error, err: %v", err)
			}
		} else {
			R.logger.Warnf("there is no process func registered with code: %d", packet.Code)
		}
	}
}
