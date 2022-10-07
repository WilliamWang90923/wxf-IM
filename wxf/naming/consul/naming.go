package consul

import (
	"errors"
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/sirupsen/logrus"
	"github.com/wangxuefeng90923/wxf"
	"github.com/wangxuefeng90923/wxf/naming"
	"sync"
	"time"
)

const (
	KeyProtocol  = "protocol"
	KeyHealthURL = "health_url"
)

type Watch struct {
	Service   string
	Callback  func([]wxf.ServiceRegistration)
	WaitIndex uint64
	Quit      chan struct{}
}

type Naming struct {
	sync.RWMutex
	cli     *api.Client
	watches map[string]*Watch
}

func (n *Naming) Find(serviceName string, tags ...string) ([]wxf.ServiceRegistration, error) {
	services, _, err := n.load(serviceName, 0, tags...)
	if err != nil {
		return nil, err
	}
	return services, nil
}

func (n *Naming) Subscribe(serviceName string, callback func([]wxf.ServiceRegistration)) error {
	n.Lock()
	defer n.Unlock()
	if _, ok := n.watches[serviceName]; ok {
		return errors.New("serviceName has already been registered")
	}
	w := &Watch{
		Service:  serviceName,
		Callback: callback,
		Quit:     make(chan struct{}, 1),
	}
	n.watches[serviceName] = w

	go n.watch(w)
	return nil
}

func (n *Naming) Unsubscribe(serviceName string) error {
	n.Lock()
	defer n.Unlock()
	wh, ok := n.watches[serviceName]
	delete(n.watches, serviceName)

	if ok {
		close(wh.Quit)
	}
	return nil
}

func (n *Naming) Register(s wxf.ServiceRegistration) error {
	reg := &api.AgentServiceRegistration{
		ID:      s.ServiceID(),
		Name:    s.ServiceName(),
		Tags:    s.GetTags(),
		Port:    s.PublicPort(),
		Address: s.PublicAddress(),
		Meta:    s.GetMeta(),
	}
	if reg.Meta == nil {
		reg.Meta = make(map[string]string)
	}
	reg.Meta[KeyProtocol] = s.GetProtocol()

	// consul health check
	healthURL := s.GetMeta()[KeyHealthURL]
	if healthURL != "" {
		check := new(api.AgentServiceCheck)
		check.CheckID = fmt.Sprintf("%s_normal", s.ServiceID())
		check.HTTP = healthURL
		check.Timeout = "1s"
		check.Interval = "10s"
		check.DeregisterCriticalServiceAfter = "20s"
		reg.Check = check
	}

	err := n.cli.Agent().ServiceRegister(reg)
	return err
}

func (n *Naming) Deregister(serviceID string) error {
	return n.cli.Agent().ServiceDeregister(serviceID)
}

func NewNaming(consulUrl string) (naming.Naming, error) {
	conf := api.DefaultConfig()
	conf.Address = consulUrl
	cli, err := api.NewClient(conf)
	if err != nil {
		return nil, err
	}
	return &Naming{
		cli:     cli,
		watches: make(map[string]*Watch, 1),
	}, nil
}

func (n *Naming) load(name string, waitIndex uint64, tags ...string) ([]wxf.ServiceRegistration, *api.QueryMeta, error) {
	opts := &api.QueryOptions{
		UseCache:  true,
		MaxAge:    time.Minute,
		WaitIndex: waitIndex,
	}
	catalogServices, meta, err := n.cli.Catalog().ServiceMultipleTags(name, tags, opts)
	if err != nil {
		return nil, meta, err
	}
	services := make([]wxf.ServiceRegistration, 0, len(catalogServices))
	for _, s := range catalogServices {
		if s.Checks.AggregatedStatus() != api.HealthPassing {
			logrus.Debugf("load service: id:%s name:%s %s:%d Status:%s",
				s.ServiceID, s.ServiceName, s.ServiceAddress, s.ServicePort, s.Checks.AggregatedStatus())
			continue
		}
		services = append(services, &naming.DefaultService{
			Id:       s.ServiceID,
			Name:     s.ServiceName,
			Address:  s.ServiceAddress,
			Port:     s.ServicePort,
			Protocol: s.ServiceMeta[KeyProtocol],
			Tags:     s.ServiceTags,
			Meta:     s.ServiceMeta,
		})
	}
	logrus.Debugf("load services: %v, meta: %v", services, meta)
	return services, meta, nil
}

func (n *Naming) watch(wh *Watch) {
	stopped := false
	doWatch := func(service string, callback func([]wxf.ServiceRegistration)) {
		services, meta, err := n.load(service, wh.WaitIndex)
		if err != nil {
			logrus.Warn(err)
			return
		}
		select {
		case <-wh.Quit:
			stopped = true
			logrus.Infof("watch %s stopped", wh.Service)
			return
		default:
		}
		wh.WaitIndex = meta.LastIndex
		if callback != nil {
			callback(services)
		}
	}
	doWatch(wh.Service, nil)
	for !stopped {
		doWatch(wh.Service, wh.Callback)
	}
}
