/**
 * @Author : liangliangtoo
 * @File : core
 * @Date: 2021/11/4 16:49
 * @Description:
 */
package core

import "io"

type Header struct {
	ServiceMethod string // format "Service.Method" 服务名.方法名
	Seq           uint64 // 请求的序号
	Error         error
}

// Codec 对消息体进行编码/解码的接口
type Codec interface {
	io.Closer
	ReadHeader(*Header) error
	ReadBody(interface{}) error
	Write(*Header, interface{}) error
}


type NewCodecFunc func(io.ReadWriteCloser) Codec

type Ttype string

const (
	GobType Ttype = "application/gob"
	JosnType Ttype = "application/json"
)



var NewCodecFuncMap map[Ttype]NewCodecFunc

func init() {
	NewCodecFuncMap = make(map[Ttype]NewCodecFunc)

	//TODO
	NewCodecFuncMap[GobType] = NewGobCodec
}







