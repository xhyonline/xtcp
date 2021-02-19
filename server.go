package xtcp

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"github.com/go-basic/uuid"
	"net"
	"sync"
)



// server 是一个服务端实例
type server struct {
	// 服务端监听句柄
	listener net.Listener
	// 服务端文件描述符的上下文集合 key 是 uid 值为 contextRecv
	fdsContext sync.Map
	// 是否注册连接方法
	haveRegisterConn bool
	// 是否注册了关闭方法
	haveRegisterClose bool
	// 是否注册了消息处理方法
	haveRegisterHandleMessage bool
	// 连接事件管道
	connChan chan *contextRecv
	// 关闭事件管道
	closeChan chan *contextRecv
	// 处理消息管道
	handleMessageChan chan *contextRecv
}

// accept 接受连接
func (s *server) accept(){
	for  {
		conn,err:=s.listener.Accept()
		if err!=nil {
			panic(err)
		}

		uid:=uuid.New()
		// 构建上下文
		ctx:=&contextRecv{
			uid:uid,
			remoteIP:conn.RemoteAddr().String(),
			conn:&ConnFD{conn:conn},
		}
		// 将连接维护到总集合中
		s.fdsContext.Store(uid,ctx)

		// 如果用户注册了以下方法
		if s.haveRegisterConn {
			s.connChan<-ctx
		}

		if s.haveRegisterHandleMessage {
			s.handleMessageChan<-ctx
		}

	}
}

// CloseByUID 通过 uid 关闭一个连接
func (s *server) CloseByUID(uid string){
	if fd,exists:=s.fdsContext.Load(uid);exists{
		ctx:=fd.(*contextRecv)
		// 关闭连接
		ctx.conn.Close()
		// 从总集合中删除
		s.fdsContext.Delete(uid)
		// 推送关闭事件
		if s.haveRegisterClose { s.closeChan<-ctx}
	}
}

// Broadcast 广播
func (s *server) Broadcast(msg StandardMessage){
	s.fdsContext.Range(func(key, ctx interface{}) bool {
		_,_=ctx.(*contextRecv).Send(msg)
		return true
	})
}

// BroadcastOther 广播给其它人,除了自己
func (s *server) BroadcastOther(uid string,msg StandardMessage){
	s.fdsContext.Range(func(key, ctx interface{}) bool {
		if id:=ctx.(*contextRecv).uid;uid==id{
			return true
		}
		_,_=ctx.(*contextRecv).Send(msg)
		return true
	})
}

// Send 发送一条消息
func (s *server) Send(uid string,msg StandardMessage){
	if ctx,ok:=s.fdsContext.Load(uid);ok{
		_,_=ctx.(*contextRecv).Send(msg)
	}
}

// Close 优雅退出
func (s *server) Close(){
	_=s.listener.Close()
	defer close(s.closeChan)
	defer close(s.handleMessageChan)
	defer close(s.connChan)
	s.fdsContext.Range(func(key, value interface{}) bool {
		s.CloseByUID(key.(string))
		return true
	})
}


// =========================== 事件回调方法 ======================================

// OnConnect 当有连接发生时
func (s *server) OnConnect(f HandleFunc){
	s.haveRegisterConn=true
	go func() {
		for ctx:=range s.connChan{
			f(ctx)
		}
	}()
}

// OnMessage 当有消息时
func (s *server) OnMessage(f HandleFunc){
	s.haveRegisterHandleMessage=true
	// 注册事件
	go func() {
		for ctx:= range s.handleMessageChan{
			// 每一条消息启动一个协成,并发处理
			go func(ctx *contextRecv) {
				fd :=ctx.GetConn()
				reader := bufio.NewReader(fd.conn)
				for {
					// 前4个字节表示数据长度
					// 此外 Peek 方法并不会减少 reader 中的实际数据量
					peek, err := reader.Peek(4)
					if err != nil {
						// 读取失败关闭连接
						s.CloseByUID(ctx.uid)
						break
					}
					buffer := bytes.NewBuffer(peek)
					var length int32
					// 读取缓冲区前4位,代表消息实体的数据长度,赋予 length 变量
					err = binary.Read(buffer, binary.BigEndian, &length)
					if err != nil {
						continue
					}
					// reader.Buffered() 返回缓存中未读取的数据的长度,
					// 如果缓存区的数据小于总长度，则意味着数据不完整
					if int32(reader.Buffered()) < length+4 {
						continue
					}
					//从缓存区读取大小为数据长度的数据
					data := make([]byte, length+4)
					_, err = reader.Read(data)
					if err != nil {
						continue
					}
					// 将消息内容赋给上下文
					ctx.body=data[4:]
					f(ctx)
				}
			}(ctx)
		}
	}()
}

// OnClose 当客户端断开时
func (s *server) OnClose(f HandleFunc){
	s.haveRegisterClose=true
	go func() {
		for ctx:=range s.closeChan{
			f(ctx)
		}
	}()
}

// ======================== 事件回调方法结束 ===================================


