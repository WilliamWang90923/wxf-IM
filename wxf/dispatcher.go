package wxf

import "github.com/wangxuefeng90923/wxf/wire/pkt"

type Dispatcher interface {
	Push(gateway string, channels []string, p *pkt.LogicPkt) error
}
