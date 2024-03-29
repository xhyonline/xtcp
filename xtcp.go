package xtcp

import (
	"github.com/go-basic/uuid"
	"net"
	"time"
)

// NewClient 实例化一个客户端连接
func NewClient(c Config) ClientHandle {
	conn, err := net.Dial("tcp", c.Host+":"+c.Port)
	if err != nil {
		panic(err)
	}
	client := &client{
		UID:               uuid.New(),
		conn:              &FD{conn: conn},
		closeChan:         make(chan *contextRecv),
		handleMessageChan: make(chan *contextRecv),
	}
	return client
}

// NewServer 获取一个实例
func NewServer(c Config) (ServerHandle, error) {
	listener, err := net.Listen("tcp", c.Host+":"+c.Port)
	if err != nil {
		return nil, err
	}
	if c.HeartBeat == 0 {
		c.HeartBeat = time.Second * 5
	}
	s := &server{
		listener:          listener,
		closeChan:         make(chan *contextRecv),
		connChan:          make(chan *contextRecv),
		handleMessageChan: make(chan *contextRecv),
		heartBeat:         c.HeartBeat,
	}
	return s, nil
}
