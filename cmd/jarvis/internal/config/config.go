package config

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf

	Mongo          MongoConfig     // mongo 相关配置
	BizRedisConfig redis.RedisConf // biz 缓存，带有业务属性的缓存

	Kodo        *KodoConfig     // Kodo 鉴权配置
	CacheConfig cache.CacheConf // cache，纯 DB 缓存
}

type KodoConfig struct {
	AccessKey string
	SecretKey string
	Bucket    string
}

type MongoConfig struct {
	Url string
	DB  string `json:",default=jarvis"`
}
