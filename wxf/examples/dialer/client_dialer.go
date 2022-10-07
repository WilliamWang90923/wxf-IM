package dialer

import (
	"bytes"
	"context"
	"fmt"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/sirupsen/logrus"
	"github.com/wangxuefeng90923/wxf"
	"github.com/wangxuefeng90923/wxf/wire"
	"github.com/wangxuefeng90923/wxf/wire/pkt"
	"github.com/wangxuefeng90923/wxf/wire/token"
	"net"
	"time"
)

type ClientDialer struct {
}

func (d *ClientDialer) DialAndHandShake(ctx wxf.DialerContext) (net.Conn, error) {
	logrus.Info("DialAndHandshake called")
	// 1. 拨号
	conn, _, _, err := ws.Dial(context.TODO(), ctx.Address)
	if err != nil {
		return nil, err
	}
	// 2. 直接使用封装的JWT包生成一个token
	tk, err := token.Generate(token.DefaultSecret, &token.Token{
		Account: ctx.Id,
		App:     "kim",
		Exp:     time.Now().AddDate(0, 0, 1).Unix(),
	})
	if err != nil {
		return nil, err
	}
	// 3. 发送一条CommandLoginSignIn消息
	loginreq := pkt.New(wire.CommandLoginSignIn).WriteBody(&pkt.LoginReq{
		Token: tk,
	})
	err = wsutil.WriteClientBinary(conn, pkt.Marshal(loginreq))
	if err != nil {
		return nil, err
	}

	// wait resp
	logrus.Info("waiting for login response")
	_ = conn.SetReadDeadline(time.Now().Add(ctx.Timeout))
	frame, err := ws.ReadFrame(conn)
	if err != nil {
		return nil, err
	}
	ack, err := pkt.MustReadLogicPkt(bytes.NewBuffer(frame.Payload))
	if err != nil {
		return nil, err
	}
	// 4. 判断是否登陆成功
	if ack.Status != pkt.Status_Success {
		return nil, fmt.Errorf("login failed: %v", &ack.Header)
	}
	var resp = new(pkt.LoginResp)
	_ = ack.ReadBody(resp)

	logrus.Info("logined ", resp.GetChannelId())
	return conn, nil
}
