package model

import (
	"hackathon/sharedmodel"

	"github.com/zeromicro/go-zero/core/stores/cache"
)

type AllowAppsModel interface {
	sharedmodel.AllowAppsModel
}

type defaultAllowAppsModel struct {
	sharedmodel.AllowAppsModel
}

func NewAllowAppsModel(url string, db string, c cache.CacheConf) AllowAppsModel {
	return &defaultAllowAppsModel{
		AllowAppsModel: sharedmodel.NewAllowAppsModel(url, db, c),
	}
}
