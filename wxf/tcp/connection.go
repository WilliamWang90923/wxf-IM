package tcp

import (
	"github.com/wangxuefeng90923/wxf"
	"github.com/wangxuefeng90923/wxf/wire/endian"
	"io"
	"net"
)

type Frame struct {
	OpCode  wxf.OpCode
	Payload []byte
}

func (f *Frame) SetOpCode(code wxf.OpCode) {
	f.OpCode = code
}

func (f *Frame) GetOpCode() wxf.OpCode {
	return f.OpCode
}

func (f *Frame) SetPayload(bytes []byte) {
	f.Payload = bytes
}

func (f *Frame) GetPayload() []byte {
	return f.Payload
}

type ConnTCP struct {
	net.Conn
}

func NewConn(conn net.Conn) *ConnTCP {
	return &ConnTCP{conn}
}

func (c *ConnTCP) ReadFrame() (wxf.Frame, error) {
	opCode, err := endian.ReadUint8(c.Conn)
	if err != nil {
		return nil, err
	}
	payload, err := endian.ReadBytes(c.Conn)
	if err != nil {
		return nil, err
	}
	return &Frame{
		OpCode:  wxf.OpCode(opCode),
		Payload: payload,
	}, nil
}

func (c *ConnTCP) WriteFrame(code wxf.OpCode, payload []byte) error {
	return WriteFrame(c.Conn, code, payload)
}

func (c *ConnTCP) Flush() error {
	panic("implement me")
}

// WriteFrame write a frame to w
func WriteFrame(w io.Writer, code wxf.OpCode, payload []byte) error {
	if err := endian.WriteUint8(w, uint8(code)); err != nil {
		return err
	}
	if err := endian.WriteBytes(w, payload); err != nil {
		return err
	}
	return nil
}
