package handler

import (
	"github.com/sirupsen/logrus"
	"github.com/wangxuefeng90923/wxf"
	"github.com/wangxuefeng90923/wxf/wire/pkt"
)

type LoginHandler struct {
}

func NewLoginHandler() *LoginHandler {
	return &LoginHandler{}
}

func (h *LoginHandler) DoSysLogin(ctx wxf.Context) {
	log := logrus.WithField("func", "DoSysLogin")
	var session pkt.Session
	if err := ctx.ReadBody(&session); err != nil {
		_ = ctx.RespWithError(pkt.Status_InvalidCommand, err)
		return
	}

	log.Infof("do login of %v ", session.String())
	// 2. if current account login somewhere else
	old, err := ctx.GetLocation(session.Account, "")
	if err != nil && err != wxf.ErrSessionNil {
		_ = ctx.RespWithError(pkt.Status_SystemException, err)
		return
	}

	if old != nil {
		// 3. notify old account to sign out
		_ = ctx.Dispatch(&pkt.KickoutNotify{ChannelId: old.ChannelId}, old)
	}
	// 4. add to session manager
	err = ctx.Add(&session)
	if err != nil {
		_ = ctx.RespWithError(pkt.Status_SystemException, err)
		return
	}
	// 5. return login success msg
	var resp = &pkt.LoginResp{
		ChannelId: session.ChannelId,
	}
	_ = ctx.Resp(pkt.Status_Success, resp)
}

func (h *LoginHandler) DoSysLogout(ctx wxf.Context) {
	logrus.WithField("func", "DoSysLogout").Infof("do Logout of %s %s ",
		ctx.Session().GetChannelId(), ctx.Session().GetAccount())
	err := ctx.Delete(ctx.Session().GetAccount(), ctx.Session().GetChannelId())
	if err != nil {
		_ = ctx.RespWithError(pkt.Status_SystemException, err)
		return
	}
	_ = ctx.Resp(pkt.Status_Success, nil)
}
