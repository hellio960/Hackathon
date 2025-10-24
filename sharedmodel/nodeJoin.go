package sharedmodel

import (
	"context"
	"strings"

	cachec "github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	CollectionNodeJoin = "nodeJoin"
)

type (
	NodeJoinField = string

	NodeJoin struct {
		Id string `bson:"_id"            json:"_id"`
	}
	defaultNodeInfoJoinModel struct {
		mc *monc.Model
	}
	NodeJoinModel interface {
		SearchV2(ctx context.Context, cond *NodeSearchCond) ([]*NodeJoin, string, int, error)
	}
)

func NewNodeInfoJoinModel(url string, db string, c cachec.CacheConf) NodeJoinModel {
	return &defaultNodeInfoJoinModel{
		mc: monc.MustNewModel(url, db, CollectionNodeJoin, c),
	}
}

func (c *NodeSearchCond) generateNodeJoinCond() bson.M {
	// 条件筛选
	m := bson.M{}

	if len(c.DeviceType) > 0 {
		m["nodeInfo.deviceType"] = bson.M{
			"$in": c.DeviceType,
		}
	}

	if c.Stage != "" {
		m["nodeStaticInfo.stage"] = bson.M{
			"$in": strings.Split(c.Stage, ","),
		}
	}

	if c.NodeType != "" {
		// 如果是all则表示查所有节点
		if c.NodeType != "all" {
			m["nodeStaticInfo.nodeType"] = c.NodeType
		}
	} else {
		m["nodeStaticInfo.nodeType"] = NodeTypeNode
	}

	if c.State != "" {
		m["nodeInfo.status"] = c.State
	}

	if len(c.CustomerIDs) > 0 {
		m["nodeStaticInfo.customerIDs"] = bson.M{"$in": c.CustomerIDs}
	}

	return m
}

// SearchV2 支持mark翻页，并返回下一个mark
// 使用mark，需要赋值 cond.MarkSortKey, cond.MarkSortType, cond.Mark, cond.Size
func (j *defaultNodeInfoJoinModel) SearchV2(ctx context.Context, cond *NodeSearchCond) ([]*NodeJoin, string, int, error) {
	pageParam := cond.PageParam
	var option = pageParam.GeneratePageOption()
	filter := cond.generateNodeJoinCond()

	var r []*NodeJoin
	err := j.mc.Find(ctx, &r, filter, option)
	if err != nil {
		return nil, "", 0, err
	}
	var cnt int64
	if !cond.NoCount {
		cnt, err = j.mc.CountDocuments(ctx, filter)
		if err != nil {
			return nil, "", 0, err
		}
	}
	return r, "", int(cnt), nil
}
