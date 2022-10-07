package container

import (
	"github.com/sirupsen/logrus"
	"github.com/wangxuefeng90923/wxf"
	"sync"
)

type ClientMap interface {
	Add(wxf.Client)
	Remove(id string)
	Get(id string) (wxf.Client, bool)
	Services(...string) []wxf.Service
}

type ClientsImpl struct {
	clients *sync.Map
}

func NewClients(num int) ClientMap {
	return &ClientsImpl{clients: new(sync.Map)}
}

func (cli *ClientsImpl) Add(client wxf.Client) {
	if client.ID() == "" {
		logrus.WithFields(logrus.Fields{
			"module": "ClientsImpl",
		}).Error("client id is required")
	}
	cli.clients.Store(client.ID(), client)
}

func (cli *ClientsImpl) Remove(id string) {
	cli.clients.Delete(id)
}

func (cli *ClientsImpl) Get(id string) (wxf.Client, bool) {
	if id == "" {
		logrus.WithFields(logrus.Fields{
			"module": "ClientsImpl",
		}).Error("client id is required")
	}

	val, ok := cli.clients.Load(id)
	if !ok {
		return nil, false
	}
	return val.(wxf.Client), true
}

func (cli *ClientsImpl) Services(kvs ...string) []wxf.Service {
	kvLen := len(kvs)
	if kvLen != 0 && kvLen != 2 {
		return nil
	}
	arr := make([]wxf.Service, 0)
	cli.clients.Range(func(key, value any) bool {
		srv := value.(wxf.Service)
		if kvLen > 0 && srv.GetMeta()[kvs[0]] != kvs[1] {
			return true
		}
		arr = append(arr, srv)
		return true
	})
	return arr
}
