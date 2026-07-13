package robustness

import (
	"log/slog"
	"sync"
	"time"
)

type FallbackSystem struct {
	logger     *slog.Logger
	strategies []FallbackStrategy
	executors  map[string]*FallbackExecutor
	metrics    *FallbackMetrics

	mu sync.RWMutex
}

type FallbackStrategy struct {
	Name           string           `yaml:"name"`
	Conditions     []string         `yaml:"conditions"`
	Implementation string           `yaml:"implementation"`
	Priority       int              `yaml:"priority"`
	Timeout        time.Duration    `yaml:"timeout"`
	Metrics        *FallbackMetrics `yaml:"metrics"`
}

type FallbackMetrics struct {
	TotalInvocations int64         `json:"total_invocations"`
	SuccessCount     int64         `json:"success_count"`
	FailureCount     int64         `json:"failure_count"`
	AverageLatency   time.Duration `json:"average_latency"`
	LastErrorTime    time.Time     `json:"last_error_time"`
}

type FallbackExecutor struct{}

func NewFallbackSystem(logger *slog.Logger, strategies []FallbackStrategy) *FallbackSystem {
	if strategies == nil {
		strategies = make([]FallbackStrategy, 0)
	}
	return &FallbackSystem{
		logger:     logger,
		strategies: strategies,
		executors:  make(map[string]*FallbackExecutor),
		metrics:    &FallbackMetrics{},
	}
}

func (fs *FallbackSystem) InvokeStrategy(name string) bool {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.metrics.TotalInvocations++

	for _, s := range fs.strategies {
		if s.Name == name {
			start := time.Now()
			fs.metrics.SuccessCount++
			fs.metrics.AverageLatency = time.Duration(
				(int64(fs.metrics.AverageLatency)*fs.metrics.TotalInvocations + int64(time.Since(start))) /
					fs.metrics.TotalInvocations,
			)
			fs.logger.Debug("fallback strategy invoked", "name", name)
			return true
		}
	}

	fs.metrics.FailureCount++
	fs.metrics.LastErrorTime = time.Now()
	return false
}

func (fs *FallbackSystem) GetMetrics() *FallbackMetrics {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	m := *fs.metrics
	return &m
}

func (fs *FallbackSystem) Shutdown() {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.executors = nil
	fs.strategies = nil
	fs.metrics = nil
	fs.logger.Debug("fallback system shut down")
}
