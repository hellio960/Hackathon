package localstorage

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"hackathon/common/device"
	"hackathon/sharedmodel"
)

type localNodeReleaseModel struct {
	coll *Collection
	qh   *QueryHelper
}

func NewLocalNodeReleaseModel(dataDir string) sharedmodel.NodeReleaseModel {
	storage := NewLocalStorage(dataDir)
	return &localNodeReleaseModel{
		coll: storage.Collection("nodeRelease"),
		qh:   &QueryHelper{},
	}
}

func (m *localNodeReleaseModel) Insert(ctx context.Context, data *sharedmodel.NodeRelease) error {
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

func (m *localNodeReleaseModel) Find(ctx context.Context, id string) (*sharedmodel.NodeRelease, error) {
	var result sharedmodel.NodeRelease
	err := m.coll.FindOne(ctx, id, &result)
	if err == ErrNotFound {
		return nil, sharedmodel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (m *localNodeReleaseModel) Update(ctx context.Context, data *sharedmodel.NodeRelease) error {
	data.UpdateAt = time.Now()
	return m.coll.Update(ctx, data.ID, data)
}

func (m *localNodeReleaseModel) Search(ctx context.Context, cond *sharedmodel.NodeReleaseListCond) ([]*sharedmodel.NodeRelease, int, error) {
	filter := func(doc interface{}) bool {
		data, _ := json.Marshal(doc)
		var nr sharedmodel.NodeRelease
		json.Unmarshal(data, &nr)
		
		if cond.App != "" && nr.App != cond.App {
			return false
		}
		
		if len(cond.DeviceTypes) > 0 {
			found := false
			for _, dt := range cond.DeviceTypes {
				if nr.DeviceType == dt {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		
		if len(cond.ReleaseTypes) > 0 {
			found := false
			for _, rt := range cond.ReleaseTypes {
				if string(nr.ReleaseType) == rt {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		
		if len(cond.States) > 0 {
			found := false
			for _, s := range cond.States {
				if string(nr.State) == s {
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
	
	var allReleases []*sharedmodel.NodeRelease
	err := m.coll.FindAll(ctx, filter, &allReleases)
	if err != nil {
		return nil, 0, err
	}
	
	sort.Slice(allReleases, func(i, j int) bool {
		return allReleases[i].CreateAt.After(allReleases[j].CreateAt)
	})
	
	total := len(allReleases)
	
	if cond.NoCount {
		return allReleases, 0, nil
	}
	
	page := cond.Page
	if page < 1 {
		page = 1
	}
	size := cond.Size
	if size > 1000 {
		size = 1000
	}
	
	start := (page - 1) * size
	end := start + size
	
	if start >= len(allReleases) {
		return []*sharedmodel.NodeRelease{}, total, nil
	}
	if end > len(allReleases) {
		end = len(allReleases)
	}
	
	return allReleases[start:end], total, nil
}

func (m *localNodeReleaseModel) FindProcessingRelease(ctx context.Context, appName string, releaseTypes []sharedmodel.ReleaseType, devType device.DevType) ([]*sharedmodel.NodeRelease, error) {
	filter := func(doc interface{}) bool {
		data, _ := json.Marshal(doc)
		var nr sharedmodel.NodeRelease
		json.Unmarshal(data, &nr)
		
		if nr.App != appName {
			return false
		}
		if nr.DeviceType != devType.String() {
			return false
		}
		if nr.State != sharedmodel.NodeReleaseStateProcessing {
			return false
		}
		
		if len(releaseTypes) > 0 {
			found := false
			for _, rt := range releaseTypes {
				if nr.ReleaseType == rt {
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
	
	var results []*sharedmodel.NodeRelease
	err := m.coll.FindAll(ctx, filter, &results)
	if err != nil {
		return nil, err
	}
	
	if len(results) == 0 {
		return nil, nil
	}
	
	return results, nil
}

func (m *localNodeReleaseModel) FindProcessingNewAppRelease(ctx context.Context, devType device.DevType) ([]*sharedmodel.NodeRelease, error) {
	filter := func(doc interface{}) bool {
		data, _ := json.Marshal(doc)
		var nr sharedmodel.NodeRelease
		json.Unmarshal(data, &nr)
		
		return nr.DeviceType == devType.String() &&
			nr.State == sharedmodel.NodeReleaseStateProcessing &&
			nr.OpType == sharedmodel.NodeReleaseOpTypeAddApp
	}
	
	var results []*sharedmodel.NodeRelease
	err := m.coll.FindAll(ctx, filter, &results)
	if err != nil {
		return nil, err
	}
	
	if len(results) == 0 {
		return nil, nil
	}
	
	return results, nil
}

func (m *localNodeReleaseModel) FindLastRelease(ctx context.Context, appName string, releaseTypes []sharedmodel.ReleaseType, devType device.DevType) (*sharedmodel.NodeRelease, error) {
	filter := func(doc interface{}) bool {
		data, _ := json.Marshal(doc)
		var nr sharedmodel.NodeRelease
		json.Unmarshal(data, &nr)
		
		if nr.App != appName {
			return false
		}
		if nr.DeviceType != devType.String() {
			return false
		}
		
		if len(releaseTypes) > 0 {
			found := false
			for _, rt := range releaseTypes {
				if nr.ReleaseType == rt {
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
	
	var results []*sharedmodel.NodeRelease
	err := m.coll.FindAll(ctx, filter, &results)
	if err != nil {
		return nil, err
	}
	
	if len(results) == 0 {
		return nil, sharedmodel.ErrNotFound
	}
	
	sort.Slice(results, func(i, j int) bool {
		return results[i].CreateAt.After(results[j].CreateAt)
	})
	
	return results[0], nil
}
