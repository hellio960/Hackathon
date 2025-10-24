package executor

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type K8SExecutor struct {
	config *K8SConfig
}

type K8SConfig struct {
	KubeConfigPath string
	Namespace      string
	Timeout        time.Duration
}

func NewK8SExecutor(config interface{}) (DeploymentExecutor, error) {
	k8sConfig, ok := config.(*K8SConfig)
	if !ok {
		k8sConfig = &K8SConfig{
			Namespace: "default",
			Timeout:   5 * time.Minute,
		}
	}

	return &K8SExecutor{
		config: k8sConfig,
	}, nil
}

func (e *K8SExecutor) Deploy(ctx context.Context, req *DeployRequest) error {
	logx.Infof("[K8SExecutor] Deploying release %s to %d nodes", req.ReleaseID, len(req.NodeIDs))

	for _, nodeID := range req.NodeIDs {
		if err := e.deployToNode(ctx, nodeID, req); err != nil {
			logx.Errorf("[K8SExecutor] Failed to deploy to node %s: %v", nodeID, err)
			return fmt.Errorf("deploy to node %s failed: %w", nodeID, err)
		}
	}

	logx.Infof("[K8SExecutor] Successfully deployed to %d nodes", len(req.NodeIDs))
	return nil
}

func (e *K8SExecutor) deployToNode(ctx context.Context, nodeID string, req *DeployRequest) error {
	logx.Infof("[K8SExecutor] Deploying to K8S pod: %s, package: %s", nodeID, req.AppConfig.PackageUrl)
	time.Sleep(100 * time.Millisecond)
	return nil
}

func (e *K8SExecutor) Rollback(ctx context.Context, req *RollbackRequest) error {
	logx.Infof("[K8SExecutor] Rolling back release %s on %d nodes", req.ReleaseID, len(req.NodeIDs))

	for _, nodeID := range req.NodeIDs {
		if err := e.rollbackNode(ctx, nodeID, req); err != nil {
			logx.Errorf("[K8SExecutor] Failed to rollback node %s: %v", nodeID, err)
			return fmt.Errorf("rollback node %s failed: %w", nodeID, err)
		}
	}

	logx.Infof("[K8SExecutor] Successfully rolled back %d nodes", len(req.NodeIDs))
	return nil
}

func (e *K8SExecutor) rollbackNode(ctx context.Context, nodeID string, req *RollbackRequest) error {
	logx.Infof("[K8SExecutor] Rolling back K8S pod: %s to package: %s", nodeID, req.AppConfig.PackageUrl)
	time.Sleep(100 * time.Millisecond)
	return nil
}

func (e *K8SExecutor) GetHealth(ctx context.Context, req *HealthCheckRequest) (*HealthStatus, error) {
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
			logx.Warnf("[K8SExecutor] Error checking health for node %s: %v", nodeID, err)
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

func (e *K8SExecutor) checkNodeHealth(ctx context.Context, nodeID string, healthURL string) (bool, error) {
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
