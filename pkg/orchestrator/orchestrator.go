package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/PonnyS/Hackathon/pkg/environment"
	"github.com/PonnyS/Hackathon/pkg/health"
	"github.com/PonnyS/Hackathon/pkg/rollback"
	"github.com/PonnyS/Hackathon/pkg/version"
)

type CanaryStrategy struct {
	InitialTraffic int
	Increment      int
	Interval       time.Duration
	MaxTraffic     int
}

type DeploymentConfig struct {
	Service         string
	Version         string
	PreviousVersion string
	Environment     environment.Environment
	Strategy        *CanaryStrategy
	HealthCheck     *health.HealthCheck
	Replicas        int
}

type Orchestrator struct {
	versionManager     *version.VersionManager
	healthChecker      *health.HealthChecker
	rollbackController *rollback.RollbackController
}

func NewOrchestrator(healthChecker *health.HealthChecker, rollbackController *rollback.RollbackController) *Orchestrator {
	return &Orchestrator{
		versionManager:     version.NewVersionManager(),
		healthChecker:      healthChecker,
		rollbackController: rollbackController,
	}
}

func (o *Orchestrator) Deploy(ctx context.Context, config *DeploymentConfig) error {
	deployment, err := o.versionManager.StartDeployment(config.Service, config.Version)
	if err != nil {
		return fmt.Errorf("failed to start deployment: %w", err)
	}

	fmt.Printf("[Orchestrator] Starting deployment of %s:%s (previous: %s)\n", 
		config.Service, config.Version, config.PreviousVersion)

	if config.Strategy == nil {
		config.Strategy = &CanaryStrategy{
			InitialTraffic: 10,
			Increment:      10,
			Interval:       2 * time.Minute,
			MaxTraffic:     100,
		}
	}

	o.healthChecker.RegisterCheck(config.Service, config.Version, config.HealthCheck)

	deployment.UpdateState(version.StateDeploying)
	
	target := &environment.DeploymentTarget{
		Service:  config.Service,
		Version:  config.Version,
		Replicas: config.Replicas,
	}
	
	err = config.Environment.Deploy(ctx, target)
	if err != nil {
		o.versionManager.FailDeployment(config.Service, config.Version, err)
		return fmt.Errorf("deployment failed: %w", err)
	}

	deployment.UpdateState(version.StateHealthCheck)
	
	err = o.canaryRollout(ctx, config, deployment)
	if err != nil {
		o.versionManager.FailDeployment(config.Service, config.Version, err)
		return fmt.Errorf("canary rollout failed: %w", err)
	}

	deployment.UpdateState(version.StateCompleted)
	o.versionManager.CompleteDeployment(config.Service, config.Version)
	
	fmt.Printf("[Orchestrator] Successfully deployed %s:%s\n", config.Service, config.Version)
	return nil
}

func (o *Orchestrator) canaryRollout(ctx context.Context, config *DeploymentConfig, deployment *version.VersionDeployment) error {
	currentTraffic := config.Strategy.InitialTraffic
	
	weights := map[string]int{
		config.Version:         currentTraffic,
		config.PreviousVersion: 100 - currentTraffic,
	}
	
	err := config.Environment.SetTraffic(ctx, config.Service, weights)
	if err != nil {
		return fmt.Errorf("failed to set initial traffic: %w", err)
	}

	fmt.Printf("[Orchestrator] Initial canary traffic: %d%% to %s\n", currentTraffic, config.Version)

	monitorCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		_, err := o.rollbackController.MonitorAndDecide(monitorCtx, config.Service, config.Version, config.PreviousVersion)
		if err != nil {
			errChan <- err
		}
	}()

	ticker := time.NewTicker(config.Strategy.Interval)
	defer ticker.Stop()

	for currentTraffic < config.Strategy.MaxTraffic {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errChan:
			cancel()
			return fmt.Errorf("health monitoring triggered rollback: %w", err)
		case <-ticker.C:
			if !o.healthChecker.IsHealthy(config.Service, config.Version) {
				fmt.Printf("[Orchestrator] Health check failed, initiating rollback\n")
				_, rollbackErr := o.rollbackController.Rollback(ctx, config.Service, config.Version, 
					config.PreviousVersion, rollback.ReasonHealthCheckFailed, "Canary health check failed")
				if rollbackErr != nil {
					return fmt.Errorf("rollback failed: %w", rollbackErr)
				}
				return fmt.Errorf("deployment rolled back due to health check failure")
			}

			currentTraffic += config.Strategy.Increment
			if currentTraffic > config.Strategy.MaxTraffic {
				currentTraffic = config.Strategy.MaxTraffic
			}

			weights[config.Version] = currentTraffic
			weights[config.PreviousVersion] = 100 - currentTraffic

			err := config.Environment.SetTraffic(ctx, config.Service, weights)
			if err != nil {
				return fmt.Errorf("failed to update traffic: %w", err)
			}

			progress := (currentTraffic * 100) / config.Strategy.MaxTraffic
			deployment.UpdateProgress(currentTraffic, config.Strategy.MaxTraffic)
			
			fmt.Printf("[Orchestrator] Canary progress: %d%% traffic to %s (overall: %d%%)\n", 
				currentTraffic, config.Version, progress)
		}
	}

	cancel()
	fmt.Printf("[Orchestrator] Canary rollout completed successfully for %s:%s\n", 
		config.Service, config.Version)
	
	return nil
}

func (o *Orchestrator) GetActiveDeployments(service string) []*version.VersionDeployment {
	return o.versionManager.GetActiveDeployments(service)
}

func (o *Orchestrator) GetDeploymentStatus(service, version string) (*version.VersionDeployment, error) {
	return o.versionManager.GetDeployment(service, version)
}
