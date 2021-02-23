package config

import (
	"fmt"
	"github.com/panjf2000/gnet"
	"thunder/internal/logging"
	"time"
)

type ServerConfig struct {
	Addr         string
	Port         int32
	Multicore    bool
	EventLoopNum int32
	TcpKeepAlive time.Duration
	LoadBalance  gnet.LoadBalancing
	Logger       logging.Logger

	PrintBanner bool
}

func NewDefaultServerConfig(port int32) *ServerConfig {
	return &ServerConfig{
		Addr:         fmt.Sprintf("tcp://:%d", port),
		Port:         port,
		Multicore:    true,
		EventLoopNum: 8,
		TcpKeepAlive: 5 * time.Second,
		Logger:       logging.DefaultLogger,
		PrintBanner:  true,
	}
}

type ClientConfig struct {
	Logger logging.Logger
}

func NewClientConfig() *ClientConfig {
	return &ClientConfig{
		Logger: logging.DefaultLogger,
	}
}


