/**
 * @Author : liangliangtoo
 * @File : client
 * @Date: 2021/11/10 21:26
 * @Description:
 */
package Trpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/LucienVen/Trpc/core"
	"io"
	"log"
	"net"
	"sync"
)

// Call 结构代表一个活动的RPC
type Call struct {
	Seq           uint64
	ServiceMethod string      // 格式化 server.method
	Args          interface{} // 参数
	Reply         interface{} // 回复
	Error         error
	Done          chan *Call // 调用结束后通知调用方
}

func (c *Call) done() {
	c.Done <- c
}

// Client 代表一个 RPC Client。
//可能有多个未完成的调用关联
//有一个客户端，一个客户端可以被使用
//同时运行多个 goroutine。
type Client struct {
	cc       core.Codec // 消息的编解码器
	opt      *Option
	sending  sync.Mutex  // 互斥锁，保证请求按顺序发送
	header   core.Header // 每个请求的请求头
	mu       sync.Mutex
	seq      uint64           // 每个请求唯一序号
	pending  map[uint64]*Call // 存储未处理完的请求，map[seq]*Call
	closing  bool
	shutdown bool
}

// TODO 这是什么写法？？
var _ io.Closer = (*Client)(nil)

var ErrShutdown = errors.New("connection is shut down")

// Close 关闭连接
func (client *Client) Close() error {
	client.mu.Lock()
	defer client.mu.Unlock()

	if client.closing {
		return ErrShutdown
	}
	client.closing = true
	return client.cc.Close()
}

// IsAvailable 检测客户端是否正常工作
func (client *Client) IsAvailable() bool {
	client.mu.Lock()
	defer client.mu.Unlock()

	return !client.shutdown && !client.closing
}

// 将参数call 添加到client.pending中，并更新client.seq
func (client *Client) registerCall(call *Call) (uint64, error) {
	client.mu.Lock()
	defer client.mu.Unlock()

	if client.closing || client.shutdown {
		return 0, ErrShutdown
	}

	call.Seq = client.seq
	client.pending[call.Seq] = call // call 添加到client.pending中
	client.seq++                    // 更新client.seq
	return call.Seq, nil
}

// 根据seq, 从client.pending 中移除对应的call
func (client Client) removeCall(seq uint64) *Call {
	client.mu.Lock()
	defer client.mu.Unlock()
	call, ok := client.pending[seq]
	if ok {
		delete(client.pending, seq)
	}

	return call
}

// 服务端或者客户端发生错误时候调用
// 将shutdown设置为true，并将错误信息通知所有的pending 状态的call
func (client *Client) terminateCalls(err error)  {
	client.sending.Lock()
	defer client.sending.Unlock()
	client.mu.Lock()
	defer client.mu.Unlock()

	client.shutdown = true
	for _, call := range client.pending {
		call.Error = err
		call.done()
	}
}

/** 请求与响应 **/

// 接收请求
func (client *Client) receive()  {
	var err error
	for err == nil {
		var h core.Header
		// 如果读取请求头失败
		if err = client.cc.ReadHeader(&h); err != nil {
			break
		}

		call := client.removeCall(h.Seq)
		switch {
		case call == nil:
			// 通常表示写入部分失败，并且调用已被删除
			err = client.cc.ReadBody(nil)
		case h.Error != "":
			call.Error = fmt.Errorf(h.Error)
			err = client.cc.ReadBody(nil)
			call.done()
		default:
			err = client.cc.ReadBody(call.Reply)
			if err != nil {
				call.Error = errors.New("reading body " + err.Error())
			}
			call.done()
		}
	}
}

// NewClient 创建client实例
// 需要处理协议交换（option）
func NewClient(conn net.Conn, opt *Option) (*Client, error) {
	f := core.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		err := fmt.Errorf("invalid core type: %s", opt.CodecType)
		log.Println("rpc client: core error:", err)
		return nil, err
	}

	// 使用服务器发送选项
	if err := json.NewEncoder(conn).Encode(opt); err != nil {
		log.Println("rpc clientL options error: ", err)
		_ = conn.Close()
		return nil, err
	}
	
	return newClientCodec(f(conn), opt), nil
}

// 建立实例，接收请求
func newClientCodec(cc core.Codec, opt *Option) *Client {
	client := &Client{
		cc:       cc,
		opt:      opt,
		seq:      1, // 序号从1开始
		pending:  make(map[uint64]*Call),
	}

	go client.receive()
	return client
}

// option 可选参数传入
func parseOptions(opts ...*Option) (*Option, error) {
	if len(opts) == 0 || opts[0] == nil {
		return DefaultOption, nil
	}

	if len(opts) != 1 {
		return nil, errors.New("number of options is more than 1")
	}

	opt := opts[0]
	opt.MagicNumber = DefaultOption.MagicNumber
	if opt.CodecType == "" {
		opt.CodecType = DefaultOption.CodecType
	}
	return opt, nil
}

// Dial 实现Dial函数
func Dial(network, address string, opts ...*Option) (client *Client, err error) {
	opt, err := parseOptions(opts...)
	if err != nil {
		return nil, err
	}
	
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}

	defer func() {
		if client == nil {
			_ = conn.Close()
		}
	}()

	return NewClient(conn, opt)
}


// 发送请求的实现
func (client *Client) send(call *Call)  {
	client.sending.Lock()
	defer client.sending.Unlock()

	// 注册结构体call
	seq, err := client.registerCall(call)
	if err != nil {
		call.Error = err
		call.done()
		return
	}

	// 准备请求头
	client.header.ServiceMethod = call.ServiceMethod
	client.header.Seq = seq
	client.header.Error = ""

	// 编码且发送请求
	if err := client.cc.Write(&client.header, call.Args); err != nil {
		call := client.removeCall(seq)
		if call != nil {
			call.Error = err
			call.done()
		}
	}

}

// Go 函数 异步调用，发起请求
func (client *Client) Go(serviceMethod string, args, reply interface{}, done chan *Call) *Call {
	if done == nil {
		done = make(chan *Call, 10)
	} else if cap(done) == 0 {
		log.Panic("rpc client: done channel is unbuffered")
	}

	call := &Call{
		ServiceMethod: serviceMethod,
		Args:          args,
		Reply:         reply,
		Done:          done,
	}

	client.send(call)
	return call
}

// Call 对Go函数的封装，阻塞等待call.Done, 等待响应返回
// 返回错误状态
func (client *Client) Call(serviceMethod string, args, relpy interface{}) error {
	call := <-client.Go(serviceMethod, args, relpy, make(chan *Call, 1)).Done
	return call.Error
}







