package model

import (
	"hackathon/sharedmodel"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/monc"
)

type NodeJoinModel interface {
	sharedmodel.NodeJoinModel
}

type defaultNodeJoinModel struct {
	sharedmodel.NodeJoinModel
	mc *monc.Model
}

func NewNodeJoinModel(url, db string, c cache.CacheConf) NodeJoinModel {
	return &defaultNodeJoinModel{
		NodeJoinModel: sharedmodel.NewNodeInfoJoinModel(url, db, c),
		mc:            monc.MustNewModel(url, db, sharedmodel.CollectionNodeJoin, c),
	}
}
