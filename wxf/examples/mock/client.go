package mock

import (
	"context"
	"fmt"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/sirupsen/logrus"
	"github.com/wangxuefeng90923/wxf"
	"github.com/wangxuefeng90923/wxf/tcp"
	"github.com/wangxuefeng90923/wxf/websocket"
	"net"
	"time"
)

type ClientDemo struct {
}

func (c *ClientDemo) Start(userID, protocol, addr string) {
	var cli wxf.Client
	if protocol == "ws" {
		cli = websocket.NewClient(userID, "client", websocket.ClientOptions{})
		// set dialer
		cli.SetDialer(&WebsocketDialer{})
	} else if protocol == "tcp" {
		cli = tcp.NewClient(userID, "client", tcp.ClientOptions{})
		cli.SetDialer(&TcpDialer{})
	}
	err := cli.Connect(addr)
	if err != nil {
		return
	}
	count := 6
	go func() {
		// step3: 发送消息然后退出
		for i := 0; i < count; i++ {
			err := cli.Send([]byte(fmt.Sprintf("hello_%d", i)))
			if err != nil {
				logrus.Error(err)
				return
			}
			time.Sleep(time.Millisecond * 1000)
		}
	}()

	recv := 0
	for {
		frame, err := cli.Read()
		if err != nil {
			logrus.Info(err)
			break
		}
		if frame.GetOpCode() != wxf.OpBinary {
			continue
		}
		recv++
		logrus.Infof("%s receive message [%s]", cli.ID(), frame.GetPayload())
		if recv == count { // 接收完消息
			break
		}
	}
	cli.Close()
}

type WebsocketDialer struct {
}

func (d *WebsocketDialer) DialAndHandShake(ctx wxf.DialerContext) (net.Conn, error) {
	logrus.Info("start ws dial: ", ctx.Address)
	ctxWithTimeout, cancelFunc := context.WithTimeout(context.TODO(), ctx.Timeout)
	defer cancelFunc()

	conn, _, _, err := ws.Dial(ctxWithTimeout, ctx.Address)
	if err != nil {
		return nil, err
	}
	// 2. 发送用户认证信息，示例就是userid
	err = wsutil.WriteClientBinary(conn, []byte(ctx.Id))
	if err != nil {
		return nil, err
	}
	// 3. return conn
	return conn, nil
}

type TcpDialer struct {
}

func (d *TcpDialer) DialAndHandShake(ctx wxf.DialerContext) (net.Conn, error) {
	logrus.Info("start tcp dial: ", ctx.Address)
	// 1 调用net.Dial拨号
	conn, err := net.DialTimeout("tcp", ctx.Address, ctx.Timeout)
	if err != nil {
		return nil, err
	}
	// 2. 发送用户认证信息，示例就是userid
	err = tcp.WriteFrame(conn, wxf.OpBinary, []byte(ctx.Id))
	if err != nil {
		return nil, err
	}
	// 3. return conn
	return conn, nil
}
