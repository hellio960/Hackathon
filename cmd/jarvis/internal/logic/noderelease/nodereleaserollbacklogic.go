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

type NodeReleaseRollbackLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewNodeReleaseRollbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) NodeReleaseRollbackLogic {
	return NodeReleaseRollbackLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *NodeReleaseRollbackLogic) NodeReleaseRollback(req *types.NodeReleaseRollbackReq) error {
	// 参数检查
	if err := l.checkBasicArgs(req); err != nil {
		return err
	}

	releasePlan, err := l.svcCtx.NodeReleaseModel.Find(l.ctx, req.ReleaseID)
	if err != nil {
		return err
	}

	// 检查发布任务状态
	if err := l.ensureReleaseState(releasePlan); err != nil {
		return err
	}

	// 正式发布回退, 区分是否进行中, 若为进行中, 标记任务为终止, 并清除灰度节点即可; 若为已完成, 创建一个新的发布任务, 回退到main版本
	releasePlanBackup := *releasePlan

	var prevMain, newMain *sharedmodel.AppConfig
	if releasePlan.State == sharedmodel.NodeReleaseStateCompleted {
		// 将配置回退到main版本
		if prevMain, newMain, err = noderelease.RollbackAppConfig(l.ctx, l.svcCtx.SysParamModel, releasePlan); err != nil {
			return err
		}
	}

	nodeType := sharedmodel.NodeTypeNode
	if device.DevType(releasePlan.DeviceType).IsSmallBoxDev() {
		nodeType = sharedmodel.NodeTypeSmallBox
	}

	topic, err := noderelease.GetAllowNodesTopic(l.ctx, nil, l.svcCtx.AllowAppsModel, releasePlan.App, nodeType.String(), releasePlan.ID)
	if err != nil {
		return err
	}

	// 保存灰度节点
	if err := noderelease.SaveAllowNodes(l.ctx, l.svcCtx.GrayNodesModel, l.svcCtx.BizRedis, topic, releasePlan.ID); err != nil {
		return err
	}

	// 清空灰度节点
	if err := noderelease.ClearAllowNodes(l.ctx, l.svcCtx.BizRedis, topic); err != nil {
		return err
	}

	// 更新发布任务
	releasePlan.State = sharedmodel.NodeReleaseStateRollbacked
	releasePlan.EndAt = time.Now()
	if err := l.svcCtx.NodeReleaseModel.Update(l.ctx, releasePlan); err != nil {
		return err
	}

	// 记录操作日志
	if err := l.svcCtx.NodeReleaseHistoryModel.Insert(l.ctx, &sharedmodel.NodeReleaseHistory{
		ReleaseID: releasePlan.ID,
		Operation: sharedmodel.NodeReleaseHistoryOperationRollback,
		Operator:  operator,
		OpTime:    time.Now(),
		CreateAt:  time.Now(),
		UpdateAt:  time.Now(),

		// 变更信息
		BeforeState: releasePlanBackup.State,
		AfterState:  releasePlan.State,
		// 回滚时, main可能变化
		AppConfigInfo: &sharedmodel.AppConfigRemark{
			BeforeMain: prevMain,
			AfterMain:  newMain,
		},
	}); err != nil {
		return err
	}

	return nil
}

func (l *NodeReleaseRollbackLogic) checkBasicArgs(req *types.NodeReleaseRollbackReq) error {
	if req.ReleaseID == "" {
		return errorx.NewDefaultError("releaseId参数无效, 请检查")
	}

	return nil
}

func (l *NodeReleaseRollbackLogic) ensureReleaseState(releasePlan *sharedmodel.NodeRelease) error {
	rollbackAllowed, err := l.checkRollbackAllowed(l.ctx, l.svcCtx, releasePlan)
	if err != nil {
		return err
	}
	if !rollbackAllowed {
		return errorx.NewDefaultError("当前任务不允许回滚")
	}

	return nil
}

func (l *NodeReleaseRollbackLogic) checkRollbackAllowed(ctx context.Context, svcCtx *svc.ServiceContext, release *sharedmodel.NodeRelease) (rollbackAllowed bool, err error) {

	if release.ReleaseType != sharedmodel.ReleaseTypeFormal {
		return false, errorx.NewDefaultError("非正式发布任务, 禁止操作回滚")
	}

	// 对于回滚, 只针对应用最新一次发布, 且为完成状态的生效;
	// 对于当前在发布中, 未完成的任务, 均可以回滚(区分正式任务和灰度验证任务)
	// 返回任务时, 标记是否可以回滚
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
