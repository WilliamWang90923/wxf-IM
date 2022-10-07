package websocket

import (
	"context"
	"errors"
	"fmt"
	"github.com/gobwas/ws"
	"github.com/segmentio/ksuid"
	"github.com/sirupsen/logrus"
	"github.com/wangxuefeng90923/wxf"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type ServerOptions struct {
	loginWait time.Duration
	readWait  time.Duration
	writeWait time.Duration
}

type Server struct {
	listen string
	wxf.ServiceRegistration
	wxf.ChannelMap
	wxf.Acceptor
	wxf.StateListener
	wxf.MessageListener
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

func (s *Server) Start() error {
	mux := http.NewServeMux()
	log := logrus.WithFields(logrus.Fields{
		"module": "ws.server",
		"listen": s.listen,
		"id":     s.ServiceID(),
	})
	if s.Acceptor == nil {
		s.Acceptor = new(defaultAcceptor)
	}
	if s.StateListener == nil {
		return fmt.Errorf("state listener is nil")
	}
	if s.ChannelMap == nil {
		s.ChannelMap = wxf.NewChannels(100)
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		rawConn, _, _, err := ws.UpgradeHTTP(r, w)
		if err != nil {
			logrus.Errorf("Http Upgrade error: %v", err)
			rawConn.Close()
			return
		}

		conn := NewConn(rawConn)
		id, err := s.Accept(conn, s.options.loginWait)
		if err != nil {
			_ = conn.WriteFrame(wxf.OpClose, []byte(err.Error()))
			conn.Close()
			return
		}
		if _, ok := s.Get(id); ok {
			log.Warnf("channel %s existed", id)
			_ = conn.WriteFrame(wxf.OpClose, []byte("channelId is repeated"))
			conn.Close()
			return
		}
		channel := wxf.NewChannel(id, conn)
		channel.SetWriteWait(s.options.writeWait)
		channel.SetReadWait(s.options.readWait)
		s.Add(channel)

		go func(ch wxf.Channel) {
			err := ch.Readloop(s.MessageListener)
			if err != nil {
				log.Info(err)
			}
			s.Remove(ch.ID())
			err = s.Disconnect(ch.ID())
			if err != nil {
				log.Warn(err)
			}
			ch.Close()
		}(channel)
	})
	log.Infoln("started")
	return http.ListenAndServe(s.listen, mux)
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

func (s *Server) Push(id string, data []byte) error {
	ch, ok := s.ChannelMap.Get(id)
	if !ok {
		return errors.New("channel no found")
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
			log.Infoln("shutdown")
		}()
		if atomic.CompareAndSwapInt32(&s.quit, 0, 1) {
			return
		}
		channels := s.ChannelMap.All()
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

type defaultAcceptor struct {
}

func (a *defaultAcceptor) Accept(conn wxf.Conn, duration time.Duration) (string, error) {
	return ksuid.New().String(), nil
}
