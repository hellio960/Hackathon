package rollback

import (
	"context"
	"fmt"
	"time"

	"github.com/PonnyS/Hackathon/pkg/environment"
	"github.com/PonnyS/Hackathon/pkg/health"
)

type RollbackReason string

const (
	ReasonHealthCheckFailed RollbackReason = "health_check_failed"
	ReasonErrorRateHigh     RollbackReason = "error_rate_high"
	ReasonManualTrigger     RollbackReason = "manual_trigger"
	ReasonTimeout           RollbackReason = "deployment_timeout"
)

type RollbackEvent struct {
	Service       string
	FromVersion   string
	ToVersion     string
	Reason        RollbackReason
	Timestamp     time.Time
	Details       string
	Success       bool
}

type RollbackPolicy struct {
	MaxHealthCheckFailures int
	HealthCheckInterval    time.Duration
	ErrorRateThreshold     float64
	AutoRollback           bool
}

type RollbackController struct {
	env           environment.Environment
	healthChecker *health.HealthChecker
	policy        *RollbackPolicy
	history       []*RollbackEvent
}

func NewRollbackController(env environment.Environment, healthChecker *health.HealthChecker, policy *RollbackPolicy) *RollbackController {
	if policy == nil {
		policy = &RollbackPolicy{
			MaxHealthCheckFailures: 3,
			HealthCheckInterval:    30 * time.Second,
			ErrorRateThreshold:     0.1,
			AutoRollback:           true,
		}
	}

	return &RollbackController{
		env:           env,
		healthChecker: healthChecker,
		policy:        policy,
		history:       make([]*RollbackEvent, 0),
	}
}

func (rc *RollbackController) MonitorAndDecide(ctx context.Context, service, version, previousVersion string) (*RollbackEvent, error) {
	failures := 0
	ticker := time.NewTicker(rc.policy.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			healthy := rc.healthChecker.IsHealthy(service, version)
			
			if !healthy {
				failures++
				fmt.Printf("[Rollback] Health check failed for %s:%s (%d/%d)\n", 
					service, version, failures, rc.policy.MaxHealthCheckFailures)
				
				if failures >= rc.policy.MaxHealthCheckFailures {
					if rc.policy.AutoRollback {
						return rc.Rollback(ctx, service, version, previousVersion, ReasonHealthCheckFailed, 
							fmt.Sprintf("Failed %d consecutive health checks", failures))
					}
					return nil, fmt.Errorf("max health check failures reached, but auto-rollback is disabled")
				}
			} else {
				fmt.Printf("[Rollback] Health check passed for %s:%s\n", service, version)
				return nil, nil
			}
		}
	}
}

func (rc *RollbackController) Rollback(ctx context.Context, service, fromVersion, toVersion string, reason RollbackReason, details string) (*RollbackEvent, error) {
	fmt.Printf("[Rollback] Initiating rollback for %s from %s to %s (reason: %s)\n", 
		service, fromVersion, toVersion, reason)

	event := &RollbackEvent{
		Service:     service,
		FromVersion: fromVersion,
		ToVersion:   toVersion,
		Reason:      reason,
		Timestamp:   time.Now(),
		Details:     details,
		Success:     false,
	}

	err := rc.env.Rollback(ctx, service, fromVersion, toVersion)
	if err != nil {
		event.Success = false
		rc.history = append(rc.history, event)
		return event, fmt.Errorf("rollback failed: %w", err)
	}

	event.Success = true
	rc.history = append(rc.history, event)
	
	fmt.Printf("[Rollback] Successfully rolled back %s from %s to %s\n", 
		service, fromVersion, toVersion)
	
	return event, nil
}

func (rc *RollbackController) GetHistory(service string) []*RollbackEvent {
	if service == "" {
		return rc.history
	}

	filtered := make([]*RollbackEvent, 0)
	for _, event := range rc.history {
		if event.Service == service {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

func (rc *RollbackController) ManualRollback(ctx context.Context, service, fromVersion, toVersion string) (*RollbackEvent, error) {
	return rc.Rollback(ctx, service, fromVersion, toVersion, ReasonManualTrigger, "Manually triggered rollback")
}
