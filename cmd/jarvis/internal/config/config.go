package config

import (
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf

	DataDir string `json:",default=./data"` // Local storage directory

	Kodo *KodoConfig // Kodo 鉴权配置
}

type KodoConfig struct {
	AccessKey string
	SecretKey string
	Bucket    string
}
