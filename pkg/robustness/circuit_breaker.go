package robustness

import (
	"log/slog"
	"sync"
	"time"
)

type CircuitBreaker struct {
	logger   *slog.Logger
	config   CircuitBreakerConfig
	breakers map[string]*CircuitState
	metrics  *CircuitMetrics

	mu sync.RWMutex
}

type CircuitBreakerConfig struct {
	FailureThreshold int           `yaml:"failure_threshold"`
	SuccessThreshold int           `yaml:"success_threshold"`
	Timeout          time.Duration `yaml:"timeout"`
	HalfOpenMaxCalls int           `yaml:"half_open_max_calls"`
	ResetTimeout     time.Duration `yaml:"reset_timeout"`
	MetricsWindow    time.Duration `yaml:"metrics_window"`
}

type CircuitState struct {
	Name         string           `json:"name"`
	State        CircuitStateEnum `json:"state"`
	FailureCount int              `json:"failure_count"`
	SuccessCount int              `json:"success_count"`
	LastError    time.Time        `json:"last_error"`
	NextRetry    time.Time        `json:"next_retry"`
	Timeout      time.Duration    `json:"timeout"`
	Metrics      *CircuitMetrics  `json:"metrics"`
}

type CircuitMetrics struct {
	TotalCalls     int64         `json:"total_calls"`
	SuccessCalls   int64         `json:"success_calls"`
	FailureCalls   int64         `json:"failure_calls"`
	TimeoutCalls   int64         `json:"timeout_calls"`
	RejectCalls    int64         `json:"reject_calls"`
	LastErrorTime  time.Time     `json:"last_error_time"`
	AverageLatency time.Duration `json:"average_latency"`
}

type CircuitBreakerState struct{}

func NewCircuitBreaker(logger *slog.Logger, config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{logger: logger, config: config, breakers: make(map[string]*CircuitState), metrics: &CircuitMetrics{}}
}
