package conf

import (
	"encoding/json"
	"fmt"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/wangxuefeng90923/wxf"
	"strings"
)

type Server struct {
}

type Config struct {
	ServiceID     string
	ServiceName   string
	Listen        string `default:":8005"`
	MonitorPort   int    `default:"8006"`
	PublicAddress string
	PublicPort    int `default:"8005"`
	Tags          []string
	ConsulURL     string
}

func (c Config) String() string {
	bts, _ := json.Marshal(c)
	return string(bts)
}

func Init(file string) (*Config, error) {
	viper.SetConfigFile(file)
	viper.AddConfigPath(".")

	var config Config
	err := envconfig.Process("wxf", &config)
	if err != nil {
		return nil, err
	}

	if err := viper.ReadInConfig(); err != nil {
		logrus.Warn(err)
	} else {
		if err := viper.Unmarshal(&config); err != nil {
			return nil, err
		}
	}
	if config.ServiceID == "" {
		localIP := wxf.GetLocalIP()
		config.ServiceID = fmt.Sprintf("server_%s", strings.ReplaceAll(localIP, ".", ""))
	}
	if config.PublicAddress == "" {
		config.PublicAddress = wxf.GetLocalIP()
	}
	logrus.Info(config)
	return &config, nil
}
