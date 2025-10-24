package noderelease

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"hackathon/cmd/jarvis/internal/svc"
	"hackathon/cmd/jarvis/internal/types"
	"hackathon/cmd/jarvis/shared/noderelease"
	"hackathon/common/device"
	"hackathon/sharedmodel"
)

type NodeReleaseAllowNodesExportLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewNodeReleaseAllowNodesExportLogic(ctx context.Context, svcCtx *svc.ServiceContext) NodeReleaseAllowNodesExportLogic {
	return NodeReleaseAllowNodesExportLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *NodeReleaseAllowNodesExportLogic) NodeReleaseAllowNodesExport(req *types.NodeReleaseAllowNodesExportReq) (resp *types.NodeReleaseAllowNodesExportResp, err error) {
	// 查询发布任务详情
	release, err := l.svcCtx.NodeReleaseModel.Find(l.ctx, req.ReleaseID)
	if err != nil {
		l.Errorf("NodeReleaseModel.Find err:%v releaseId:%s", err, req.ReleaseID)
		return nil, err
	}

	var nodes []string
	switch release.State {
	case sharedmodel.NodeReleaseStateProcessing:
		// 查询全量灰度节点
		nodes, err = l.exportAllowNodesFromRedis(l.ctx, l.svcCtx, release)
		if err != nil {
			l.Errorf("exportAllowNodes err:%v release:%+v", err, release)
			return nil, err
		}
	case sharedmodel.NodeReleaseStateCompleted, sharedmodel.NodeReleaseStateRollbacked:
		// 查询全量灰度节点
		nodes, err = l.exportAllowNodesFromMongo(l.ctx, release)
		if err != nil {
			l.Errorf("exportAllowNodes err:%v release:%+v", err, release)
			return nil, err
		}
	}

	resp = &types.NodeReleaseAllowNodesExportResp{
		Nodes: nodes,
	}

	return
}

func (l *NodeReleaseAllowNodesExportLogic) exportAllowNodesFromMongo(ctx context.Context, release *sharedmodel.NodeRelease) (nodes []string, err error) {
	cond := &sharedmodel.GrayNodesSearchCond{
		PageParam: sharedmodel.PageParam{
			Page: 1,
			Size: 1000,
		},
		ReleaseID: release.ID,
		NoCount:   true,
	}

	var mark string
	cond.PageParam.SetMarkSort("_id", sharedmodel.SortTypeAsc)
	for {
		cond.PageParam.SetMark(mark)
		items, newMark, _, err := l.svcCtx.GrayNodesModel.Search(ctx, cond)
		if err != nil {
			l.Errorf("GrayNodesModel.Find err:%v release:%+v", err, release)
			return nil, err
		}

		for _, i := range items {
			nodes = append(nodes, i.NodeId)
		}

		if len(items) < cond.PageParam.Size {
			break
		}
		mark = newMark
	}

	return nodes, nil
}

func (l *NodeReleaseAllowNodesExportLogic) exportAllowNodesFromRedis(ctx context.Context, svcCtx *svc.ServiceContext, release *sharedmodel.NodeRelease) (nodes []string, err error) {
	nodeType, err := device.DevType(release.DeviceType).NodeType()
	if err != nil {
		l.Errorf("GetNodeType err:%v release:%+v", err, release)
		return nil, err
	}

	topic, err := noderelease.GetAllowNodesTopic(l.ctx, nil, l.svcCtx.AllowAppsModel, release.App, nodeType, release.ID)
	if err != nil {
		l.Errorf("GetAllowNodesTopic err:%v release:%+v", err, release)
		return nil, err
	}

	nodes, err = noderelease.GetAllowNodes(l.ctx, l.svcCtx.BizRedis, topic)
	if err != nil {
		l.Errorf("GetAllowNodes err:%v release:%+v", err, release)
		return nil, err
	}

	return nodes, nil
}
