package noderelease

import (
	"context"
	"fmt"
	"sort"
	"strings"
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
	prefix := req.Path
	var autoCursor bool
	if prefix == "" {
		// 从allowAppModel中获取
		nodeType, err := device.DevType(req.DevType).NodeType()
		if err != nil {
			return nil, err
		}
		allowApp, err := l.svcCtx.AllowAppsModel.FindByName(l.ctx, nodeType, req.App)
		if err != nil {
			return nil, err
		}
		prefix = allowApp.Path
		autoCursor = true
	}

	if prefix == "" {
		return nil, errorx.NewDefaultError("未匹配到正确包路径, 请配置应用默认路径或指定路径")
	}

	var packages []types.PackageInfo
	if autoCursor {
		// 根据应用配置自动遍历, 默认检查包命名格式
		packages, err = l.cursorPackagesPast30Days(KodoBucket, prefix, device.DevType(req.DevType), req.App, 50, true)
		if err != nil {
			return nil, err
		}
	} else {
		// 根据指定路径遍历, 不检查包命名格式
		packages, err = l.cursorPackagesByPath(KodoBucket, prefix, device.DevType(req.DevType), req.App, false)
		if err != nil {
			return nil, err
		}
	}

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

	for _, file := range files {
		// 获取文件md5
		url := KodoUrlPrefix + file.Key
		md5, err := kodo.GetKodoFileMd5(url)
		if err != nil {
			l.Errorf("getKodoFileMd5 url:%s error:%s", url, err)
			continue
		}
		packages = append(packages, types.PackageInfo{
			Url:  KodoUrlPrefix + file.Key,
			File: file.Key,
			Md5:  md5,
			Size: unit.ByteConvert(unit.Byte(file.Fsize)),
		})
	}

	if len(packages) > 50 {
		packages = packages[:50]
	}

	return packages, nil
}

func (l *NodeReleasePackagesLogic) cursorPackagesPast30Days(bucket, prefix string, devType device.DevType, app string, limit int, checkFormat bool) (packages []types.PackageInfo, err error) {
	// 按{prefix}/2025080, {prefix}/2025072, {prefix}/2025071, ... 列出最近30天的包
	paths := generatePathsForPastDays(prefix, 30)
	var files []storage.ListItem
	for _, path := range paths {
		pkgs, err := l.getPackages(bucket, path, devType, app, checkFormat)
		if err != nil {
			return nil, err
		}
		files = append(files, pkgs...)
		if len(files) >= limit {
			l.Infof("cursorPackagesPast30Days match max count:%d, stop cursor.", limit)
			break
		}
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].PutTime > files[j].PutTime
	})

	for _, file := range files {
		url := KodoUrlPrefix + file.Key
		md5, err := kodo.GetKodoFileMd5(url)
		if err != nil {
			l.Errorf("getKodoFileMd5 url:%s error:%s", url, err)
			continue
		}
		packages = append(packages, types.PackageInfo{
			Url:  url,
			File: file.Key,
			Md5:  md5,
			Size: unit.ByteConvert(unit.Byte(file.Fsize)),
		})
	}

	l.Errorf("got packages count:%d", len(packages))
	if len(packages) > limit {
		packages = packages[:limit]
	}

	return packages, nil
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
			devTypeContained := strings.Contains(entry.Key, "ant.") || strings.Contains(entry.Key, "jarvis.") || strings.Contains(entry.Key, "droid.")
			if checkFormat && devTypeContained && !strings.Contains(entry.Key, devType.String()) {
				continue
			}

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
