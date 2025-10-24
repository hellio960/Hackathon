package sharedmodel

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/stores/mon"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"hackathon/common/device"
	"hackathon/common/utils"
)

const (
	CollectionNodeRelease = "nodeRelease"
)

const (
	ReleaseTypeFormal ReleaseType = "formal"
	ReleaseTypeBeta   ReleaseType = "beta"
)

const (
	NodeReleaseStateProcessing NodeReleaseState = "processing"
	NodeReleaseStateCompleted  NodeReleaseState = "complete"
	NodeReleaseStateRollbacked NodeReleaseState = "rollbacked"

	NodeReleaseOpTypeAddApp    NodeReleaseOpType = "add"
	NodeReleaseOpTypeUpdateApp NodeReleaseOpType = "update"
	NodeReleaseOpTypeDeleteApp NodeReleaseOpType = "delete"
)

type NodeReleaseState string
type NodeReleaseOpType string

func (r NodeReleaseOpType) String() string {
	return string(r)
}

func (r NodeReleaseOpType) Valid() bool {
	return utils.InAnySlice(r, []NodeReleaseOpType{NodeReleaseOpTypeAddApp, NodeReleaseOpTypeUpdateApp, NodeReleaseOpTypeDeleteApp})
}

func (r NodeReleaseState) String() string {
	return string(r)
}

func (r NodeReleaseState) Valid() bool {
	return utils.InAnySlice(r, []NodeReleaseState{NodeReleaseStateProcessing, NodeReleaseStateCompleted, NodeReleaseStateRollbacked})
}

type ReleaseType string

func (r ReleaseType) String() string {
	return string(r)
}

func (r ReleaseType) Valid() bool {
	return utils.InAnySlice(r, []ReleaseType{ReleaseTypeFormal, ReleaseTypeBeta})
}

type NodeRelease struct {
	ID          string            `bson:"_id"         json:"id"`                // 任务id
	App         string            `bson:"app"         json:"app"`               // 应用名
	DeviceType  string            `bson:"deviceType"         json:"deviceType"` // 节点类型
	ReleaseType ReleaseType       `bson:"releaseType" json:"releaseType"`       // 发布类型 (正式发布/功能验证)
	OpType      NodeReleaseOpType `bson:"opType" json:"opType"`                 // 操作类型(新增组件/升级组件/移除组件)
	MainConfig  *AppConfig        `bson:"mainConfig" json:"mainConfig"`         // 当前版本配置
	AlterConfig *AppConfig        `bson:"alterConfig" json:"alterConfig"`       // 灰度版本配置
	GrayPolicy  GrayPolicy        `bson:"grayPolicy" json:"grayPolicy"`         // 匹配策略
	State       NodeReleaseState  `bson:"state"         json:"state"`           // 发布状态（进行中,完成,回滚）
	Operator    string            `bson:"operator"         json:"operator"`     // 操作人
	Describe    string            `bson:"desc" json:"desc"`                     // 发布功能说明
	CreateAt    time.Time         `bson:"createAt" json:"createAt"`             // 开始发布时间
	UpdateAt    time.Time         `bson:"updateAt" json:"updateAt"`             // 最近更新时间
	EndAt       time.Time         `bson:"endAt" json:"endAt"`                   // 发布完成时间
}

type GrayPolicy struct {
	NodeIds    []string    `bson:"nodeIds" json:"nodeIds"`       // 灰度节点id
	Filter     *GrayFilter `bson:"filter" json:"filter"`         // 灰度节点过滤规则
	Percentage int         `bson:"percentage" json:"percentage"` // 灰度比例
	StatInfo   *StatInfo   `bson:"statInfo" json:"statInfo"`     // 统计信息
}

type GrayFilter struct {
	DevType     device.DevType `bson:"devType" json:"devType"`         // 设备类型
	CustomerIds []uint32       `bson:"customerIds" json:"customerIds"` // 当前业务
	Status      string         `bson:"status" json:"status"`           // 节点状态, online/offline
	Stages      []string       `bson:"stages" json:"stages"`           // 节点阶段
}

type StatInfo struct {
	MarkID             string    `bson:"markID" json:"markID"`                         // 渐进查找标记
	TotalCount         int64     `bson:"totalCount" json:"totalCount"`                 // 总节点数
	AllowCount         int64     `bson:"allowCount" json:"allowCount"`                 // 当前灰度中节点数
	TotalCountUpdateAt time.Time `bson:"totalCountUpdateAt" json:"totalCountUpdateAt"` // 总节点数更新时间
	AllowCountUpdateAt time.Time `bson:"allowCountUpdateAt" json:"allowCountUpdateAt"` // 当前灰度中节点数更新时间
}

type AppConfig struct {
	PackageUrl  string   `bson:"url" json:"url"`             // 包名
	PackageType string   `bson:"type" json:"type"`           // 包类型
	Cmd         string   `bson:"cmd" json:"cmd"`             // 程序启动名
	Args        []string `bson:"args" json:"args"`           // 应用启动参数
	WorkDir     string   `bson:"dir" json:"dir"`             // 工作目录
	HealthUrl   string   `bson:"healthUrl" json:"healthUrl"` // 保活url（目前不使用）
	PackageMd5  string   `bson:"md5" json:"md5"`             // 包md5
}

type NodeReleaseListCond struct {
	PageParam
	FieldsCond
	App          string
	DeviceTypes  []string
	ReleaseTypes []string
	States       []string

	NoCount bool
}

func (c NodeReleaseListCond) generateCond() bson.M {
	filter := bson.M{}
	if c.App != "" {
		filter["app"] = c.App
	}
	if len(c.DeviceTypes) > 0 {
		filter["deviceType"] = bson.M{"$in": c.DeviceTypes}
	}
	if len(c.ReleaseTypes) > 0 {
		filter["releaseType"] = bson.M{"$in": c.ReleaseTypes}
	}
	if len(c.States) > 0 {
		filter["state"] = bson.M{"$in": c.States}
	}

	if c.Page < 1 {
		c.Page = 1
	}
	if c.Size > 1000 {
		c.Size = 1000
	}

	return filter
}

type NodeReleaseModel interface {
	Insert(ctx context.Context, data *NodeRelease) error
	Find(ctx context.Context, id string) (*NodeRelease, error)
	Update(ctx context.Context, data *NodeRelease) error
	Search(ctx context.Context, cond *NodeReleaseListCond) ([]*NodeRelease, int, error)
	FindProcessingRelease(ctx context.Context, appName string, releaseTypes []ReleaseType, devType device.DevType) ([]*NodeRelease, error)
	FindProcessingNewAppRelease(ctx context.Context, devType device.DevType) ([]*NodeRelease, error)
	FindLastRelease(ctx context.Context, appName string, releaseTypes []ReleaseType, devType device.DevType) (*NodeRelease, error)
}

type defaultNodeReleaseModel struct {
	model *mon.Model
}

func NewNodeReleaseModel(url, db string) NodeReleaseModel {
	return &defaultNodeReleaseModel{
		model: mon.MustNewModel(url, db, CollectionNodeRelease),
	}
}

func (m *defaultNodeReleaseModel) Insert(ctx context.Context, data *NodeRelease) error {
	if data.ID == "" {
		data.ID = primitive.NewObjectID().Hex()
	}

	if data.CreateAt.IsZero() {
		data.CreateAt = time.Now()
	}
	if data.UpdateAt.IsZero() {
		data.UpdateAt = time.Now()
	}

	_, err := m.model.InsertOne(ctx, data)
	return err
}

func (m *defaultNodeReleaseModel) Find(ctx context.Context, id string) (*NodeRelease, error) {
	filter := bson.M{
		"_id": id,
	}
	var data NodeRelease
	err := m.model.FindOne(ctx, &data, filter)
	switch err {
	case nil:
		return &data, nil
	case mongo.ErrNoDocuments:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}

func (m *defaultNodeReleaseModel) Update(ctx context.Context, data *NodeRelease) error {
	data.UpdateAt = time.Now()
	_, err := m.model.UpdateOne(ctx, bson.M{"_id": data.ID}, bson.M{"$set": data})
	return err
}

func (m *defaultNodeReleaseModel) Search(ctx context.Context, cond *NodeReleaseListCond) ([]*NodeRelease, int, error) {
	var r []*NodeRelease
	option := cond.GeneratePageOption()
	query := cond.generateCond()

	if err := m.model.Find(ctx, &r, query, option); err != nil {
		return nil, 0, err
	}

	if cond.NoCount {
		return r, 0, nil
	}
	c, err := m.model.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	return r, int(c), nil
}

func (m *defaultNodeReleaseModel) FindProcessingRelease(ctx context.Context, appName string, releaseTypes []ReleaseType, devType device.DevType) ([]*NodeRelease, error) {
	filter := bson.M{
		"app":        appName,
		"deviceType": devType,
		"state":      NodeReleaseStateProcessing,
	}

	if len(releaseTypes) > 0 {
		filter["releaseType"] = bson.M{"$in": releaseTypes}
	}

	var r []*NodeRelease
	err := m.model.Find(ctx, &r, filter)
	if err != nil {
		return nil, err
	}
	if len(r) == 0 {
		return nil, nil
	}
	return r, nil
}

func (m *defaultNodeReleaseModel) FindProcessingNewAppRelease(ctx context.Context, devType device.DevType) ([]*NodeRelease, error) {
	filter := bson.M{
		"deviceType": devType,
		"state":      NodeReleaseStateProcessing,
		"opType":     NodeReleaseOpTypeAddApp,
	}

	var r []*NodeRelease
	err := m.model.Find(ctx, &r, filter)
	if err != nil {
		return nil, err
	}
	if len(r) == 0 {
		return nil, nil
	}
	return r, nil
}

func (m *defaultNodeReleaseModel) FindLastRelease(ctx context.Context, appName string, releaseTypes []ReleaseType, devType device.DevType) (*NodeRelease, error) {
	filter := bson.M{
		"app":        appName,
		"deviceType": devType,
	}

	if len(releaseTypes) > 0 {
		filter["releaseType"] = bson.M{"$in": releaseTypes}
	}

	option := options.Find()
	option.SetSort(bson.M{"createAt": -1})
	option.SetLimit(1)

	var r []*NodeRelease
	err := m.model.Find(ctx, &r, filter, option)
	if err != nil {
		return nil, err
	}
	if len(r) == 0 {
		return nil, ErrNotFound
	}
	return r[0], nil
}
