package executor

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type PhysicalExecutor struct {
	config *PhysicalConfig
}

type PhysicalConfig struct {
	SSHUser    string
	SSHKeyPath string
	Timeout    time.Duration
}

func NewPhysicalExecutor(config interface{}) (DeploymentExecutor, error) {
	physicalConfig, ok := config.(*PhysicalConfig)
	if !ok {
		physicalConfig = &PhysicalConfig{
			SSHUser: "root",
			Timeout: 5 * time.Minute,
		}
	}

	return &PhysicalExecutor{
		config: physicalConfig,
	}, nil
}

func (e *PhysicalExecutor) Deploy(ctx context.Context, req *DeployRequest) error {
	logx.Infof("[PhysicalExecutor] Deploying release %s to %d physical nodes", req.ReleaseID, len(req.NodeIDs))

	for _, nodeID := range req.NodeIDs {
		if err := e.deployToNode(ctx, nodeID, req); err != nil {
			logx.Errorf("[PhysicalExecutor] Failed to deploy to node %s: %v", nodeID, err)
			return fmt.Errorf("deploy to node %s failed: %w", nodeID, err)
		}
	}

	logx.Infof("[PhysicalExecutor] Successfully deployed to %d nodes", len(req.NodeIDs))
	return nil
}

func (e *PhysicalExecutor) deployToNode(ctx context.Context, nodeID string, req *DeployRequest) error {
	logx.Infof("[PhysicalExecutor] Deploying to physical machine: %s", nodeID)
	logx.Infof("[PhysicalExecutor] Package URL: %s", req.AppConfig.PackageUrl)
	logx.Infof("[PhysicalExecutor] Work Dir: %s, Cmd: %s", req.AppConfig.WorkDir, req.AppConfig.Cmd)

	time.Sleep(100 * time.Millisecond)
	return nil
}

func (e *PhysicalExecutor) Rollback(ctx context.Context, req *RollbackRequest) error {
	logx.Infof("[PhysicalExecutor] Rolling back release %s on %d physical nodes", req.ReleaseID, len(req.NodeIDs))

	for _, nodeID := range req.NodeIDs {
		if err := e.rollbackNode(ctx, nodeID, req); err != nil {
			logx.Errorf("[PhysicalExecutor] Failed to rollback node %s: %v", nodeID, err)
			return fmt.Errorf("rollback node %s failed: %w", nodeID, err)
		}
	}

	logx.Infof("[PhysicalExecutor] Successfully rolled back %d nodes", len(req.NodeIDs))
	return nil
}

func (e *PhysicalExecutor) rollbackNode(ctx context.Context, nodeID string, req *RollbackRequest) error {
	logx.Infof("[PhysicalExecutor] Rolling back physical machine: %s", nodeID)
	logx.Infof("[PhysicalExecutor] Package URL: %s", req.AppConfig.PackageUrl)

	time.Sleep(100 * time.Millisecond)
	return nil
}

func (e *PhysicalExecutor) GetHealth(ctx context.Context, req *HealthCheckRequest) (*HealthStatus, error) {
	status := &HealthStatus{
		ReleaseID:    req.ReleaseID,
		TotalNodes:   len(req.NodeIDs),
		CheckTime:    time.Now().Unix(),
	}

	healthyNodes := []string{}
	unhealthyNodes := []string{}

	for _, nodeID := range req.NodeIDs {
		healthy, err := e.checkNodeHealth(ctx, nodeID, req.HealthURL)
		if err != nil {
			logx.Warnf("[PhysicalExecutor] Error checking health for node %s: %v", nodeID, err)
			unhealthyNodes = append(unhealthyNodes, nodeID)
			continue
		}

		if healthy {
			healthyNodes = append(healthyNodes, nodeID)
		} else {
			unhealthyNodes = append(unhealthyNodes, nodeID)
		}
	}

	status.HealthyNodes = healthyNodes
	status.UnhealthyNodes = unhealthyNodes
	status.HealthyCount = len(healthyNodes)
	status.UnhealthyCount = len(unhealthyNodes)

	if status.TotalNodes > 0 {
		status.HealthRate = float64(status.HealthyCount) / float64(status.TotalNodes)
	}

	return status, nil
}

func (e *PhysicalExecutor) checkNodeHealth(ctx context.Context, nodeID string, healthURL string) (bool, error) {
	if healthURL == "" {
		return true, nil
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	url := fmt.Sprintf("http://%s%s", nodeID, healthURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}
