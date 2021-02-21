# xtcp 介绍

xtcp 是一款 Golang 轻量级 TCP 框架,以回调事件形式处理消息。作为初学者,您可以查看源码,源码内含大量中文注释。

使用方式说明:

**服务端**

```go
package main

import (
	"fmt"
	"github.com/xhyonline/xtcp"
	"time"
)

func main() {
	// 建立连接
	server, err := xtcp.NewServer(xtcp.Config{
		Host: "127.0.0.1",
		Port: "8888",
	})
	if err != nil {
		panic(err)
	}
	// 当客户端连接时
	server.OnConnect(func(ctx xtcp.Context) {
		fmt.Printf("有一个客户端连接进来了,他的 IP 为 %s  他的 uid 为 %s\n", ctx.RemoteIP(), ctx.GetConnUID())
		// 发送欢迎语
		err := ctx.GetConn().SendText("您好新来的客户端" + ctx.GetConnUID())
		if err != nil {
			panic(err)
		}
	})

	// 当收到客户端消息时
	server.OnMessage(func(ctx xtcp.Context) {
		fmt.Println(ctx.String())
		// 当然你也可以这样
		//fmt.Println(string(ctx.Byte()))
	})

	// 当有连接断开时
	server.OnClose(func(ctx xtcp.Context) {
		fmt.Println("有一个连接断开了 IP 为:", ctx.RemoteIP(), "uid为", ctx.GetConnUID())
	})

	// 启动服务端,它是一个异步操作,请自行添加阻塞
	server.Run()
	//
	time.Sleep(time.Second * 15)
	//
	server.BroadcastText("这是一条广播消息")
	//
	select {}
}
```

**客户端**

```go
package main

import (
	"fmt"
	"github.com/xhyonline/xtcp"
)

func main() {

	client := xtcp.NewClient(xtcp.Config{
		Host: "127.0.0.1",
		Port: "8888",
	})

	client.OnConnect(func(ctx xtcp.Context) {
		fmt.Println("客户端建立了连接")
		err := ctx.GetConn().SendText("你好我是客户端的消息")
		if err != nil {
			panic(err)
		}
	})
	client.OnMessage(func(ctx xtcp.Context) {
		fmt.Println("客户端收到了一条消息", ctx.String())
	})

	client.OnClose(func(ctx xtcp.Context) {
		fmt.Println("客户端断开了连接")
	})

	client.Run()

	select {}
}

```





