package loader

import (
	"context"
	"sync"

	"hackathon/sharedmodel"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

var (
	loaderOnce        sync.Once
	NodeReleaseLoader *nodeReleaseLoader
)

type LoaderContext struct {
	NodeReleaseModel sharedmodel.NodeReleaseModel
	BizRedis         *redis.Redis
}

type Loader interface {
	Load()
}

func StartLoaders(loaderCtx *LoaderContext) {
	loaderOnce.Do(func() {
		ctx := context.Background()

		NodeReleaseLoader = NewNodeReleaseLoader(ctx, loaderCtx)
		loaders := []Loader{NodeReleaseLoader}
		for _, l := range loaders {
			l.Load()
		}
	})
}
