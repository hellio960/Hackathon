package model

import (
	"context"

	cachec "github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"

	"hackathon/sharedmodel"
)

const (
	DetecterFileSizeSysParam = "detecter_file_size"
	JarvisUpdConfigSysParam  = "jarvis_upd_config"
)

type SysParamModel interface {
	sharedmodel.SysParamModel
	Insert(ctx context.Context, data *sharedmodel.SysParam) error
	DeleteByName(ctx context.Context, name string) error
	Drop(ctx context.Context) error // only for test
}

type defaultSysParamModel struct {
	sharedmodel.SysParamModel
	mc *monc.Model
}

func NewSysParamModel(url string, db string, c cachec.CacheConf) SysParamModel {
	return &defaultSysParamModel{
		SysParamModel: sharedmodel.NewSysParamModel(url, db, c),
		mc:            monc.MustNewModel(url, db, "sysParam", c),
	}
}

func (m *defaultSysParamModel) Insert(ctx context.Context, data *sharedmodel.SysParam) error {

	_, err := m.mc.InsertOneNoCache(ctx, data)
	return err
}

func (m *defaultSysParamModel) DeleteByName(ctx context.Context, name string) error {
	_, err := m.mc.DeleteOne(ctx, m.Key(name), bson.M{"name": name})
	return err
}

// Drop only for test
func (d *defaultSysParamModel) Drop(ctx context.Context) error {
	return d.mc.Drop(ctx)
}

func (m *defaultSysParamModel) Key(name string) string {
	return sharedmodel.PrefixSysParamCacheKey + name
}
