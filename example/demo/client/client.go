package main

import (
	"fmt"
	"github.com/xhyonline/xtcp"
	"time"
)

func main() {
	c:=xtcp.NewClient(xtcp.Config{
		Host: "121.5.62.93",
		Port: "9000",
	})

	c.OnConnect(func(ctx xtcp.Context) {
		// 心跳请自行保证
		for   {
			if err:=ctx.GetConn().SendText("PING");err!=nil{
				break
			}
			time.Sleep(time.Second*3)
		}

	})

	c.OnClose(func(ctx xtcp.Context) {
		fmt.Println("客户端超时断开了链接")
	})

	c.Run()

	select {

	}
}

