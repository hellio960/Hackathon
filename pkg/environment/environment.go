package environment

import (
	"context"
	"fmt"
)

type DeploymentStatus string

const (
	StatusPending   DeploymentStatus = "pending"
	StatusRunning   DeploymentStatus = "running"
	StatusCompleted DeploymentStatus = "completed"
	StatusFailed    DeploymentStatus = "failed"
	StatusRolledBack DeploymentStatus = "rolled_back"
)

type Instance struct {
	ID      string
	Version string
	Status  string
	Host    string
}

type DeploymentTarget struct {
	Service   string
	Version   string
	Instances []string
	Replicas  int
}

type Environment interface {
	Deploy(ctx context.Context, target *DeploymentTarget) error
	GetInstances(ctx context.Context, service string, version string) ([]*Instance, error)
	Rollback(ctx context.Context, service string, fromVersion string, toVersion string) error
	GetTraffic(ctx context.Context, service string) (map[string]int, error)
	SetTraffic(ctx context.Context, service string, weights map[string]int) error
	GetType() string
}

type EnvironmentFactory interface {
	Create(config map[string]interface{}) (Environment, error)
}

var factories = make(map[string]EnvironmentFactory)

func RegisterEnvironment(envType string, factory EnvironmentFactory) {
	factories[envType] = factory
}

func NewEnvironment(envType string, config map[string]interface{}) (Environment, error) {
	factory, ok := factories[envType]
	if !ok {
		return nil, fmt.Errorf("unknown environment type: %s", envType)
	}
	return factory.Create(config)
}
