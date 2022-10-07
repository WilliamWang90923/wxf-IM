package websocket

import (
	"github.com/gobwas/ws"
	"github.com/wangxuefeng90923/wxf"
	"net"
)

type Frame struct {
	raw ws.Frame
}

func (f *Frame) SetOpCode(code wxf.OpCode) {
	f.raw.Header.OpCode = ws.OpCode(code)
}

func (f *Frame) GetOpCode() wxf.OpCode {
	return wxf.OpCode(f.raw.Header.OpCode)
}

func (f *Frame) SetPayload(payload []byte) {
	f.raw.Payload = payload
}

func (f *Frame) GetPayload() []byte {
	if f.raw.Header.Masked {
		ws.Cipher(f.raw.Payload, f.raw.Header.Mask, 0)
	}
	f.raw.Header.Masked = false
	return f.raw.Payload
}

type WsConn struct {
	net.Conn
}

func NewConn(conn net.Conn) *WsConn {
	return &WsConn{conn}
}

func (wc *WsConn) ReadFrame() (wxf.Frame, error) {
	f, err := ws.ReadFrame(wc)
	if err != nil {
		return nil, err
	}
	return &Frame{raw: f}, nil
}

// WriteFrame we assume our max package size will not exceed
// websocket package size restriction, so fin always true
func (wc *WsConn) WriteFrame(code wxf.OpCode, payload []byte) error {
	f := ws.NewFrame(ws.OpCode(code), true, payload)
	return ws.WriteFrame(wc.Conn, f)
}

func (wc *WsConn) Flush() error {
	//TODO implement me
	panic("implement me")
}
