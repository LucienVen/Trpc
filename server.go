/**
 * @Author : liangliangtoo
 * @File : server
 * @Date: 2021/11/6 0:32
 * @Description: 通讯过程
 */

package Trpc

import "github.com/LucienVen/Trpc/core"

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
