package health

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusUnhealthy HealthStatus = "unhealthy"
	StatusUnknown   HealthStatus = "unknown"
)

type HealthCheck struct {
	Type     string
	Endpoint string
	Timeout  time.Duration
	Interval time.Duration
}

type HealthResult struct {
	Status    HealthStatus
	Timestamp time.Time
	Message   string
	Latency   time.Duration
}

type ServiceHealth struct {
	Service    string
	Version    string
	Instances  map[string]*HealthResult
	mu         sync.RWMutex
}

func (sh *ServiceHealth) UpdateInstance(instanceID string, result *HealthResult) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.Instances[instanceID] = result
}

func (sh *ServiceHealth) GetHealthyCount() int {
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	
	count := 0
	for _, result := range sh.Instances {
		if result.Status == StatusHealthy {
			count++
		}
	}
	return count
}

func (sh *ServiceHealth) GetHealthyRatio() float64 {
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	
	if len(sh.Instances) == 0 {
		return 0
	}
	
	healthy := 0
	for _, result := range sh.Instances {
		if result.Status == StatusHealthy {
			healthy++
		}
	}
	
	return float64(healthy) / float64(len(sh.Instances))
}

type HealthChecker struct {
	checks    map[string]*HealthCheck
	results   map[string]*ServiceHealth
	mu        sync.RWMutex
	stopChan  chan struct{}
	threshold float64
}

func NewHealthChecker(threshold float64) *HealthChecker {
	if threshold <= 0 || threshold > 1 {
		threshold = 0.8
	}

	return &HealthChecker{
		checks:    make(map[string]*HealthCheck),
		results:   make(map[string]*ServiceHealth),
		stopChan:  make(chan struct{}),
		threshold: threshold,
	}
}

func (hc *HealthChecker) RegisterCheck(service, version string, check *HealthCheck) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	
	key := fmt.Sprintf("%s:%s", service, version)
	hc.checks[key] = check
	
	if _, exists := hc.results[key]; !exists {
		hc.results[key] = &ServiceHealth{
			Service:   service,
			Version:   version,
			Instances: make(map[string]*HealthResult),
		}
	}
}

func (hc *HealthChecker) Check(ctx context.Context, service, version string, instances []string) (*ServiceHealth, error) {
	key := fmt.Sprintf("%s:%s", service, version)
	
	hc.mu.RLock()
	check, exists := hc.checks[key]
	serviceHealth := hc.results[key]
	hc.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("no health check registered for %s", key)
	}

	var wg sync.WaitGroup
	for _, instanceID := range instances {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			result := hc.performCheck(ctx, check, id)
			serviceHealth.UpdateInstance(id, result)
		}(instanceID)
	}
	
	wg.Wait()
	return serviceHealth, nil
}

func (hc *HealthChecker) performCheck(ctx context.Context, check *HealthCheck, instanceID string) *HealthResult {
	start := time.Now()
	
	checkCtx, cancel := context.WithTimeout(ctx, check.Timeout)
	defer cancel()

	select {
	case <-checkCtx.Done():
		return &HealthResult{
			Status:    StatusUnhealthy,
			Timestamp: time.Now(),
			Message:   "health check timeout",
			Latency:   time.Since(start),
		}
	case <-time.After(100 * time.Millisecond):
		return &HealthResult{
			Status:    StatusHealthy,
			Timestamp: time.Now(),
			Message:   "ok",
			Latency:   time.Since(start),
		}
	}
}

func (hc *HealthChecker) IsHealthy(service, version string) bool {
	key := fmt.Sprintf("%s:%s", service, version)
	
	hc.mu.RLock()
	serviceHealth, exists := hc.results[key]
	hc.mu.RUnlock()
	
	if !exists {
		return false
	}
	
	return serviceHealth.GetHealthyRatio() >= hc.threshold
}

func (hc *HealthChecker) GetHealth(service, version string) (*ServiceHealth, error) {
	key := fmt.Sprintf("%s:%s", service, version)
	
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	
	serviceHealth, exists := hc.results[key]
	if !exists {
		return nil, fmt.Errorf("no health data for %s", key)
	}
	
	return serviceHealth, nil
}

func (hc *HealthChecker) Stop() {
	close(hc.stopChan)
}
