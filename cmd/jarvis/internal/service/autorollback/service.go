package autorollback

import (
	"context"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"hackathon/cmd/jarvis/internal/executor"
	"hackathon/cmd/jarvis/internal/svc"
	"hackathon/cmd/jarvis/shared/noderelease"
	"hackathon/common/device"
	"hackathon/sharedmodel"
)

type AutoRollbackService struct {
	svcCtx   *svc.ServiceContext
	executor executor.DeploymentExecutor
}

func NewAutoRollbackService(svcCtx *svc.ServiceContext, executor executor.DeploymentExecutor) *AutoRollbackService {
	return &AutoRollbackService{
		svcCtx:   svcCtx,
		executor: executor,
	}
}

func (s *AutoRollbackService) ExecuteRollback(ctx context.Context, releaseID string, reason string) error {
	logx.Infof("[AutoRollback] Starting auto-rollback for release %s, reason: %s", releaseID, reason)

	release, err := s.svcCtx.NodeReleaseModel.Find(ctx, releaseID)
	if err != nil {
		logx.Errorf("[AutoRollback] Failed to find release %s: %v", releaseID, err)
		return fmt.Errorf("find release failed: %w", err)
	}

	if release.State != sharedmodel.NodeReleaseStateProcessing {
		logx.Warnf("[AutoRollback] Release %s is not in processing state, skipping rollback", releaseID)
		return fmt.Errorf("release is not in processing state")
	}

	nodeType := sharedmodel.NodeTypeNode
	if device.DevType(release.DeviceType).IsSmallBoxDev() {
		nodeType = sharedmodel.NodeTypeSmallBox
	}

	topic, err := noderelease.GetAllowNodesTopic(ctx, nil, s.svcCtx.AllowAppsModel, release.App, nodeType.String(), release.ID)
	if err != nil {
		return fmt.Errorf("get allow nodes topic failed: %w", err)
	}

	grayNodes, err := s.svcCtx.BizRedis.Smembers(topic)
	if err != nil {
		return fmt.Errorf("get gray nodes failed: %w", err)
	}

	if len(grayNodes) > 0 && release.MainConfig != nil {
		rollbackReq := &executor.RollbackRequest{
			ReleaseID:  releaseID,
			NodeIDs:    grayNodes,
			AppConfig:  release.MainConfig,
			DeviceType: release.DeviceType,
			AppName:    release.App,
		}

		if err := s.executor.Rollback(ctx, rollbackReq); err != nil {
			logx.Errorf("[AutoRollback] Failed to execute rollback for release %s: %v", releaseID, err)
			return fmt.Errorf("execute rollback failed: %w", err)
		}
	}

	if err := noderelease.SaveAllowNodes(ctx, s.svcCtx.GrayNodesModel, s.svcCtx.BizRedis, topic, release.ID); err != nil {
		return fmt.Errorf("save allow nodes failed: %w", err)
	}

	if err := noderelease.ClearAllowNodes(ctx, s.svcCtx.BizRedis, topic); err != nil {
		return fmt.Errorf("clear allow nodes failed: %w", err)
	}

	releaseBackup := *release
	release.State = sharedmodel.NodeReleaseStateRollbacked
	release.EndAt = time.Now()
	release.Describe = fmt.Sprintf("[AUTO-ROLLBACK] %s - %s", reason, release.Describe)

	if err := s.svcCtx.NodeReleaseModel.Update(ctx, release); err != nil {
		return fmt.Errorf("update release failed: %w", err)
	}

	if err := s.svcCtx.NodeReleaseHistoryModel.Insert(ctx, &sharedmodel.NodeReleaseHistory{
		ReleaseID:   release.ID,
		Operation:   "auto_rollback",
		Operator:    "system",
		OpTime:      time.Now(),
		CreateAt:    time.Now(),
		UpdateAt:    time.Now(),
		BeforeState: releaseBackup.State,
		AfterState:  release.State,
	}); err != nil {
		logx.Errorf("[AutoRollback] Failed to record history: %v", err)
	}

	logx.Infof("[AutoRollback] Successfully completed auto-rollback for release %s", releaseID)
	return nil
}
