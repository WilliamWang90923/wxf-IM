package wxf

import (
	"github.com/golang/protobuf/proto"
	"github.com/sirupsen/logrus"
	"github.com/wangxuefeng90923/wxf/wire/pkt"
	"sync"
)

type Session interface {
	GetChannelId() string
	GetGateId() string
	GetAccount() string
	GetRemoteIP() string
	GetApp() string
	GetTags() []string
}

type Context interface {
	Dispatcher
	SessionStorage
	Header() *pkt.Header
	ReadBody(val proto.Message) error
	Session() Session
	RespWithError(status pkt.Status, err error) error
	Resp(status pkt.Status, body proto.Message) error
	Dispatch(body proto.Message, recvs ...*Location) error
	Next()
}

type HandlerFunc func(Context)

type HandlersChain []HandlerFunc

type ContextImpl struct {
	sync.Mutex
	Dispatcher
	SessionStorage

	handlers HandlersChain
	index    int
	request  *pkt.LogicPkt
	session  Session
}

func (c *ContextImpl) Header() *pkt.Header {
	return &c.request.Header
}

func (c *ContextImpl) Session() Session {
	//TODO implement me
	panic("implement me")
}

func (c *ContextImpl) RespWithError(status pkt.Status, err error) error {
	//TODO implement me
	panic("implement me")
}

// Resp used to response a message to sender
func (c *ContextImpl) Resp(status pkt.Status, body proto.Message) error {
	packet := pkt.NewFrom(&c.request.Header)
	packet.Status = status
	packet.WriteBody(body)
	packet.Flag = pkt.Flag_Response
	logrus.Debugf("<-- Resp to %s command:%s  status: %v body: %s",
		c.Session(), &c.request.Header, status, body)
	err := c.Push(c.Session().GetGateId(), []string{c.Session().GetChannelId()}, packet)
	if err != nil {
		logrus.Error(err)
	}
	return err
}

// Dispatch use Push(gateway, channels, packet) method
// inside to send messages, Push method belong to Dispatcher
// interface.
func (c *ContextImpl) Dispatch(body proto.Message, recvs ...*Location) error {
	if len(recvs) == 0 {
		return nil
	}
	packet := pkt.NewFrom(&c.request.Header)
	packet.Flag = pkt.Flag_Push
	packet.WriteBody(body)
	logrus.Debugf("<-- Dispatch to %d users command:%s",
		len(recvs), &c.request.Header)

	group := make(map[string][]string)
	for _, recv := range recvs {
		// Do not send to itself
		if recv.ChannelId == c.Session().GetChannelId() {
			continue
		}
		if _, ok := group[recv.GateId]; !ok {
			group[recv.GateId] = make([]string, 0)
		}
		group[recv.GateId] = append(group[recv.GateId], recv.ChannelId)
	}
	for gateWay, ids := range group {
		err := c.Push(gateWay, ids, packet)
		if err != nil {
			logrus.Error(err)
		}
		return err
	}
	return nil
}

func (c *ContextImpl) Next() {
	if c.index >= len(c.handlers) {
		return
	}
	f := c.handlers[c.index]
	c.index++
	if f == nil {
		logrus.Warn("arrived unknown handler function")
		return
	}
	f(c)
	c.Next()
}

func (c *ContextImpl) ReadBody(val proto.Message) error {
	return c.request.ReadBody(val)
}

func (c *ContextImpl) reset() {
	c.request = nil
	c.index = 0
	c.handlers = nil
	c.session = nil
}
