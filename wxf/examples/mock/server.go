package mock

import (
	"errors"
	"github.com/sirupsen/logrus"
	"github.com/wangxuefeng90923/wxf"
	"github.com/wangxuefeng90923/wxf/naming"
	"github.com/wangxuefeng90923/wxf/tcp"
	"github.com/wangxuefeng90923/wxf/websocket"
	"time"
)

type ServerDemo struct {
}

func (s *ServerDemo) Start(id, protocol, addr string) {
	var srv wxf.Server
	service := &naming.DefaultService{
		Id:       "",
		Protocol: "",
	}
	if protocol == "ws" {
		srv = websocket.NewServer(addr, service)
	} else if protocol == "tcp" {
		srv = tcp.NewServer(addr, service)
	}
	handler := &ServerHandler{}

	srv.SetReadWait(time.Minute)
	srv.SetAcceptor(handler)
	srv.SetMessageListener(handler)
	srv.SetStateListener(handler)

	err := srv.Start()
	if err != nil {
		panic(err)
	}
}

type ServerHandler struct {
}

func (s *ServerHandler) Accept(conn wxf.Conn, timeout time.Duration) (string, error) {
	frame, err := conn.ReadFrame()
	if err != nil {
		return "", err
	}
	userID := string(frame.GetPayload())
	if userID == "" {
		return "", errors.New("user id is invalid")
	}
	return userID, nil
}

func (s *ServerHandler) Receive(agent wxf.Agent, payload []byte) {
	ack := agent.ID() + " receive " + string(payload) + " from server "
	agent.Push([]byte(ack))
}

func (s *ServerHandler) Disconnect(id string) error {
	logrus.Warnf("disconnect %s", id)
	return nil
}
