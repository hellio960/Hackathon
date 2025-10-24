package localstorage

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"hackathon/cmd/jarvis/internal/model"
	"hackathon/sharedmodel"
)

type localAllowAppsModel struct {
	coll *Collection
}

func NewLocalAllowAppsModel(dataDir string) sharedmodel.AllowAppsModel {
	storage := NewLocalStorage(dataDir)
	return &localAllowAppsModel{
		coll: storage.Collection("allowApps"),
	}
}

func (m *localAllowAppsModel) Insert(ctx context.Context, data *sharedmodel.AllowAppsTemplate) error {
	if data.ID == "" {
		data.ID = primitive.NewObjectID().Hex()
	}
	if data.CreateAt.IsZero() {
		data.CreateAt = time.Now()
	}
	if data.UpdateAt.IsZero() {
		data.UpdateAt = time.Now()
	}
	return m.coll.Insert(ctx, data.ID, data)
}

func (m *localAllowAppsModel) FindByID(ctx context.Context, id string) (*sharedmodel.AllowAppsTemplate, error) {
	var result sharedmodel.AllowAppsTemplate
	err := m.coll.FindOne(ctx, id, &result)
	if err == ErrNotFound {
		return nil, sharedmodel.ErrNotFound
	}
	return &result, err
}

func (m *localAllowAppsModel) FindByName(ctx context.Context, nodeType, name string) (*sharedmodel.AllowAppsTemplate, error) {
	filter := func(doc interface{}) bool {
		data, _ := json.Marshal(doc)
		var app sharedmodel.AllowAppsTemplate
		json.Unmarshal(data, &app)
		return app.Name == name && app.NodeType == nodeType
	}
	
	var results []*sharedmodel.AllowAppsTemplate
	err := m.coll.FindAll(ctx, filter, &results)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, sharedmodel.ErrNotFound
	}
	return results[0], nil
}

func (m *localAllowAppsModel) Update(ctx context.Context, data *sharedmodel.AllowAppsTemplate) error {
	data.UpdateAt = time.Now()
	return m.coll.Update(ctx, data.ID, data)
}

func (m *localAllowAppsModel) SetAppPathAndDesc(ctx context.Context, id, path, desc string) error {
	var app sharedmodel.AllowAppsTemplate
	if err := m.coll.FindOne(ctx, id, &app); err != nil {
		return err
	}
	app.Path = path
	app.Desc = desc
	app.UpdateAt = time.Now()
	return m.coll.Update(ctx, id, &app)
}

func (m *localAllowAppsModel) DeleteByID(ctx context.Context, id string) error {
	return m.coll.Delete(ctx, id)
}

func (m *localAllowAppsModel) DeleteByName(ctx context.Context, nodeType, name string) error {
	filter := func(doc interface{}) bool {
		data, _ := json.Marshal(doc)
		var app sharedmodel.AllowAppsTemplate
		json.Unmarshal(data, &app)
		return app.Name == name && app.NodeType == nodeType
	}
	
	var results []*sharedmodel.AllowAppsTemplate
	m.coll.FindAll(ctx, filter, &results)
	if len(results) > 0 {
		return m.coll.Delete(ctx, results[0].ID)
	}
	return ErrNotFound
}

func (m *localAllowAppsModel) Search(ctx context.Context, cond sharedmodel.AllowAppsListCond) ([]*sharedmodel.AllowAppsTemplate, int, error) {
	filter := func(doc interface{}) bool {
		data, _ := json.Marshal(doc)
		var app sharedmodel.AllowAppsTemplate
		json.Unmarshal(data, &app)
		
		if len(cond.IDs) > 0 {
			found := false
			for _, id := range cond.IDs {
				if app.ID == id {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		
		if len(cond.Names) > 0 {
			found := false
			for _, name := range cond.Names {
				if app.Name == name {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		
		if len(cond.NodeTypes) > 0 {
			found := false
			for _, nt := range cond.NodeTypes {
				if app.NodeType == nt {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		
		return true
	}
	
	var results []*sharedmodel.AllowAppsTemplate
	err := m.coll.FindAll(ctx, filter, &results)
	if err != nil {
		return nil, 0, err
	}
	
	sort.Slice(results, func(i, j int) bool {
		return results[i].CreateAt.After(results[j].CreateAt)
	})
	
	total := len(results)
	
	page := cond.Page
	if page < 1 {
		page = 1
	}
	size := cond.Size
	if size < 1 {
		size = 20
	}
	if size > 1000 {
		size = 1000
	}
	
	start := (page - 1) * size
	end := start + size
	
	if start >= len(results) {
		return []*sharedmodel.AllowAppsTemplate{}, total, nil
	}
	if end > len(results) {
		end = len(results)
	}
	
	return results[start:end], total, nil
}

func (m *localAllowAppsModel) Drop(ctx context.Context) error {
	return m.coll.Drop(ctx)
}

type localSysParamModel struct {
	coll *Collection
}

func NewLocalSysParamModel(dataDir string) model.SysParamModel {
	storage := NewLocalStorage(dataDir)
	return &localSysParamModel{
		coll: storage.Collection("sysParam"),
	}
}

func (m *localSysParamModel) Insert(ctx context.Context, data *sharedmodel.SysParam) error {
	return m.coll.Insert(ctx, data.Name, data)
}

func (m *localSysParamModel) FindOne(ctx context.Context, name string) (*sharedmodel.SysParam, error) {
	var result sharedmodel.SysParam
	err := m.coll.FindOne(ctx, name, &result)
	if err == ErrNotFound {
		return nil, sharedmodel.ErrNotFound
	}
	return &result, err
}

func (m *localSysParamModel) Update(ctx context.Context, data *sharedmodel.SysParam) error {
	return m.coll.Update(ctx, data.Name, data)
}

func (m *localSysParamModel) DeleteByName(ctx context.Context, name string) error {
	return m.coll.Delete(ctx, name)
}

func (m *localSysParamModel) Drop(ctx context.Context) error {
	return m.coll.Drop(ctx)
}

func (m *localSysParamModel) Key(name string) string {
	return sharedmodel.PrefixSysParamCacheKey + name
}

type localUpdRecordModel struct {
	coll *Collection
}

func NewLocalUpdRecordModel(dataDir string) model.UpdRecordModel {
	storage := NewLocalStorage(dataDir)
	return &localUpdRecordModel{
		coll: storage.Collection("updRecord"),
	}
}

func (m *localUpdRecordModel) Insert(ctx context.Context, data *sharedmodel.UpdRecord) error {
	if data.ID == "" {
		data.ID = primitive.NewObjectID().Hex()
	}
	if data.CreateAt.IsZero() {
		data.CreateAt = time.Now()
	}
	return m.coll.Insert(ctx, data.ID, data)
}

func (m *localUpdRecordModel) FindOne(ctx context.Context, id string) (*sharedmodel.UpdRecord, error) {
	var result sharedmodel.UpdRecord
	err := m.coll.FindOne(ctx, id, &result)
	if err == ErrNotFound {
		return nil, sharedmodel.ErrNotFound
	}
	return &result, err
}

func (m *localUpdRecordModel) Update(ctx context.Context, data *sharedmodel.UpdRecord) error {
	return m.coll.Update(ctx, data.ID, data)
}

func (m *localUpdRecordModel) Search(ctx context.Context, cond sharedmodel.UpdRecordListCond) ([]*sharedmodel.UpdRecord, int, error) {
	var results []*sharedmodel.UpdRecord
	err := m.coll.FindAll(ctx, nil, &results)
	if err != nil {
		return nil, 0, err
	}
	
	sort.Slice(results, func(i, j int) bool {
		return results[i].CreateAt.After(results[j].CreateAt)
	})
	
	total := len(results)
	return results, total, nil
}

func (m *localUpdRecordModel) Drop(ctx context.Context) error {
	return m.coll.Drop(ctx)
}

func (m *localUpdRecordModel) Key(id string) string {
	return "cache:updRecord:" + id
}

type localGrayNodesModel struct {
	coll *Collection
}

func NewLocalGrayNodesModel(dataDir string) sharedmodel.GrayNodesModel {
	storage := NewLocalStorage(dataDir)
	return &localGrayNodesModel{
		coll: storage.Collection("grayNodes"),
	}
}

func (m *localGrayNodesModel) Insert(ctx context.Context, data *sharedmodel.GrayNodesRecord) error {
	if data.ID == "" {
		data.ID = primitive.NewObjectID().Hex()
	}
	if data.CreateAt.IsZero() {
		data.CreateAt = time.Now()
	}
	return m.coll.Insert(ctx, data.ID, data)
}

func (m *localGrayNodesModel) Update(ctx context.Context, data *sharedmodel.GrayNodesRecord) error {
	return m.coll.Update(ctx, data.ID, data)
}

func (m *localGrayNodesModel) FindOne(ctx context.Context, nodeId, releaseId string) (*sharedmodel.GrayNodesRecord, error) {
	filter := func(doc interface{}) bool {
		data, _ := json.Marshal(doc)
		var gn sharedmodel.GrayNodesRecord
		json.Unmarshal(data, &gn)
		return gn.NodeId == nodeId && gn.ReleaseId == releaseId
	}
	
	var results []*sharedmodel.GrayNodesRecord
	err := m.coll.FindAll(ctx, filter, &results)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, sharedmodel.ErrNotFound
	}
	return results[0], nil
}

type localNodeReleaseHistoryModel struct {
	coll *Collection
}

func NewLocalNodeReleaseHistoryModel(dataDir string) sharedmodel.NodeReleaseHistoryModel {
	storage := NewLocalStorage(dataDir)
	return &localNodeReleaseHistoryModel{
		coll: storage.Collection("nodeReleaseHistory"),
	}
}

func (m *localNodeReleaseHistoryModel) Insert(ctx context.Context, data *sharedmodel.NodeReleaseHistory) error {
	if data.ID == "" {
		data.ID = primitive.NewObjectID().Hex()
	}
	if data.CreateAt.IsZero() {
		data.CreateAt = time.Now()
	}
	return m.coll.Insert(ctx, data.ID, data)
}

func (m *localNodeReleaseHistoryModel) Search(ctx context.Context, cond sharedmodel.NodeReleaseHistoryListCond) ([]*sharedmodel.NodeReleaseHistory, int, error) {
	var results []*sharedmodel.NodeReleaseHistory
	err := m.coll.FindAll(ctx, nil, &results)
	if err != nil {
		return nil, 0, err
	}
	
	sort.Slice(results, func(i, j int) bool {
		return results[i].CreateAt.After(results[j].CreateAt)
	})
	
	total := len(results)
	return results, total, nil
}

type localNodeJoinModel struct {
	coll *Collection
}

func NewLocalNodeJoinModel(dataDir string) sharedmodel.NodeJoinModel {
	storage := NewLocalStorage(dataDir)
	return &localNodeJoinModel{
		coll: storage.Collection("nodeJoin"),
	}
}

func (m *localNodeJoinModel) Search(ctx context.Context, cond *sharedmodel.NodeSearchCond) ([]*sharedmodel.NodeJoin, error) {
	var results []*sharedmodel.NodeJoin
	err := m.coll.FindAll(ctx, nil, &results)
	return results, err
}

func (m *localNodeJoinModel) FindOneByNodeId(ctx context.Context, nodeId string) (*sharedmodel.NodeJoin, error) {
	filter := func(doc interface{}) bool {
		data, _ := json.Marshal(doc)
		var nj sharedmodel.NodeJoin
		json.Unmarshal(data, &nj)
		return nj.NodeId == nodeId
	}
	
	var results []*sharedmodel.NodeJoin
	err := m.coll.FindAll(ctx, filter, &results)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, sharedmodel.ErrNotFound
	}
	return results[0], nil
}

func (m *localNodeJoinModel) FindNodeIdsByPool(ctx context.Context, nodePool string) ([]string, error) {
	filter := func(doc interface{}) bool {
		data, _ := json.Marshal(doc)
		var nj sharedmodel.NodeJoin
		json.Unmarshal(data, &nj)
		return nj.NodePoolId == nodePool
	}
	
	var results []*sharedmodel.NodeJoin
	err := m.coll.FindAll(ctx, filter, &results)
	if err != nil {
		return nil, err
	}
	
	nodeIds := make([]string, 0, len(results))
	for _, nj := range results {
		nodeIds = append(nodeIds, nj.NodeId)
	}
	return nodeIds, nil
}

func (m *localNodeJoinModel) FindNodePoolsByNodeId(ctx context.Context, nodeId string) ([]string, error) {
	nj, err := m.FindOneByNodeId(ctx, nodeId)
	if err != nil {
		return nil, err
	}
	return []string{nj.NodePoolId}, nil
}
