package version

import (
	"fmt"
	"sync"
	"time"
)

type DeploymentState string

const (
	StateInitializing DeploymentState = "initializing"
	StateDeploying    DeploymentState = "deploying"
	StateHealthCheck  DeploymentState = "health_checking"
	StateStable       DeploymentState = "stable"
	StateRollingBack  DeploymentState = "rolling_back"
	StateFailed       DeploymentState = "failed"
	StateCompleted    DeploymentState = "completed"
)

type VersionDeployment struct {
	Service     string
	Version     string
	State       DeploymentState
	StartTime   time.Time
	CurrentStep int
	TotalSteps  int
	Progress    int
	Error       error
	mu          sync.RWMutex
}

func (v *VersionDeployment) UpdateState(state DeploymentState) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.State = state
}

func (v *VersionDeployment) GetState() DeploymentState {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.State
}

func (v *VersionDeployment) UpdateProgress(current, total int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.CurrentStep = current
	v.TotalSteps = total
	if total > 0 {
		v.Progress = (current * 100) / total
	}
}

type VersionManager struct {
	deployments map[string]map[string]*VersionDeployment
	mu          sync.RWMutex
}

func NewVersionManager() *VersionManager {
	return &VersionManager{
		deployments: make(map[string]map[string]*VersionDeployment),
	}
}

func (vm *VersionManager) StartDeployment(service, version string) (*VersionDeployment, error) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if _, exists := vm.deployments[service]; !exists {
		vm.deployments[service] = make(map[string]*VersionDeployment)
	}

	deployment := &VersionDeployment{
		Service:   service,
		Version:   version,
		State:     StateInitializing,
		StartTime: time.Now(),
	}

	vm.deployments[service][version] = deployment
	return deployment, nil
}

func (vm *VersionManager) GetDeployment(service, version string) (*VersionDeployment, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	serviceDeployments, exists := vm.deployments[service]
	if !exists {
		return nil, fmt.Errorf("no deployments found for service: %s", service)
	}

	deployment, exists := serviceDeployments[version]
	if !exists {
		return nil, fmt.Errorf("version %s not found for service: %s", version, service)
	}

	return deployment, nil
}

func (vm *VersionManager) GetActiveDeployments(service string) []*VersionDeployment {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	serviceDeployments, exists := vm.deployments[service]
	if !exists {
		return []*VersionDeployment{}
	}

	var active []*VersionDeployment
	for _, deployment := range serviceDeployments {
		state := deployment.GetState()
		if state != StateCompleted && state != StateFailed {
			active = append(active, deployment)
		}
	}

	return active
}

func (vm *VersionManager) CompleteDeployment(service, version string) error {
	deployment, err := vm.GetDeployment(service, version)
	if err != nil {
		return err
	}

	deployment.UpdateState(StateCompleted)
	return nil
}

func (vm *VersionManager) FailDeployment(service, version string, err error) error {
	deployment, ferr := vm.GetDeployment(service, version)
	if ferr != nil {
		return ferr
	}

	deployment.mu.Lock()
	deployment.State = StateFailed
	deployment.Error = err
	deployment.mu.Unlock()

	return nil
}
