package main

import (
	"flag"
	"fmt"

	_ "github.com/qiniu/version"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"

	"hackathon/cmd/jarvis/internal/config"
	"hackathon/cmd/jarvis/internal/handler"
	"hackathon/cmd/jarvis/internal/loader"
	"hackathon/cmd/jarvis/internal/svc"
)

var configFile = flag.String("f", "etc/jarvis-api.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	ctx := svc.NewServiceContext(c)
	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	loader.StartLoaders(&loader.LoaderContext{
		NodeReleaseModel: ctx.NodeReleaseModel,
		BizRedis:         ctx.BizRedis,
	})

	handler.RegisterHandlers(server, ctx)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
