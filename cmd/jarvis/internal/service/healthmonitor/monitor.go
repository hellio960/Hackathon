package healthmonitor

import (
	"context"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"hackathon/cmd/jarvis/internal/executor"
	"hackathon/cmd/jarvis/internal/svc"
	"hackathon/sharedmodel"
)

type HealthMonitor struct {
	svcCtx               *svc.ServiceContext
	executor             executor.DeploymentExecutor
	checkInterval        time.Duration
	healthThreshold      float64
	mu                   sync.RWMutex
	monitoredTasks       map[string]*MonitorTask
	stopChan             chan struct{}
	autoRollbackCallback func(ctx context.Context, releaseID string, healthStatus *executor.HealthStatus) error
}

type MonitorTask struct {
	ReleaseID       string
	AppName         string
	DeviceType      string
	HealthURL       string
	NodeIDs         []string
	StartTime       time.Time
	LastCheckTime   time.Time
	LastHealthRate  float64
	CheckCount      int
	UnhealthyCount  int
	Status          string
}

type MonitorConfig struct {
	CheckInterval   time.Duration
	HealthThreshold float64
}

func NewHealthMonitor(svcCtx *svc.ServiceContext, executor executor.DeploymentExecutor, config *MonitorConfig) *HealthMonitor {
	if config == nil {
		config = &MonitorConfig{
			CheckInterval:   30 * time.Second,
			HealthThreshold: 0.8,
		}
	}

	return &HealthMonitor{
		svcCtx:          svcCtx,
		executor:        executor,
		checkInterval:   config.CheckInterval,
		healthThreshold: config.HealthThreshold,
		monitoredTasks:  make(map[string]*MonitorTask),
		stopChan:        make(chan struct{}),
	}
}

func (m *HealthMonitor) Start(ctx context.Context) {
	logx.Info("[HealthMonitor] Starting health monitor service")

	ticker := time.NewTicker(m.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logx.Info("[HealthMonitor] Context cancelled, stopping health monitor")
			return
		case <-m.stopChan:
			logx.Info("[HealthMonitor] Stop signal received, stopping health monitor")
			return
		case <-ticker.C:
			m.checkAllTasks(ctx)
		}
	}
}

func (m *HealthMonitor) Stop() {
	close(m.stopChan)
}

func (m *HealthMonitor) AddMonitorTask(release *sharedmodel.NodeRelease, nodeIDs []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	task := &MonitorTask{
		ReleaseID:  release.ID,
		AppName:    release.App,
		DeviceType: release.DeviceType,
		NodeIDs:    nodeIDs,
		StartTime:  time.Now(),
		Status:     "monitoring",
	}

	if release.AlterConfig != nil {
		task.HealthURL = release.AlterConfig.HealthUrl
	}

	m.monitoredTasks[release.ID] = task
	logx.Infof("[HealthMonitor] Added monitoring task for release %s with %d nodes", release.ID, len(nodeIDs))
}

func (m *HealthMonitor) RemoveMonitorTask(releaseID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.monitoredTasks, releaseID)
	logx.Infof("[HealthMonitor] Removed monitoring task for release %s", releaseID)
}

func (m *HealthMonitor) GetTaskStatus(releaseID string) *MonitorTask {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.monitoredTasks[releaseID]
}

func (m *HealthMonitor) checkAllTasks(ctx context.Context) {
	m.mu.RLock()
	tasks := make([]*MonitorTask, 0, len(m.monitoredTasks))
	for _, task := range m.monitoredTasks {
		tasks = append(tasks, task)
	}
	m.mu.RUnlock()

	for _, task := range tasks {
		if err := m.checkTask(ctx, task); err != nil {
			logx.Errorf("[HealthMonitor] Error checking task %s: %v", task.ReleaseID, err)
		}
	}
}

func (m *HealthMonitor) checkTask(ctx context.Context, task *MonitorTask) error {
	if len(task.NodeIDs) == 0 {
		return nil
	}

	req := &executor.HealthCheckRequest{
		ReleaseID:  task.ReleaseID,
		NodeIDs:    task.NodeIDs,
		HealthURL:  task.HealthURL,
		AppName:    task.AppName,
		DeviceType: task.DeviceType,
	}

	status, err := m.executor.GetHealth(ctx, req)
	if err != nil {
		return err
	}

	m.mu.Lock()
	task.LastCheckTime = time.Now()
	task.LastHealthRate = status.HealthRate
	task.CheckCount++
	m.mu.Unlock()

	logx.Infof("[HealthMonitor] Release %s health check: %d/%d nodes healthy (%.2f%%)",
		task.ReleaseID, status.HealthyCount, status.TotalNodes, status.HealthRate*100)

	if status.HealthRate < m.healthThreshold {
		m.mu.Lock()
		task.UnhealthyCount++
		m.mu.Unlock()

		logx.Warnf("[HealthMonitor] Release %s health rate (%.2f%%) below threshold (%.2f%%), unhealthy count: %d",
			task.ReleaseID, status.HealthRate*100, m.healthThreshold*100, task.UnhealthyCount)

		if task.UnhealthyCount >= 3 {
			logx.Errorf("[HealthMonitor] Release %s has failed health check %d times, triggering auto-rollback",
				task.ReleaseID, task.UnhealthyCount)

			if m.autoRollbackCallback != nil {
				go func() {
					if err := m.autoRollbackCallback(ctx, task.ReleaseID, status); err != nil {
						logx.Errorf("[HealthMonitor] Auto-rollback failed for release %s: %v", task.ReleaseID, err)
					}
				}()
			} else {
				go m.triggerAutoRollback(ctx, task.ReleaseID, status)
			}
		}
	} else {
		m.mu.Lock()
		task.UnhealthyCount = 0
		m.mu.Unlock()
	}

	return nil
}

func (m *HealthMonitor) triggerAutoRollback(ctx context.Context, releaseID string, healthStatus *executor.HealthStatus) {
	logx.Infof("[HealthMonitor] Triggering auto-rollback for release %s", releaseID)

	m.mu.Lock()
	task := m.monitoredTasks[releaseID]
	if task != nil {
		task.Status = "auto_rollback_triggered"
	}
	m.mu.Unlock()
}

func (m *HealthMonitor) SetAutoRollbackCallback(callback func(ctx context.Context, releaseID string, healthStatus *executor.HealthStatus) error) {
	m.autoRollbackCallback = callback
}
