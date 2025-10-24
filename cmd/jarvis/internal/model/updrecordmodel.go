package model

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"hackathon/sharedmodel"

	cachec "github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/monc"
)

type AppChange struct {
	Previous string `bson:"previous" json:"previous"` //上一版本
	Latest   string `bson:"latest"   json:"latest"`   //最新版本
}

type UpdRecordTemplate struct {
	ID        string                `bson:"_id"            json:"id"`
	AppChange map[string]*AppChange `bson:"appChange"      json:"appChange"`      // 升级变更
	Desc      string                `bson:"desc,omitempty" json:"desc,omitempty"` // task 描述
	UpdConf   string                `bson:"updConf"        json:"updConf"`        //升级配置
	CreateAt  time.Time             `bson:"createAt"       json:"createAt"`
	UpdateAt  time.Time             `bson:"updateAt"       json:"updateAt"`
}

func NewUpdRecordID() string {
	return primitive.NewObjectID().Hex()
}

type UpdRecordModel interface {
	Insert(ctx context.Context, data *UpdRecordTemplate) error
	FindOne(ctx context.Context, id string) (*UpdRecordTemplate, error)
	Update(ctx context.Context, data *UpdRecordTemplate) error
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, cond UpdRecordListCond) ([]*UpdRecordTemplate, int, error)
	// only for test
	Drop(ctx context.Context) error
}

type defaultUpdRecordModel struct {
	mc *monc.Model
}

func NewUpdRecordModel(url string, db string, c cachec.CacheConf) UpdRecordModel {
	return &defaultUpdRecordModel{
		mc: monc.MustNewModel(url, db, "updRecord", c),
	}
}

var prefixUpdRecordCacheKey = "cache:UpdRecord:"

func (m *defaultUpdRecordModel) key(id string) string {
	return fmt.Sprintf("%s%s", prefixUpdRecordCacheKey, id)
}

func (m *defaultUpdRecordModel) Insert(ctx context.Context, data *UpdRecordTemplate) error {

	data.CreateAt = time.Now()
	data.UpdateAt = time.Now()
	_, err := m.mc.InsertOneNoCache(ctx, data)
	return err
}

func (m *defaultUpdRecordModel) FindOne(ctx context.Context, id string) (*UpdRecordTemplate, error) {
	var r UpdRecordTemplate
	filter := bson.M{"_id": id}
	err := m.mc.FindOneNoCache(ctx, &r, filter)
	return &r, err
}

func (m *defaultUpdRecordModel) Update(ctx context.Context, data *UpdRecordTemplate) error {
	data.UpdateAt = time.Now()
	_, err := m.mc.ReplaceOneNoCache(ctx, bson.M{"_id": data.ID}, data)
	return err
}

func (m *defaultUpdRecordModel) Delete(ctx context.Context, id string) error {
	_, err := m.mc.DeleteOne(ctx, m.key(id), bson.M{"_id": id})
	return err
}

type UpdRecordListCond struct {
	IDs             []string // ID
	Sort            bson.D   // 排序方式
	CreateTimeRange *sharedmodel.TimeRange
	Page            int
	Size            int
}

func (c *UpdRecordListCond) generateCond() bson.M {
	m := bson.M{}

	if len(c.IDs) > 0 {
		m["_id"] = bson.M{"$in": c.IDs}
	}

	if c.Page < 1 {
		c.Page = 1
	}
	if c.Size > 1000 {
		c.Size = 1000
	}

	if c.Sort == nil {
		c.Sort = bson.D{{Key: "createAt", Value: -1}}
	}

	if c.CreateTimeRange != nil {
		m["createAt"] = bson.M{"$gte": c.CreateTimeRange.Start, "$lte": c.CreateTimeRange.End}
	}
	return m
}

func (m *defaultUpdRecordModel) Search(ctx context.Context, cond UpdRecordListCond) ([]*UpdRecordTemplate, int, error) {
	var r []*UpdRecordTemplate
	option := options.Find()
	query := cond.generateCond()
	option.SetSort(cond.Sort)
	option.SetLimit(int64(cond.Size))
	option.SetSkip(int64((cond.Page - 1) * cond.Size))

	if err := m.mc.Find(ctx, &r, query, option); err != nil {
		return nil, 0, err
	}

	c, err := m.mc.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	return r, int(c), nil
}

// Drop only for test
func (d *defaultUpdRecordModel) Drop(ctx context.Context) error {
	return d.mc.Drop(ctx)
}
