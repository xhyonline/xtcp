package xtcp

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"net"
)

type HandleFunc func(ctx Context)

// Handle 这是一个公共处理方法的接口
// client 实现了它, server 也实现了它
type Handle interface {
	// 当连接时触发的回调方法
	OnConnect(handleFunc HandleFunc)
	// 当连接关闭时触发的方法
	OnClose(handleFunc HandleFunc)
	// 当有消息时触发的方法
	OnMessage(handleFunc HandleFunc)
	// Close 优雅退出
	Close()
	// 启动
	Run()
}

// ServerHandle 服务端接口
type ServerHandle interface {
	Handle
	// 广播消息
	BroadcastText(msg string)
	// 广播除了自己的其它用户
	BroadcastTextOther(uid string, msg string)
	// 发送消息
	SendText(uid string, msg string) error
	// 发送消息
	SendByte(uid string, msg []byte) error
	// 关闭一个连接
	CloseByUID(uid string)
}

// ClientHandle 客户端接口
type ClientHandle interface {
	Handle
	SendText(msg string) error
	// 发送消息
	SendByte(msg []byte) error
}

// Context 标准上下文信息
type Context interface {
	// 返回字符串
	String() string
	// 原始内容
	Byte() []byte
	// 获取远程 IP 端口信息
	RemoteIP() string
	// 获取该连接的 uid
	GetConnUID() string
	// 获取连接句柄
	GetConn() *FD
	// 发送文本消息
	SendText(msg string) error
	// 发送字节
	SendByte(msg []byte) error
}

// contextRecv 上下文接收器
type contextRecv struct {
	// 该连接的唯一 uid
	uid string
	// IP
	remoteIP string
	// 句柄
	conn *FD
	// 消息内容
	body string
}

// String 获取消息字符串
func (c *contextRecv) String() string {
	return c.body
}

// Byte 获取消息
func (c *contextRecv) Byte() []byte {
	return []byte(c.body)
}

// RemoteIP 获取远程客户端信息
func (c *contextRecv) RemoteIP() string {
	return c.remoteIP
}

// GetConnUID 获取该连接唯一的 uid
func (c *contextRecv) GetConnUID() string {
	return c.uid
}

// GetConn 获取连接句柄
func (c *contextRecv) GetConn() *FD {
	return c.conn
}

// SendText 发送消息
func (c *contextRecv) SendText(msg string) error {
	return c.conn.SendText(msg)
}

// SendByte 发送字节
func (c *contextRecv) SendByte(msg []byte) error {
	return c.conn.SendByte(msg)
}

// FD 连接描述符的具体实现
type FD struct {
	conn net.Conn
}

// SendText 发送消息
func (c FD) SendText(msg string) error {
	m := &Message{Body: msg}
	body, err := m.encode()
	if err != nil {
		return err
	}
	_, err = c.conn.Write(body)
	return err
}

// SendByte 发送字节
func (c FD) SendByte(msg []byte) error {
	m := &Message{Body: string(msg)}
	body, err := m.encode()
	if err != nil {
		return err
	}
	_, err = c.conn.Write(body)
	return err
}

// Close 关闭文件描述符
func (c FD) Close() {
	_ = c.conn.Close()
}

// StandardMessage 一条标准消息
type StandardMessage interface {
	// 一条标准消息实现编码方法
	encode() ([]byte, error)
}

// Message 是一条标准消息的实现
type Message struct {
	Body string `json:"body"`
}

// encode 消息编码
func (m *Message) encode() ([]byte, error) {
	// 序列化为 json
	message, _ := json.Marshal(m)

	// 读取该 json 的长度
	var length = int32(len(message))
	var pkg = new(bytes.Buffer)
	// 写入消息头
	err := binary.Write(pkg, binary.BigEndian, length)
	if err != nil {
		return nil, err
	}
	// 写入消息实体
	err = binary.Write(pkg, binary.BigEndian, message)
	if err != nil {
		return nil, err
	}
	return pkg.Bytes(), nil
}

// Config 公共配置
type Config struct {
	Host string
	Port string
}
