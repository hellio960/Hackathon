package noderelease

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"hackathon/cmd/jarvis/internal/svc"
	"hackathon/cmd/jarvis/internal/types"
	"hackathon/sharedmodel"
)

type NodesSearchLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewNodesSearchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NodesSearchLogic {
	return &NodesSearchLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *NodesSearchLogic) NodesSearch(req *types.NodesSearchReq) (resp *types.NodesSearchResp, err error) {
	// 构建搜索条件
	cond := &sharedmodel.NodeSearchCond{
		PageParam: sharedmodel.PageParam{
			Size: req.Size,
		},
	}

	// 如果指定了节点ID，使用过滤方式
	// NodeSearchCond没有NodeIDs字段，需要通过其他方式实现
	// 暂时只支持按条件搜索

	// 设备类型
	if req.DevType != "" {
		cond.DeviceType = []string{req.DevType}
	}

	// 节点阶段
	if req.Stage != "" {
		cond.Stage = req.Stage
	}

	// 节点状态
	if req.Status != "" {
		cond.State = req.Status
	}

	// 业务ID
	if len(req.CustomerIds) > 0 {
		cond.CustomerIDs = req.CustomerIds
	}

	// 查询节点
	nodes, _, total, err := l.svcCtx.NodeJoinModel.Search(l.ctx, cond)
	if err != nil {
		l.Logger.Errorf("search nodes failed: %v", err)
		return nil, err
	}

	// 转换为响应格式
	resp = &types.NodesSearchResp{
		Nodes: make([]types.NodeInfo, 0, len(nodes)),
		Total: total,
	}

	for _, node := range nodes {
		resp.Nodes = append(resp.Nodes, types.NodeInfo{
			NodeId:      node.NodeId,
			DeviceType:  node.DeviceType.String(),
			Stage:       node.Stage,
			Status:      node.Status,
			CustomerIDs: node.CustomerIDs,
		})
	}

	return resp, nil
}
