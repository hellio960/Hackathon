package executor

import (
	"context"
	"hackathon/sharedmodel"
)

type DeploymentExecutor interface {
	Deploy(ctx context.Context, req *DeployRequest) error
	Rollback(ctx context.Context, req *RollbackRequest) error
	GetHealth(ctx context.Context, req *HealthCheckRequest) (*HealthStatus, error)
}

type DeployRequest struct {
	ReleaseID   string
	NodeIDs     []string
	AppConfig   *sharedmodel.AppConfig
	DeviceType  string
	AppName     string
}

type RollbackRequest struct {
	ReleaseID   string
	NodeIDs     []string
	AppConfig   *sharedmodel.AppConfig
	DeviceType  string
	AppName     string
}

type HealthCheckRequest struct {
	ReleaseID  string
	NodeIDs    []string
	HealthURL  string
	AppName    string
	DeviceType string
}

type HealthStatus struct {
	ReleaseID      string
	HealthyNodes   []string
	UnhealthyNodes []string
	TotalNodes     int
	HealthyCount   int
	UnhealthyCount int
	HealthRate     float64
	CheckTime      int64
}

type ExecutorType string

const (
	ExecutorTypeK8S      ExecutorType = "k8s"
	ExecutorTypePhysical ExecutorType = "physical"
)

func NewExecutor(executorType ExecutorType, config interface{}) (DeploymentExecutor, error) {
	switch executorType {
	case ExecutorTypeK8S:
		return NewK8SExecutor(config)
	case ExecutorTypePhysical:
		return NewPhysicalExecutor(config)
	default:
		return NewPhysicalExecutor(config)
	}
}
