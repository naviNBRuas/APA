package robustness

import (
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"sync"
	"time"
)

type RetryManager struct {
	logger    *slog.Logger
	policies  map[string]RetryPolicy
	executors map[string]*RetryExecutor
	metrics   *RetryMetrics

	mu sync.RWMutex
}

type RetryPolicy struct {
	MaxAttempts   int           `yaml:"max_attempts"`
	InitialDelay  time.Duration `yaml:"initial_delay"`
	MaxDelay      time.Duration `yaml:"max_delay"`
	BackoffFactor float64       `yaml:"backoff_factor"`
	Jitter        bool          `yaml:"jitter"`
	Timeout       time.Duration `yaml:"timeout"`
	Condition     string        `yaml:"condition"`
}

type RetryMetrics struct {
	TotalAttempts     int64         `json:"total_attempts"`
	SuccessfulRetries int64         `json:"successful_retries"`
	FailedRetries     int64         `json:"failed_retries"`
	AverageDelay      time.Duration `json:"average_delay"`
	MaxDelay          time.Duration `json:"max_delay"`
}

type RetryExecutor struct {
	Success      bool
	LastAttempt  time.Time
	AttemptsMade int
}

func NewRetryManager(logger *slog.Logger, policies map[string]RetryPolicy) *RetryManager {
	return &RetryManager{logger: logger, policies: policies, executors: make(map[string]*RetryExecutor), metrics: &RetryMetrics{}}
}

func (rm *RetryManager) ExecuteWithRetry(operation func() error, policyName string) error {
	rm.mu.RLock()
	policy, exists := rm.policies[policyName]
	rm.mu.RUnlock()

	if !exists {
		policy = RetryPolicy{
			MaxAttempts:   3,
			InitialDelay:  100 * time.Millisecond,
			MaxDelay:      5 * time.Second,
			BackoffFactor: 2.0,
			Jitter:        true,
		}
	}

	executor := &RetryExecutor{}
	rm.mu.Lock()
	rm.executors[policyName] = executor
	rm.mu.Unlock()

	var lastErr error
	delay := policy.InitialDelay

	for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
		err := operation()
		executor.LastAttempt = time.Now()
		executor.AttemptsMade = attempt

		rm.mu.Lock()
		rm.metrics.TotalAttempts++
		if delay > rm.metrics.MaxDelay {
			rm.metrics.MaxDelay = delay
		}
		rm.mu.Unlock()

		if err == nil {
			executor.Success = true
			rm.mu.Lock()
			rm.metrics.SuccessfulRetries++
			rm.mu.Unlock()
			return nil
		}

		lastErr = err

		if attempt == policy.MaxAttempts {
			break
		}

		delay = time.Duration(float64(delay) * policy.BackoffFactor)
		if delay > policy.MaxDelay {
			delay = policy.MaxDelay
		}

		if policy.Jitter {
			jitter := time.Duration(rand.Int63n(int64(delay) / 4))
			delay += jitter
		}

		time.Sleep(delay)

		rm.mu.Lock()
		avgAttempts := float64(rm.metrics.TotalAttempts)
		if avgAttempts > 0 {
			totalDelay := float64(rm.metrics.AverageDelay) * (avgAttempts - 1) / avgAttempts
			rm.metrics.AverageDelay = time.Duration(totalDelay + float64(delay)/avgAttempts)
		}
		rm.mu.Unlock()
	}

	rm.mu.Lock()
	rm.metrics.FailedRetries++
	rm.mu.Unlock()

	return fmt.Errorf("all %d retry attempts failed: %w", policy.MaxAttempts, lastErr)
}

func (rm *RetryManager) GetMetrics() *RetryMetrics {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	m := *rm.metrics
	return &m
}

func (rm *RetryManager) Shutdown() {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.executors = nil
	rm.policies = nil
	rm.metrics = nil
	rm.logger.Debug("retry manager shut down")
}

func backoffDelay(attempt int, policy RetryPolicy) time.Duration {
	delay := float64(policy.InitialDelay) * math.Pow(policy.BackoffFactor, float64(attempt-1))
	if delay > float64(policy.MaxDelay) {
		delay = float64(policy.MaxDelay)
	}
	d := time.Duration(delay)
	if policy.Jitter {
		jitter := time.Duration(rand.Int63n(int64(d) / 4))
		d += jitter
	}
	return d
}
