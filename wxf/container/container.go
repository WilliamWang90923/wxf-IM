package container

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/wangxuefeng90923/wxf"
	"github.com/wangxuefeng90923/wxf/naming"
	"github.com/wangxuefeng90923/wxf/tcp"
	"github.com/wangxuefeng90923/wxf/wire"
	"github.com/wangxuefeng90923/wxf/wire/pkt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	stateUninitialized = iota
	stateInitialized
	stateStarted
	stateClosed
)

const (
	StateYoung = "young"
	StateAdult = "adult"
)

const (
	KeyServiceState = "service_state"
)

type Container struct {
	sync.RWMutex
	Naming     naming.Naming
	Srv        wxf.Server
	state      uint32
	srvClients map[string]ClientMap
	selector   Selector
	dialer     wxf.Dialer
	deps       map[string]struct{}
}

var log = logrus.WithField("module", "container")

var c = &Container{
	state:    0,
	selector: &HashSelector{},
	deps:     make(map[string]struct{}),
}

func Default() *Container {
	return c
}

func Init(srv wxf.Server, deps ...string) error {
	if !atomic.CompareAndSwapUint32(&c.state, stateUninitialized, stateInitialized) {
		return errors.New("has Initialized")
	}
	c.Srv = srv
	for _, dep := range deps {
		if _, ok := c.deps[dep]; ok {
			continue
		}
		c.deps[dep] = struct{}{}
	}
	log.WithField("func", "Init").
		Infof("srv %s:%s - deps %v", srv.ServiceID(), srv.ServiceName(), c.deps)
	c.srvClients = make(map[string]ClientMap, len(deps))
	return nil
}

func Start() error {
	if c.Naming == nil {
		return fmt.Errorf("naming is nil")
	}
	if !atomic.CompareAndSwapUint32(&c.state, stateInitialized, stateStarted) {
		return errors.New("has started")
	}
	// 1.start Server
	go func(srv wxf.Server) {
		err := srv.Start()
		if err != nil {
			log.Errorln(err)
		}
	}(c.Srv)
	// 2. connect to dependent services
	for service := range c.deps {
		go func(service string) {
			err := connectToService(service)
			if err != nil {
				log.Errorln(err)
			}
		}(service)
	}
	// 3. service registration
	if c.Srv.PublicAddress() != "" && c.Srv.PublicPort() != 0 {
		err := c.Naming.Register(c.Srv)
		if err != nil {
			log.Errorln(err)
		}
	}
	// wait for quit signal of system
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	log.Infoln("shutdown", <-c)
	// 4. 退出
	return shutdown()
}

// Push (logic service downstream) push message to server(gateway)
func Push(server string, p *pkt.LogicPkt) error {
	p.AddStringMeta(wire.MetaDestServer, server)
	return c.Srv.Push(server, pkt.Marshal(p))
}

// Forward (gateway service upstream) forward message to server(logic service)
func Forward(serviceName string, packet *pkt.LogicPkt) error {
	if packet == nil {
		return errors.New("packet is nil")
	}
	if packet.Command == "" {
		return errors.New("command is empty in packet")
	}
	if packet.ChannelId == "" {
		return errors.New("ChannelId is empty in packet")
	}
	return ForwardWithSelector(serviceName, packet, c.selector)
}

func ForwardWithSelector(serviceName string, packet *pkt.LogicPkt, selector Selector) error {
	cli, err := lookup(serviceName, &packet.Header, selector)
	if err != nil {
		return err
	}
	packet.AddStringMeta(wire.MetaDestServer, c.Srv.ServiceID())
	log.Debugf("forward message to %v with %s", cli.ID(), &packet.Header)
	return cli.Send(pkt.Marshal(packet))
}

func lookup(serviceName string, header *pkt.Header, selector Selector) (wxf.Client, error) {
	clients, ok := c.srvClients[serviceName]
	if !ok {
		return nil, fmt.Errorf("service %s not found", serviceName)
	}
	// only get service with state being StateAdult
	srvs := clients.Services(KeyServiceState, StateAdult)
	if len(srvs) == 0 {
		return nil, fmt.Errorf("no services found for %s", serviceName)
	}
	id := selector.Lookup(header, srvs)
	if cli, ok := clients.Get(id); ok {
		return cli, nil
	}
	return nil, fmt.Errorf("no client found")
}

// PushMessage (gateway service downstream) push message to Channel
func pushMessage(packet *pkt.LogicPkt) error {
	server, _ := packet.GetMeta(wire.MetaDestServer)
	if server != c.Srv.ServiceID() {
		return fmt.Errorf("dest_server is incorrect, %s != %s",
			server, c.Srv.ServiceID())
	}
	channels, ok := packet.GetMeta(wire.MetaDestChannels)
	if !ok {
		return fmt.Errorf("dest_channels is nil")
	}
	channelIds := strings.Split(channels.(string), ",")
	packet.DelMeta(wire.MetaDestServer)
	packet.DelMeta(wire.MetaDestChannels)
	payload := pkt.Marshal(packet)
	log.Debugf("Push to %v %v", channelIds, packet)

	for _, channel := range channelIds {
		err := c.Srv.Push(channel, payload)
		if err != nil {
			log.Debug(err)
		}
	}
	return nil
}

func shutdown() error {
	if !atomic.CompareAndSwapUint32(&c.state, stateStarted, stateClosed) {
		return errors.New("has closed")
	}
	ctx, cancelFunc := context.WithTimeout(context.TODO(), time.Second*10)
	defer cancelFunc()
	err := c.Srv.Shutdown(ctx)
	if err != nil {
		log.Error(err)
	}
	err = c.Naming.Deregister(c.Srv.ServiceID())
	if err != nil {
		log.Error(err)
	}
	for dep := range c.deps {
		_ = c.Naming.Unsubscribe(dep)
	}
	log.Infoln("shutdown")
	return nil
}

func connectToService(serviceName string) error {
	clients := NewClients(10)
	c.srvClients[serviceName] = clients
	// 1. watch a new added service
	delay := time.Second * 10
	err := c.Naming.Subscribe(serviceName, func(services []wxf.ServiceRegistration) {
		for _, service := range services {
			if _, ok := clients.Get(service.ServiceID()); ok {
				continue
			}
			log.WithField("func", "connectToService").Infof("Watch a new service: %v", service)
			service.GetMeta()[KeyServiceState] = StateYoung
			go func(service wxf.ServiceRegistration) {
				time.Sleep(delay)
				service.GetMeta()[KeyServiceState] = StateAdult
			}(service)

			_, err := buildClient(clients, service)
			if err != nil {
				logrus.Warn(err)
			}
		}
	})
	if err != nil {
		return err
	}
	// 2. find existed services
	services, err := c.Naming.Find(serviceName)
	if err != nil {
		return err
	}
	log.Info("find service: ", services)
	for _, service := range services {
		service.GetMeta()[KeyServiceState] = StateAdult
		_, err := buildClient(clients, service)
		if err != nil {
			logrus.Warn(err)
		}
	}
	return nil
}

// BuildClient: connect to the service after being discovered
func buildClient(clients ClientMap, service wxf.ServiceRegistration) (wxf.Client, error) {
	c.Lock()
	defer c.Unlock()
	var (
		id   = service.ServiceID()
		name = service.ServiceName()
		meta = service.GetMeta()
	)
	if _, ok := clients.Get(id); ok {
		return nil, nil
	}
	// tcp allowed only in between services
	if service.GetProtocol() != string(wire.ProtocolTCP) {
		return nil, fmt.Errorf("unexpected service Protocol: %s", service.GetProtocol())
	}
	cli := tcp.NewClientWithProps(id, name, meta, tcp.ClientOptions{
		Heartbeat: wxf.DefaultHeartbeat,
		ReadWait:  wxf.DefaultReadWait,
		WriteWait: wxf.DefaultWriteWait,
	})
	if c.dialer == nil {
		return nil, fmt.Errorf("dialer is nil")
	}
	cli.SetDialer(c.dialer)
	err := cli.Connect(service.DialURL())
	if err != nil {
		return nil, err
	}
	// read messages
	go func(cli wxf.Client) {
		readLoop(cli)
	}(cli)
	clients.Add(cli)
	return cli, nil
}

func readLoop(cli wxf.Client) error {
	logrus.WithFields(logrus.Fields{
		"module": "container",
		"func":   "readLoop",
	})
	log.Infof("readLoop started of %s %s", cli.ID(), cli.Name())
	for {
		frame, err := cli.Read()
		if err != nil {
			return err
		}
		if frame.GetOpCode() != wxf.OpBinary {
			continue
		}
		buf := bytes.NewBuffer(frame.GetPayload())
		logicPkt, err := pkt.MustReadLogicPkt(buf)
		if err != nil {
			log.Info(err)
			continue
		}
		err = pushMessage(logicPkt)
		if err != nil {
			log.Info(err)
		}
	}
}

func SetDialer(dialer wxf.Dialer) {
	c.dialer = dialer
}

func SetSelector(selector Selector) {
	c.selector = selector
}

func SetServiceNaming(nm naming.Naming) {
	c.Naming = nm
}
