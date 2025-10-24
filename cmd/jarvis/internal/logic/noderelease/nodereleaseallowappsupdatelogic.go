package noderelease

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"

	"hackathon/cmd/jarvis/internal/svc"
	"hackathon/cmd/jarvis/internal/types"
	"hackathon/cmd/jarvis/shared/noderelease"
	"hackathon/common/device"
	"hackathon/common/errorx"
	"hackathon/sharedmodel"
)

type NodeReleaseAllowAppsUpdateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewNodeReleaseAllowAppsUpdateLogic(ctx context.Context, svcCtx *svc.ServiceContext) NodeReleaseAllowAppsUpdateLogic {
	return NodeReleaseAllowAppsUpdateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *NodeReleaseAllowAppsUpdateLogic) NodeReleaseAllowAppsUpdate(req *types.NodeReleaseAllowAppsUpdateReq) error {
	if err := l.isOperationAllow(req); err != nil {
		return err
	}

	return l.updateApps(req, operator)
}

func (l *NodeReleaseAllowAppsUpdateLogic) updateApps(req *types.NodeReleaseAllowAppsUpdateReq, operator string) error {
	switch req.Operation {
	case "add":
		for _, nodeType := range req.NodeTypes {
			if err := l.svcCtx.AllowAppsModel.Insert(l.ctx, &sharedmodel.AllowAppsTemplate{
				Path:     req.Path,
				Name:     req.Name,
				NodeType: nodeType,
				Operator: operator,
				Desc:     req.Desc,
			}); err != nil {
				l.Errorf("insert allowApps %s-%s err: %v", nodeType, req.Name, err)
				return err
			}
		}

	case "del":
		return l.svcCtx.AllowAppsModel.DeleteByID(l.ctx, req.ID)

	case "update":
		// 目前只支持更新路径和描述
		return l.svcCtx.AllowAppsModel.SetAppPathAndDesc(l.ctx, req.ID, req.Path, req.Desc)
	}
	return nil
}

func (l *NodeReleaseAllowAppsUpdateLogic) isOperationAllow(req *types.NodeReleaseAllowAppsUpdateReq) (err error) {

	switch req.Operation {
	case "add":
		// 检查所创建的组件是否已经存在
		var apps []*sharedmodel.AllowAppsTemplate
		if apps, _, err = l.svcCtx.AllowAppsModel.Search(l.ctx, sharedmodel.AllowAppsListCond{
			Names:     []string{req.Name},
			NodeTypes: req.NodeTypes,
		}); err != nil {
			l.Errorf("allowApps search err:%v", err)
			return err
		}

		if len(apps) > 0 {
			var existApps []string
			for _, app := range apps {
				existApps = append(existApps, fmt.Sprintf("%s-%s", app.NodeType, app.Name))
			}
			return errorx.NewDefaultError("组件已存在:" + strings.Join(existApps, " "))
		}

		return nil

	case "del":
		allowApp, err := l.svcCtx.AllowAppsModel.FindByID(l.ctx, req.ID)
		if err != nil {
			l.Errorf("find allowApps err: %v", err)
			return err
		}

		var devTypes []device.DevType
		for _, dev := range device.GetAllDevTypes() {
			if dev.IsSmallBoxDev() && sharedmodel.NodeTypeSmallBox.String() == allowApp.NodeType {
				devTypes = append(devTypes, dev)
			}

			if dev.IsJarvisGroup() && sharedmodel.NodeTypeNode.String() == allowApp.NodeType {
				devTypes = append(devTypes, dev)
			}
		}

		appName := allowApp.Name
		exist, err := l.hasProcessingRelease(appName, devTypes)
		if err != nil {
			l.Errorf("hasProcessingRelease err: %v", err)
			return err
		}

		if exist {
			return errorx.NewDefaultError(fmt.Sprintf("组件%s存在未结束发布任务，请先终止", appName))
		}

		if exist, err = l.hasAppCfgInUpdConfig(appName, devTypes); err != nil {
			l.Errorf("check hasAppCfgInUpdConfig err: %v", err)
			return err
		}

		if exist {
			return errorx.NewDefaultError(fmt.Sprintf("组件%s在upd配置中仍存在，请先移除", req.Name))
		}

		if fails := l.checkAllowNodesCleanInDefault(appName, req.NodeTypes); len(fails) > 0 {
			l.Errorf("checkAllowNodesClean fail info: %+v", fails)
			return errorx.NewDefaultError(fmt.Sprintf("组件%s在灰度节点中仍存在，请先移除", req.Name))
		}

		return nil

	case "update":
		if req.ID == "" {
			return errorx.NewDefaultError("ID不能为空")
		}

		return nil

	default:
		return errorx.NewDefaultError("无效操作:" + req.Operation)
	}
}

func (l *NodeReleaseAllowAppsUpdateLogic) hasProcessingRelease(app string, devTypes []device.DevType) (bool, error) {
	// 检查allowApp是否存在进行中的发布任务
	for _, devType := range devTypes {
		releasePlan, err := l.svcCtx.NodeReleaseModel.FindProcessingRelease(l.ctx, app, nil, devType)
		if err != nil {
			l.Errorf("find release plan err: %v", err)
			return false, err
		}

		if len(releasePlan) > 0 {
			return true, nil
		}
	}

	return false, nil
}

func (l *NodeReleaseAllowAppsUpdateLogic) hasAppCfgInUpdConfig(appName string, devTypes []device.DevType) (bool, error) {
	for _, devType := range devTypes {
		// 检查upd配置中是否仍包含app配置
		updConfKey, err := noderelease.GenUpdConfKey(devType)
		if err != nil {
			return false, err
		}

		cfg, err := l.svcCtx.SysParamModel.FindByName(l.ctx, updConfKey)
		if err != nil && err != sharedmodel.ErrNotFound {
			return false, err
		}

		if err == sharedmodel.ErrNotFound {
			continue
		}

		var jarvisUpdConf types.JarvisUpdConf
		if err = json.Unmarshal([]byte(cfg.Value), &jarvisUpdConf); err != nil {
			return false, err
		}

		if _, ok := jarvisUpdConf.Apps[appName]; ok {
			return true, nil
		}

	}

	return false, nil
}

func (l *NodeReleaseAllowAppsUpdateLogic) checkAllowNodesCleanInDefault(appName string, nodeTypes []string) (failInfos []string) {
	// 检查灰度节点中是否仍包含当前app
	allowNodesExistCheckFn := func(nodeType string) (bool, error) {
		topic, err := noderelease.GetAllowNodesTopic(l.ctx, nil, l.svcCtx.AllowAppsModel, appName, nodeType, "")
		if err != nil {
			l.Errorf("get allow nodes topic err: %v", err)
			return false, err
		}

		// 检查相应的redis set中是否仍存在元素
		cnt, err := noderelease.GetAllowNodesCount(l.ctx, l.svcCtx.BizRedis, topic)
		if err != nil {
			return false, err
		}

		return cnt > 0, nil
	}

	for _, nodeType := range nodeTypes {
		exist, err := allowNodesExistCheckFn(nodeType)
		if err != nil {
			failInfos = append(failInfos, fmt.Sprintf("获取组件%s灰度节点失败:%s", appName, err.Error()))
			break
		}

		if exist {
			failInfos = append(failInfos, fmt.Sprintf("组件%s在%s中仍存在灰度节点", appName, nodeType))
			break
		}

	}

	return failInfos
}
