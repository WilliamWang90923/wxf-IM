package container

import (
	"github.com/wangxuefeng90923/wxf"
	"github.com/wangxuefeng90923/wxf/wire/pkt"
)

type HashSelector struct {
}

func (s *HashSelector) Lookup(header *pkt.Header, services []wxf.Service) string {
	l := len(services)
	code := HashCode(header.ChannelId)
	return services[code%l].ServiceID()
}
