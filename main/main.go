/**
 * @Author : liangliangtoo
 * @File : main
 * @Date: 2021/11/9 18:23
 * @Description: 客户端实现
 */
package main

import (
	"context"
	"fmt"
	"github.com/LucienVen/Trpc"
	"log"
	"net"
	"reflect"
	"strings"
	"sync"
	"time"
)


// 使用chan，确保服务端端口监听成功再发起请求
func startServer(addr chan string)  {
	// 注册Foo到服务中
	var foo Foo
	if err := Trpc.Register(&foo); err != nil {
		log.Fatal("register error:", err)
	}

	// TODO 读取配置
	l, err := net.Listen("tcp", ":8765")
	if err != nil {
		log.Fatal("network error:", err)
	}

	log.Println("start rpc server on: ", l.Addr())
	addr <- l.Addr().String()
	Trpc.Accept(l)
}

// Day-1
//func main()  {
//	addr := make(chan string)
//	go startServer(addr)
//
//	conn, _ := net.Dial("tcp", <-addr)
//	defer func() {
//		_ = conn.Close()
//	}()
//
//	time.Sleep(time.Second)
//
//	// 发送option
//	json.NewEncoder(conn).Encode(Trpc.DefaultOption)
//	cc := core.NewGobCodec(conn)
//
//	for i := 0; i < 5; i++ {
//		h := &core.Header{
//			ServiceMethod: "Foo.sum",
//			Seq:           uint64(i),
//			Error:         "",
//		}
//
//		// write 模拟发送请求
//		cc.Write(h, fmt.Sprintf("Trpc req: %d", h.Seq))
//		cc.ReadHeader(h)
//
//		var reply string
//		cc.ReadBody(&reply)
//		log.Println("reply: ", reply)
//	}
//}

// Day-2
func main()  {
	log.SetFlags(0)
	addr := make(chan string)
	go startServer(addr)

	client, _ := Trpc.Dial("tcp", <-addr)
	defer func() {
		_ = client.Close()
	}()

	time.Sleep(time.Second)

	// 发送请求和接受响应
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// Day-2
			//args := fmt.Sprintf("trpc req: %d", i)

			//day-3
			args := &Args{
				Num1: i,
				Num2: i+1,
			}
			//day-2
			//var reply string

			//day-3
			ctx, _ := context.WithTimeout(context.Background(), time.Second)
			var reply int
			if err := client.Call(ctx, "Foo.Sum", args, &reply); err != nil {
				log.Fatal("call Foo.Sum error: ", err)
			}

			log.Println("reply: ", reply)
		}(i)
	}

	wg.Wait()
}

func CheckWg()  {
	var wg sync.WaitGroup
	typ := reflect.TypeOf(&wg)
	//fmt.Println(typ.NumMethod())
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)
		fmt.Println("method:", method.Type)

		argv := make([]string, 0, method.Type.NumIn())
		returns := make([]string, 0, method.Type.NumOut())

		for j := 1; j < method.Type.NumIn(); j++ {
			fmt.Println("method in:", method.Type.In(j))
			argv = append(argv, method.Type.In(j).Name())
		}

		for j := 0; j < method.Type.NumOut(); j++ {
			returns = append(returns, method.Type.Out(j).Name())
			fmt.Println("method out:", method.Type.Out(j))

		}


		log.Printf("func (w *%s) %s(%s) %s",
				typ.Elem().Name(),
				method.Name,
				strings.Join(argv, ","),
				strings.Join(returns, ","),
			)
	}
}



// day-3

type Foo int

type Args struct {
	Num1, Num2 int
}

func (f Foo) Sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}















