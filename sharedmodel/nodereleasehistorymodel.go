package sharedmodel

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/stores/mon"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	CollectionNodeReleaseHistory = "nodeReleaseHistory"
)

type NodeReleaseHistoryOperation string

type NodeReleaseFilterChangeMode string

const (
	NodeReleaseHistoryOperationCreate       NodeReleaseHistoryOperation = "create"
	NodeReleaseHistoryOperationContinue     NodeReleaseHistoryOperation = "continue"
	NodeReleaseHistoryOperationRollback     NodeReleaseHistoryOperation = "rollback"
	NodeReleaseHistoryOperationComplete     NodeReleaseHistoryOperation = "complete"
	NodeReleaseHistoryOperationSwitchFilter NodeReleaseHistoryOperation = "switch_filter"

	NodeReleaseFilterChangeModeNodes2Filter NodeReleaseFilterChangeMode = "nodes2filter"
	NodeReleaseFilterChangeModeFilter2Nodes NodeReleaseFilterChangeMode = "filter2nodes"
)

type NodeReleaseHistoryModel interface {
	Insert(ctx context.Context, data *NodeReleaseHistory) error
	Search(ctx context.Context, cond *NodeReleaseHistoryCond) ([]*NodeReleaseHistory, int, error)
}

type NodeReleaseHistory struct {
	ID        string                      `bson:"_id" json:"id"`              // 主键
	ReleaseID string                      `bson:"releaseId" json:"releaseId"` // 发布ID
	Operation NodeReleaseHistoryOperation `bson:"operation" json:"operation"` // 操作类型
	Operator  string                      `bson:"operator" json:"operator"`   // 操作人
	OpTime    time.Time                   `bson:"opTime" json:"opTime"`       // 操作时间
	Remark    string                      `bson:"remark" json:"remark"`       // 操作详情

	BeforeState NodeReleaseState `bson:"beforeState" json:"beforeState"` // 变更前任务状态
	AfterState  NodeReleaseState `bson:"afterState" json:"afterState"`   // 变更后任务状态

	GrayPolicyInfo *GrayPolicyRemark `bson:"grayPolicyInfo" json:"grayPolicyInfo"` // 灰度配置变更, 创建发布|继续发布涉及
	AppConfigInfo  *AppConfigRemark  `bson:"appConfigInfo" json:"appConfigInfo"`   // 全量/回滚操作相关信息

	CreateAt time.Time `bson:"createAt" json:"createAt"` // 创建时间
	UpdateAt time.Time `bson:"updateAt" json:"updateAt"` // 更新时间
}

type AppConfigRemark struct {
	BeforeMain *AppConfig `bson:"beforeMain" json:"beforeMain"`
	AfterMain  *AppConfig `bson:"afterMain" json:"afterMain"`
}

type GrayPolicyRemark struct {
	NodeIdsAdd       []string                    `bson:"nodeIdsAdd" json:"nodeIdsAdd"`
	NodeIdsDel       []string                    `bson:"nodeIdsDel" json:"nodeIdsDel"`
	AfterFilter      *GrayFilter                 `bson:"afterFilter" json:"afterFilter"`
	FilterChangeMode NodeReleaseFilterChangeMode `bson:"filterChangeMode" json:"filterChangeMode"`
	BeforePercentage int                         `bson:"beforePercentage" json:"beforePercentage"`
	AfterPercentage  int                         `bson:"afterPercentage" json:"afterPercentage"`
}

type NodeReleaseHistoryCond struct {
	PageParam
	FieldsCond

	OpTimeCond *TimeCond
	Operator   string
	Operation  []string
	ReleaseID  string

	NoCount bool
}

func (c NodeReleaseHistoryCond) generateCond() bson.M {
	filter := bson.M{}
	if c.OpTimeCond != nil {
		filter["opTime"] = bson.M{
			"$gte": c.OpTimeCond.Start,
			"$lte": c.OpTimeCond.End,
		}
	}
	if c.Operator != "" {
		filter["operator"] = c.Operator
	}
	if len(c.Operation) > 0 {
		filter["operation"] = bson.M{"$in": c.Operation}
	}
	if c.ReleaseID != "" {
		filter["releaseId"] = c.ReleaseID
	}
	if c.Page < 1 {
		c.Page = 1
	}
	if c.Size > 1000 {
		c.Size = 1000
	}

	return filter
}

func (m NodeReleaseHistoryOperation) String() string {
	return string(m)
}

type defaultNodeReleaseHistoryModel struct {
	model *mon.Model
}

func NewNodeReleaseHistoryModel(url, db string) NodeReleaseHistoryModel {
	return &defaultNodeReleaseHistoryModel{
		model: mon.MustNewModel(url, db, CollectionNodeReleaseHistory),
	}
}

func (m *defaultNodeReleaseHistoryModel) Insert(ctx context.Context, data *NodeReleaseHistory) error {
	if data.ID == "" {
		data.ID = primitive.NewObjectID().Hex()
	}
	data.CreateAt = time.Now()
	data.UpdateAt = time.Now()

	_, err := m.model.InsertOne(ctx, data)
	return err
}

func (m *defaultNodeReleaseHistoryModel) Search(ctx context.Context, cond *NodeReleaseHistoryCond) ([]*NodeReleaseHistory, int, error) {
	option := cond.GeneratePageOption()
	filter := cond.generateCond()

	var r []*NodeReleaseHistory
	err := m.model.Find(ctx, &r, filter, option)
	if err != nil {
		return nil, 0, err
	}

	if cond.NoCount {
		return r, 0, nil
	}

	c, err := m.model.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	return r, int(c), nil
}
