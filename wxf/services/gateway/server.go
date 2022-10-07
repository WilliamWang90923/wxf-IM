package gateway

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/wangxuefeng90923/wxf"
	"github.com/wangxuefeng90923/wxf/container"
	"github.com/wangxuefeng90923/wxf/naming"
	"github.com/wangxuefeng90923/wxf/naming/consul"
	"github.com/wangxuefeng90923/wxf/services/gateway/serv"
	"github.com/wangxuefeng90923/wxf/services/server/conf"
	"github.com/wangxuefeng90923/wxf/websocket"
	"github.com/wangxuefeng90923/wxf/wire"
	"time"
)

type ServerStartOptions struct {
	config   string
	protocol string
	route    string
}

func RunServerStart(ctx context.Context, opts *ServerStartOptions, version string) error {
	config, err := conf.Init(opts.config)
	if err != nil {
		return err
	}
	level, _ := logrus.ParseLevel("trace")
	logrus.SetLevel(level)
	handler := &serv.Handler{ServiceID: config.ServiceID}

	var srv wxf.Server
	service := &naming.DefaultService{
		Id:       config.ServiceID,
		Name:     config.ServiceName,
		Address:  config.PublicAddress,
		Port:     config.PublicPort,
		Protocol: opts.protocol,
		Tags:     config.Tags,
	}
	if opts.protocol == "ws" {
		srv = websocket.NewServer(config.Listen, service)
	}
	srv.SetReadWait(time.Minute)
	srv.SetAcceptor(handler)
	srv.SetMessageListener(handler)
	srv.SetStateListener(handler)

	err = container.Init(srv, wire.SNChat, wire.SNLogin)
	if err != nil {
		fmt.Errorf("gateway container fail to init with error: %s", err)
		return err
	}
	ns, err := consul.NewNaming(config.ConsulURL)
	if err != nil {
		return err
	}
	container.SetServiceNaming(ns)
	container.SetDialer(serv.NewDialer(config.ServiceID))
	return container.Start()
}
