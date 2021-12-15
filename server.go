/**
 * @Author : liangliangtoo
 * @File : server
 * @Date: 2021/11/6 0:32
 * @Description: 通讯过程
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
	"reflect"
	"strings"
	"sync"
)

// @TODO 改为读取配置
const DefaultMagicNumber = 0x3bef5c

type Option struct {
	MagicNumber int
	CodecType   core.Ttype
}

var DefaultOption = &Option{
	MagicNumber: DefaultMagicNumber,
	CodecType:   core.GobType,
}

/** 服务端的实现 **/

type Server struct {
	serviceMap sync.Map
}

func NewServer() *Server {
	return &Server{}
}

var DeafaultServer = NewServer()

func (s *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println("rpc server: accept error:", err.Error())
			return
		}

		go s.ServeConn(conn)
	}
}

func Accept(lis net.Listener) {
	DeafaultServer.Accept(lis)
}

func (s *Server) ServeConn(conn io.ReadWriteCloser) {
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
var invalidRequest = struct{}{}

func (s *Server) serveCodec(cc core.Codec) {
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
	h      *core.Header
	argv   reflect.Value
	replyv reflect.Value
	mtype  *methodType
	svc    *service
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
//即通过 newArgv() 和 newReplyv() 两个方法创建出两个入参实例，
//然后通过 cc.ReadBody() 将请求报文反序列化为第一个入参 argv，
//在这里同样需要注意 argv 可能是值类型，也可能是指针类型，所以处理方式有点差异
func (s *Server) readRequest(cc core.Codec) (*request, error) {
	h, err := s.readRequestHeader(cc)
	fmt.Println("log: h", h.ServiceMethod)
	if err != nil {
		return nil, err
	}

	req := &request{h: h}

	req.svc, req.mtype, err = s.findService(h.ServiceMethod)
	if err != nil {
		return nil, err
	}

	req.argv = req.mtype.newArgv()
	req.replyv = req.mtype.newReplyv()

	// 确保 argvi 是一个指针，ReadBody 需要一个指针作为参数
	argvi := req.argv.Interface()
	if req.argv.Type().Kind() != reflect.Ptr {
		argvi = req.argv.Addr().Interface()
	}

	if err = cc.ReadBody(argvi); err != nil {
		log.Println("rpc server: read body err:", err)
		return req, err
	}


	// TODO 仅支持string
	//req.argv = reflect.New(reflect.TypeOf(""))
	//if err = cc.ReadBody(req.argv.Interface()); err != nil {
	//	log.Println("rpc server: read argv err: ", err)
	//}

	return req, nil
}

// 处理请求
func (s *Server) handleRequest(cc core.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()
	//log.Println(req.h, req.argv.Elem())
	//req.replyv = reflect.ValueOf(fmt.Sprintf("Trpc resp: %v", req.h.Seq))

	err := req.svc.call(req.mtype, req.argv, req.replyv)
	if err != nil {
		req.h.Error = err.Error()
		s.sendResponse(cc, req.h, invalidRequest, sending)
		return
	}

	s.sendResponse(cc, req.h, req.replyv.Interface(), sending)
}

// 回复请求
func (s *Server) sendResponse(cc core.Codec, h *core.Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()

	if err := cc.Write(h, body); err != nil {
		log.Println("rpc server: write response error:", err)
	}
}

// 服务注册（注册在服务器上发布的方案集）
func (s *Server) Register(rcvr interface{}) error {
	newServer := newService(rcvr)
	if _, dup := s.serviceMap.LoadOrStore(newServer.name, newServer); dup {
		return errors.New("rpc: service already defined: " + newServer.name)
	}

	return nil
}

// 注册在 DefaultServer 中发布接收者的方法
func Register(rcvr interface{}) error {
	return DeafaultServer.Register(rcvr)
}

// 寻找服务（通过ServiceMethod 从 serviceMap 中找到对应的service）
func (s *Server) findService(serviceMethod string) (svc *service, mtype *methodType, err error) {
	dot := strings.LastIndex(serviceMethod, ".")
	if dot < 0 {
		err = errors.New("rpc server: service/method request ill-formed: " + serviceMethod)
		return
	}

	serviceName, methodName := serviceMethod[:dot], serviceMethod[dot+1:]
	svci, ok := s.serviceMap.Load(serviceName)
	if !ok {
		err = errors.New("rpc server: can't find service " + serviceName)
		return
	}
	fmt.Println("*************** first **************")
	svc = svci.(*service)
	fmt.Println("*************** after **************")
	mtype = svc.method[methodName]
	if mtype == nil {
		err = errors.New("rpc server: can't find method " + methodName)
	}
	return
}
