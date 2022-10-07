package wxf

import (
	"net"
	"time"
)

type Client interface {
	ID() string
	Name() string
	Connect(string) error
	SetDialer(Dialer)
	Send([]byte) error
	Read() (Frame, error)
	Close()
}

type Dialer interface {
	DialAndHandShake(DialerContext) (net.Conn, error)
}

type DialerContext struct {
	Id      string
	Name    string
	Address string
	Timeout time.Duration
}
