package robustness

import (
	"log/slog"
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

type RetryExecutor struct{ Success bool }

func NewRetryManager(logger *slog.Logger, policies map[string]RetryPolicy) *RetryManager {
	return &RetryManager{logger: logger, policies: policies, executors: make(map[string]*RetryExecutor), metrics: &RetryMetrics{}}
}
