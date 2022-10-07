package wxf

import (
	"errors"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type ChannelImpl struct {
	sync.Mutex
	id string
	Conn
	writeChan chan []byte
	sync.Once
	writeWait time.Duration
	readWait  time.Duration
	closed    *Event
}

func NewChannel(id string, conn Conn) Channel {
	log := logrus.WithFields(logrus.Fields{
		"module": "tcp_channel",
		"id":     id,
	})
	ch := &ChannelImpl{
		id:        id,
		Conn:      conn,
		writeChan: make(chan []byte, 5),
		writeWait: time.Second * 10,
		closed:    NewEvent(),
	}
	go func() {
		err := ch.writeLoop()
		if err != nil {
			log.Info(err)
		}
	}()
	return ch
}

func (c *ChannelImpl) writeLoop() error {
	for {
		select {
		case payload := <-c.writeChan:
			err := c.WriteFrame(OpBinary, payload)
			if err != nil {
				return err
			}
			chanLen := len(c.writeChan)
			for i := 0; i < chanLen; i++ {
				payload = <-c.writeChan
				err := c.WriteFrame(OpBinary, payload)
				if err != nil {
					return err
				}
			}
			//err = c.Conn.Flush()
			if err != nil {
				return err
			}
		case <-c.closed.Done():
			return nil
		}
	}
}

// Readloop could only be visited by one thread one time
// it is a block method
func (c *ChannelImpl) Readloop(msgLst MessageListener) error {
	c.Lock()
	defer c.Unlock()
	log := logrus.WithFields(logrus.Fields{
		"struct": "ChannelImpl",
		"func":   "Readloop",
		"id":     c.id,
	})
	for {
		_ = c.SetWriteDeadline(time.Now().Add(c.readWait))
		frame, err := c.ReadFrame()
		if err != nil {
			return err
		}
		if frame.GetOpCode() == OpClose {
			return errors.New("remote side close the channel")
		}
		if frame.GetOpCode() == OpPing {
			log.Trace("recv a ping; resp with a pong")
			_ = c.WriteFrame(OpPong, nil)
			continue
		}
		payload := frame.GetPayload()
		if len(payload) == 0 {
			continue
		}
		go msgLst.Receive(c, payload)
	}
}

// WriteFrame overwrite Conn
// enforcing it with setting write deadline
func (c *ChannelImpl) WriteFrame(code OpCode, payload []byte) error {
	_ = c.Conn.SetWriteDeadline(time.Now().Add(c.writeWait))
	return c.Conn.WriteFrame(code, payload)
}

func (c *ChannelImpl) ID() string {
	return c.id
}

func (c *ChannelImpl) Push(payload []byte) error {
	if c.closed.HasFired() {
		return errors.New("channel has closed")
	}
	c.writeChan <- payload
	return nil
}

func (c *ChannelImpl) SetWriteWait(duration time.Duration) {
	if duration == 0 {
		return
	}
	c.writeWait = duration
}

func (c *ChannelImpl) SetReadWait(duration time.Duration) {
	if duration == 0 {
		return
	}
	c.readWait = duration
}
