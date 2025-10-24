package deploymanager

import (
	"context"
	"fmt"
	"sync"

	"github.com/zeromicro/go-zero/core/logx"

	"hackathon/cmd/jarvis/internal/executor"
	"hackathon/cmd/jarvis/internal/service/autorollback"
	"hackathon/cmd/jarvis/internal/service/healthmonitor"
	"hackathon/cmd/jarvis/internal/svc"
	"hackathon/sharedmodel"
)

type DeployManager struct {
	svcCtx             *svc.ServiceContext
	executors          map[string]executor.DeploymentExecutor
	healthMonitor      *healthmonitor.HealthMonitor
	autoRollbackSvc    *autorollback.AutoRollbackService
	deployConfigs      map[string]*sharedmodel.DeployConfig
	mu                 sync.RWMutex
	monitorCtx         context.Context
	monitorCancel      context.CancelFunc
}

func NewDeployManager(svcCtx *svc.ServiceContext) (*DeployManager, error) {
	dm := &DeployManager{
		svcCtx:        svcCtx,
		executors:     make(map[string]executor.DeploymentExecutor),
		deployConfigs: make(map[string]*sharedmodel.DeployConfig),
	}

	if err := dm.loadDeployConfigs(context.Background()); err != nil {
		logx.Errorf("[DeployManager] Failed to load deploy configs: %v", err)
	}

	return dm, nil
}

func (dm *DeployManager) Start(ctx context.Context) error {
	dm.monitorCtx, dm.monitorCancel = context.WithCancel(ctx)

	defaultExecutor, err := executor.NewExecutor(executor.ExecutorTypePhysical, nil)
	if err != nil {
		return fmt.Errorf("failed to create default executor: %w", err)
	}

	dm.healthMonitor = healthmonitor.NewHealthMonitor(dm.svcCtx, defaultExecutor, &healthmonitor.MonitorConfig{})
	dm.autoRollbackSvc = autorollback.NewAutoRollbackService(dm.svcCtx, defaultExecutor)

	dm.healthMonitor.SetAutoRollbackCallback(func(ctx context.Context, releaseID string, healthStatus *executor.HealthStatus) error {
		reason := fmt.Sprintf("Health check failed: %.2f%% healthy nodes", healthStatus.HealthRate*100)
		return dm.autoRollbackSvc.ExecuteRollback(ctx, releaseID, reason)
	})

	go dm.healthMonitor.Start(dm.monitorCtx)

	logx.Info("[DeployManager] Deploy manager started successfully")
	return nil
}

func (dm *DeployManager) Stop() {
	if dm.monitorCancel != nil {
		dm.monitorCancel()
	}
	if dm.healthMonitor != nil {
		dm.healthMonitor.Stop()
	}
	logx.Info("[DeployManager] Deploy manager stopped")
}

func (dm *DeployManager) loadDeployConfigs(ctx context.Context) error {
	return nil
}

func (dm *DeployManager) getExecutor(deviceType string) (executor.DeploymentExecutor, error) {
	dm.mu.RLock()
	config, ok := dm.deployConfigs[deviceType]
	dm.mu.RUnlock()

	if !ok {
		return executor.NewExecutor(executor.ExecutorTypePhysical, nil)
	}

	executorType := executor.ExecutorType(config.ExecutorType)

	dm.mu.RLock()
	exec, exists := dm.executors[deviceType]
	dm.mu.RUnlock()

	if exists {
		return exec, nil
	}

	var execConfig interface{}
	switch executorType {
	case executor.ExecutorTypeK8S:
		if config.K8SConfig != nil {
			execConfig = &executor.K8SConfig{
				KubeConfigPath: config.K8SConfig.KubeConfigPath,
				Namespace:      config.K8SConfig.Namespace,
			}
		}
	case executor.ExecutorTypePhysical:
		if config.PhysicalCfg != nil {
			execConfig = &executor.PhysicalConfig{
				SSHUser:    config.PhysicalCfg.SSHUser,
				SSHKeyPath: config.PhysicalCfg.SSHKeyPath,
			}
		}
	}

	newExec, err := executor.NewExecutor(executorType, execConfig)
	if err != nil {
		return nil, err
	}

	dm.mu.Lock()
	dm.executors[deviceType] = newExec
	dm.mu.Unlock()

	return newExec, nil
}

func (dm *DeployManager) Deploy(ctx context.Context, release *sharedmodel.NodeRelease, nodeIDs []string) error {
	if release.AlterConfig == nil {
		return fmt.Errorf("alter config is nil")
	}

	exec, err := dm.getExecutor(release.DeviceType)
	if err != nil {
		return fmt.Errorf("failed to get executor: %w", err)
	}

	req := &executor.DeployRequest{
		ReleaseID:  release.ID,
		NodeIDs:    nodeIDs,
		AppConfig:  release.AlterConfig,
		DeviceType: release.DeviceType,
		AppName:    release.App,
	}

	if err := exec.Deploy(ctx, req); err != nil {
		return fmt.Errorf("deploy failed: %w", err)
	}

	dm.mu.RLock()
	config, ok := dm.deployConfigs[release.DeviceType]
	dm.mu.RUnlock()

	if ok && config.AutoRollback && dm.healthMonitor != nil {
		dm.healthMonitor.AddMonitorTask(release, nodeIDs)
		logx.Infof("[DeployManager] Added health monitoring for release %s", release.ID)
	}

	return nil
}

func (dm *DeployManager) Rollback(ctx context.Context, release *sharedmodel.NodeRelease, nodeIDs []string) error {
	if release.MainConfig == nil {
		return fmt.Errorf("main config is nil")
	}

	exec, err := dm.getExecutor(release.DeviceType)
	if err != nil {
		return fmt.Errorf("failed to get executor: %w", err)
	}

	req := &executor.RollbackRequest{
		ReleaseID:  release.ID,
		NodeIDs:    nodeIDs,
		AppConfig:  release.MainConfig,
		DeviceType: release.DeviceType,
		AppName:    release.App,
	}

	if err := exec.Rollback(ctx, req); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	if dm.healthMonitor != nil {
		dm.healthMonitor.RemoveMonitorTask(release.ID)
	}

	return nil
}

func (dm *DeployManager) GetHealthStatus(ctx context.Context, releaseID string) *healthmonitor.MonitorTask {
	if dm.healthMonitor == nil {
		return nil
	}
	return dm.healthMonitor.GetTaskStatus(releaseID)
}

func (dm *DeployManager) TriggerAutoRollback(ctx context.Context, releaseID string, reason string) error {
	if dm.autoRollbackSvc == nil {
		return fmt.Errorf("auto rollback service not initialized")
	}
	return dm.autoRollbackSvc.ExecuteRollback(ctx, releaseID, reason)
}
