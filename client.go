package xtcp

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"github.com/go-basic/uuid"
	"net"
)

// client 客户端实例
type client struct {
	// 服务端连接的唯一 ID
	UID string
	// 连接句柄
	conn *ConnFD
	// 是否注册了关闭方法
	haveRegisterClose bool
	// 是否注册了消息处理方法
	haveRegisterHandleMessage bool
	// 关闭事件管道
	closeChan chan *contextRecv
	// 处理消息管道
	handleMessageChan chan *contextRecv
}

// ================ 客户端回调方法集 ===============

// OnMessage 当收到消息时触发
func (c *client) OnMessage(f HandleFunc){
	c.haveRegisterHandleMessage=true
	go func() {
		for ctx:=range c.handleMessageChan {
			f(ctx)
		}
	}()
}

// OnConnect 建立连接触发的回调
func (c *client) OnConnect(f HandleFunc){
	f(&contextRecv{
		uid:      c.UID,
		remoteIP: c.conn.conn.RemoteAddr().String(),
		conn:     c.conn,
		body:     nil,	// 刚连接,没有消息,自然为空
	})
}

// OnClose 连接断开触发的回调
func (c *client) OnClose(f HandleFunc){
	c.haveRegisterClose=true
	go func() {
		for ctx:=range c.closeChan{
			f(ctx)
		}
	}()
}

// ==================== 客户端回调方法集结束 ==========================

// Send 实现客户端发送消息给服务端的方法
func (c *client) Send(msg StandardMessage)  {
	_,_=c.conn.Send(msg)
}

// listen 客户端消息监听
func (c *client) listen(){
	reader := bufio.NewReader(c.conn.conn)
	for {
		// 前4个字节表示数据长度
		// 此外 Peek 方法并不会减少 reader 中的实际数据量
		peek, err := reader.Peek(4)
		if err != nil {
			c.Close()
			break
		}
		buffer := bytes.NewBuffer(peek)
		var length int32
		// 读取缓冲区前4位,代表消息实体的数据长度,赋予 length 变量
		err = binary.Read(buffer, binary.BigEndian, &length)
		if err != nil {
			panic(err)
		}
		// reader.Buffered() 返回缓存中未读取的数据的长度,
		// 如果缓存区的数据小于总长度，则意味着数据不完整,很可能是内核态没有完全拷贝数据到用户态中
		// 因此下一轮就齐活了
		if int32(reader.Buffered()) < length+4 {
			continue
		}
		//从缓存区读取大小为数据长度的数据
		data := make([]byte, length+4)
		_, err = reader.Read(data)
		if err != nil {
			panic(err)
		}
		// 如果没有注册该方法,则丢弃这条消息,继续下一轮
		if !c.haveRegisterHandleMessage {
			continue
		}
		// 管道分发,事件处理
		c.handleMessageChan<- &contextRecv{
			uid:      c.UID,
			remoteIP: c.conn.conn.RemoteAddr().String(),
			conn:     c.conn,
			body:    data[4:],
		}
	}
}

// Close 优雅退出,该方法只允许被调用一回
func (c *client) Close(){
	// 关闭连接
	defer c.conn.Close()
	defer close(c.closeChan)
	defer close(c.handleMessageChan)
	// 事件通知
	if c.haveRegisterClose {
		c.closeChan<-&contextRecv{
			uid:      c.UID,
			remoteIP: c.conn.conn.RemoteAddr().String(),
			conn:     c.conn,
			body:     nil,
		}
	}
}


