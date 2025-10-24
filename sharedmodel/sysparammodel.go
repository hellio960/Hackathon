package sharedmodel

import (
	"context"
	"time"

	mopt "go.mongodb.org/mongo-driver/mongo/options"

	cachec "github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	JarvisUpdConf = "jarvis_upd_config"
	AntUpdConf    = "ant_upd_config"
	DroidUpdConf  = "droid_upd_config"
)

var PrefixSysParamCacheKey = "cache:SysParam:"

type (
	SysParam struct {
		Name     string    `bson:"name"               json:"name"`
		Value    string    `bson:"value"              json:"value"`
		Remark   string    `bson:"remark"             json:"remark"`
		CreateAt time.Time `bson:"createAt,omitempty" json:"createAt,omitempty"`
		UpdateAt time.Time `bson:"updateAt,omitempty" json:"updateAt,omitempty"`
	}

	DefaultSysParamModel struct {
		mc *monc.Model
	}
)

type SysParamModel interface {
	Upsert(ctx context.Context, data *SysParam) error
	FindByName(ctx context.Context, name string) (*SysParam, error)
	FindByNamePattern(ctx context.Context, namePrefix string) ([]*SysParam, error)
}

func NewSysParamModel(url string, db string, c cachec.CacheConf) SysParamModel {
	return &DefaultSysParamModel{
		mc: monc.MustNewModel(url, db, "sysParam", c),
	}
}

func (m *DefaultSysParamModel) Upsert(ctx context.Context, data *SysParam) error {
	upsert := true
	_, err := m.mc.UpdateOne(ctx, m.Key(data.Name), bson.M{"name": data.Name}, bson.M{"$set": data}, &mopt.UpdateOptions{Upsert: &upsert})

	return err
}

func (m *DefaultSysParamModel) FindByName(ctx context.Context, name string) (*SysParam, error) {
	var data SysParam

	err := m.mc.FindOne(ctx, m.Key(name), &data, bson.M{"name": name})
	switch err {
	case nil:
		return &data, nil
	case monc.ErrNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}

func (m *DefaultSysParamModel) FindByNamePattern(ctx context.Context, namePattern string) ([]*SysParam, error) {
	var data []*SysParam
	query := bson.M{"name": bson.M{"$regex": namePattern}}

	err := m.mc.Find(ctx, &data, query)
	return data, err
}

func (m *DefaultSysParamModel) Key(name string) string {
	return PrefixSysParamCacheKey + name
}
