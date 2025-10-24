package noderelease

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"hackathon/cmd/jarvis/internal/svc"
	"hackathon/cmd/jarvis/internal/types"
	"hackathon/cmd/jarvis/shared/noderelease"
	"hackathon/common/device"
	"hackathon/common/errorx"
	"hackathon/sharedmodel"
)

type NodeReleaseCompleteLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewNodeReleaseCompleteLogic(ctx context.Context, svcCtx *svc.ServiceContext) NodeReleaseCompleteLogic {
	return NodeReleaseCompleteLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *NodeReleaseCompleteLogic) NodeReleaseComplete(req *types.NodeReleaseCompleteReq) error {

	// 查询发布任务
	release, err := l.svcCtx.NodeReleaseModel.Find(l.ctx, req.ReleaseID)
	if err != nil {
		return err
	}

	// 检查发布任务状态
	if err := l.ensureReleaseState(release); err != nil {
		return err
	}

	// 正式发布任务调整配置并清空灰度节点, 功能验证发布无须修改配置
	nodeType, err := device.DevType(release.DeviceType).NodeType()
	if err != nil {
		return err
	}
	releaseBackup := *release
	var prev, after *sharedmodel.AppConfig
	if release.ReleaseType == sharedmodel.ReleaseTypeFormal {
		var defaultTopic string // 发布工具对应灰度列表
		defaultTopic, err = noderelease.GetAllowNodesTopic(l.ctx, nil, l.svcCtx.AllowAppsModel, release.App, nodeType, "")
		if err != nil {
			return err
		}

		prev, after, err = noderelease.CompleteAppConfig(l.ctx, l.svcCtx.SysParamModel, release, l.svcCtx.BizRedis, defaultTopic)
		if err != nil {
			return err
		}
	}

	topic, err := noderelease.GetAllowNodesTopic(l.ctx, nil, l.svcCtx.AllowAppsModel, release.App, nodeType, release.ID)
	if err != nil {
		return err
	}
	if err := noderelease.ClearAllowNodes(l.ctx, l.svcCtx.BizRedis, topic); err != nil {
		return err
	}

	release.State = sharedmodel.NodeReleaseStateCompleted
	release.EndAt = time.Now()
	if err = l.svcCtx.NodeReleaseModel.Update(l.ctx, release); err != nil {
		return err
	}

	// 插入操作历史
	historyItem := &sharedmodel.NodeReleaseHistory{
		ReleaseID: release.ID,
		Operation: sharedmodel.NodeReleaseHistoryOperationComplete,
		Operator:  operator,
		OpTime:    time.Now(),
		CreateAt:  time.Now(),
		UpdateAt:  time.Now(),

		// 变更信息
		BeforeState: releaseBackup.State,
		AfterState:  release.State,
		GrayPolicyInfo: &sharedmodel.GrayPolicyRemark{
			BeforePercentage: releaseBackup.GrayPolicy.Percentage,
			AfterPercentage:  release.GrayPolicy.Percentage,
		},
	}
	if prev != nil || after != nil {
		historyItem.AppConfigInfo = &sharedmodel.AppConfigRemark{
			BeforeMain: prev,
			AfterMain:  after,
		}
	}
	return l.svcCtx.NodeReleaseHistoryModel.Insert(l.ctx, historyItem)

}

func (l *NodeReleaseCompleteLogic) ensureReleaseState(release *sharedmodel.NodeRelease) error {
	if release.State != sharedmodel.NodeReleaseStateProcessing {
		return errorx.NewDefaultError("发布任务状态不正确")
	}

	// 功能验证发布无须做以下检查
	if release.ReleaseType == sharedmodel.ReleaseTypeBeta {
		return nil
	}

	if len(release.GrayPolicy.NodeIds) > 0 && release.GrayPolicy.Percentage < 80 {
		return errorx.NewDefaultError("指定灰度节点池中灰度比例过低, 请先完成灰度节点池的灰度")
	}

	if release.ReleaseType == sharedmodel.ReleaseTypeFormal {
		if release.GrayPolicy.Filter != nil && release.GrayPolicy.Percentage < 30 {
			return errorx.NewDefaultError("当前灰度比例过低(最小比例为30%), 请分批灰度后执行全量")
		}

		// 检查操作历史, 至少需要经过3次灰度比例调整
		history, _, err := l.svcCtx.NodeReleaseHistoryModel.Search(l.ctx, &sharedmodel.NodeReleaseHistoryCond{
			ReleaseID: release.ID,
			Operation: []string{sharedmodel.NodeReleaseHistoryOperationContinue.String()},
		})
		if err != nil {
			return err
		}

		var percentageIncreCount int
		for _, h := range history {
			if h.GrayPolicyInfo == nil {
				continue
			}

			if h.GrayPolicyInfo.NodeIdsAdd != nil || h.GrayPolicyInfo.NodeIdsDel != nil {
				continue
			}

			if h.GrayPolicyInfo.AfterPercentage > h.GrayPolicyInfo.BeforePercentage {
				percentageIncreCount++
			}
		}

		if percentageIncreCount < 3 {
			return errorx.NewDefaultError("至少需要经过3次灰度比例调整, 否则禁止执行全量")
		}
	}

	return nil
}
