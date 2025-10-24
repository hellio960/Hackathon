package sharedmodel

import (
	"context"
	"fmt"
	"time"

	cachec "github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AllowAppsTemplate struct {
	ID       string    `bson:"_id"            json:"id"`
	Name     string    `bson:"name"           json:"name"`
	NodeType string    `bson:"nodeType"        json:"nodeType"`
	Path     string    `bson:"path"        json:"path"` // 包所在目录
	Desc     string    `bson:"desc,omitempty" json:"desc,omitempty"`
	Operator string    `bson:"operator"       json:"operator"`
	CreateAt time.Time `bson:"createAt"       json:"createAt"`
	UpdateAt time.Time `bson:"updateAt"       json:"updateAt"`
}

func NewAllowAppID() string {
	return primitive.NewObjectID().Hex()
}

type AllowAppsModel interface {
	Insert(ctx context.Context, data *AllowAppsTemplate) error
	FindByID(ctx context.Context, id string) (*AllowAppsTemplate, error)
	FindByName(ctx context.Context, nodeType, name string) (*AllowAppsTemplate, error)
	Update(ctx context.Context, data *AllowAppsTemplate) error
	SetAppPathAndDesc(ctx context.Context, id, fileDir, desc string) error
	DeleteByID(ctx context.Context, id string) error
	DeleteByName(ctx context.Context, nodeType, name string) error
	Search(ctx context.Context, cond AllowAppsListCond) ([]*AllowAppsTemplate, int, error)
	// only for test
	Drop(ctx context.Context) error
}

type DefaultAllowAppsModel struct {
	mc *monc.Model
}

func NewAllowAppsModel(url string, db string, c cachec.CacheConf) AllowAppsModel {
	return &DefaultAllowAppsModel{
		mc: monc.MustNewModel(url, db, "allowApps", c),
	}
}

var prefixAllowAppsCacheKey = "cache:AllowApps:"

type AllowAppsListCond struct {
	IDs             []string // ID
	Names           []string
	NodeTypes       []string
	Sort            bson.D // 排序方式
	CreateTimeRange *TimeRange
	Page            int
	Size            int
}

func (c *AllowAppsListCond) generateCond() bson.M {
	m := bson.M{}

	if len(c.IDs) > 0 {
		m["_id"] = bson.M{"$in": c.IDs}
	}

	if len(c.Names) > 0 {
		m["name"] = bson.M{"$in": c.Names}
	}

	if len(c.NodeTypes) > 0 {
		m["nodeType"] = bson.M{"$in": c.NodeTypes}
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

func (m *DefaultAllowAppsModel) key(id string) string {
	return fmt.Sprintf("%s%s", prefixAllowAppsCacheKey, id)
}

func (m *DefaultAllowAppsModel) Insert(ctx context.Context, data *AllowAppsTemplate) error {

	if data.ID == "" {
		data.ID = primitive.NewObjectID().Hex()
	}
	if data.CreateAt.IsZero() {
		data.CreateAt = time.Now()
	}
	if data.UpdateAt.IsZero() {
		data.UpdateAt = time.Now()
	}
	_, err := m.mc.InsertOneNoCache(ctx, data)
	return err
}

func (m *DefaultAllowAppsModel) FindByID(ctx context.Context, id string) (*AllowAppsTemplate, error) {
	var r AllowAppsTemplate
	filter := bson.M{"_id": id}
	err := m.mc.FindOneNoCache(ctx, &r, filter)
	return &r, err
}

func (m *DefaultAllowAppsModel) FindByName(ctx context.Context, nodeType, name string) (*AllowAppsTemplate, error) {
	var r AllowAppsTemplate
	filter := bson.M{"name": name, "nodeType": nodeType}
	err := m.mc.FindOneNoCache(ctx, &r, filter)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &r, err
}

func (m *DefaultAllowAppsModel) Update(ctx context.Context, data *AllowAppsTemplate) error {
	data.UpdateAt = time.Now()
	_, err := m.mc.ReplaceOneNoCache(ctx, bson.M{"_id": data.ID}, data)
	return err
}

func (m *DefaultAllowAppsModel) SetAppPathAndDesc(ctx context.Context, id, path, desc string) error {
	_, err := m.mc.UpdateOne(ctx, m.key(id), bson.M{"_id": id}, bson.M{"$set": bson.M{"path": path, "desc": desc, "updateAt": time.Now()}})
	return err
}

func (m *DefaultAllowAppsModel) DeleteByID(ctx context.Context, id string) error {
	_, err := m.mc.DeleteOne(ctx, m.key(id), bson.M{"_id": id})
	return err
}

func (m *DefaultAllowAppsModel) DeleteByName(ctx context.Context, nodeType, name string) error {
	_, err := m.mc.DeleteOne(ctx, m.key(name), bson.M{"name": name, "nodeType": nodeType})
	return err
}

// Drop only for test
func (d *DefaultAllowAppsModel) Drop(ctx context.Context) error {
	return d.mc.Drop(ctx)
}

func (m *DefaultAllowAppsModel) Search(ctx context.Context, cond AllowAppsListCond) ([]*AllowAppsTemplate, int, error) {
	var r []*AllowAppsTemplate
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
