package serv

import (
	"bytes"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/wangxuefeng90923/wxf"
	"github.com/wangxuefeng90923/wxf/container"
	"github.com/wangxuefeng90923/wxf/wire"
	"github.com/wangxuefeng90923/wxf/wire/pkt"
	"github.com/wangxuefeng90923/wxf/wire/token"
	"regexp"
	"time"
)

const (
	MetaKeyApp     = "app"
	MetaKeyAccount = "account"
)

var log = logrus.WithFields(logrus.Fields{
	"service": "gateway",
	"pkg":     "serv",
})

type Handler struct {
	ServiceID string
}

func (x *Handler) Disconnect(id string) error {
	log.Infof("disconnect %s", id)
	logoutPkt := pkt.New(wire.CommandLoginSignOut, pkt.WithChannel(id))
	err := container.Forward(wire.SNLogin, logoutPkt)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"module": "handler",
			"id":     id,
		}).Error(err)
	}
	return nil
}

func (x *Handler) Receive(agent wxf.Agent, payload []byte) {
	buf := bytes.NewBuffer(payload)
	packet, err := pkt.Read(buf)
	if err != nil {
		return
	}
	// case BasicPkt, heartbeat handling
	if basicPkt, ok := packet.(*pkt.BasicPkt); ok {
		if basicPkt.Code == pkt.CodePing {
			_ = agent.Push(pkt.Marshal(&pkt.BasicPkt{
				Code: pkt.CodePong,
			}))
		}
		return
	}
	// case LogicPkt, transfer to logic service
	if logicPkt, ok := packet.(*pkt.LogicPkt); ok {
		logicPkt.ChannelId = agent.ID()

		err = container.Forward(logicPkt.ServiceName(), logicPkt)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"module": "handler",
				"id":     agent.ID(),
				"cmd":    logicPkt.Command,
				"dest":   logicPkt.Dest,
			}).Error(err)
		}
	}
}

func (x *Handler) Accept(conn wxf.Conn, timeout time.Duration) (string, error) {
	log := logrus.WithFields(logrus.Fields{
		"ServiceID": x.ServiceID,
		"module":    "Handler",
		"handler":   "Accept",
	})
	log.Infoln("enter")
	// 1. read login packet
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	frame, err := conn.ReadFrame()
	if err != nil {
		return "", err
	}

	buf := bytes.NewBuffer(frame.GetPayload())
	req, err := pkt.MustReadLogicPkt(buf)
	if err != nil {
		return "", err
	}
	// 2. it must be a login packet
	if req.Command != wire.CommandLoginSignIn {
		resp := pkt.NewFrom(&req.Header)
		resp.Status = pkt.Status_InvalidCommand
		_ = conn.WriteFrame(wxf.OpBinary, pkt.Marshal(resp))
		return "", fmt.Errorf("acceptor receive a InvalidCommand command")
	}
	// 3. Unmarshal body
	var login pkt.LoginReq
	err = req.ReadBody(&login)
	if err != nil {
		return "", err
	}
	// 4. decode token with DefaultSecret
	tk, err := token.Parse(token.DefaultSecret, login.Token)
	if err != nil {
		// 5. ineffective token, return to SDK an Unauthorized Msg
		resq := pkt.NewFrom(&req.Header)
		resq.Status = pkt.Status_Unauthorized
		_ = conn.WriteFrame(wxf.OpBinary, pkt.Marshal(resq))
		return "", err
	}
	// 6. generate a global unique ChannelID
	id := generateChannelID(x.ServiceID, tk.Account)

	req.ChannelId = id
	req.WriteBody(&pkt.Session{
		ChannelId: id,
		GateId:    x.ServiceID,
		Account:   tk.Account,
		RemoteIP:  getIP(conn.RemoteAddr().String()),
		App:       tk.App,
	})
	// 7. transfer login to Login service
	err = container.Forward(wire.SNLogin, req)
	if err != nil {
		return "", err
	}
	return id, nil
}

var ipExp = regexp.MustCompile(string("\\:[0-9]+$"))

func getIP(remoteAddr string) string {
	if remoteAddr == "" {
		return ""
	}
	return ipExp.ReplaceAllString(remoteAddr, "")
}

func generateChannelID(serviceID, account string) string {
	return fmt.Sprintf("%s_%s_%d", serviceID, account, wire.Seq.Next())
}
