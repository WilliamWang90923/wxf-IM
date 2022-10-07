package tcp

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/wangxuefeng90923/wxf"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

type Client struct {
	sync.Mutex
	wxf.Dialer
	sync.Once
	id      string
	name    string
	conn    wxf.Conn
	state   int32
	options ClientOptions
	Meta    map[string]string
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

func NewClientWithProps(id, name string, meta map[string]string, opts ClientOptions) wxf.Client {
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
		Meta:    meta,
	}
	return cli
}

func (c *Client) Connect(addr string) error {
	_, err := url.Parse(addr)
	if err != nil {
		return err
	}
	if !atomic.CompareAndSwapInt32(&c.state, 0, 1) {
		return fmt.Errorf("client has connected")
	}
	rawConn, err := c.DialAndHandShake(wxf.DialerContext{
		Id:      c.id,
		Name:    c.name,
		Address: addr,
		Timeout: wxf.DefaultLoginWait,
	})
	if err != nil {
		atomic.CompareAndSwapInt32(&c.state, 1, 0)
		return err
	}
	if rawConn == nil {
		return fmt.Errorf("conn is nil")
	}
	c.conn = NewConn(rawConn)

	if c.options.Heartbeat > 0 {
		go func() {
			err := c.heartbeatLoop()
			if err != nil {
				logrus.WithField("module", "tcp.client").Warn("heartbeatLoop stopped - ", err)
			}
		}()
	}
	return nil
}

func (c *Client) Send(bytes []byte) error {
	if atomic.LoadInt32(&c.state) == 0 {
		return fmt.Errorf("connection is nil")
	}
	c.Lock()
	defer c.Unlock()
	err := c.conn.SetWriteDeadline(time.Now().Add(c.options.WriteWait))
	if err != nil {
		return err
	}
	return c.conn.WriteFrame(wxf.OpBinary, bytes)
}

func (c *Client) Read() (wxf.Frame, error) {
	if c.conn == nil {
		return nil, errors.New("connection is nil")
	}
	if c.options.Heartbeat > 0 {
		_ = c.conn.SetReadDeadline(time.Now().Add(c.options.ReadWait))
	}
	frame, err := c.conn.ReadFrame()
	if err != nil {
		return nil, err
	}
	if frame.GetOpCode() == wxf.OpClose {
		return nil, errors.New("remote side close the channel")
	}
	return frame, nil
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
	logrus.WithField("module", "tcp client").
		Trace("%s send ping to server", c.id)
	err := c.conn.SetWriteDeadline(time.Now().Add(c.options.WriteWait))
	if err != nil {
		return err
	}
	return c.conn.WriteFrame(wxf.OpPing, nil)
}

func (c *Client) SetDialer(dialer wxf.Dialer) {
	c.Dialer = dialer
}

func (c *Client) ID() string {
	return c.id
}

func (c *Client) Name() string {
	return c.name
}

func (c *Client) Close() {
	c.Do(func() {
		if c.conn == nil {
			return
		}
		_ = c.conn.WriteFrame(wxf.OpClose, nil)
		_ = c.conn.Close()
		atomic.CompareAndSwapInt32(&c.state, 1, 0)
	})
}

type ClientOptions struct {
	Heartbeat time.Duration
	ReadWait  time.Duration
	WriteWait time.Duration
}
