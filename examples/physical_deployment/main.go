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
	env, err := environment.NewEnvironment("physical", map[string]interface{}{
		"hosts": []string{"192.168.1.10", "192.168.1.11", "192.168.1.12"},
		"ssh": map[string]interface{}{
			"user":     "deploy",
			"key_path": "~/.ssh/deploy_key",
			"port":     22,
		},
	})
	if err != nil {
		panic(err)
	}

	healthChecker := health.NewHealthChecker(0.75)

	rollbackCtrl := rollback.NewRollbackController(env, healthChecker, &rollback.RollbackPolicy{
		MaxHealthCheckFailures: 2,
		HealthCheckInterval:    20 * time.Second,
		ErrorRateThreshold:     0.15,
		AutoRollback:           true,
	})

	orch := orchestrator.NewOrchestrator(healthChecker, rollbackCtrl)

	ctx := context.Background()

	config := &orchestrator.DeploymentConfig{
		Service:         "api-gateway",
		Version:         "v2.0.0",
		PreviousVersion: "v1.9.0",
		Environment:     env,
		Strategy: &orchestrator.CanaryStrategy{
			InitialTraffic: 5,
			Increment:      15,
			Interval:       3 * time.Minute,
			MaxTraffic:     100,
		},
		HealthCheck: &health.HealthCheck{
			Type:     "tcp",
			Endpoint: ":8080",
			Timeout:  3 * time.Second,
			Interval: 15 * time.Second,
		},
		Replicas: 3,
	}

	fmt.Println("Starting canary deployment to physical machines...")
	err = orch.Deploy(ctx, config)
	if err != nil {
		fmt.Printf("Deployment failed: %v\n", err)
		return
	}

	fmt.Println("Deployment completed successfully!")
}
