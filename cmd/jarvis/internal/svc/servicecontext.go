package svc

import (
	"github.com/zeromicro/go-zero/core/stores/redis"

	"hackathon/cmd/jarvis/internal/config"
	"hackathon/cmd/jarvis/internal/model"
	"hackathon/sharedmodel"
)

type ServiceContext struct {
	Config   config.Config
	BizRedis *redis.Redis

	NodeReleaseModel        sharedmodel.NodeReleaseModel
	AllowAppsModel          sharedmodel.AllowAppsModel
	SysParamModel           model.SysParamModel
	UpdRecordModel          model.UpdRecordModel
	GrayNodesModel          sharedmodel.GrayNodesModel
	NodeReleaseHistoryModel sharedmodel.NodeReleaseHistoryModel
	NodeJoinModel           sharedmodel.NodeJoinModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	bizRedis := c.BizRedisConfig.NewRedis()

	return &ServiceContext{
		Config: c,

		BizRedis: bizRedis,

		NodeReleaseModel:        sharedmodel.NewNodeReleaseModel(c.Mongo.Url, c.Mongo.DB),
		AllowAppsModel:          sharedmodel.NewAllowAppsModel(c.Mongo.Url, c.Mongo.DB, c.CacheConfig),
		SysParamModel:           model.NewSysParamModel(c.Mongo.Url, c.Mongo.DB, c.CacheConfig),
		UpdRecordModel:          model.NewUpdRecordModel(c.Mongo.Url, c.Mongo.DB, c.CacheConfig),
		GrayNodesModel:          sharedmodel.NewGrayNodesModel(c.Mongo.Url, c.Mongo.DB),
		NodeReleaseHistoryModel: sharedmodel.NewNodeReleaseHistoryModel(c.Mongo.Url, c.Mongo.DB),
		NodeJoinModel:           sharedmodel.NewNodeJoinModel(c.Mongo.Url, c.Mongo.DB, c.CacheConfig),
	}
}
