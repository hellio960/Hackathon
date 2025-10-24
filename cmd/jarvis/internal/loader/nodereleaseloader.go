package loader

import (
	"context"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"

	"hackathon/sharedmodel"
)

type nodeReleaseLoader struct {
	ctx       context.Context
	loaderCtx *LoaderContext

	mu                    sync.RWMutex
	processingReleaseList []*sharedmodel.NodeRelease
}

func NewNodeReleaseLoader(ctx context.Context, loaderCtx *LoaderContext) *nodeReleaseLoader {
	return &nodeReleaseLoader{ctx: ctx, loaderCtx: loaderCtx}
}

func (l *nodeReleaseLoader) Load() {
	l.load(100)
	threading.GoSafe(func() {
		for range time.NewTicker(time.Minute).C {
			l.load(100)
		}
	})
}

func (l *nodeReleaseLoader) LoadOnce() {
	l.load(100)
}

func (l *nodeReleaseLoader) load(ps int) {
	var processingReleaseList []*sharedmodel.NodeRelease

	// 获取所有的进行中任务
	lg := logx.WithContext(l.ctx)
	cond := &sharedmodel.NodeReleaseListCond{
		PageParam: sharedmodel.PageParam{
			Page: 1,
			Size: ps,
		},
		//FieldsCond:   nil,
		States:  []string{sharedmodel.NodeReleaseStateProcessing.String()},
		NoCount: true,
	}

	for {
		releases, _, err := l.loaderCtx.NodeReleaseModel.Search(l.ctx, cond)
		if err != nil {
			lg.Errorf("search node release model error: %v", err)
			return
		}

		for _, r := range releases {
			processingReleaseList = append(processingReleaseList, r)
		}

		if len(releases) < cond.Size {
			break
		}

		cond.Page++
	}

	lg.Infof("succeed load %d records", len(processingReleaseList))
	l.SetProcessingList(processingReleaseList)
}

func (l *nodeReleaseLoader) SetProcessingList(rls []*sharedmodel.NodeRelease) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.processingReleaseList = rls
}

func (l *nodeReleaseLoader) GetProcessingNewAppRelease(devType string) (res []*sharedmodel.NodeRelease) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, rl := range l.processingReleaseList {
		if rl.State != sharedmodel.NodeReleaseStateProcessing {
			continue
		}

		if rl.OpType != sharedmodel.NodeReleaseOpTypeAddApp {
			continue
		}

		if rl.DeviceType != devType {
			continue
		}

		res = append(res, rl)
	}

	return res
}

func (l *nodeReleaseLoader) GetProcessingRelease(name, devType string) (res []*sharedmodel.NodeRelease) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, rl := range l.processingReleaseList {
		if rl.State != sharedmodel.NodeReleaseStateProcessing {
			continue
		}

		if rl.App != name {
			continue
		}

		if rl.DeviceType != devType {
			continue
		}

		res = append(res, rl)
	}

	return res
}
