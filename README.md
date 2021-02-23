# thunder

Thunder is a network transport tool kit powered by [gnet](https://github.com/panjf2000/gnet)

it can build a high performance RPCServer or RPCClient easily. It supports send request synchronously、asynchronously and oneway

## How to use

### server

```go
func main() {
    s := NewRPCServer(config.NewDefaultServerConfig(9003))
    // register your func corresponding the code to process the packet with the code
    s.RegisterProcessor(1, func(p *protocol.Packet, addr net.Addr) *protocol.Packet {
        resp := protocol.NewPacket(1, nil, nil)
        resp.Remark = "response message"
        return resp
    })
    // call the start to start the server
    s.Start()
}
```

### client
```go
func main() {
    addr, err := net.ResolveTCPAddr("", "127.0.0.1:9003")
    if err != nil {
        panic(err)
    }
    c := NewRPCClient(config.NewClientConfig())
    // async
    err = c.InvokeAsync(context.TODO(), addr, protocol.NewPacket(1, nil, nil), func(future *ResponseFuture) {
        c.logger.Infof("%+v", future.Response)
    }, time.Second * 3000)
    if err != nil {
        panic(err)
    }
    
    // sync
    p, err := c.InvokeSync(context.TODO(), addr, protocol.NewPacket(1, nil, nil), time.Second * 3000)
    if err != nil {
        panic(err)
    }
    fmt.Printf("%+v", p)
}
```