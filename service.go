/**
 * @Author : liangliangtoo
 * @File : service
 * @Date: 2021/11/12 17:18
 * @Description: 通过反射实现结构体与服务的映射关系
 */
package Trpc

import (
	"fmt"
	"go/ast"
	"log"
	"reflect"
	"sync/atomic"
)

type methodType struct {
	method    reflect.Method
	ArgType   reflect.Type // 第一个参数类型
	ReplyType reflect.Type // 第二个参数的类型
	numCalls  uint64       // 统计方法调用次数
}

func (m *methodType) NumCalls() uint64 {
	return atomic.LoadUint64(&m.numCalls)
}

// 创建ArgType类型实例
func (m *methodType) newArgv() reflect.Value {
	var argv reflect.Value
	// 参数 指针类型或值类型
	if m.ArgType.Kind() == reflect.Ptr {
		argv = reflect.New(m.ArgType.Elem())
	} else {
		argv = reflect.New(m.ArgType).Elem()
	}

	return argv
}

// 创建ReplyType类型实例
func (m *methodType) newReplyv() reflect.Value {
	// 响应类型必须为指针类型
	replyv := reflect.New(m.ReplyType.Elem())

	switch m.ReplyType.Elem().Kind() {
	case reflect.Map:
		replyv.Elem().Set(reflect.MakeMap(m.ReplyType.Elem()))
	case reflect.Slice:
		replyv.Elem().Set(reflect.MakeSlice(m.ReplyType.Elem(), 0, 0))
	}

	return replyv
}

type service struct {
	name   string	// 映射的结构体名称
	typ    reflect.Type	// 结构体的类型
	rcvr   reflect.Value	// 结构体的实例本身
	method map[string]*methodType	// 存储映射的结构体的所有符合条件的方法
}

func newService(rcvr interface{}) *service {
	s := new(service)
	s.rcvr = reflect.ValueOf(rcvr)
	// reflect.Indirect 间接返回 s.rcvr 指向的值
	s.name = reflect.Indirect(s.rcvr).Type().Name()
	fmt.Println("s.name,indirect:", reflect.Indirect(s.rcvr).Type().Name())

	s.typ = reflect.TypeOf(rcvr)
	// TODO 打印
	if !ast.IsExported(s.name) {
		log.Fatalf("rpc server: %s is not a valid service name", s.name)
	}
	s.registerMethods()

	return s
}


// 注册method
// 过滤筛出符合条件的方法：两个导
func (s *service) registerMethods()  {
	s.method = make(map[string]*methodType)
	for i := 0; i < s.typ.NumMethod(); i++ {
		method := s.typ.Method(i) // 获取其中一个单一的方法
		mType := method.Type // 获取该方法的类型，如 func(*sync.WaitGroup, int)

		// 判断方法的入参和出参 数目
		// 按照rpc调用定义，需要3个入参，和1个出参（反射时为三个，第0个是自身）
		// 返回值有且只有一个，类型为error
		if mType.NumIn() != 3 || mType.NumOut() != 1 {
			continue
		}

		// 判断出参第一个参数类型
		fmt.Println("mType.Out(0):", mType.Out(0) )
		if mType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
			continue
		}

		// 判断调用的第一和第二个参数的类型
		argType, replyType := mType.In(1), mType.In(2)
		if !isExportedOrBuildInType(argType) || !isExportedOrBuildInType(replyType) {
			continue
		}

		s.method[method.Name] = &methodType{
			method:    method,
			ArgType:   argType,
			ReplyType: replyType,
		}

		log.Printf("rpc server: register %s.%s\n", s.name, method.Name)
	}
}

func isExportedOrBuildInType(t reflect.Type) bool {
	// PkgPath 返回定义类型的包路径，即导入路径
	// ast.IsExported 判断是否可导出类型
	return ast.IsExported(t.Name()) || t.PkgPath() == ""
}


// 实现call方法，即能够通过反射值调用方法
func (s *service) call(m *methodType, argv, replyv reflect.Value) error {
	atomic.AddUint64(&m.numCalls, 1)
	// 以接收者为第一个参数的函数
	f := m.method.Func
	returnValues := f.Call([]reflect.Value{s.rcvr, argv, replyv})
	if errInter := returnValues[0].Interface(); errInter != nil {
		// TODO 这是什么意思
		return errInter.(error)
	}

	return nil
}






















