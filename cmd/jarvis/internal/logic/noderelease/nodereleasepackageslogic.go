package noderelease

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/qiniu/go-sdk/v7/auth"
	"github.com/qiniu/go-sdk/v7/storage"
	"github.com/zeromicro/go-zero/core/logx"

	"hackathon/cmd/client/kodo"
	"hackathon/cmd/jarvis/internal/svc"
	"hackathon/cmd/jarvis/internal/types"
	"hackathon/common/device"
	"hackathon/common/errorx"
	"hackathon/common/unit"
)

const (
	KodoBucket    = "niulink"
	KodoUrlPrefix = "https://nbaililk.ootaiwaevufe.com/"
)

type NodeReleasePackagesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewNodeReleasePackagesLogic(ctx context.Context, svcCtx *svc.ServiceContext) NodeReleasePackagesLogic {
	return NodeReleasePackagesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *NodeReleasePackagesLogic) NodeReleasePackages(req *types.NodeReleasePackagesReq) (resp *types.NodeReleasePackagesResp, err error) {
	startTime := time.Now()
	l.Infof("[NodeReleasePackages] 开始处理, app:%s devType:%s path:%s", req.App, req.DevType, req.Path)

	prefix := req.Path
	var autoCursor bool
	if prefix == "" {
		// 从allowAppModel中获取
		t1 := time.Now()
		nodeType, err := device.DevType(req.DevType).NodeType()
		if err != nil {
			l.Errorf("[NodeReleasePackages] DevType转换失败, devType:%s error:%v", req.DevType, err)
			return nil, err
		}
		allowApp, err := l.svcCtx.AllowAppsModel.FindByName(l.ctx, nodeType, req.App)
		if err != nil {
			l.Errorf("[NodeReleasePackages] 查询AllowApp失败, nodeType:%s app:%s error:%v", nodeType, req.App, err)
			return nil, err
		}
		prefix = allowApp.Path
		autoCursor = true
		l.Infof("[NodeReleasePackages] 获取AllowApp耗时:%v path:%s", time.Since(t1), prefix)
	}

	if prefix == "" {
		return nil, errorx.NewDefaultError("未匹配到正确包路径, 请配置应用默认路径或指定路径")
	}

	var packages []types.PackageInfo
	t2 := time.Now()
	if autoCursor {
		// 根据应用配置自动遍历, 默认检查包命名格式
		l.Infof("[NodeReleasePackages] 开始自动遍历最近30天包")
		packages, err = l.cursorPackagesPast30Days(KodoBucket, prefix, device.DevType(req.DevType), req.App, 50, true)
		if err != nil {
			l.Errorf("[NodeReleasePackages] cursorPackagesPast30Days失败, error:%v", err)
			return nil, err
		}
	} else {
		// 根据指定路径遍历, 不检查包命名格式
		l.Infof("[NodeReleasePackages] 开始按路径遍历包")
		packages, err = l.cursorPackagesByPath(KodoBucket, prefix, device.DevType(req.DevType), req.App, false)
		if err != nil {
			l.Errorf("[NodeReleasePackages] cursorPackagesByPath失败, error:%v", err)
			return nil, err
		}
	}
	l.Infof("[NodeReleasePackages] 获取包列表耗时:%v 包数量:%d", time.Since(t2), len(packages))
	l.Infof("[NodeReleasePackages] 总耗时:%v", time.Since(startTime))

	return &types.NodeReleasePackagesResp{
		Packages: packages,
	}, nil
}

func (l *NodeReleasePackagesLogic) cursorPackagesByPath(bucket, prefix string, devType device.DevType, app string, checkFormat bool) (packages []types.PackageInfo, err error) {
	files, err := l.getPackages(bucket, prefix, devType, app, checkFormat)
	if err != nil {
		return nil, err
	}

	l.Infof("files count:%d", len(files))
	sort.Slice(files, func(i, j int) bool {
		return files[i].PutTime > files[j].PutTime
	})

	// 并发获取 MD5
	packages = l.fetchPackagesWithMd5Concurrently(files)

	if len(packages) > 50 {
		packages = packages[:50]
	}

	return packages, nil
}

func (l *NodeReleasePackagesLogic) cursorPackagesPast30Days(bucket, prefix string, devType device.DevType, app string, limit int, checkFormat bool) (packages []types.PackageInfo, err error) {
	// 按{prefix}/2025080, {prefix}/2025072, {prefix}/2025071, ... 列出最近30天的包
	startTime := time.Now()
	paths := generatePathsForPastDays(prefix, 30)
	l.Infof("[cursorPackagesPast30Days] 生成路径数量:%d", len(paths))

	var files []storage.ListItem
	for i, path := range paths {
		t1 := time.Now()
		pkgs, err := l.getPackages(bucket, path, devType, app, checkFormat)
		if err != nil {
			l.Errorf("[cursorPackagesPast30Days] getPackages失败, path:%s error:%v", path, err)
			return nil, err
		}
		l.Infof("[cursorPackagesPast30Days] 路径[%d/%d] %s 获取%d个包, 耗时:%v", i+1, len(paths), path, len(pkgs), time.Since(t1))
		files = append(files, pkgs...)
		if len(files) >= limit {
			l.Infof("[cursorPackagesPast30Days] 已达到上限 %d, 停止遍历", limit)
			break
		}
	}
	l.Infof("[cursorPackagesPast30Days] 获取文件列表总耗时:%v 文件数:%d", time.Since(startTime), len(files))

	t2 := time.Now()
	sort.Slice(files, func(i, j int) bool {
		return files[i].PutTime > files[j].PutTime
	})
	l.Infof("[cursorPackagesPast30Days] 排序耗时:%v", time.Since(t2))

	// 并发获取 MD5
	t3 := time.Now()
	packages = l.fetchPackagesWithMd5Concurrently(files)
	l.Infof("[cursorPackagesPast30Days] 并发获取MD5耗时:%v 包数量:%d", time.Since(t3), len(packages))

	if len(packages) > limit {
		packages = packages[:limit]
	}

	return packages, nil
}

// 并发获取包的 MD5 信息
func (l *NodeReleasePackagesLogic) fetchPackagesWithMd5Concurrently(files []storage.ListItem) []types.PackageInfo {
	startTime := time.Now()
	l.Infof("[fetchPackagesWithMd5Concurrently] 开始并发获取MD5, 文件数:%d", len(files))

	var wg sync.WaitGroup
	var mu sync.Mutex
	packages := make([]types.PackageInfo, 0, len(files))

	// 使用固定数量的 goroutine 池，避免创建过多 goroutine
	const maxConcurrent = 10
	sem := make(chan struct{}, maxConcurrent)

	successCount := 0
	failCount := 0

	for i, file := range files {
		wg.Add(1)
		go func(idx int, f storage.ListItem) {
			defer wg.Done()

			// 获取信号量
			sem <- struct{}{}
			defer func() { <-sem }()

			t1 := time.Now()
			url := KodoUrlPrefix + f.Key
			md5, err := kodo.GetKodoFileMd5(url)
			if err != nil {
				l.Errorf("[fetchPackagesWithMd5Concurrently] [%d/%d] 获取MD5失败, url:%s error:%v 耗时:%v", idx+1, len(files), url, err, time.Since(t1))
				mu.Lock()
				failCount++
				mu.Unlock()
				return
			}

			pkg := types.PackageInfo{
				Url:  url,
				File: f.Key,
				Md5:  md5,
				Size: unit.ByteConvert(unit.Byte(f.Fsize)),
			}

			mu.Lock()
			packages = append(packages, pkg)
			successCount++
			mu.Unlock()

			l.Infof("[fetchPackagesWithMd5Concurrently] [%d/%d] 获取MD5成功, file:%s 耗时:%v", idx+1, len(files), f.Key, time.Since(t1))
		}(i, file)
	}

	wg.Wait()
	l.Infof("[fetchPackagesWithMd5Concurrently] 完成, 成功:%d 失败:%d 总耗时:%v", successCount, failCount, time.Since(startTime))
	return packages
}

func generatePathsForPastDays(prefix string, days int) []string {
	paths := []string{}
	for i := 0; i < days/10; i++ {
		dayTime := time.Now().AddDate(0, 0, -10*i)
		paths = append(paths, fmt.Sprintf("%s/%s", strings.TrimSuffix(prefix, "/"), dayTime.Format("20060102")[:7]))
	}
	return paths
}

func (l *NodeReleasePackagesLogic) getPackages(bucket, prefix string, devType device.DevType, app string, checkFormat bool) (files []storage.ListItem, err error) {
	mac := auth.New(l.svcCtx.Config.Kodo.AccessKey, l.svcCtx.Config.Kodo.SecretKey)
	bucketManager := storage.NewBucketManager(mac, &storage.Config{UseHTTPS: false})

	limit := 100
	delimiter := ""
	marker := ""

	// 最多查10次, 共1000个文件
	for i := 0; i < 10; i++ {
		l.Infof("ListFiles prefix:%s delimiter:%s marker:%s", prefix, delimiter, marker)
		entries, _, nextMarker, hashNext, err := bucketManager.ListFiles(bucket, prefix, delimiter, marker, limit)
		if err != nil {
			l.Errorf("kodo ListFiles error:%s", err)
			break
		}

		l.Infof("ListFiles entries count:%d round:%d", len(entries), i+1)
		for _, entry := range entries {
			// 包格式{PATH}/{app}_{devType}.{pkgType},  如NiuLinkNodeApps/jarvis/20250828/jarvisagent_jarvis.A.tar.gz
			//if checkFormat && !strings.Contains(entry.Key, fmt.Sprintf("/%s_%s", app, devType.String())) {
			//	continue
			//}

			// 如果文件路径包含devType信息, 则进行匹配, 否则不做匹配
			// devTypeContained := strings.Contains(entry.Key, "ant.") || strings.Contains(entry.Key, "jarvis.") || strings.Contains(entry.Key, "droid.")
			// if checkFormat && devTypeContained && !strings.Contains(entry.Key, devType.String()) {
			// 	continue
			// }

			if checkFormat && !strings.Contains(entry.Key, app) {
				continue
			}

			if strings.HasSuffix(entry.Key, ".md5") {
				continue
			}

			files = append(files, entry)

			if len(files) >= 50 {
				return nil, errorx.NewDefaultError("指定路径下包含包数量过多, 请指定更精确的包存放路径")
			}
		}
		if hashNext {
			marker = nextMarker
		} else {
			break
		}
	}

	return files, nil
}
