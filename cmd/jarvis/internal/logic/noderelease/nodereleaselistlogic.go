package noderelease

import (
	"context"

	"hackathon/common/utils"

	"github.com/zeromicro/go-zero/core/logx"

	"hackathon/cmd/jarvis/internal/svc"
	"hackathon/cmd/jarvis/internal/types"

	"hackathon/sharedmodel"
)

type NodeReleaseListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewNodeReleaseListLogic(ctx context.Context, svcCtx *svc.ServiceContext) NodeReleaseListLogic {
	return NodeReleaseListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *NodeReleaseListLogic) NodeReleaseList(req *types.NodeReleaseListReq) (resp *types.NodeReleaseListResp, err error) {
	// 查询发布任务列表
	releases, count, err := l.svcCtx.NodeReleaseModel.Search(l.ctx, &sharedmodel.NodeReleaseListCond{
		PageParam: sharedmodel.PageParam{
			SortKey:  "createAt",
			SortType: sharedmodel.SortTypeDesc,
			Page:     req.Page,
			Size:     req.Size,
		},
		App:          req.App,
		DeviceTypes:  req.DevTypes,
		States:       req.States,
		ReleaseTypes: req.ReleaseTypes,
	})
	if err != nil {
		l.Errorf("NodeReleaseModel.Search(l.ctx, %+v):%v", req, err)
		return nil, err
	}

	items, err := transformRelease(releases)
	if err != nil {
		l.Errorf("transformRelease(l.ctx, %+v):%v", req, err)
		return nil, err
	}

	resp = &types.NodeReleaseListResp{
		Items: items,
		Total: count,
	}

	return
}

func transformRelease(releases []*sharedmodel.NodeRelease) (result []types.NodeReleaseBrief, err error) {

	for _, release := range releases {
		var item types.NodeReleaseBrief
		if err := utils.TransformStruct(release, &item); err != nil {
			// 结构中包含时间相关字段, 类型存在差异, err为预期
			// return nil, err
		}

		// 时间做单独转换
		item.CreateAt = release.CreateAt.Unix()
		result = append(result, item)
	}

	return result, nil
}
