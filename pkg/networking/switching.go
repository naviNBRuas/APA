//go:build enhanced

package networking

import (
	"log/slog"
	"sync"
	"time"
)

type ProtocolFailureDetector struct {
	logger         *slog.Logger
	failureHistory map[ProtocolType][]*FailureEvent
	thresholds     FailureThresholds

	mu sync.RWMutex
}

type ProtocolSwitchingEngine struct {
	logger            *slog.Logger
	switchingHistory  []*SwitchEvent
	switchingCriteria SwitchingCriteria
	cooldownPeriod    time.Duration

	mu              sync.RWMutex
	lastSwitchTime  time.Time
	currentProtocol ProtocolType
}

type FailureThresholds struct {
	ConsecutiveFailures int           `yaml:"consecutive_failures"`
	TimeWindow          time.Duration `yaml:"time_window"`
	ErrorRateThreshold  float64       `yaml:"error_rate_threshold"`
}

type SwitchingCriteria struct {
	LatencyImprovementThreshold    float64       `yaml:"latency_improvement_threshold"`
	ThroughputImprovementThreshold float64       `yaml:"throughput_improvement_threshold"`
	AvailabilityThreshold          float64       `yaml:"availability_threshold"`
	MinTimeBetweenSwitches         time.Duration `yaml:"min_time_between_switches"`
}

func NewProtocolFailureDetector(logger *slog.Logger, timeout time.Duration) *ProtocolFailureDetector {
	return &ProtocolFailureDetector{
		logger:         logger,
		failureHistory: make(map[ProtocolType][]*FailureEvent),
		thresholds:     FailureThresholds{ConsecutiveFailures: 3, TimeWindow: timeout, ErrorRateThreshold: 0.1},
	}
}

func NewProtocolSwitchingEngine(logger *slog.Logger, adaptive bool) *ProtocolSwitchingEngine {
	return &ProtocolSwitchingEngine{
		logger:           logger,
		switchingHistory: make([]*SwitchEvent, 0),
		switchingCriteria: SwitchingCriteria{
			LatencyImprovementThreshold:    0.2,
			ThroughputImprovementThreshold: 0.15,
			AvailabilityThreshold:          0.95,
			MinTimeBetweenSwitches:         5 * time.Minute,
		},
		cooldownPeriod: 30 * time.Second,
	}
}

func (pse *ProtocolSwitchingEngine) RecordSwitch(event *SwitchEvent) {
	pse.mu.Lock()
	defer pse.mu.Unlock()
	pse.switchingHistory = append(pse.switchingHistory, event)
	pse.lastSwitchTime = event.Timestamp
	pse.currentProtocol = event.ToProtocol
}

func (pfd *ProtocolFailureDetector) RecordFailure(event *FailureEvent) {
	pfd.mu.Lock()
	defer pfd.mu.Unlock()
	pfd.failureHistory[event.Protocol] = append(pfd.failureHistory[event.Protocol], event)
}
