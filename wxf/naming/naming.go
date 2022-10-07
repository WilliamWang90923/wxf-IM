package naming

import "github.com/wangxuefeng90923/wxf"

type Naming interface {
	Find(serviceName string, tags ...string) ([]wxf.ServiceRegistration, error)
	Subscribe(string, func([]wxf.ServiceRegistration)) error
	Unsubscribe(string) error
	Register(registration wxf.ServiceRegistration) error
	Deregister(string) error
}
