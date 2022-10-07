package consul

import (
	"github.com/stretchr/testify/assert"
	"github.com/wangxuefeng90923/wxf"
	"github.com/wangxuefeng90923/wxf/naming"
	"sync"
	"testing"
	"time"
)

func Test_Naming(t *testing.T) {
	ns, err := NewNaming("localhost:8500")
	assert.Nil(t, err)

	_ = ns.Deregister("test_1")
	_ = ns.Deregister("test_2")

	serviceName := "for_test"
	err = ns.Register(&naming.DefaultService{
		Id:       "test_1",
		Name:     serviceName,
		Address:  "localhost",
		Port:     8000,
		Protocol: "ws",
		Tags:     []string{"tag1", "gate"},
	})
	assert.Nil(t, err)

	servs, err := ns.Find(serviceName)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(servs))
	t.Log(servs)

	wg := sync.WaitGroup{}
	wg.Add(1)

	_ = ns.Subscribe(serviceName, func(services []wxf.ServiceRegistration) {
		t.Log(len(services))

		assert.Equal(t, 2, len(services))
		assert.Equal(t, "test_2", services[1].ServiceID())
		wg.Done()
	})
	time.Sleep(time.Second)

	err = ns.Register(&naming.DefaultService{
		Id:        "test_2",
		Name:      serviceName,
		Namespace: "",
		Address:   "localhost",
		Port:      8001,
		Protocol:  "ws",
		Tags:      []string{"tab2", "gate"},
	})
	assert.Nil(t, err)

	// 等 Watch 回调中的方法执行完成
	wg.Wait()

	_ = ns.Unsubscribe(serviceName)

	// 5. 服务发现
	servs, _ = ns.Find(serviceName, "gate")
	assert.Equal(t, 2, len(servs)) // <-- 必须有两个

	// 6. 服务发现, 验证tag查询
	servs, _ = ns.Find(serviceName, "tab2")
	assert.Equal(t, 1, len(servs)) // <-- 必须有1个
	assert.Equal(t, "test_2", servs[0].ServiceID())

	// 7. 注销test_2
	err = ns.Deregister("test_2")
	assert.Nil(t, err)

	// 8. 服务发现
	servs, err = ns.Find(serviceName)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(servs))
	assert.Equal(t, "test_1", servs[0].ServiceID())

	// 9. 注销test_1
	err = ns.Deregister("test_1")
	assert.Nil(t, err)
}
