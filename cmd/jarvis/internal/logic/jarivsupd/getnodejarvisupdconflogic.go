package jarivsupd

import (
	"context"
	"encoding/json"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"hackathon/cmd/jarvis/internal/loader"
	"hackathon/cmd/jarvis/internal/svc"
	"hackathon/cmd/jarvis/internal/types"
	"hackathon/cmd/jarvis/shared/noderelease"
	"hackathon/common/device"
	"hackathon/common/errorx"
	"hackathon/common/utils"
	"hackathon/sharedmodel"
)

type GetNodeJarvisUpdConfLogic struct {
	logx.Logger
	ctx                  context.Context
	svcCtx               *svc.ServiceContext
	allowNodesTopicCache noderelease.AllowNodesTopicCache
}

func NewGetNodeJarvisUpdConfLogic(ctx context.Context, svcCtx *svc.ServiceContext) GetNodeJarvisUpdConfLogic {
	return GetNodeJarvisUpdConfLogic{
		Logger:               logx.WithContext(ctx),
		ctx:                  ctx,
		svcCtx:               svcCtx,
		allowNodesTopicCache: noderelease.NewAllowNodesTopicCache(),
	}
}

func (l *GetNodeJarvisUpdConfLogic) GetNodeJarvisUpdConf(req types.GetJarvisUpdConfReq) (resp *types.JarvisUpdConf, err error) {

	// 生成updConfKey和updAllowNodesKey
	updConfKey := sharedmodel.JarvisUpdConf
	devType := device.DevType(req.DevType)
	switch {
	case devType.IsAntGroup():
		updConfKey = sharedmodel.AntUpdConf + "_" + req.DevType
	case devType.IsDroidGroup():
		updConfKey = sharedmodel.DroidUpdConf + "_" + req.DevType
	case devType.IsJarvisGroup():
	default:
		return nil, errorx.NewDefaultError("设备类型无效")
	}

	sysParam, err := l.svcCtx.SysParamModel.FindByName(l.ctx, updConfKey)
	if err != nil && err != sharedmodel.ErrNotFound {
		l.Errorf("find system param %s error: %v", updConfKey, err)
		return nil, errorx.NewDefaultError("获取配置失败")
	}
	if sysParam == nil {
		return nil, errorx.NewDefaultError("配置不存在")
	}

	resp = &types.JarvisUpdConf{}
	if err = json.Unmarshal([]byte(sysParam.Value), resp); err != nil {
		l.Logger.Errorf("fail unmarshal %s, %s", updConfKey, err.Error())
		return
	}

	// 分别确认当前节点的各app是否在灰度中, 是则返回alter, 否则返回main+main
	for name, item := range resp.Apps {
		if err = l.implementAlterByAllowNodes(name, req.NodeID, device.DevType(req.DevType), item); err != nil {
			l.Errorf("implement alter for %s failed,err: %s", name, err.Error())
			return nil, err
		}

		// 加应用/移除应用时，统一为alter
		if (item.Main == nil && item.Alter != nil) || (item.Main != nil && item.Alter == nil) {
			item.Main = item.Alter
		}

		// 加应用时，白名单外节点会拿到全空配置，删除
		if item.Main == nil && item.Alter == nil {
			delete(resp.Apps, name)
		}
	}

	// 从内存缓存做查询, 确保性能
	releases := loader.NodeReleaseLoader.GetProcessingNewAppRelease(req.DevType)
	//releases, err := l.svcCtx.NodeReleaseModel.FindProcessingNewAppRelease(l.ctx, device.DevType(req.DevType))
	//if err != nil {
	//	l.Errorf("find processing new app release err: %v", err)
	//	return nil, err
	//}
	if len(releases) > 0 {
		for _, release := range releases {
			// 检查是否在灰度列表里
			nodeType := sharedmodel.NodeTypeNode
			if device.DevType(release.DeviceType).IsSmallBoxDev() {
				nodeType = sharedmodel.NodeTypeSmallBox
			}
			topic, err := noderelease.GetAllowNodesTopicNoCheck(release.App, nodeType.String(), release.ID)
			if err != nil {
				return nil, err
			}

			allowed, err := noderelease.CheckNodeIdIsAllowed(l.ctx, l.svcCtx.BizRedis, topic, req.NodeID)
			if err != nil {
				return nil, err
			}

			if allowed {
				var config types.AppUpdInfo
				if err = utils.TransformStruct(release.AlterConfig, &config); err != nil {
					l.Errorf("transform alter config err: %s", err.Error())
					return nil, err
				}
				resp.Apps[release.App] = &types.Apps{
					Main:  &config,
					Alter: &config,
				}
			}
		}
	}

	// 默认填充升级时间段
	if resp.Name == "" {
		resp.Name = updConfKey
	}
	resp.Timestamp = time.Now().UnixMilli()
	if resp.UpdTimeRange == nil {
		resp.UpdTimeRange = &defaultUpdTimeRange
	}

	if len(resp.Apps) == 0 {
		return nil, errorx.NewDefaultError("配置不存在, 请检查")
	}
	return
}

func (l *GetNodeJarvisUpdConfLogic) implementAlterByAllowNodes(name, nodeId string, devType device.DevType, item *types.Apps) error {
	// 归属默认灰度任务, 从sysParam取到alter; 归属发布任务, 从nodeRelease取到alter
	nodeType, err := devType.NodeType()
	if err != nil {
		return err
	}

	topic, err := noderelease.GetAllowNodesTopicNoCheck(name, nodeType, "")
	if err != nil {
		l.Errorf("fail get allow nodes topic for %s, %s", name, err.Error())
		return err
	}
	// 不在灰度节点中, main和alter都展示为main版本
	exist, err := noderelease.CheckNodeIdIsAllowed(l.ctx, l.svcCtx.BizRedis, topic, nodeId)
	if err != nil {
		l.Errorf("check node id %s in default is allowed err: %s", nodeId, err.Error())
		return err
	}

	if exist {
		return nil
	}

	// 检查是否在某个灰度任务中
	// 从内存缓存做查询, 确保性能
	plans := loader.NodeReleaseLoader.GetProcessingRelease(name, devType.String())
	//plans, err := l.svcCtx.NodeReleaseModel.GetProcessingRelease(l.ctx, name, nil, devType)
	//if err != nil {
	//	l.Logger.Errorf("fail get processing release for %s, %s", name, err.Error())
	//	return err
	//}

	for _, plan := range plans {
		if topic, err = noderelease.GetAllowNodesTopicNoCheck(name, nodeType, plan.ID); err != nil {
			l.Logger.Errorf("fail get processing release for %s, %s", name, err.Error())
			return err
		}

		if exist, err = noderelease.CheckNodeIdIsAllowed(l.ctx, l.svcCtx.BizRedis, topic, nodeId); err != nil {
			l.Logger.Errorf("check node %s allowed err: %s", nodeId, err.Error())
			return err
		}

		if exist {
			// 对于移除app, 直接将alter置空即可
			if plan.OpType == sharedmodel.NodeReleaseOpTypeDeleteApp {
				item.Alter = nil
			} else {
				// 使用发布任务中的alter替换掉默认的alter
				if err = utils.TransformStruct(plan.AlterConfig, item.Alter); err != nil {
					l.Errorf("transform alter config err: %s", err.Error())
					return err
				}
			}

			return nil
		}
	}

	// 节点不存在于任何发布的灰度节点中, 返回main+main
	item.Alter = item.Main
	return nil
}
