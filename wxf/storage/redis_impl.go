package storage

import (
	"github.com/wangxuefeng90923/wxf"
	"github.com/wangxuefeng90923/wxf/wire/pkt"
	"time"
)

const (
	LocationExpired = time.Hour * 48
)

type RedisStorage struct {
}

func (r *RedisStorage) Add(session *pkt.Session) error {
	//TODO implement me
	panic("implement me")
}

func (r *RedisStorage) Delete(account string, channleId string) error {
	//TODO implement me
	panic("implement me")
}

func (r *RedisStorage) Get(channelId string) (*pkt.Session, error) {
	//TODO implement me
	panic("implement me")
}

func (r *RedisStorage) GetLocations(account ...string) ([]*wxf.Location, error) {
	//TODO implement me
	panic("implement me")
}

func (r *RedisStorage) GetLocation(account string, device string) (*wxf.Location, error) {
	//TODO implement me
	panic("implement me")
}
