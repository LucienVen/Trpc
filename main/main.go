/**
 * @Author : liangliangtoo
 * @File : main
 * @Date: 2021/11/9 18:23
 * @Description: 客户端实现
 */
package main

import (
	"encoding/json"
	"fmt"
	"github.com/LucienVen/Trpc"
	"github.com/LucienVen/Trpc/core"
	"log"
	"net"
	"time"

)


// 使用chan，确保服务端端口监听成功再发起请求
func startServer(addr chan string)  {
	// TODO 读取配置
	l, err := net.Listen("tcp", ":8765")
	if err != nil {
		log.Fatalf("network error:", err)
	}

	log.Println("start rpc server on: ", l.Addr())
	addr <- l.Addr().String()
	Trpc.Accept(l)
}

func main()  {
	addr := make(chan string)
	go startServer(addr)

	conn, _ := net.Dial("tcp", <-addr)
	defer func() {
		_ = conn.Close()
	}()

	time.Sleep(time.Second)

	// 发送option
	json.NewEncoder(conn).Encode(Trpc.DefaultOption)
	cc := core.NewGobCodec(conn)

	for i := 0; i < 5; i++ {
		h := &core.Header{
			ServiceMethod: "Foo.sum",
			Seq:           uint64(i),
			Error:         "",
		}

		cc.Write(h, fmt.Sprintf("Trpc req: %d", h.Seq))
		cc.ReadHeader(h)

		var reply string
		cc.ReadBody(&reply)
		log.Println("reply: ", reply)
	}
}
