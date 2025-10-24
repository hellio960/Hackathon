package jarivsupd

import (
	"context"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"

	"hackathon/cmd/jarvis/internal/svc"
	"hackathon/cmd/jarvis/internal/types"
	"hackathon/cmd/jarvis/shared/noderelease"
)

type GetAllowNodesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetAllowNodesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAllowNodesLogic {
	return &GetAllowNodesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetAllowNodesLogic) GetAllowNodes(req *types.GetAllowNodesReq) (resp *types.AllowNodeInfo, err error) {
	req.App = strings.ToLower(req.App)
	topic, err := noderelease.GetAllowNodesTopic(l.ctx, nil, l.svcCtx.AllowAppsModel, req.App, req.NodeType, "")
	if err != nil {
		l.Errorf("GetAllowNodes err: %v", err)
		return nil, err
	}

	resp = &types.AllowNodeInfo{}
	v, err := l.svcCtx.BizRedis.Smembers(topic)
	if err != nil {
		l.Logger.Errorf("fail get %s, %s", topic, err.Error())
		return
	}
	resp.Info = append(resp.Info, types.AllowNode{
		App:     req.App,
		NodeIds: v,
	})

	return
}
