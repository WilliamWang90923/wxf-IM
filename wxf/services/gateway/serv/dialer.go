package serv

import (
	"github.com/golang/protobuf/proto"
	"github.com/sirupsen/logrus"
	"github.com/wangxuefeng90923/wxf"
	"github.com/wangxuefeng90923/wxf/tcp"
	"github.com/wangxuefeng90923/wxf/wire/pkt"
	"net"
)

type TcpDialer struct {
	ServiceId string
}

func (d *TcpDialer) DialAndHandShake(ctx wxf.DialerContext) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", ctx.Address, ctx.Timeout)
	if err != nil {
		return nil, err
	}
	req := &pkt.InnerHandshakeReq{ServiceId: d.ServiceId}
	logrus.Infof("send request %v to dial", req)
	bts, err := proto.Marshal(req)
	if err != nil {
		return nil, err
	}
	err = tcp.WriteFrame(conn, wxf.OpBinary, bts)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func NewDialer(serviceId string) wxf.Dialer {
	return &TcpDialer{ServiceId: serviceId}
}
