package wxf

import (
	"github.com/sirupsen/logrus"
	"sync"
)

type ChannelsImpl struct {
	channels *sync.Map
}

func NewChannels(num int) ChannelMap {
	return &ChannelsImpl{channels: new(sync.Map)}
}

func (c *ChannelsImpl) Add(channel Channel) {
	if channel.ID() == "" {
		logrus.WithFields(logrus.Fields{
			"module": "ChannelsImpl",
		}).Error("channel id is required")
	}
	c.channels.Store(channel.ID(), channel)
}

func (c *ChannelsImpl) Remove(id string) {
	c.channels.Delete(id)
}

func (c *ChannelsImpl) Get(id string) (Channel, bool) {
	if id == "" {
		logrus.WithFields(logrus.Fields{
			"module": "ChannelsImpl",
		}).Error("channel id is required")
	}
	val, ok := c.channels.Load(id)
	if !ok {
		return nil, false
	}
	return val.(Channel), true
}

func (c *ChannelsImpl) All() []Channel {
	arr := make([]Channel, 0)
	c.channels.Range(func(key, value interface{}) bool {
		arr = append(arr, value.(Channel))
		return true
	})
	return arr
}
