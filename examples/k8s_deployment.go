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
	env, err := environment.NewEnvironment("kubernetes", map[string]interface{}{
		"namespace":  "production",
		"kubeconfig": "~/.kube/config",
	})
	if err != nil {
		panic(err)
	}

	healthChecker := health.NewHealthChecker(0.8)

	rollbackCtrl := rollback.NewRollbackController(env, healthChecker, &rollback.RollbackPolicy{
		MaxHealthCheckFailures: 3,
		HealthCheckInterval:    30 * time.Second,
		ErrorRateThreshold:     0.1,
		AutoRollback:           true,
	})

	orch := orchestrator.NewOrchestrator(healthChecker, rollbackCtrl)

	ctx := context.Background()

	config := &orchestrator.DeploymentConfig{
		Service:         "user-service",
		Version:         "v1.2.0",
		PreviousVersion: "v1.1.0",
		Environment:     env,
		Strategy: &orchestrator.CanaryStrategy{
			InitialTraffic: 10,
			Increment:      10,
			Interval:       2 * time.Minute,
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

	fmt.Println("Starting canary deployment to Kubernetes...")
	err = orch.Deploy(ctx, config)
	if err != nil {
		fmt.Printf("Deployment failed: %v\n", err)
		return
	}

	fmt.Println("Deployment completed successfully!")
}
