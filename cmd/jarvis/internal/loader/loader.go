package loader

import (
	"context"
	"sync"

	"hackathon/cmd/jarvis/internal/localstorage"
	"hackathon/sharedmodel"
)

var (
	loaderOnce        sync.Once
	NodeReleaseLoader *nodeReleaseLoader
)

type LoaderContext struct {
	NodeReleaseModel sharedmodel.NodeReleaseModel
	BizRedis         *localstorage.LocalRedis
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
