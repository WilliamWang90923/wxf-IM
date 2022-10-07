package unittest

import (
	"github.com/stretchr/testify/assert"
	"github.com/wangxuefeng90923/wxf"
	"github.com/wangxuefeng90923/wxf/examples/dialer"
	"github.com/wangxuefeng90923/wxf/websocket"
	"testing"
	"time"
)

func login(account string) (wxf.Client, error) {
	cli := websocket.NewClient(account, "unittest", websocket.ClientOptions{})
	cli.SetDialer(&dialer.ClientDialer{})

	err := cli.Connect("ws://localhost:8000")
	if err != nil {
		return nil, err
	}
	return cli, nil
}

func Test_Login(t *testing.T) {
	cli, err := login("test1")
	assert.Nil(t, err)
	time.Sleep(time.Second * 5)
	cli.Close()
}
