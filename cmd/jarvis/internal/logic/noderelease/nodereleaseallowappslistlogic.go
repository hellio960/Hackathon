package noderelease

import (
	"context"
	"hackathon/common/utils"
	"hackathon/sharedmodel"

	"go.mongodb.org/mongo-driver/bson"

	"hackathon/cmd/jarvis/internal/svc"
	"hackathon/cmd/jarvis/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type NodeReleaseAllowAppsListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewNodeReleaseAllowAppsListLogic(ctx context.Context, svcCtx *svc.ServiceContext) NodeReleaseAllowAppsListLogic {
	return NodeReleaseAllowAppsListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *NodeReleaseAllowAppsListLogic) NodeReleaseAllowAppsList(req *types.NodeReleaseAllowAppsListReq) (resp *types.NodeReleaseAllowAppsListResp, err error) {

	cond := sharedmodel.AllowAppsListCond{
		Sort: bson.D{{Key: "createAt", Value: -1}},
	}
	if req.NodeType != "" {
		cond.NodeTypes = append(cond.NodeTypes, req.NodeType)
	}
	if req.App != "" {
		cond.Names = append(cond.Names, req.App)
	}
	apps, _, err := l.svcCtx.AllowAppsModel.Search(l.ctx, cond)
	if err != nil {
		l.Errorf("AllowApps search error: %v", err)
		return nil, err
	}

	resp = &types.NodeReleaseAllowAppsListResp{}
	for _, app := range apps {
		var item types.AllowApp
		if errN := utils.TransformStruct(app, &item); errN != nil {
			l.Errorf("transform app err: %v", errN)
			// 结构包含时间戳, 类型不一致, 预期报错
			//return nil, err
		}

		item.CreateAt = app.CreateAt.Unix()
		resp.Apps = append(resp.Apps, item)
	}

	return
}
