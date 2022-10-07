package server

import (
	"context"
	"github.com/sirupsen/logrus"
	"github.com/wangxuefeng90923/wxf"
	"github.com/wangxuefeng90923/wxf/container"
	"github.com/wangxuefeng90923/wxf/naming"
	"github.com/wangxuefeng90923/wxf/naming/consul"
	"github.com/wangxuefeng90923/wxf/services/server/conf"
	"github.com/wangxuefeng90923/wxf/services/server/handler"
	"github.com/wangxuefeng90923/wxf/services/server/serv"
	"github.com/wangxuefeng90923/wxf/tcp"
	"github.com/wangxuefeng90923/wxf/wire"
)

type ServerStartOptions struct {
	config      string
	serviceName string
}

func RunServerStart(ctx context.Context, opts *ServerStartOptions, version string) error {
	config, err := conf.Init(opts.config)
	if err != nil {
		return err
	}
	level, _ := logrus.ParseLevel("trace")
	logrus.SetLevel(level)

	r := wxf.NewRouter()

	loginHandler := handler.NewLoginHandler()
	r.Handle(wire.CommandLoginSignIn, loginHandler.DoSysLogin)
	r.Handle(wire.CommandLoginSignOut, loginHandler.DoSysLogout)

	// TODO: init Redis
	// TODO: session management

	service := &naming.DefaultService{
		Id:       config.ServiceID,
		Name:     opts.serviceName,
		Address:  config.PublicAddress,
		Port:     config.PublicPort,
		Protocol: string(wire.ProtocolTCP),
		Tags:     config.Tags,
	}

	servHandler := &serv.ServeHandler{}

	srv := tcp.NewServer(config.Listen, service)
	srv.SetReadWait(wxf.DefaultReadWait)
	srv.SetAcceptor(servHandler)
	srv.SetMessageListener(servHandler)
	srv.SetStateListener(servHandler)

	if err := container.Init(srv); err != nil {
		return err
	}

	ns, err := consul.NewNaming(config.ConsulURL)
	if err != nil {
		return err
	}
	container.SetServiceNaming(ns)

	return container.Start()
}
