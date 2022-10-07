package websocket

import (
	"errors"
	"fmt"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/sirupsen/logrus"
	"github.com/wangxuefeng90923/wxf"
	"net"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

type ClientOptions struct {
	Heartbeat time.Duration
	ReadWait  time.Duration
	WriteWait time.Duration
}

type Client struct {
	sync.Mutex
	wxf.Dialer
	sync.Once
	id      string
	name    string
	conn    net.Conn
	state   int32
	options ClientOptions
}

func NewClient(id, name string, opts ClientOptions) *Client {
	if opts.WriteWait == 0 {
		opts.WriteWait = wxf.DefaultWriteWait
	}
	if opts.ReadWait == 0 {
		opts.ReadWait = wxf.DefaultReadWait
	}
	cli := &Client{
		id:      id,
		name:    name,
		options: opts,
	}
	return cli
}

func (c *Client) Connect(addr string) error {
	_, err := url.Parse(addr)
	if err != nil {
		return err
	}
	if !atomic.CompareAndSwapInt32(&c.state, 0, 1) {
		return fmt.Errorf("client has already connected")
	}
	conn, err := c.DialAndHandShake(wxf.DialerContext{
		Id:      c.id,
		Name:    c.name,
		Address: addr,
		Timeout: wxf.DefaultLoginWait,
	})
	if err != nil {
		atomic.CompareAndSwapInt32(&c.state, 1, 0)
		return err
	}
	if conn == nil {
		return fmt.Errorf("conn is nil")
	}
	c.conn = conn

	if c.options.Heartbeat > 0 {
		go func() {
			err := c.heartbeatLoop()
			if err != nil {
				logrus.Error("heartbeatLoop stopped: ", err)
			}
		}()
	}
	return nil
}

// this method is not thread secured!
func (c *Client) Read() (wxf.Frame, error) {
	if c.conn == nil {
		return nil, errors.New("connection is nil")
	}
	if c.options.Heartbeat > 0 {
		_ = c.conn.SetReadDeadline(time.Now().Add(c.options.ReadWait))
	}
	frame, err := ws.ReadFrame(c.conn)
	if err != nil {
		return nil, err
	}
	if frame.Header.OpCode == ws.OpClose {
		return nil, errors.New("remote side close the channel")
	}
	return &Frame{raw: frame}, nil
}

func (c *Client) Send(payload []byte) error {
	if atomic.LoadInt32(&c.state) == 0 {
		return fmt.Errorf("connection is nil")
	}
	c.Lock()
	defer c.Unlock()
	err := c.conn.SetWriteDeadline(time.Now().Add(c.options.WriteWait))
	if err != nil {
		return err
	}
	return wsutil.WriteClientMessage(c.conn, ws.OpBinary, payload)
}

func (c *Client) ID() string {
	return c.id
}

func (c *Client) Name() string {
	return c.name
}

func (c *Client) SetDialer(dialer wxf.Dialer) {
	c.Dialer = dialer
}

func (c *Client) Close() {
	c.Once.Do(func() {
		if c.conn == nil {
			return
		}
	})
	_ = wsutil.WriteClientMessage(c.conn, ws.OpClose, nil)
	c.conn.Close()
	atomic.CompareAndSwapInt32(&c.state, 1, 0)
}

func (c *Client) heartbeatLoop() error {
	tick := time.NewTicker(c.options.Heartbeat)
	for range tick.C {
		if err := c.ping(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) ping() error {
	c.Lock()
	defer c.Unlock()
	err := c.conn.SetWriteDeadline(time.Now().Add(c.options.WriteWait))
	if err != nil {
		return err
	}
	logrus.Tracef("%s send ping to server", c.id)
	return wsutil.WriteClientMessage(c.conn, ws.OpPing, nil)
}
