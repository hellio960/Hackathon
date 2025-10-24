package noderelease

import (
	"context"

	"hackathon/common/utils"

	"github.com/zeromicro/go-zero/core/logx"

	"hackathon/cmd/jarvis/internal/svc"
	"hackathon/cmd/jarvis/internal/types"
	"hackathon/cmd/jarvis/shared/noderelease"
	"hackathon/common/device"
	"hackathon/sharedmodel"
)

type NodeReleaseDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewNodeReleaseDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) NodeReleaseDetailLogic {
	return NodeReleaseDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *NodeReleaseDetailLogic) NodeReleaseDetail(req *types.NodeReleaseDetailReq) (resp *types.NodeReleaseDetailResp, err error) {
	// 查询发布任务详情
	release, err := l.svcCtx.NodeReleaseModel.Find(l.ctx, req.ReleaseId)
	if err != nil {
		l.Errorf("NodeReleaseModel.Find err:%v releaseId:%s", err, req.ReleaseId)
		return nil, err
	}

	// 检查当前任务是否可以回滚
	rollbackAllowed, err := l.checkRollbackAllowed(l.ctx, l.svcCtx, release)
	if err != nil {
		l.Errorf("checkRollbackAllowed err:%v release:%+v", err, release)
		return nil, err
	}

	var item types.NodeReleaseDetail
	if err := utils.TransformStruct(release, &item); err != nil {
		l.Errorf("utils.TransformStruct err:%v release:%+v", err, release)
		// 结构中包含时间相关字段, 类型存在差异, err为预期
		//return nil, err
	}

	item.CreateAt = release.CreateAt.Unix()
	item.UpdateAt = release.UpdateAt.Unix()
	item.EndAt = release.EndAt.Unix()

	// 填充当前灰度中节点信息
	nodes, total, err := l.getAllowNodes(release)
	if err != nil {
		l.Errorf("getAllowNodes err:%v release:%+v", err, release)
		return nil, err
	}

	item.AllowNodes = nodes
	item.AllowNodesTotal = int(total)
	item.RollbackAllowed = rollbackAllowed
	resp = &types.NodeReleaseDetailResp{
		NodeReleaseDetail: item,
	}

	return
}

func (l *NodeReleaseDetailLogic) checkRollbackAllowed(ctx context.Context, svcCtx *svc.ServiceContext, release *sharedmodel.NodeRelease) (rollbackAllowed bool, err error) {
	if release.ReleaseType != sharedmodel.ReleaseTypeFormal {
		return false, nil
	}

	// 对于回滚, 只针对应用最新一次发布, 且为完成状态的生效;
	// 对于当前在发布中, 未完成的任务, 均可以回滚(区分正式任务和灰度验证任务)
	switch release.State {
	case sharedmodel.NodeReleaseStateProcessing:
		return true, nil
	case sharedmodel.NodeReleaseStateCompleted:
		// 检查是否为最后一次正式发布
		releasePlan, err := svcCtx.NodeReleaseModel.FindLastRelease(ctx, release.App, []sharedmodel.ReleaseType{sharedmodel.ReleaseTypeFormal}, device.DevType(release.DeviceType))
		if err != nil {
			return false, err
		}
		if releasePlan.ID == release.ID {
			return true, nil
		}
		return false, nil
	default:
		return false, nil
	}
}

func (l *NodeReleaseDetailLogic) getAllowNodes(release *sharedmodel.NodeRelease) (nodes []string, total int64, err error) {
	nodeType, err := device.DevType(release.DeviceType).NodeType()
	if err != nil {
		l.Errorf("GetNodeType err:%v release:%+v", err, release)
		return nil, 0, err
	}

	topic, err := noderelease.GetAllowNodesTopic(l.ctx, nil, l.svcCtx.AllowAppsModel, release.App, nodeType, release.ID)
	if err != nil {
		l.Errorf("GetAllowNodesTopic err:%v release:%+v", err, release)
		return nil, 0, err
	}

	nodes, err = noderelease.GetAllowNodesWithLimit(l.ctx, l.svcCtx.BizRedis, topic, 100)
	if err != nil {
		l.Errorf("GetAllowNodesWithLimit err:%v release:%+v", err, release)
		return nil, 0, err
	}

	total, err = noderelease.GetAllowNodesCount(l.ctx, l.svcCtx.BizRedis, topic)
	if err != nil {
		l.Errorf("GetAllowNodesCount err:%v release:%+v", err, release)
		return nil, 0, err
	}

	return nodes, total, nil
}
