/**
 * @Author : liangliangtoo
 * @File : gob
 * @Date: 2021/11/4 17:03
 * @Description:
 */
package core

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

type GobCodec struct {
	conn io.ReadWriteCloser
	buf *bufio.Writer
	dec *gob.Decoder
	enc *gob.Encoder
}

var _ Codec = (*GobCodec)(nil)

func NewGobCodec(conn io.ReadWriteCloser) Codec {
	buf := bufio.NewWriter(conn)
	return &GobCodec{
		conn: conn,
		buf:  buf,
		dec:  gob.NewDecoder(conn),
		enc:  gob.NewEncoder(buf),
	}
}


/** 实现codec方法 **/

func (c *GobCodec) ReadHeader(h *Header) error {
	return c.dec.Decode(h)
}

func (c *GobCodec) ReadBody(body interface{}) error {
	return c.dec.Decode(body)
}

func (c *GobCodec) Write(h *Header, body interface{}) error {
	defer func() {
		err := c.buf.Flush()
		if err != nil {
			_ = c.Close()
		}
	}()

	if err := c.enc.Encode(h); err != nil {
		log.Println("rpc codec: gob error encoding Header:", err)
		return err
	}

	if err := c.enc.Encode(body); err != nil {
		log.Println("rpc codec: gob error encoding Body:", err)
		return err
	}

	return nil
}

func (c *GobCodec) Close() error {
	return c.conn.Close()
}