package sharedmodel

import (
	"context"
	"hackathon/common/device"
	"strings"

	cachec "github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	CollectionNodeJoin = "nodeJoin"
)

type (
	NodeJoin struct {
		Id          string         `bson:"_id" json:"_id"`
		NodeId      string         `bson:"nodeId" json:"nodeId"`
		DeviceType  device.DevType `bson:"deviceType" json:"deviceType"`
		Stage       string         `bson:"stage" json:"stage"`
		NodeType    NodeType       `bson:"nodeType" json:"nodeType"`
		Status      string         `bson:"status" json:"status"`
		CustomerIDs []uint32       `bson:"customerIDs"  json:"customerIDs"`
	}
	defaultNodeInfoJoinModel struct {
		mc *monc.Model
	}
	NodeJoinModel interface {
		Search(ctx context.Context, cond *NodeSearchCond) ([]*NodeJoin, string, int, error)
	}
)

func NewNodeJoinModel(url string, db string, c cachec.CacheConf) NodeJoinModel {
	return &defaultNodeInfoJoinModel{
		mc: monc.MustNewModel(url, db, CollectionNodeJoin, c),
	}
}

func (c *NodeSearchCond) generateNodeJoinCond() bson.M {
	// 条件筛选
	m := bson.M{}

	if len(c.DeviceType) > 0 {
		m["deviceType"] = bson.M{
			"$in": c.DeviceType,
		}
	}

	if c.Stage != "" {
		m["stage"] = bson.M{
			"$in": strings.Split(c.Stage, ","),
		}
	}

	if c.NodeType != "" {
		// 如果是all则表示查所有节点
		if c.NodeType != "all" {
			m["nodeType"] = c.NodeType
		}
	} else {
		m["nodeType"] = NodeTypeNode
	}

	if c.State != "" {
		m["status"] = c.State
	}

	if len(c.CustomerIDs) > 0 {
		m["customerIDs"] = bson.M{"$in": c.CustomerIDs}
	}

	return m
}

// Search 支持mark翻页，并返回下一个mark
// 使用mark，需要赋值 cond.MarkSortKey, cond.MarkSortType, cond.Mark, cond.Size
func (j *defaultNodeInfoJoinModel) Search(ctx context.Context, cond *NodeSearchCond) ([]*NodeJoin, string, int, error) {
	pageParam := cond.PageParam
	if cond.IsMarkPage() {
		// 使用mark，不能用其他非mark的字段进行排序
		pageParam.SortKey = cond.MarkSortKey
		pageParam.SortType = cond.MarkSortType
	} else {
		pageParam.SortKey = "_id"
	}
	var option = pageParam.GeneratePageOption()
	filter := cond.generateNodeJoinCond()
	cond.MarkCond(filter) // 添加 mark 相关的排序字段，注意可能覆盖filter里的字段
	var r []*NodeJoin
	var newMark string
	err := j.mc.Find(ctx, &r, filter, option)
	if err != nil {
		return nil, newMark, 0, err
	}
	if len(r) > 0 && cond.IsMarkPage() && len(r) >= cond.Size {
		// 没有返回或者不是mark或者返回的内容已经小于size要求(最后一页)，不返回mark
		newMark = r[len(r)-1].Id
	}
	var cnt int64
	if !cond.NoCount {
		cnt, err = j.mc.CountDocuments(ctx, filter)
		if err != nil {
			return nil, newMark, 0, err
		}
	}
	return r, newMark, int(cnt), nil
}
