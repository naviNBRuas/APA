package robustness

import (
	"errors"
	"fmt"
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

var ErrCircuitOpen = errors.New("circuit breaker is open")

func NewCircuitBreaker(logger *slog.Logger, config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{logger: logger, config: config, breakers: make(map[string]*CircuitState), metrics: &CircuitMetrics{}}
}

func (cb *CircuitBreaker) Execute(fn func() error) error {
	return cb.executeForBreaker("default", fn)
}

func (cb *CircuitBreaker) ExecuteWithFallback(fn func() error, fallback func() error) error {
	err := cb.Execute(fn)
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrCircuitOpen) && fallback != nil {
		return fallback()
	}
	if fallback != nil {
		if fbErr := fallback(); fbErr != nil {
			return fmt.Errorf("original: %w; fallback: %v", err, fbErr)
		}
		return nil
	}
	return err
}

func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.breakers = make(map[string]*CircuitState)
	cb.metrics = &CircuitMetrics{}
	cb.logger.Debug("circuit breaker reset")
}

func (cb *CircuitBreaker) GetState() CircuitStateEnum {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	state, exists := cb.breakers["default"]
	if !exists {
		return CircuitClosed
	}
	return state.State
}

func (cb *CircuitBreaker) GetMetrics() CircuitMetrics {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return *cb.metrics
}

func (cb *CircuitBreaker) Shutdown() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.breakers = nil
	cb.metrics = nil
	cb.logger.Debug("circuit breaker shut down")
}

func (cb *CircuitBreaker) executeForBreaker(name string, fn func() error) error {
	cb.mu.Lock()
	state := cb.getOrCreateState(name)

	if !state.allow() {
		cb.metrics.RejectCalls++
		cb.mu.Unlock()
		return ErrCircuitOpen
	}

	start := time.Now()
	cb.mu.Unlock()

	err := fn()

	elapsed := time.Since(start)

	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.metrics.TotalCalls++
	avg := (cb.metrics.AverageLatency.Nanoseconds()*cb.metrics.TotalCalls + elapsed.Nanoseconds()) / (cb.metrics.TotalCalls + 1)
	cb.metrics.AverageLatency = time.Duration(avg)
	cb.metrics.TotalCalls++ // increment after avg calculation

	if err != nil {
		cb.metrics.FailureCalls++
		cb.metrics.LastErrorTime = time.Now()
		state.onFailure(cb.config)
		return err
	}

	cb.metrics.SuccessCalls++
	state.onSuccess(cb.config)
	return nil
}

func (cb *CircuitBreaker) getOrCreateState(name string) *CircuitState {
	state, exists := cb.breakers[name]
	if !exists {
		state = &CircuitState{
			Name:    name,
			State:   CircuitClosed,
			Timeout: cb.config.Timeout,
			Metrics: cb.metrics,
		}
		cb.breakers[name] = state
	}
	return state
}

func (cs *CircuitState) allow() bool {
	switch cs.State {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if time.Now().After(cs.NextRetry) {
			cs.State = CircuitHalfOpen
			return true
		}
		return false
	case CircuitHalfOpen:
		return cs.SuccessCount < 1
	default:
		return false
	}
}

func (cs *CircuitState) onFailure(config CircuitBreakerConfig) {
	cs.FailureCount++
	cs.LastError = time.Now()

	switch cs.State {
	case CircuitClosed:
		if config.FailureThreshold > 0 && cs.FailureCount >= config.FailureThreshold {
			cs.State = CircuitOpen
			timeout := config.Timeout
			if timeout <= 0 {
				timeout = 30 * time.Second
			}
			cs.NextRetry = time.Now().Add(timeout)
		}
	case CircuitHalfOpen:
		cs.State = CircuitOpen
		cs.NextRetry = time.Now().Add(config.Timeout)
	}
}

func (cs *CircuitState) onSuccess(config CircuitBreakerConfig) {
	cs.SuccessCount++

	switch cs.State {
	case CircuitHalfOpen:
		if config.SuccessThreshold > 0 && cs.SuccessCount >= config.SuccessThreshold {
			cs.State = CircuitClosed
			cs.FailureCount = 0
			cs.SuccessCount = 0
		}
	case CircuitClosed:
		cs.FailureCount = 0
	}
}
