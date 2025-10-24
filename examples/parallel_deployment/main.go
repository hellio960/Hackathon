package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/PonnyS/Hackathon/pkg/environment"
	"github.com/PonnyS/Hackathon/pkg/health"
	"github.com/PonnyS/Hackathon/pkg/orchestrator"
	"github.com/PonnyS/Hackathon/pkg/rollback"
)

func main() {
	env, err := environment.NewEnvironment("kubernetes", map[string]interface{}{
		"namespace": "production",
	})
	if err != nil {
		panic(err)
	}

	healthChecker := health.NewHealthChecker(0.8)
	rollbackCtrl := rollback.NewRollbackController(env, healthChecker, nil)
	orch := orchestrator.NewOrchestrator(healthChecker, rollbackCtrl)

	ctx := context.Background()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		config := &orchestrator.DeploymentConfig{
			Service:         "user-service",
			Version:         "v1.2.0",
			PreviousVersion: "v1.1.0",
			Environment:     env,
			Strategy: &orchestrator.CanaryStrategy{
				InitialTraffic: 10,
				Increment:      20,
				Interval:       1 * time.Minute,
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

		fmt.Println("Starting deployment of user-service v1.2.0...")
		err := orch.Deploy(ctx, config)
		if err != nil {
			fmt.Printf("user-service v1.2.0 deployment failed: %v\n", err)
		} else {
			fmt.Println("user-service v1.2.0 deployment completed!")
		}
	}()

	time.Sleep(30 * time.Second)

	wg.Add(1)
	go func() {
		defer wg.Done()
		config := &orchestrator.DeploymentConfig{
			Service:         "user-service",
			Version:         "v1.3.0",
			PreviousVersion: "v1.2.0",
			Environment:     env,
			Strategy: &orchestrator.CanaryStrategy{
				InitialTraffic: 10,
				Increment:      20,
				Interval:       1 * time.Minute,
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

		fmt.Println("Starting deployment of user-service v1.3.0 (parallel)...")
		err := orch.Deploy(ctx, config)
		if err != nil {
			fmt.Printf("user-service v1.3.0 deployment failed: %v\n", err)
		} else {
			fmt.Println("user-service v1.3.0 deployment completed!")
		}
	}()

	time.Sleep(2 * time.Second)
	activeDeployments := orch.GetActiveDeployments("user-service")
	fmt.Printf("\nActive parallel deployments: %d\n", len(activeDeployments))
	for _, dep := range activeDeployments {
		fmt.Printf("  - Version %s: %s (progress: %d%%)\n", dep.Version, dep.State, dep.Progress)
	}

	wg.Wait()
	fmt.Println("\nAll deployments completed!")
}
