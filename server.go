/**
 * @Author : liangliangtoo
 * @File : server
 * @Date: 2021/11/6 0:32
 * @Description: 通讯过程
 */

package Trpc

import (
	"encoding/json"
	"fmt"
	"github.com/LucienVen/Trpc/core"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
)

// @TODO 改为读取配置
const DefaultMagicNumber = 0x3bef5c

type Option struct {
	MagicNumber int
	CodecType core.Ttype
}


var DefaultOption = &Option{
	MagicNumber: DefaultMagicNumber,
	CodecType:   core.GobType,
}


/** 服务端的实现 **/

type Server struct {
	
}

func NewServer() *Server {
	return &Server{}
}

var DeafaultServer = NewServer()

func (s *Server) Accept(lis net.Listener)  {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println("rpc server: accept error:", err.Error())
			return
		}

		go s.ServeConn(conn)
	}
}

func Accept(lis net.Listener)  {
	DeafaultServer.Accept(lis)
}

func (s *Server) ServeConn(conn io.ReadWriteCloser)  {
	defer func() {
		_ = conn.Close()
	}()

	var opt Option
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc server: options error: ", err)
		return
	}

	if opt.MagicNumber != DefaultMagicNumber {
		log.Printf("rpc server: invalid magic number %x", opt.MagicNumber)
		return
	}

	f := core.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		log.Printf("rpc server: invalid codec type %s", opt.CodecType)
		return
	}

	s.serveCodec(f(conn))
}


// 发生错误时候的占位符
var invalidRequest = struct {}{}

func (s *Server) serveCodec(cc core.Codec)  {
	sending := new(sync.Mutex)
	wg := new(sync.WaitGroup)

	for {
		req, err := s.readRequest(cc)
		if err != nil {
			if err != nil {
				if req == nil {
					break
				}
				req.h.Error = err.Error()
				s.sendResponse(cc, req.h, invalidRequest, sending)
				continue
			}
		}

		wg.Add(1)
		go s.handleRequest(cc, req, sending, wg)
	}

	wg.Wait()
	_ = cc.Close()
}


/** request 请求 **/
type request struct {
	h *core.Header
	argv reflect.Value
	replyv reflect.Value
}

// 读取请求头
func (s *Server) readRequestHeader(cc core.Codec) (*core.Header, error) {
	var h core.Header
	if err := cc.ReadHeader(&h); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server: read header err: ", err)
		}
		return nil, err
	}
	return &h, nil
}

// 读取请求
func (s *Server) readRequest(cc core.Codec) (*request, error) {
	h, err := s.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}

	req := &request{h: h}

	// TODO 仅支持string
	req.argv = reflect.New(reflect.TypeOf(""))
	if err = cc.ReadBody(req.argv.Interface()); err != nil {
		log.Println("rpc server: read argv err: ", err)
	}

	return req, nil
}

// 处理请求
func (s *Server) handleRequest(cc core.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup)  {
	defer wg.Done()
	log.Println(req.h, req.argv.Elem())

	req.replyv = reflect.ValueOf(fmt.Sprintf("Trpc resp: %v", req.h.Seq))
	s.sendResponse(cc, req.h, req.replyv.Interface(), sending)
}

// 回复请求
func (s *Server) sendResponse(cc core.Codec, h *core.Header, body interface{}, sending *sync.Mutex)  {
	sending.Lock()
	defer sending.Unlock()

	if err := cc.Write(h, body); err != nil {
		log.Println("rpc server: write response error:", err)
	}
}





















