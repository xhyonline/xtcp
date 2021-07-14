package main

import (
	"fmt"
	"github.com/xhyonline/xtcp"
	"github.com/xhyonline/xutil/sig"
	"github.com/xhyonline/xutil/xlog"
	"time"
)

var logger=xlog.Get().Debugger()

func main() {
	s,err:=xtcp.NewServer(xtcp.Config{
		Host: "0.0.0.0",
		Port: "9000",
		HeartBeat:time.Second*5,	// 心跳超时,必须保证客户端每 5s 内都有消息发送至此
	})
	if err!=nil{
		panic(err)
	}
	s.OnConnect(func(ctx xtcp.Context) {
		fmt.Println("有一个链接进来了")
	})
	s.OnMessage(func(ctx xtcp.Context) {
		fmt.Println(time.Now().String(),"客户端发来消息:",ctx.String())

	})
	s.OnClose(func(ctx xtcp.Context) {
		fmt.Println("有一个客户端断开了链接",time.Now().String())
	})
	s.Run()


	ctx:=sig.Get().RegisterClose(s)

	select {
	case <-ctx.Done():
		break
	}
	logger.Infof("服务已优雅退出")

}
