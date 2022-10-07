package tcp

import (
	"context"
	"errors"
	"fmt"
	"github.com/segmentio/ksuid"
	"github.com/sirupsen/logrus"
	"github.com/wangxuefeng90923/wxf"
	"github.com/wangxuefeng90923/wxf/container"
	"github.com/wangxuefeng90923/wxf/naming"
	"github.com/wangxuefeng90923/wxf/naming/consul"
	"github.com/wangxuefeng90923/wxf/services/gateway/serv"
	"github.com/wangxuefeng90923/wxf/services/server/conf"
	"github.com/wangxuefeng90923/wxf/websocket"
	"github.com/wangxuefeng90923/wxf/wire"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type Server struct {
	listen string
	wxf.ServiceRegistration
	wxf.ChannelMap
	wxf.Acceptor
	wxf.MessageListener
	wxf.StateListener
	sync.Once
	options ServerOptions
	quit    int32
}

func NewServer(listen string, service wxf.ServiceRegistration) wxf.Server {
	return &Server{
		listen:              listen,
		ServiceRegistration: service,
		options: ServerOptions{
			loginWait: wxf.DefaultLoginWait,
			readWait:  wxf.DefaultReadWait,
			writeWait: wxf.DefaultWriteWait,
		},
	}
}

type ServerStartOptions struct {
	config      string
	serviceName string
	protocol    string
}

func RunServerStart(ctx context.Context, opts *ServerStartOptions, version string) error {
	config, err := conf.Init(opts.config)
	if err != nil {
		return err
	}
	handler := &serv.Handler{ServiceID: config.ServiceID}

	var srv wxf.Server
	service := &naming.DefaultService{
		Id:       config.ServiceID,
		Name:     opts.serviceName,
		Address:  config.PublicAddress,
		Port:     config.PublicPort,
		Protocol: opts.protocol,
		Tags:     config.Tags,
	}
	if opts.protocol == "ws" {
		srv = websocket.NewServer(config.Listen, service)
	}
	srv.SetReadWait(time.Minute * 2)
	srv.SetAcceptor(handler)
	srv.SetMessageListener(handler)
	srv.SetStateListener(handler)

	_ = container.Init(srv, wire.SNChat, wire.SNLogin)
	ns, err := consul.NewNaming(config.ConsulURL)
	if err != nil {
		return err
	}
	container.SetServiceNaming(ns)
	container.SetDialer(serv.NewDialer(config.ServiceID))

	return container.Start()
}

func (s *Server) Start() error {
	log := logrus.WithFields(logrus.Fields{
		"module": "tcp.server",
		"listen": s.listen,
		"id":     s.ServiceID(),
	})

	if s.StateListener == nil {
		return fmt.Errorf("StateListener is nil")
	}
	if s.Acceptor == nil {
		s.Acceptor = new(defaultAcceptor)
	}
	if s.ChannelMap == nil {
		s.ChannelMap = wxf.NewChannels(100)
	}

	lst, err := net.Listen("tcp", s.listen)
	if err != nil {
		return err
	}
	log.Info("tcp started")
	for {
		rawConn, err := lst.Accept()
		if err != nil {
			_ = rawConn.Close()
			log.Warn(err)
			return err
		}
		go func(rawConn net.Conn) {
			conn := NewConn(rawConn)
			id, err := s.Accept(conn, s.options.loginWait)
			if err != nil {
				_ = conn.WriteFrame(wxf.OpClose, []byte(err.Error()))
				_ = conn.Close()
				return
			}
			if _, ok := s.Get(id); ok {
				log.Warnf("channel %s existed", id)
				_ = conn.WriteFrame(wxf.OpClose, []byte("channelId is repeated"))
				_ = conn.Close()
				return
			}
			channel := wxf.NewChannel(id, conn)
			channel.SetReadWait(s.options.readWait)
			channel.SetWriteWait(s.options.writeWait)
			s.Add(channel)
			log.Info("accept channel: ", channel)
			err = channel.Readloop(s.MessageListener)
			if err != nil {
				log.Info(err)
			}
			s.Remove(channel.ID())
			_ = s.Disconnect(channel.ID())
			channel.Close()
		}(rawConn)
	}
}

func (s *Server) Push(id string, data []byte) error {
	ch, ok := s.Get(id)
	if !ok {
		return errors.New("channel not found")
	}
	return ch.Push(data)
}

func (s *Server) Shutdown(ctx context.Context) error {
	log := logrus.WithFields(logrus.Fields{
		"module": s.ServiceName(),
		"id":     s.ServiceID(),
	})
	s.Do(func() {
		defer func() {
			log.Infof("service %s shutdown", s.ServiceName())
		}()
		// already closed
		if !atomic.CompareAndSwapInt32(&s.quit, 0, 1) {
			return
		}
		channels := s.All()
		for _, ch := range channels {
			_ = ch.Close()
			select {
			case <-ctx.Done():
				return
			default:
				continue
			}
		}
	})
	return nil
}

func (s *Server) SetAcceptor(acceptor wxf.Acceptor) {
	s.Acceptor = acceptor
}

func (s *Server) SetMessageListener(listener wxf.MessageListener) {
	s.MessageListener = listener
}

func (s *Server) SetStateListener(listener wxf.StateListener) {
	s.StateListener = listener
}

func (s *Server) SetReadWait(duration time.Duration) {
	s.options.readWait = duration
}

func (s *Server) SetChannelMap(channelMap wxf.ChannelMap) {
	s.ChannelMap = channelMap
}

type ServerOptions struct {
	loginWait time.Duration
	readWait  time.Duration
	writeWait time.Duration
}

type defaultAcceptor struct {
}

func (a *defaultAcceptor) Accept(conn wxf.Conn, duration time.Duration) (string, error) {
	return ksuid.New().String(), nil
}
