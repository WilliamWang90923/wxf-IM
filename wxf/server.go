package wxf

import (
	"context"
	"net"
	"time"
)

// Server handles:
// 1. Start itself for upper service to activate services
// (however handshake is not handled here, needs to send callback to upper accept layer)
// 2. when a connection is accepted, readloop is started to listen to msg
// 3. when disconnect detected, notify upper layer
// 4. provide Push method so that upper service could send msg to channel designated
// 5. basic configurations
type Server interface {
	ServiceRegistration
	SetAcceptor(Acceptor)
	SetMessageListener(MessageListener)
	SetStateListener(StateListener)
	SetReadWait(time.Duration)
	SetChannelMap(ChannelMap)

	Start() error
	Push(string, []byte) error
	Shutdown(context.Context) error
}

// Acceptor
// return channelId(unique)
type Acceptor interface {
	Accept(Conn, time.Duration) (string, error)
}

type MessageListener interface {
	Receive(Agent, []byte)
}

type StateListener interface {
	Disconnect(string) error
}

type ChannelMap interface {
	Add(channel Channel)
	Remove(id string)
	Get(id string) (Channel, bool)
	All() []Channel
}

type Frame interface {
	SetOpCode(OpCode)
	GetOpCode() OpCode
	SetPayload([]byte)
	GetPayload() []byte
}

type Conn interface {
	net.Conn
	ReadFrame() (Frame, error)
	WriteFrame(OpCode, []byte) error
	Flush() error
}

// Agent return msg to upper service layer
// return channelId of connection
type Agent interface {
	ID() string
	Push([]byte) error
}

type Channel interface {
	Conn
	Agent
	Close() error
	Readloop(msgLst MessageListener) error
	SetWriteWait(time.Duration)
	SetReadWait(time.Duration)
}

type OpCode byte
