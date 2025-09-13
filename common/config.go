package common

import (
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

var (
	configOnce sync.Once
	config     *Config
)

type Config struct {
	Server       *ServerConfig       `yaml:"server"`
	MQ           *MQConfig           `yaml:"mq"`
	DB           *DBConfig           `yaml:"db"`
	ThirdService *ThirdServiceConfig `yaml:"thirdService"`
	Event        *EventConfig        `yaml:"event"`
}

type ServerConfig struct {
	PublicAddr  string `yaml:"publicAddr"`
	PrivateAddr string `yaml:"privateAddr"`
}

type MQConfig struct {
	Type     string `yaml:"type"`
	NSQDAddr string `yaml:"nsqdAddr"`
}

type WebsocketConfig struct {
	Addr string `yaml:"addr"`
}

type ThirdServiceConfig struct {
	IdentifyServiceAddr string `yaml:"identifyServiceAddr"`
}

type DBConfig struct {
	User   string `yaml:"user"`
	Passwd string `yaml:"passwd"`
	Host   string `yaml:"host"`
	Port   int    `yaml:"port"`
	DBName string `yaml:"dbName"`
}

type EventConfig struct {
	Topics []string `yaml:"topics"`
}

func NewConfig() *Config {
	configOnce.Do(func() {
		content, err := os.ReadFile("config.yaml")
		if err != nil {
			panic(err)
		}

		config = &Config{}
		err = yaml.Unmarshal(content, config)
		if err != nil {
			panic(err)
		}
	})

	return config
}
