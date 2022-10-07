package container

import (
	"github.com/wangxuefeng90923/wxf"
	"github.com/wangxuefeng90923/wxf/wire/pkt"
	"hash/crc32"
)

func HashCode(key string) int {
	hash32 := crc32.NewIEEE()
	_, _ = hash32.Write([]byte(key))
	return int(hash32.Sum32())
}

type Selector interface {
	Lookup(header *pkt.Header, services []wxf.Service) string
}
