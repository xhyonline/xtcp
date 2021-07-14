package xtcp

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"github.com/go-basic/uuid"
	"github.com/xhyonline/xutil/xlog"
	"net"
	"sync"
	"time"
)

var logger=xlog.Get().Debugger()

// server 是一个服务端实例
type server struct {
	// 心跳
	heartBeat time.Duration
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
	// 是否关闭
	isShutdown bool
}

// accept 接受连接
func (s *server) accept() {
	for {
		conn, err := s.listener.Accept()
		if err != nil && s.isShutdown {
			break
		}
		if err != nil {
			panic(err)
		}

		uid := uuid.New()
		// 构建上下文
		ctx := &contextRecv{
			uid:      uid,
			remoteIP: conn.RemoteAddr().String(),
			conn:     &FD{conn: conn},
		}
		// 将连接维护到总集合中
		s.fdsContext.Store(uid, ctx)

		// 如果用户注册了以下方法
		if s.haveRegisterConn {
			s.connChan <- ctx
		}

		if s.haveRegisterHandleMessage {
			s.handleMessageChan <- ctx
		}

	}
}

// Run 启动方法
func (s *server) Run() {
	go s.accept()
}

// CloseByUID 通过 uid 关闭一个连接
func (s *server) CloseByUID(uid string) {
	if fd, exists := s.fdsContext.Load(uid); exists {
		// 优先从集合中删除,再关闭连接,请注意执行顺序不可颠倒
		s.fdsContext.Delete(uid)
		ctx := fd.(*contextRecv)
		// 关闭连接
		ctx.conn.close()
		// 推送关闭事件
		if s.haveRegisterClose {
			s.closeChan <- ctx
		}
	}
}

// BroadcastText 广播
func (s *server) BroadcastText(msg string) {
	s.fdsContext.Range(func(key, ctx interface{}) bool {
		_ = ctx.(*contextRecv).SendText(msg)
		return true
	})
}

// BroadcastTextOther 广播给其它人,除了自己
func (s *server) BroadcastTextOther(uid string, msg string) {
	s.fdsContext.Range(func(key, ctx interface{}) bool {
		if id := ctx.(*contextRecv).uid; uid == id {
			return true
		}
		_ = ctx.(*contextRecv).SendText(msg)
		return true
	})
}

// BroadcastByte 广播
func (s *server) BroadcastByte(msg []byte) {
	s.fdsContext.Range(func(key, ctx interface{}) bool {
		_ = ctx.(*contextRecv).SendByte(msg)
		return true
	})
}

// BroadcastByteOther 广播给其它人,除了自己
func (s *server) BroadcastByteOther(uid string, msg []byte) {
	s.fdsContext.Range(func(key, ctx interface{}) bool {
		if id := ctx.(*contextRecv).uid; uid == id {
			return true
		}
		_ = ctx.(*contextRecv).SendByte(msg)
		return true
	})
}

// SendText 发送一条消息
func (s *server) SendText(uid string, msg string) error {
	if ctx, ok := s.fdsContext.Load(uid); ok {
		return ctx.(*contextRecv).SendText(msg)
	}
	return nil
}

// SendByte 发送一条消息
func (s *server) SendByte(uid string, msg []byte) error {
	if ctx, ok := s.fdsContext.Load(uid); ok {
		return ctx.(*contextRecv).SendByte(msg)
	}
	return nil
}

// GracefulClose 优雅退出
func (s *server) GracefulClose() {
	if s.isShutdown {
		return
	}
	s.isShutdown = true
	_ = s.listener.Close()
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
func (s *server) OnConnect(f HandleFunc) {
	s.haveRegisterConn = true
	go func() {
		for ctx := range s.connChan {
			go f(ctx)
		}
	}()
}

// OnMessage 当有消息时
func (s *server) OnMessage(f HandleFunc) {
	s.haveRegisterHandleMessage = true
	// 注册事件
	go func() {
		for ctx := range s.handleMessageChan {
			// 每一条消息启动一个协程,并发处理
			go func(ctx *contextRecv) {
				fd := ctx.GetConn()
				reader := bufio.NewReader(fd.conn)
				for {
					err := fd.conn.SetDeadline(time.Now().Add(s.heartBeat))
					if err != nil {
						logger.Error(err)
					}
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
					m := new(Message)
					_ = json.Unmarshal(data[4:], m)
					// 将消息内容赋给上下文
					ctx.body = m.Body
					f(ctx)
				}
			}(ctx)
		}
	}()
}

// OnClose 当客户端断开时
func (s *server) OnClose(f HandleFunc) {
	s.haveRegisterClose = true
	go func() {
		for ctx := range s.closeChan {
			go f(ctx)
		}
	}()
}

// ======================== 事件回调方法结束 ===================================

