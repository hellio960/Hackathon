package sharedmodel

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/stores/mon"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"hackathon/common/pointerutil"
)

const (
	CollectionGrayNodes = "grayNodes"
)

type GrayNode struct {
	ID        string    `bson:"_id" json:"id"`
	ReleaseID string    `bson:"releaseId" json:"releaseId"`
	NodeId    string    `bson:"nodeId" json:"nodeId"`
	CreateAt  time.Time `bson:"createAt" json:"createAt"`
}

type GrayNodesModel interface {
	Upsert(ctx context.Context, data *GrayNode) error
	UpsertBulk(ctx context.Context, datas []*GrayNode) error
	Find(ctx context.Context, releaseID string) ([]GrayNode, error)
	Search(ctx context.Context, cond *GrayNodesSearchCond) ([]*GrayNode, string, int, error)
	DeletePartNodesInRelease(ctx context.Context, releaseID string, nodeIds []string) error
	Drop(ctx context.Context) error
}

type defaultGrayNodesModel struct {
	model *mon.Model
}

type GrayNodesSearchCond struct {
	PageParam
	FieldsCond
	ReleaseID string
	NoCount   bool
}

func (c *GrayNodesSearchCond) generateCond() bson.M {
	filter := bson.M{}
	if c.ReleaseID != "" {
		filter["releaseId"] = c.ReleaseID
	}

	return filter
}

func (c *GrayNodesSearchCond) ensureSearchMarkField() {
	if len(c.FieldsCond) == 0 {
		return
	}

	var exist bool
	for _, field := range c.FieldsCond {
		if field == "_id" {
			exist = true
			break
		}
	}
	if !exist {
		c.FieldsCond = append(c.FieldsCond, "_id")
	}
}

func NewGrayNodesModel(url string, db string) GrayNodesModel {
	return &defaultGrayNodesModel{
		model: mon.MustNewModel(url, db, CollectionGrayNodes),
	}
}

func (m *defaultGrayNodesModel) Upsert(ctx context.Context, data *GrayNode) error {
	if data.ID == "" {
		data.ID = primitive.NewObjectID().Hex()
	}
	if data.CreateAt.IsZero() {
		data.CreateAt = time.Now()
	}
	_, err := m.model.UpdateOne(ctx, bson.M{"_id": data.ID}, bson.M{"$set": data}, &options.UpdateOptions{Upsert: pointerutil.Pointer(true)})
	return err
}

func (m *defaultGrayNodesModel) UpsertBulk(ctx context.Context, datas []*GrayNode) error {
	var updateModels []mongo.WriteModel
	for _, node := range datas {
		if node.CreateAt.IsZero() {
			node.CreateAt = time.Now()
		}
		if node.ID == "" {
			node.ID = primitive.NewObjectID().Hex()
		}
		updateModels = append(updateModels, mongo.NewUpdateOneModel().
			SetFilter(bson.M{"_id": node.ID}).
			SetUpdate(bson.M{"$set": node}).
			SetUpsert(true))
	}
	_, err := m.model.BulkWrite(ctx, updateModels)
	return err
}

func (m *defaultGrayNodesModel) Find(ctx context.Context, releaseID string) ([]GrayNode, error) {
	var nodes []GrayNode
	err := m.model.Find(ctx, &nodes, bson.M{"releaseId": releaseID})
	return nodes, err
}

func (m *defaultGrayNodesModel) Drop(ctx context.Context) error {
	return m.model.Drop(ctx)
}

// 适用于删除发布任务下的部分灰度节点
func (m *defaultGrayNodesModel) DeletePartNodesInRelease(ctx context.Context, releaseID string, nodeIds []string) error {
	_, err := m.model.DeleteMany(ctx, bson.M{"releaseId": releaseID, "nodeId": bson.M{"$in": nodeIds}})
	return err
}

func (m *defaultGrayNodesModel) Search(ctx context.Context, cond *GrayNodesSearchCond) ([]*GrayNode, string, int, error) {
	if cond.IsMarkPage() {
		cond.PageParam.SortKey = cond.MarkSortKey
		cond.PageParam.SortType = cond.MarkSortType
		cond.ensureSearchMarkField()
	}

	option := cond.GeneratePageOption()
	cond.SetProjection(option)

	filter := cond.generateCond()
	cond.MarkCond(filter)

	var nodes []*GrayNode
	err := m.model.Find(ctx, &nodes, filter, option)

	var newMark string
	if len(nodes) > 0 && cond.IsMarkPage() && len(nodes) >= cond.Size {
		// 没有返回或者不是mark或者返回的内容已经小于size要求(最后一页)，不返回mark
		newMark = nodes[len(nodes)-1].ID
	}

	var cnt int64
	if !cond.NoCount {
		cnt, err = m.model.CountDocuments(ctx, filter)
		if err != nil {
			return nil, newMark, 0, err
		}
	}

	return nodes, newMark, int(cnt), err
}
