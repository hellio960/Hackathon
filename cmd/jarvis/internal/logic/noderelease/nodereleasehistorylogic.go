package noderelease

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"hackathon/cmd/jarvis/internal/svc"
	"hackathon/cmd/jarvis/internal/types"
	"hackathon/common/utils"
	"hackathon/sharedmodel"
)

type NodeReleaseHistoryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewNodeReleaseHistoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) NodeReleaseHistoryLogic {
	return NodeReleaseHistoryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *NodeReleaseHistoryLogic) NodeReleaseHistory(req *types.NodeReleaseHistoryReq) (resp *types.NodeReleaseHistoryResp, err error) {
	// 查询发布历史
	releases, _, err := l.svcCtx.NodeReleaseHistoryModel.Search(l.ctx, &sharedmodel.NodeReleaseHistoryCond{
		PageParam: sharedmodel.PageParam{
			SortKey:       "opTime",
			SortType:      sharedmodel.SortTypeDesc,
			NoDefaultSort: true,
		},
		ReleaseID: req.ReleaseID,
	})
	if err != nil {
		l.Errorf("NodeReleaseHistoryModel.Search(l.ctx, %+v):%v", req, err)
		return nil, err
	}

	resp = &types.NodeReleaseHistoryResp{
		Items: make([]types.NodeReleaseHistoryItem, 0, len(releases)),
	}
	for _, release := range releases {
		var item types.NodeReleaseHistoryItem
		if errN := utils.TransformStruct(release, &item); errN != nil {
			l.Errorf("TransformStruct(release, &item):%v", errN)
		}

		item.OpTime = release.OpTime.Unix()
		resp.Items = append(resp.Items, item)
	}

	return
}
