package main

import (
	"context"
	"fmt"
	"time"

	"github.com/PonnyS/Hackathon/pkg/environment"
	"github.com/PonnyS/Hackathon/pkg/health"
	"github.com/PonnyS/Hackathon/pkg/orchestrator"
	"github.com/PonnyS/Hackathon/pkg/rollback"
)

func main() {
	fmt.Println("=== Intelligent Deployment System ===")
	fmt.Println()

	k8sEnv, err := environment.NewEnvironment("kubernetes", map[string]interface{}{
		"namespace": "production",
	})
	if err != nil {
		panic(err)
	}

	physicalEnv, err := environment.NewEnvironment("physical", map[string]interface{}{
		"hosts": []string{"192.168.1.10", "192.168.1.11", "192.168.1.12"},
	})
	if err != nil {
		panic(err)
	}

	healthChecker := health.NewHealthChecker(0.8)
	rollbackCtrl := rollback.NewRollbackController(k8sEnv, healthChecker, &rollback.RollbackPolicy{
		MaxHealthCheckFailures: 3,
		HealthCheckInterval:    10 * time.Second,
		AutoRollback:           true,
	})

	orch := orchestrator.NewOrchestrator(healthChecker, rollbackCtrl)

	ctx := context.Background()

	fmt.Println("Example 1: Deploy to Kubernetes with canary strategy")
	fmt.Println("-------------------------------------------------------")
	k8sConfig := &orchestrator.DeploymentConfig{
		Service:         "user-service",
		Version:         "v1.2.0",
		PreviousVersion: "v1.1.0",
		Environment:     k8sEnv,
		Strategy: &orchestrator.CanaryStrategy{
			InitialTraffic: 10,
			Increment:      20,
			Interval:       30 * time.Second,
			MaxTraffic:     100,
		},
		HealthCheck: &health.HealthCheck{
			Type:     "http",
			Endpoint: "/health",
			Timeout:  5 * time.Second,
			Interval: 10 * time.Second,
		},
		Replicas: 5,
	}

	err = orch.Deploy(ctx, k8sConfig)
	if err != nil {
		fmt.Printf("Deployment failed: %v\n", err)
	}

	fmt.Println()
	fmt.Println("Example 2: Deploy to physical machines")
	fmt.Println("---------------------------------------")
	
	rollbackCtrlPhysical := rollback.NewRollbackController(physicalEnv, healthChecker, &rollback.RollbackPolicy{
		MaxHealthCheckFailures: 2,
		HealthCheckInterval:    15 * time.Second,
		AutoRollback:           true,
	})
	
	orchPhysical := orchestrator.NewOrchestrator(healthChecker, rollbackCtrlPhysical)
	
	physicalConfig := &orchestrator.DeploymentConfig{
		Service:         "api-gateway",
		Version:         "v2.0.0",
		PreviousVersion: "v1.9.0",
		Environment:     physicalEnv,
		Strategy: &orchestrator.CanaryStrategy{
			InitialTraffic: 5,
			Increment:      15,
			Interval:       45 * time.Second,
			MaxTraffic:     100,
		},
		HealthCheck: &health.HealthCheck{
			Type:     "tcp",
			Endpoint: ":8080",
			Timeout:  3 * time.Second,
			Interval: 10 * time.Second,
		},
		Replicas: 3,
	}

	err = orchPhysical.Deploy(ctx, physicalConfig)
	if err != nil {
		fmt.Printf("Deployment failed: %v\n", err)
	}

	fmt.Println()
	fmt.Println("Example 3: Parallel deployments")
	fmt.Println("--------------------------------")
	
	go func() {
		config := &orchestrator.DeploymentConfig{
			Service:         "user-service",
			Version:         "v1.3.0",
			PreviousVersion: "v1.2.0",
			Environment:     k8sEnv,
			Strategy: &orchestrator.CanaryStrategy{
				InitialTraffic: 10,
				Increment:      10,
				Interval:       20 * time.Second,
				MaxTraffic:     100,
			},
			HealthCheck: &health.HealthCheck{
				Type:     "http",
				Endpoint: "/health",
				Timeout:  5 * time.Second,
				Interval: 10 * time.Second,
			},
			Replicas: 5,
		}
		
		err := orch.Deploy(ctx, config)
		if err != nil {
			fmt.Printf("v1.3.0 deployment failed: %v\n", err)
		} else {
			fmt.Println("v1.3.0 deployment completed successfully")
		}
	}()

	time.Sleep(2 * time.Second)
	
	activeDeployments := orch.GetActiveDeployments("user-service")
	fmt.Printf("Active deployments for user-service: %d\n", len(activeDeployments))
	for _, dep := range activeDeployments {
		fmt.Printf("  - Version %s: %s (progress: %d%%)\n", dep.Version, dep.State, dep.Progress)
	}

	time.Sleep(5 * time.Second)
	
	fmt.Println()
	fmt.Println("=== Deployment System Demo Completed ===")
}
