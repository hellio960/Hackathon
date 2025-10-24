package svc

import (
	"hackathon/cmd/jarvis/internal/config"
	"hackathon/cmd/jarvis/internal/localstorage"
	"hackathon/cmd/jarvis/internal/model"
	"hackathon/sharedmodel"
)

type ServiceContext struct {
	Config   config.Config
	BizRedis *localstorage.LocalRedis

	NodeReleaseModel        sharedmodel.NodeReleaseModel
	AllowAppsModel          sharedmodel.AllowAppsModel
	SysParamModel           model.SysParamModel
	UpdRecordModel          model.UpdRecordModel
	GrayNodesModel          sharedmodel.GrayNodesModel
	NodeReleaseHistoryModel sharedmodel.NodeReleaseHistoryModel
	NodeJoinModel           sharedmodel.NodeJoinModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	dataDir := c.DataDir
	if dataDir == "" {
		dataDir = "./data"
	}

	return &ServiceContext{
		Config: c,

		BizRedis: localstorage.NewLocalRedis(dataDir),

		NodeReleaseModel:        localstorage.NewLocalNodeReleaseModel(dataDir),
		AllowAppsModel:          localstorage.NewLocalAllowAppsModel(dataDir),
		SysParamModel:           localstorage.NewLocalSysParamModel(dataDir),
		UpdRecordModel:          localstorage.NewLocalUpdRecordModel(dataDir),
		GrayNodesModel:          localstorage.NewLocalGrayNodesModel(dataDir),
		NodeReleaseHistoryModel: localstorage.NewLocalNodeReleaseHistoryModel(dataDir),
		NodeJoinModel:           localstorage.NewLocalNodeJoinModel(dataDir),
	}
}
