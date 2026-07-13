package robustness

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRobustnessManager_NilLogger(t *testing.T) {
	t.Parallel()
	rm, err := NewRobustnessManager(nil, RobustnessConfig{})
	assert.Nil(t, rm)
	require.Error(t, err)
	assert.Equal(t, "logger is required", err.Error())
}

func TestNewRobustnessManager_DefaultConfig(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rm, err := NewRobustnessManager(logger, RobustnessConfig{})
	require.NoError(t, err)
	require.NotNil(t, rm)
	assert.Equal(t, logger, rm.logger)
	assert.False(t, rm.isRunning)
	assert.Nil(t, rm.errorHandler)
	assert.Nil(t, rm.recoveryEngine)
	assert.Nil(t, rm.healthMonitor)
	assert.Nil(t, rm.degradationManager)
	assert.Nil(t, rm.emergencyProtocols)
	assert.NotNil(t, rm.resilienceAnalyzer)
}

func TestNewRobustnessManager_EnabledConfig(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	config := RobustnessConfig{
		EnableErrorHandling:      true,
		EnableSelfHealing:        true,
		EnableHealthMonitoring:   true,
		EnableDegradation:        true,
		EnableEmergencyProtocols: true,
	}
	rm, err := NewRobustnessManager(logger, config)
	require.NoError(t, err)
	assert.NotNil(t, rm.errorHandler)
	assert.NotNil(t, rm.recoveryEngine)
	assert.NotNil(t, rm.healthMonitor)
	assert.NotNil(t, rm.degradationManager)
	assert.NotNil(t, rm.emergencyProtocols)
}

func TestStartStop_Lifecycle(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	config := RobustnessConfig{
		EnableErrorHandling:      true,
		EnableSelfHealing:        true,
		EnableHealthMonitoring:   true,
		EnableDegradation:        true,
		EnableEmergencyProtocols: true,
	}
	rm, err := NewRobustnessManager(logger, config)
	require.NoError(t, err)

	err = rm.Start()
	require.NoError(t, err, "Start()")
	assert.True(t, rm.isRunning)

	err = rm.Start()
	require.Error(t, err)
	assert.Equal(t, "robustness manager is already running", err.Error())

	rm.Stop()
	assert.False(t, rm.isRunning)

	rm.Stop()
}

func TestStartStop_NoComponents(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rm, err := NewRobustnessManager(logger, RobustnessConfig{})
	require.NoError(t, err)

	err = rm.Start()
	require.NoError(t, err, "Start()")
	assert.True(t, rm.isRunning)

	rm.Stop()
}

func TestNewErrorHandler(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	config := ErrorHandlingConfig{
		CircuitBreakerConfig: CircuitBreakerConfig{
			FailureThreshold: 5,
			SuccessThreshold: 2,
			Timeout:          10 * time.Second,
		},
	}

	eh := NewErrorHandler(logger, config)
	require.NotNil(t, eh)
	assert.Equal(t, logger, eh.logger)
	assert.NotNil(t, eh.errorClassifier)
	assert.NotNil(t, eh.errorReporter)
	assert.NotNil(t, eh.retryManager)
	assert.NotNil(t, eh.circuitBreaker)
	assert.NotNil(t, eh.fallbackSystem)
	assert.NotNil(t, eh.errorHistory)
	assert.NotNil(t, eh.classificationCache)
	assert.Equal(t, 5, eh.config.CircuitBreakerConfig.FailureThreshold)
	assert.Equal(t, 2, eh.config.CircuitBreakerConfig.SuccessThreshold)
}

func TestErrorHandler_GetPendingErrors(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	eh := NewErrorHandler(logger, ErrorHandlingConfig{})

	errors := eh.GetPendingErrors()
	require.NotNil(t, errors)
	assert.Empty(t, errors)
}

func TestErrorHandler_ClassifyError(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	eh := NewErrorHandler(logger, ErrorHandlingConfig{})

	err := errors.New("test error")
	classification := eh.ClassifyError(err)
	require.NotNil(t, classification)
	assert.Empty(t, classification.Categories)
	assert.Empty(t, classification.Severity)
}

func TestErrorHandler_ReportError(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	eh := NewErrorHandler(logger, ErrorHandlingConfig{})

	event := &ErrorEvent{
		ID:        "test-1",
		Timestamp: time.Now(),
		Error:     errors.New("test error"),
		Context:   map[string]interface{}{"key": "value"},
	}

	eh.ReportError(event)
}

func TestErrorHandler_Shutdown(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	eh := NewErrorHandler(logger, ErrorHandlingConfig{})

	eh.Shutdown()
}

func TestNewRetryManager(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	policies := map[string]RetryPolicy{
		"default": {
			MaxAttempts:   3,
			InitialDelay:  100 * time.Millisecond,
			MaxDelay:      5 * time.Second,
			BackoffFactor: 2.0,
			Jitter:        true,
		},
	}

	rm := NewRetryManager(logger, policies)
	require.NotNil(t, rm)
	assert.Equal(t, logger, rm.logger)
	assert.Len(t, rm.policies, 1)
	assert.Equal(t, 3, rm.policies["default"].MaxAttempts)
	assert.Equal(t, 2.0, rm.policies["default"].BackoffFactor)
	assert.NotNil(t, rm.executors)
	assert.NotNil(t, rm.metrics)
}

func TestRetryManager_MetricsInitialState(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rm := NewRetryManager(logger, map[string]RetryPolicy{})

	assert.Zero(t, rm.metrics.TotalAttempts)
	assert.Zero(t, rm.metrics.SuccessfulRetries)
	assert.Zero(t, rm.metrics.FailedRetries)
	assert.Zero(t, rm.metrics.AverageDelay)
	assert.Zero(t, rm.metrics.MaxDelay)
}

func TestRetryManager_EmptyPolicies(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rm := NewRetryManager(logger, nil)

	assert.NotNil(t, rm.metrics)
	assert.NotNil(t, rm.executors)
}

func TestNewCircuitBreaker(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	config := CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
		HalfOpenMaxCalls: 3,
		ResetTimeout:     60 * time.Second,
		MetricsWindow:    5 * time.Minute,
	}

	cb := NewCircuitBreaker(logger, config)
	require.NotNil(t, cb)
	assert.Equal(t, logger, cb.logger)
	assert.Equal(t, 5, cb.config.FailureThreshold)
	assert.Equal(t, 2, cb.config.SuccessThreshold)
	assert.Equal(t, 60*time.Second, cb.config.ResetTimeout)
	assert.Equal(t, 3, cb.config.HalfOpenMaxCalls)
	assert.NotNil(t, cb.breakers)
	assert.NotNil(t, cb.metrics)
}

func TestCircuitBreaker_ConfigDefaults(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cb := NewCircuitBreaker(logger, CircuitBreakerConfig{})

	assert.Zero(t, cb.config.FailureThreshold)
	assert.Zero(t, cb.config.Timeout)
	assert.Zero(t, cb.config.MetricsWindow)
}

func TestCircuitBreaker_MetricsInitialState(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cb := NewCircuitBreaker(logger, CircuitBreakerConfig{})

	assert.Zero(t, cb.metrics.TotalCalls)
	assert.Zero(t, cb.metrics.SuccessCalls)
	assert.Zero(t, cb.metrics.FailureCalls)
	assert.Zero(t, cb.metrics.RejectCalls)
	assert.Zero(t, cb.metrics.AverageLatency)
}

func TestNewFallbackSystem(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	strategies := []FallbackStrategy{
		{
			Name:       "cache_fallback",
			Priority:   1,
			Timeout:    5 * time.Second,
			Conditions: []string{"error == timeout"},
		},
		{
			Name:       "degraded_response",
			Priority:   2,
			Timeout:    1 * time.Second,
			Conditions: []string{"error == unavailable"},
		},
	}

	fs := NewFallbackSystem(logger, strategies)
	require.NotNil(t, fs)
	assert.Equal(t, logger, fs.logger)
	assert.Len(t, fs.strategies, 2)
	assert.Equal(t, "cache_fallback", fs.strategies[0].Name)
	assert.Equal(t, 1, fs.strategies[0].Priority)
	assert.Equal(t, "degraded_response", fs.strategies[1].Name)
	assert.NotNil(t, fs.metrics)
	assert.NotNil(t, fs.executors)
}

func TestFallbackSystem_EmptyStrategies(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	fs := NewFallbackSystem(logger, []FallbackStrategy{})

	assert.Empty(t, fs.strategies)
	assert.NotNil(t, fs.metrics)
}

func TestFallbackSystem_MetricsDefaultState(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	fs := NewFallbackSystem(logger, []FallbackStrategy{})

	assert.Zero(t, fs.metrics.TotalInvocations)
	assert.Zero(t, fs.metrics.SuccessCount)
	assert.Zero(t, fs.metrics.FailureCount)
	assert.Zero(t, fs.metrics.AverageLatency)
}

func TestNewRecoveryEngine(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	config := RecoveryConfig{}

	re := NewRecoveryEngine(logger, config)
	require.NotNil(t, re)
	assert.Equal(t, logger, re.logger)
	assert.NotNil(t, re.diagnosticEngine)
	assert.NotNil(t, re.repairCoordinator)
	assert.NotNil(t, re.restoreManager)
	assert.NotNil(t, re.mitigationEngine)
	assert.NotNil(t, re.preventionSystem)
	assert.NotNil(t, re.recoveryHistory)
	assert.NotNil(t, re.diagnosticCache)
}

func TestRecoveryEngine_InitiateRecovery(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	re := NewRecoveryEngine(logger, RecoveryConfig{})

	diag := &DiagnosticResult{
		TestName: "system_check",
		Status:   TestFailed,
		Output:   "high memory usage detected",
		Duration: 500 * time.Millisecond,
		Issues:   []string{"memory_leak"},
	}

	event := re.InitiateRecovery(RecoveryAutomatic, diag)
	require.NotNil(t, event)
}

func TestRecoveryEngine_RepairComponent(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	re := NewRecoveryEngine(logger, RecoveryConfig{})

	re.RepairComponent("database")
	re.RepairComponent("cache")
	re.RepairComponent("")
}

func TestRecoveryEngine_Shutdown(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	re := NewRecoveryEngine(logger, RecoveryConfig{})

	re.Shutdown()
}

func TestNewHealthMonitor(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	config := HealthMonitoringConfig{}

	hm := NewHealthMonitor(logger, config)
	require.NotNil(t, hm)
	assert.Equal(t, logger, hm.logger)
	assert.NotNil(t, hm.metricsCollector)
	assert.NotNil(t, hm.healthChecker)
	assert.NotNil(t, hm.anomalyDetector)
	assert.NotNil(t, hm.alertManager)
	assert.NotNil(t, hm.healthStatus)
	assert.Equal(t, HealthUnknown, hm.healthStatus.OverallStatus)
	assert.NotNil(t, hm.monitoringData)
}

func TestHealthMonitor_UpdateHealthStatus(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	hm := NewHealthMonitor(logger, HealthMonitoringConfig{})

	metrics := &HealthMetrics{
		Uptime:       1 * time.Hour,
		ResponseTime: 50 * time.Millisecond,
		ErrorRate:    0.01,
		Throughput:   500.0,
		ResourceUsage: &ResourceUsage{
			CPU:     45.0,
			Memory:  60.0,
			Disk:    30.0,
			Network: 20.0,
		},
		Availability: 0.999,
		Reliability:  0.995,
	}

	hm.UpdateHealthStatus(metrics)
}

func TestHealthMonitor_DetectAnomalies(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	hm := NewHealthMonitor(logger, HealthMonitoringConfig{})

	metrics := &HealthMetrics{
		ErrorRate: 0.15,
		ResourceUsage: &ResourceUsage{
			CPU:    95.0,
			Memory: 90.0,
		},
	}

	anomalies := hm.DetectAnomalies(metrics)
	require.NotNil(t, anomalies)
	assert.Empty(t, anomalies)
}

func TestHealthMonitor_GenerateAlerts(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	hm := NewHealthMonitor(logger, HealthMonitoringConfig{})

	metrics := &HealthMetrics{
		ErrorRate:    0.25,
		Availability: 0.85,
	}

	alerts := hm.GenerateAlerts(metrics)
	require.NotNil(t, alerts, "alerts should not be nil")
	assert.Empty(t, alerts, "expected 0 alerts")
}

func TestHealthMonitor_GetDegradedComponents(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	hm := NewHealthMonitor(logger, HealthMonitoringConfig{})

	components := hm.GetDegradedComponents()
	require.NotNil(t, components, "components should not be nil")
	assert.Empty(t, components, "expected 0 components")
}

func TestHealthMonitor_GetCurrentMetrics(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	hm := NewHealthMonitor(logger, HealthMonitoringConfig{})

	metrics := hm.GetCurrentMetrics()
	require.NotNil(t, metrics, "expected non-nil HealthMetrics")
}

func TestHealthMonitor_Shutdown(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	hm := NewHealthMonitor(logger, HealthMonitoringConfig{})

	hm.Shutdown()
}

func TestNewFaultInjector(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	fi := NewFaultInjector(logger, FaultInjectionConfig{})

	require.NotNil(t, fi, "expected non-nil FaultInjector")
	assert.Equal(t, logger, fi.logger, "logger not stored")
	assert.NotNil(t, fi.injectionPoints, "expected non-nil injectionPoints map")
	assert.NotNil(t, fi.scenarios, "expected non-nil scenarios map")
	assert.NotNil(t, fi.activeInjections, "expected non-nil activeInjections map")
}

func TestFaultInjector_Inject(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	fi := NewFaultInjector(logger, FaultInjectionConfig{})

	require.NoError(t, fi.InjectFault(FaultNetwork), "InjectFault(FaultNetwork)")
	require.NoError(t, fi.InjectFault(FaultDisk), "InjectFault(FaultDisk)")
	require.NoError(t, fi.InjectFault(FaultMemory), "InjectFault(FaultMemory)")
	require.NoError(t, fi.InjectFault(FaultCPU), "InjectFault(FaultCPU)")
	require.NoError(t, fi.InjectFault(FaultProcess), "InjectFault(FaultProcess)")
	require.NoError(t, fi.InjectFault(FaultService), "InjectFault(FaultService)")
}

func TestFaultInjector_Shutdown(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	fi := NewFaultInjector(logger, FaultInjectionConfig{})
	fi.Shutdown()
}

func TestNewDegradationManager(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	dm := NewDegradationManager(logger, DegradationConfig{})

	require.NotNil(t, dm, "expected non-nil DegradationManager")
	assert.Equal(t, logger, dm.logger, "logger not stored")
	assert.NotNil(t, dm.degradationLevels, "expected non-nil degradationLevels map")
	assert.NotNil(t, dm.modeSelector, "expected non-nil modeSelector")
	assert.NotNil(t, dm.resourceScaler, "expected non-nil resourceScaler")
	assert.NotNil(t, dm.qualityManager, "expected non-nil qualityManager")
	assert.Equal(t, DegradationNone, dm.currentLevel)
	assert.NotNil(t, dm.degradationHistory, "expected non-nil degradationHistory")
}

func TestDegradationManager_LevelsAndTransitions(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	dm := NewDegradationManager(logger, DegradationConfig{})

	assert.Equal(t, DegradationNone, dm.GetCurrentLevel())

	dm.ApplyDegradation(DegradationModerate)
	dm.ApplyDegradation(DegradationSevere)
	dm.ApplyDegradation(DegradationNone)
}

func TestDegradationManager_AssessDegradationLevel(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	dm := NewDegradationManager(logger, DegradationConfig{})

	metrics := &HealthMetrics{
		ErrorRate: 0.5,
		ResourceUsage: &ResourceUsage{
			CPU:    90.0,
			Memory: 85.0,
		},
	}

	level := dm.AssessDegradationLevel(metrics)
	assert.Equal(t, DegradationNone, level)
}

func TestDegradationManager_ApplyDegradation(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	dm := NewDegradationManager(logger, DegradationConfig{})

	assert.Equal(t, DegradationNone, dm.GetCurrentLevel())

	dm.ApplyDegradation(DegradationModerate)
	dm.ApplyDegradation(DegradationSevere)
	dm.ApplyDegradation(DegradationNone)
}

func TestDegradationManager_Shutdown(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	dm := NewDegradationManager(logger, DegradationConfig{})
	dm.Shutdown()
}

func TestNewEmergencyProtocols(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ep := NewEmergencyProtocols(logger, EmergencyConfig{})

	require.NotNil(t, ep, "expected non-nil EmergencyProtocols")
	assert.Equal(t, logger, ep.logger, "logger not stored")
	assert.NotNil(t, ep.protocols, "expected non-nil protocols map")
	assert.NotNil(t, ep.responseEngine, "expected non-nil responseEngine")
	assert.NotNil(t, ep.coordination, "expected non-nil coordination")
	assert.NotNil(t, ep.escalation, "expected non-nil escalation")
	assert.NotNil(t, ep.activeEmergencies, "expected non-nil activeEmergencies map")
}

func TestEmergencyProtocols_Activate(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ep := NewEmergencyProtocols(logger, EmergencyConfig{})

	for _, eType := range []EmergencyType{
		EmergencySystemCrash,
		EmergencyResourceExhaustion,
		EmergencySecurityBreach,
		EmergencyNetworkFailure,
		EmergencyDataLoss,
		EmergencyServiceOutage,
	} {
		ep.ActivateProtocol(eType)
	}
}

func TestEmergencyProtocols_Shutdown(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ep := NewEmergencyProtocols(logger, EmergencyConfig{})
	ep.Shutdown()
}

func TestNewResilienceAnalyzer(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ra := NewResilienceAnalyzer(logger, ResilienceConfig{})

	require.NotNil(t, ra, "expected non-nil ResilienceAnalyzer")
	assert.Equal(t, logger, ra.logger, "logger not stored")
	assert.NotNil(t, ra.stressTester, "expected non-nil stressTester")
	assert.NotNil(t, ra.failureAnalyzer, "expected non-nil failureAnalyzer")
	assert.NotNil(t, ra.improvementEngine, "expected non-nil improvementEngine")
	assert.NotNil(t, ra.resilienceMetrics, "expected non-nil resilienceMetrics")
	assert.NotNil(t, ra.analysisHistory, "expected non-nil analysisHistory")
}

func TestResilienceAnalyzer_StoreAnalysis(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ra := NewResilienceAnalyzer(logger, ResilienceConfig{})

	analysis := &ResilienceAnalysis{
		ID:        "analysis_1",
		Timestamp: time.Now(),
		TestType:  "comprehensive",
		Results:   &TestResults{Passed: 10, Failed: 2},
		Metrics:   &ResilienceMetrics{MTBF: 24 * time.Hour, Availability: 0.99},
		Findings: []Finding{
			{ID: "F1", Category: "performance", Severity: "medium"},
		},
		Recommendations: []string{"increase timeout", "add retry"},
		Priority:        1,
	}

	ra.StoreAnalysis(analysis)
}

func TestResilienceAnalyzer_Shutdown(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ra := NewResilienceAnalyzer(logger, ResilienceConfig{})
	ra.Shutdown()
}

func TestErrorClassifier(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rules := []ClassificationRule{
		{
			Name:       "network_error",
			Patterns:   []string{"connection refused", "timeout"},
			Categories: []ErrorCategory{ErrorCategoryNetwork},
			Severity:   SeverityHigh,
			Actions:    []string{"retry", "fallback"},
		},
		{
			Name:       "db_error",
			Patterns:   []string{"connection failed", "query timeout"},
			Categories: []ErrorCategory{ErrorCategoryDatabase},
			Severity:   SeverityCritical,
		},
	}

	ec := NewErrorClassifier(logger, rules)
	require.NotNil(t, ec)
	assert.Equal(t, logger, ec.logger)
	assert.Len(t, ec.rules, 2)
	assert.Equal(t, "network_error", ec.rules[0].Name)
	assert.Equal(t, "db_error", ec.rules[1].Name)
	assert.NotNil(t, ec.cache)
}

func TestErrorReporter(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	er := NewErrorReporter(logger, ErrorReportingConfig{})

	require.NotNil(t, er)
	assert.Equal(t, logger, er.logger)
}

func TestRobustnessManager_Concurrency(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	config := RobustnessConfig{
		EnableErrorHandling:    true,
		EnableSelfHealing:      true,
		EnableHealthMonitoring: true,
	}

	rm, err := NewRobustnessManager(logger, config)
	require.NoError(t, err)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rm.mu.Lock()
			_ = rm.isRunning
			rm.mu.Unlock()
		}()
	}
	wg.Wait()

	err = rm.Start()
	require.NoError(t, err, "Start()")
	time.Sleep(50 * time.Millisecond)

	rm.Stop()
}

func TestRobustnessManager_MultipleStarts(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rm, err := NewRobustnessManager(logger, RobustnessConfig{
		EnableHealthMonitoring: true,
	})
	require.NoError(t, err)

	require.NoError(t, rm.Start(), "first Start()")

	require.Error(t, rm.Start(), "expected error on second Start()")

	rm.Stop()
}

func TestRobustnessManager_ErrorHandlerIntegration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rm, err := NewRobustnessManager(logger, RobustnessConfig{
		EnableErrorHandling: true,
		ErrorHandlingConfig: ErrorHandlingConfig{
			CircuitBreakerConfig: CircuitBreakerConfig{
				FailureThreshold: 3,
				SuccessThreshold: 1,
				Timeout:          5 * time.Second,
				ResetTimeout:     10 * time.Second,
			},
		},
	})
	require.NoError(t, err)

	require.NotNil(t, rm.errorHandler)

	classification := rm.errorHandler.ClassifyError(errors.New("test error"))
	assert.NotNil(t, classification)

	pending := rm.errorHandler.GetPendingErrors()
	assert.NotNil(t, pending)

	rm.errorHandler.Shutdown()
}

func TestRobustnessManager_RecoveryEngineIntegration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rm, err := NewRobustnessManager(logger, RobustnessConfig{
		EnableSelfHealing: true,
	})
	require.NoError(t, err)

	require.NotNil(t, rm.recoveryEngine)

	event := rm.recoveryEngine.InitiateRecovery(RecoveryAutomatic, &DiagnosticResult{
		TestName: "integration_test",
		Status:   TestFailed,
	})
	assert.NotNil(t, event)

	rm.recoveryEngine.RepairComponent("test_component")
	rm.recoveryEngine.Shutdown()
}

func TestRobustnessManager_HealthMonitorIntegration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rm, err := NewRobustnessManager(logger, RobustnessConfig{
		EnableHealthMonitoring: true,
	})
	require.NoError(t, err)

	require.NotNil(t, rm.healthMonitor)

	metrics := rm.healthMonitor.GetCurrentMetrics()
	assert.NotNil(t, metrics)

	components := rm.healthMonitor.GetDegradedComponents()
	assert.NotNil(t, components)

	rm.healthMonitor.UpdateHealthStatus(&HealthMetrics{
		Uptime:       10 * time.Minute,
		ResponseTime: 100 * time.Millisecond,
		ErrorRate:    0.02,
	})

	rm.healthMonitor.Shutdown()
}

func TestNewRobustnessManager_AllComponentsEnabled(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	config := RobustnessConfig{
		EnableErrorHandling:      true,
		EnableSelfHealing:        true,
		EnableFaultInjection:     true,
		EnableHealthMonitoring:   true,
		EnableDegradation:        true,
		EnableEmergencyProtocols: true,
	}

	rm, err := NewRobustnessManager(logger, config)
	require.NoError(t, err)

	assert.NotNil(t, rm.errorHandler)
	assert.NotNil(t, rm.recoveryEngine)
	assert.NotNil(t, rm.faultInjector)
	assert.NotNil(t, rm.healthMonitor)
	assert.NotNil(t, rm.degradationManager)
	assert.NotNil(t, rm.emergencyProtocols)
	assert.NotNil(t, rm.resilienceAnalyzer)
}

func TestRobustnessManager_Context(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rm, err := NewRobustnessManager(logger, RobustnessConfig{})
	require.NoError(t, err)

	require.NotNil(t, rm.ctx)

	select {
	case <-rm.ctx.Done():
		require.Fail(t, "context should not be cancelled initially")
	default:
	}

	rm.cancel()

	select {
	case <-rm.ctx.Done():
	default:
		require.Fail(t, "context should be cancelled after cancel()")
	}
}

func TestRobustnessManager_Logging(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	rm, err := NewRobustnessManager(logger, RobustnessConfig{EnableErrorHandling: true})
	require.NoError(t, err)

	err = rm.Start()
	require.NoError(t, err, "Start()")

	rm.Stop()
}

func TestRobustnessManager_DegradationManagerIntegration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rm, err := NewRobustnessManager(logger, RobustnessConfig{
		EnableDegradation: true,
	})
	require.NoError(t, err)

	require.NotNil(t, rm.degradationManager)

	level := rm.degradationManager.AssessDegradationLevel(&HealthMetrics{
		ResourceUsage: &ResourceUsage{CPU: 99.9, Memory: 95.0},
	})
	assert.Equal(t, DegradationNone, level)

	rm.degradationManager.ApplyDegradation(DegradationModerate)
	rm.degradationManager.Shutdown()
}

func TestRobustnessManager_EmergencyProtocolsIntegration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rm, err := NewRobustnessManager(logger, RobustnessConfig{
		EnableEmergencyProtocols: true,
	})
	require.NoError(t, err)

	require.NotNil(t, rm.emergencyProtocols, "emergencyProtocols should be initialized")

	rm.emergencyProtocols.ActivateProtocol(EmergencySystemCrash)
	rm.emergencyProtocols.ActivateProtocol(EmergencySecurityBreach)
	rm.emergencyProtocols.Shutdown()
}

func TestRobustnessManager_FaultInjectorIntegration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rm, err := NewRobustnessManager(logger, RobustnessConfig{
		EnableFaultInjection: true,
	})
	require.NoError(t, err)

	require.NotNil(t, rm.faultInjector)

	require.NoError(t, rm.faultInjector.InjectFault(FaultNetwork))
	rm.faultInjector.Shutdown()
}

func TestRobustnessManager_ResilienceAnalyzerIntegration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rm, err := NewRobustnessManager(logger, RobustnessConfig{})
	require.NoError(t, err)

	require.NotNil(t, rm.resilienceAnalyzer)

	analysis := &ResilienceAnalysis{
		ID:      "test_001",
		Results: &TestResults{Passed: 5, Failed: 0},
		Metrics: &ResilienceMetrics{Availability: 0.999},
	}
	rm.resilienceAnalyzer.StoreAnalysis(analysis)
	rm.resilienceAnalyzer.Shutdown()
}

func TestErrorEvent_Defaults(t *testing.T) {
	t.Parallel()
	event := &ErrorEvent{}

	assert.Empty(t, event.ID)
	assert.Zero(t, event.RetryCount)
	assert.False(t, event.FallbackUsed)
	assert.False(t, event.Handled)
	assert.Nil(t, event.Context)
}

func TestErrorEvent_Full(t *testing.T) {
	t.Parallel()
	now := time.Now()
	event := &ErrorEvent{
		ID:        "evt-001",
		Timestamp: now,
		Error:     errors.New("connection timeout"),
		Context:   map[string]interface{}{"service": "db", "timeout_ms": 5000},
		Classification: &ErrorClassification{
			Categories: []ErrorCategory{ErrorCategoryNetwork},
			Severity:   SeverityHigh,
			Transient:  true,
			Retryable:  true,
		},
		RetryCount:   2,
		FallbackUsed: true,
		Handled:      true,
	}

	assert.Equal(t, "evt-001", event.ID)
	assert.True(t, event.Timestamp.Equal(now))
	assert.Equal(t, "connection timeout", event.Error.Error())
	assert.Equal(t, []ErrorCategory{ErrorCategoryNetwork}, event.Classification.Categories)
	assert.Equal(t, SeverityHigh, event.Classification.Severity)
	assert.True(t, event.Classification.Transient)
	assert.True(t, event.Classification.Retryable)
	assert.Equal(t, 2, event.RetryCount)
	assert.True(t, event.FallbackUsed)
	assert.True(t, event.Handled)
}

func TestCircuitState_Defaults(t *testing.T) {
	t.Parallel()
	cs := &CircuitState{}

	assert.Empty(t, cs.State)
	assert.Zero(t, cs.FailureCount)
	assert.Zero(t, cs.SuccessCount)
	assert.Nil(t, cs.Metrics)
}

func TestCircuitState_Full(t *testing.T) {
	t.Parallel()
	now := time.Now()
	cs := &CircuitState{
		Name:         "db_circuit",
		State:        CircuitClosed,
		FailureCount: 0,
		SuccessCount: 5,
		LastError:    now,
		NextRetry:    now.Add(30 * time.Second),
		Timeout:      10 * time.Second,
		Metrics:      &CircuitMetrics{TotalCalls: 100, SuccessCalls: 95, FailureCalls: 5},
	}

	assert.Equal(t, "db_circuit", cs.Name)
	assert.Equal(t, CircuitClosed, cs.State)
	assert.Zero(t, cs.FailureCount)
	assert.Equal(t, 5, cs.SuccessCount)
	assert.Equal(t, 10*time.Second, cs.Timeout)
	assert.Equal(t, int64(100), cs.Metrics.TotalCalls)
	assert.Equal(t, int64(95), cs.Metrics.SuccessCalls)
}

func TestCircuitStateEnumValues(t *testing.T) {
	t.Parallel()

	assert.Equal(t, CircuitStateEnum("closed"), CircuitClosed)
	assert.Equal(t, CircuitStateEnum("open"), CircuitOpen)
	assert.Equal(t, CircuitStateEnum("half_open"), CircuitHalfOpen)
}

func TestErrorCategoryValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		got  ErrorCategory
		want ErrorCategory
	}{
		{ErrorCategoryNetwork, "network"},
		{ErrorCategoryDatabase, "database"},
		{ErrorCategoryFilesystem, "filesystem"},
		{ErrorCategoryMemory, "memory"},
		{ErrorCategoryCPU, "cpu"},
		{ErrorCategorySecurity, "security"},
		{ErrorCategoryValidation, "validation"},
		{ErrorCategoryBusiness, "business"},
		{ErrorCategoryExternal, "external"},
		{ErrorCategoryInternal, "internal"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.got, "ErrorCategory(%q) mismatch", tt.got)
	}
}

func TestErrorSeverityValues(t *testing.T) {
	t.Parallel()

	assert.Equal(t, ErrorSeverity("low"), SeverityLow)
	assert.Equal(t, ErrorSeverity("medium"), SeverityMedium)
	assert.Equal(t, ErrorSeverity("high"), SeverityHigh)
	assert.Equal(t, ErrorSeverity("critical"), SeverityCritical)
	assert.Equal(t, ErrorSeverity("fatal"), SeverityFatal)
}

func TestHealthStatusValues(t *testing.T) {
	t.Parallel()

	assert.Equal(t, HealthStatus("healthy"), HealthHealthy)
	assert.Equal(t, HealthStatus("degraded"), HealthDegraded)
	assert.Equal(t, HealthStatus("unhealthy"), HealthUnhealthy)
	assert.Equal(t, HealthStatus("critical"), HealthCritical)
	assert.Equal(t, HealthStatus("unknown"), HealthUnknown)
}

func TestDegradationLevelValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		got  DegradationLevel
		want DegradationLevel
	}{
		{DegradationNone, "none"},
		{DegradationMinimal, "minimal"},
		{DegradationModerate, "moderate"},
		{DegradationSevere, "severe"},
		{DegradationCritical, "critical"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.got, "DegradationLevel(%q) mismatch", tt.got)
	}
}

func TestRecoveryTypeValues(t *testing.T) {
	t.Parallel()

	assert.Equal(t, RecoveryType("automatic"), RecoveryAutomatic)
	assert.Equal(t, RecoveryType("manual"), RecoveryManual)
	assert.Equal(t, RecoveryType("forced"), RecoveryForced)
}

func TestRecoveryStatusValues(t *testing.T) {
	t.Parallel()

	assert.Equal(t, RecoveryStatus("pending"), RecoveryPending)
	assert.Equal(t, RecoveryStatus("running"), RecoveryRunning)
	assert.Equal(t, RecoveryStatus("completed"), RecoveryCompleted)
	assert.Equal(t, RecoveryStatus("failed"), RecoveryFailed)
}

func TestTestStatusValues(t *testing.T) {
	t.Parallel()

	assert.Equal(t, TestStatus("passed"), TestPassed)
	assert.Equal(t, TestStatus("failed"), TestFailed)
	assert.Equal(t, TestStatus("skipped"), TestSkipped)
	assert.Equal(t, TestStatus("timeout"), TestTimeout)
}

func TestEmergencyTypeValues(t *testing.T) {
	t.Parallel()

	assert.Equal(t, EmergencyType("system_crash"), EmergencySystemCrash)
	assert.Equal(t, EmergencyType("resource_exhaustion"), EmergencyResourceExhaustion)
	assert.Equal(t, EmergencyType("security_breach"), EmergencySecurityBreach)
	assert.Equal(t, EmergencyType("network_failure"), EmergencyNetworkFailure)
	assert.Equal(t, EmergencyType("data_loss"), EmergencyDataLoss)
	assert.Equal(t, EmergencyType("service_outage"), EmergencyServiceOutage)
}

func TestFaultTypeValues(t *testing.T) {
	t.Parallel()

	assert.Equal(t, FaultType("network"), FaultNetwork)
	assert.Equal(t, FaultType("disk"), FaultDisk)
	assert.Equal(t, FaultType("memory"), FaultMemory)
	assert.Equal(t, FaultType("cpu"), FaultCPU)
	assert.Equal(t, FaultType("process"), FaultProcess)
	assert.Equal(t, FaultType("service"), FaultService)
}

func TestDiagnosticTypeValues(t *testing.T) {
	t.Parallel()

	assert.Equal(t, DiagnosticType("system"), DiagnosticSystem)
	assert.Equal(t, DiagnosticType("network"), DiagnosticNetwork)
	assert.Equal(t, DiagnosticType("storage"), DiagnosticStorage)
	assert.Equal(t, DiagnosticType("memory"), DiagnosticMemory)
	assert.Equal(t, DiagnosticType("cpu"), DiagnosticCPU)
	assert.Equal(t, DiagnosticType("security"), DiagnosticSecurity)
}

func TestAlertSeverityValues(t *testing.T) {
	t.Parallel()

	assert.Equal(t, AlertSeverity("low"), AlertLow)
	assert.Equal(t, AlertSeverity("medium"), AlertMedium)
	assert.Equal(t, AlertSeverity("high"), AlertHigh)
	assert.Equal(t, AlertSeverity("critical"), AlertCritical)
}

func TestSystemHealthStatus_Defaults(t *testing.T) {
	t.Parallel()

	hs := &SystemHealthStatus{}

	assert.Empty(t, hs.OverallStatus)
	assert.Nil(t, hs.ComponentStatus)
	assert.Nil(t, hs.Metrics)
	assert.Nil(t, hs.Alerts)
}

func TestHealthMetrics_Defaults(t *testing.T) {
	t.Parallel()

	hm := &HealthMetrics{}

	assert.Zero(t, hm.Uptime)
	assert.Zero(t, hm.ErrorRate)
	assert.Zero(t, hm.Throughput)
	assert.Nil(t, hm.ResourceUsage)
}

func TestHealthMetrics_Full(t *testing.T) {
	t.Parallel()

	hm := &HealthMetrics{
		Uptime:       24 * time.Hour,
		ResponseTime: 200 * time.Millisecond,
		ErrorRate:    0.03,
		Throughput:   1500.0,
		ResourceUsage: &ResourceUsage{
			CPU:     50.0,
			Memory:  70.0,
			Disk:    40.0,
			Network: 30.0,
		},
		Availability: 0.9999,
		Reliability:  0.999,
	}

	assert.Equal(t, 24*time.Hour, hm.Uptime)
	assert.Equal(t, 200*time.Millisecond, hm.ResponseTime)
	assert.Equal(t, 0.03, hm.ErrorRate)
	assert.Equal(t, 1500.0, hm.Throughput)
	assert.Equal(t, 50.0, hm.ResourceUsage.CPU)
	assert.Equal(t, 70.0, hm.ResourceUsage.Memory)
	assert.Equal(t, 0.9999, hm.Availability)
	assert.Equal(t, 0.999, hm.Reliability)
}

func TestResourceUsage_Defaults(t *testing.T) {
	t.Parallel()

	ru := &ResourceUsage{}

	assert.Zero(t, ru.CPU)
	assert.Zero(t, ru.Memory)
	assert.Zero(t, ru.Disk)
	assert.Zero(t, ru.Network)
}

func TestRobustnessConfig_Defaults(t *testing.T) {
	t.Parallel()

	cfg := RobustnessConfig{}

	assert.False(t, cfg.EnableErrorHandling)
	assert.False(t, cfg.EnableSelfHealing)
	assert.False(t, cfg.EnableFaultInjection)
	assert.False(t, cfg.EnableHealthMonitoring)
	assert.False(t, cfg.EnableDegradation)
	assert.False(t, cfg.EnableEmergencyProtocols)
}

func TestAnomaly_Defaults(t *testing.T) {
	t.Parallel()

	a := &Anomaly{}

	assert.Empty(t, a.Type)
	assert.Empty(t, a.Severity)
}

func TestErrorClassification_Defaults(t *testing.T) {
	t.Parallel()

	ec := &ErrorClassification{}

	assert.Empty(t, ec.Categories)
	assert.Empty(t, ec.Severity)
	assert.False(t, ec.Transient)
	assert.False(t, ec.Retryable)
}

func TestNewErrorClassifier_EmptyRules(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ec := NewErrorClassifier(logger, []ClassificationRule{})

	assert.Empty(t, ec.rules)
}

func TestNewErrorClassifier_NilRules(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ec := NewErrorClassifier(logger, nil)

	assert.Equal(t, logger, ec.logger)
	assert.NotNil(t, ec.cache, "expected non-nil cache map")
}

func TestNewErrorReporter_NilConfig(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	er := NewErrorReporter(logger, ErrorReportingConfig{})

	assert.Equal(t, ErrorReportingConfig{}, er.config, "expected empty config")
}

func TestNewRetryManager_NilPolicies(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rm := NewRetryManager(logger, nil)

	assert.NotNil(t, rm.metrics, "expected non-nil metrics")
	assert.NotNil(t, rm.executors, "expected non-nil executors map")
}

func TestNewFallbackSystem_NilStrategies(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	fs := NewFallbackSystem(logger, nil)

	assert.NotNil(t, fs.metrics, "expected non-nil metrics")
	assert.NotNil(t, fs.executors, "expected non-nil executors map")
}

func TestRobustnessManager_ConcurrentStop(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rm, err := NewRobustnessManager(logger, RobustnessConfig{
		EnableHealthMonitoring: true,
		EnableErrorHandling:    true,
	})
	require.NoError(t, err)

	require.NoError(t, rm.Start(), "Start() should succeed")

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rm.Stop()
		}()
	}
	wg.Wait()

	assert.False(t, rm.isRunning, "expected isRunning=false after Stop()")
}

func TestRobustnessManager_GetErrorPolicyName(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rm, _ := NewRobustnessManager(logger, RobustnessConfig{}) //nolint:errcheck

	tests := []struct {
		name       string
		categories []ErrorCategory
		expected   string
	}{
		{"network", []ErrorCategory{ErrorCategoryNetwork}, "network_retry"},
		{"database", []ErrorCategory{ErrorCategoryDatabase}, "database_retry"},
		{"filesystem", []ErrorCategory{ErrorCategoryFilesystem}, "filesystem_retry"},
		{"default from category", []ErrorCategory{ErrorCategoryMemory}, "default_retry"},
		{"default from nil", nil, "default_retry"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &ErrorEvent{
				Classification: &ErrorClassification{
					Categories: tt.categories,
				},
			}
			result := rm.getErrorPolicyName(event)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRecoveryEvent_Defaults(t *testing.T) {
	t.Parallel()

	re := &RecoveryEvent{}

	assert.Empty(t, re.ID)
	assert.False(t, re.Success)
	assert.Empty(t, re.Status)
	assert.Zero(t, re.Duration)
}

func TestDiagnosticResult_Defaults(t *testing.T) {
	t.Parallel()

	dr := &DiagnosticResult{}

	assert.Empty(t, dr.TestName)
	assert.Empty(t, dr.Status)
	assert.Zero(t, dr.Duration)
	assert.Nil(t, dr.Issues)
}

func TestRetryPolicy_Defaults(t *testing.T) {
	t.Parallel()

	rp := RetryPolicy{}

	assert.Zero(t, rp.MaxAttempts)
	assert.Zero(t, rp.InitialDelay)
	assert.Zero(t, rp.BackoffFactor)
	assert.False(t, rp.Jitter)
}

func TestRetryPolicy_Full(t *testing.T) {
	t.Parallel()

	rp := RetryPolicy{
		MaxAttempts:   5,
		InitialDelay:  200 * time.Millisecond,
		MaxDelay:      10 * time.Second,
		BackoffFactor: 3.0,
		Jitter:        true,
		Timeout:       30 * time.Second,
		Condition:     "error == transient",
	}

	assert.Equal(t, 5, rp.MaxAttempts)
	assert.Equal(t, 200*time.Millisecond, rp.InitialDelay)
	assert.Equal(t, 10*time.Second, rp.MaxDelay)
	assert.Equal(t, 3.0, rp.BackoffFactor)
	assert.True(t, rp.Jitter)
	assert.Equal(t, 30*time.Second, rp.Timeout)
	assert.Equal(t, "error == transient", rp.Condition)
}

func TestCircuitBreakerConfig_Defaults(t *testing.T) {
	t.Parallel()

	cfg := CircuitBreakerConfig{}

	assert.Zero(t, cfg.FailureThreshold)
	assert.Zero(t, cfg.SuccessThreshold)
	assert.Zero(t, cfg.Timeout)
	assert.Zero(t, cfg.HalfOpenMaxCalls)
	assert.Zero(t, cfg.ResetTimeout)
}

func TestCircuitBreakerConfig_Full(t *testing.T) {
	t.Parallel()

	cfg := CircuitBreakerConfig{
		FailureThreshold: 10,
		SuccessThreshold: 5,
		Timeout:          15 * time.Second,
		HalfOpenMaxCalls: 3,
		ResetTimeout:     30 * time.Second,
		MetricsWindow:    1 * time.Hour,
	}

	assert.Equal(t, 10, cfg.FailureThreshold)
	assert.Equal(t, 5, cfg.SuccessThreshold)
	assert.Equal(t, 15*time.Second, cfg.Timeout)
	assert.Equal(t, 3, cfg.HalfOpenMaxCalls)
	assert.Equal(t, 30*time.Second, cfg.ResetTimeout)
	assert.Equal(t, 1*time.Hour, cfg.MetricsWindow)
}

func TestErrorHandlingConfig_Defaults(t *testing.T) {
	t.Parallel()

	cfg := ErrorHandlingConfig{}

	assert.Empty(t, cfg.ClassificationRules)
	assert.Empty(t, cfg.RetryPolicies)
	assert.Empty(t, cfg.FallbackStrategies)
}

func TestRobustnessManager_CollectHealthMetrics(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "startup_time", time.Now())
	rm := &RobustnessManager{
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}

	metrics := rm.collectHealthMetrics()
	require.NotNil(t, metrics)

	assert.True(t, metrics.Uptime > 0)
	assert.True(t, metrics.ResponseTime > 0)
	assert.True(t, metrics.ErrorRate >= 0 && metrics.ErrorRate <= 1)
	assert.True(t, metrics.Throughput >= 0)
	require.NotNil(t, metrics.ResourceUsage)
	assert.True(t, metrics.ResourceUsage.CPU >= 0 && metrics.ResourceUsage.CPU <= 100)
	assert.True(t, metrics.ResourceUsage.Memory >= 0 && metrics.ResourceUsage.Memory <= 100)
	assert.True(t, metrics.Availability >= 0.9 && metrics.Availability <= 1.0)
	assert.True(t, metrics.Reliability >= 0.95 && metrics.Reliability <= 1.0)
}

func TestRobustnessManager_PerformResilienceAnalysis(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rm, err := NewRobustnessManager(logger, RobustnessConfig{})
	require.NoError(t, err)

	rm.performResilienceAnalysis()
}

func TestRobustnessManager_ProcessErrors(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rm, err := NewRobustnessManager(logger, RobustnessConfig{})
	require.NoError(t, err)

	rm.processErrors()

	rm.errorHandler = NewErrorHandler(logger, ErrorHandlingConfig{})
	rm.processErrors()
}

func TestRobustnessManager_TryFallback(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rm, _ := NewRobustnessManager(logger, RobustnessConfig{ //nolint:errcheck
		EnableErrorHandling: true,
		ErrorHandlingConfig: ErrorHandlingConfig{
			FallbackStrategies: []FallbackStrategy{
				{Name: "mock_fallback", Priority: 1},
			},
		},
	})

	err := &ErrorEvent{
		ID:    "test-fallback",
		Error: errors.New("test error"),
	}

	result := rm.tryFallback(err)
	assert.False(t, result)
}

func TestRobustnessManager_TriggerRecovery(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rm, _ := NewRobustnessManager(logger, RobustnessConfig{}) //nolint:errcheck

	err := &ErrorEvent{
		ID:    "test-recovery",
		Error: errors.New("critical error"),
		Classification: &ErrorClassification{
			Severity: SeverityCritical,
		},
	}

	rm.triggerRecovery(err)

	rm.recoveryEngine = NewRecoveryEngine(logger, RecoveryConfig{})
	rm.triggerRecovery(err)
}

func TestRobustnessManager_PerformRecovery(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rm, _ := NewRobustnessManager(logger, RobustnessConfig{}) //nolint:errcheck

	rm.performRecovery()

	rm.recoveryEngine = NewRecoveryEngine(logger, RecoveryConfig{})
	rm.performRecovery()
}

func TestRobustnessManager_MonitorHealth(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rm, _ := NewRobustnessManager(logger, RobustnessConfig{}) //nolint:errcheck

	rm.monitorHealth()
}

func TestRobustnessManager_HandleAlert(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rm, _ := NewRobustnessManager(logger, RobustnessConfig{ //nolint:errcheck
		EnableEmergencyProtocols: true,
	})

	alert := &HealthAlert{
		ID:        "alert-001",
		Severity:  AlertCritical,
		Status:    HealthUnhealthy,
		Component: "database",
		Message:   "database connection lost",
	}

	rm.handleAlert(alert)

	alert.Severity = AlertLow
	rm.handleAlert(alert)
}

func TestRobustnessManager_ManageDegradation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rm, _ := NewRobustnessManager(logger, RobustnessConfig{}) //nolint:errcheck

	rm.manageDegradation()

	rm.healthMonitor = NewHealthMonitor(logger, HealthMonitoringConfig{})
	rm.degradationManager = NewDegradationManager(logger, DegradationConfig{})
	rm.manageDegradation()
}

func TestFallbackStrategy_Defaults(t *testing.T) {
	t.Parallel()

	fs := FallbackStrategy{}

	assert.Empty(t, fs.Name)
	assert.Zero(t, fs.Priority)
	assert.Zero(t, fs.Timeout)
	assert.Nil(t, fs.Metrics)
}

func TestClassificationRule_Defaults(t *testing.T) {
	t.Parallel()

	cr := ClassificationRule{}

	assert.Empty(t, cr.Name)
	assert.Empty(t, cr.Patterns)
	assert.Empty(t, cr.Categories)
	assert.Empty(t, cr.Severity)
	assert.Zero(t, cr.Timeout)
}

func TestNewFaultInjector_EmptyConfig(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	fi := NewFaultInjector(logger, FaultInjectionConfig{})

	assert.NotNil(t, fi.injectionPoints)
	assert.NotNil(t, fi.scenarios)
	assert.NotNil(t, fi.activeInjections)
}

func TestServiceTypeValues(t *testing.T) {
	t.Parallel()

	assert.Equal(t, ServiceType("core"), ServiceCore)
	assert.Equal(t, ServiceType("secondary"), ServiceSecondary)
	assert.Equal(t, ServiceType("auxiliary"), ServiceAuxiliary)
	assert.Equal(t, ServiceType("debug"), ServiceDebug)
}

func TestRepairTypeValues(t *testing.T) {
	t.Parallel()

	assert.Equal(t, RepairType("restart"), RepairRestart)
	assert.Equal(t, RepairType("reconfigure"), RepairReconfigure)
	assert.Equal(t, RepairType("replace"), RepairReplace)
	assert.Equal(t, RepairType("cleanup"), RepairCleanup)
	assert.Equal(t, RepairType("update"), RepairUpdate)
}

func TestEmergencyStatusValues(t *testing.T) {
	t.Parallel()

	assert.Equal(t, EmergencyStatus("detected"), EmergencyDetected)
	assert.Equal(t, EmergencyStatus("responding"), EmergencyResponding)
	assert.Equal(t, EmergencyStatus("resolved"), EmergencyResolved)
	assert.Equal(t, EmergencyStatus("failed"), EmergencyFailed)
}

func TestStepStatusValues(t *testing.T) {
	t.Parallel()

	assert.Equal(t, StepStatus("pending"), StepPending)
	assert.Equal(t, StepStatus("executing"), StepExecuting)
	assert.Equal(t, StepStatus("completed"), StepCompleted)
	assert.Equal(t, StepStatus("failed"), StepFailed)
	assert.Equal(t, StepStatus("skipped"), StepSkipped)
}

func TestResilienceAnalysis_Defaults(t *testing.T) {
	t.Parallel()

	ra := &ResilienceAnalysis{}

	assert.Empty(t, ra.ID)
	assert.Nil(t, ra.Results)
	assert.Nil(t, ra.Metrics)
	assert.Empty(t, ra.Findings)
	assert.False(t, ra.Implemented)
}

func TestResilienceMetrics_Defaults(t *testing.T) {
	t.Parallel()

	rm := &ResilienceMetrics{}

	assert.Zero(t, rm.MTBF)
	assert.Zero(t, rm.MTTR)
	assert.Zero(t, rm.Availability)
	assert.Zero(t, rm.Reliability)
}

func TestNewHealthMonitor_HealthStatusInitialization(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	hm := NewHealthMonitor(logger, HealthMonitoringConfig{})

	assert.Equal(t, HealthUnknown, hm.healthStatus.OverallStatus)
	assert.Nil(t, hm.healthStatus.ComponentStatus)
	assert.Nil(t, hm.healthStatus.Metrics)
	assert.Nil(t, hm.healthStatus.Alerts)
}

func TestNewHealthChecker(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	hc := NewHealthChecker(logger)

	require.NotNil(t, hc)
	assert.Equal(t, logger, hc.logger)
	assert.NotNil(t, hc.checks)
	assert.NotNil(t, hc.evaluator)
	assert.NotNil(t, hc.reporter)
}

func TestNewMetricsCollector(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	mc := NewMetricsCollector(logger)

	require.NotNil(t, mc)
	assert.NotNil(t, mc.collectors)
	assert.NotNil(t, mc.aggregator)
	assert.NotNil(t, mc.exporter)
}

func TestNewAnomalyDetector(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ad := NewAnomalyDetector(logger)

	require.NotNil(t, ad)
	assert.NotNil(t, ad.detectors)
	assert.NotNil(t, ad.profiler)
	assert.NotNil(t, ad.alertEngine)
}

func TestNewAlertManager(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	am := NewAlertManager(logger)

	require.NotNil(t, am)
	assert.NotNil(t, am.channels)
	assert.NotNil(t, am.router)
	assert.NotNil(t, am.escalator)
}

func TestHealthAlert_Defaults(t *testing.T) {
	t.Parallel()

	ha := &HealthAlert{}

	assert.Empty(t, ha.ID)
	assert.Empty(t, ha.Severity)
	assert.False(t, ha.Resolved)
}

func TestHealthAlert_Full(t *testing.T) {
	t.Parallel()
	now := time.Now()

	ha := &HealthAlert{
		ID:             "alert-999",
		Timestamp:      now,
		Component:      "api-gateway",
		Status:         HealthCritical,
		Message:        "latency spike above threshold",
		Severity:       AlertHigh,
		Resolved:       true,
		ResolutionTime: now.Add(5 * time.Minute),
	}

	assert.Equal(t, "alert-999", ha.ID)
	assert.Equal(t, "api-gateway", ha.Component)
	assert.Equal(t, HealthCritical, ha.Status)
	assert.Equal(t, AlertHigh, ha.Severity)
	assert.True(t, ha.Resolved)
}

func TestNewModeSelector(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ms := NewModeSelector(logger)

	require.NotNil(t, ms)
	assert.NotNil(t, ms.modes)
	assert.NotNil(t, ms.selector)
	assert.NotNil(t, ms.transitions)
}

func TestNewResourceScaler(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rs := NewResourceScaler(logger)

	require.NotNil(t, rs)
	assert.NotNil(t, rs.scalers)
	assert.NotNil(t, rs.controller)
	assert.NotNil(t, rs.optimizer)
}

func TestNewQualityManager(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	qm := NewQualityManager(logger)

	require.NotNil(t, qm)
	assert.NotNil(t, qm.qualityMetrics)
	assert.NotNil(t, qm.controller)
	assert.NotNil(t, qm.prioritizer)
}

func TestNewEmergencyResponseEngine(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ere := NewEmergencyResponseEngine(logger)

	require.NotNil(t, ere)
	assert.NotNil(t, ere.responsePlans)
	assert.NotNil(t, ere.executor)
	assert.NotNil(t, ere.coordinator)
}

func TestNewEmergencyCoordination(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ec := NewEmergencyCoordination(logger)

	require.NotNil(t, ec)
	assert.NotNil(t, ec.coordinators)
	assert.NotNil(t, ec.synchronizer)
	assert.NotNil(t, ec.communicator)
}

func TestNewEscalationManager(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	em := NewEscalationManager(logger)

	require.NotNil(t, em)
	assert.NotNil(t, em.escalationPaths)
	assert.NotNil(t, em.trigger)
	assert.NotNil(t, em.notifier)
}

func TestNewStressTester(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	st := NewStressTester(logger)

	require.NotNil(t, st)
	assert.NotNil(t, st.testScenarios)
	assert.NotNil(t, st.executor)
	assert.NotNil(t, st.analyzer)
}

func TestNewFailureAnalyzer(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	fa := NewFailureAnalyzer(logger)

	require.NotNil(t, fa)
	assert.NotNil(t, fa.analyzers)
	assert.NotNil(t, fa.correlator)
	assert.NotNil(t, fa.predictor)
}

func TestNewImprovementEngine(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ie := NewImprovementEngine(logger)

	require.NotNil(t, ie)
	assert.NotNil(t, ie.improvementStrategies)
	assert.NotNil(t, ie.prioritizer)
	assert.NotNil(t, ie.implementer)
}

func TestNewDiagnosticEngine(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	de := NewDiagnosticEngine(logger)

	require.NotNil(t, de)
	assert.NotNil(t, de.diagnosticTests)
	assert.NotNil(t, de.analyzer)
	assert.NotNil(t, de.reporter)
}

func TestNewRepairCoordinator(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rc := NewRepairCoordinator(logger)

	require.NotNil(t, rc)
	assert.NotNil(t, rc.repairActions)
	assert.NotNil(t, rc.scheduler)
	assert.NotNil(t, rc.executor)
	assert.NotNil(t, rc.validator)
}

func TestNewRestoreManager(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rm := NewRestoreManager(logger)

	require.NotNil(t, rm)
	assert.NotNil(t, rm.restorePoints)
	assert.NotNil(t, rm.backupManager)
	assert.NotNil(t, rm.recoveryPlanner)
}

func TestNewMitigationEngine(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	me := NewMitigationEngine(logger)

	require.NotNil(t, me)
	assert.NotNil(t, me.mitigationStrategies)
	assert.NotNil(t, me.impactAssessor)
	assert.NotNil(t, me.priorityManager)
}

func TestNewPreventionSystem(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ps := NewPreventionSystem(logger)

	require.NotNil(t, ps)
	assert.NotNil(t, ps.preventionRules)
	assert.NotNil(t, ps.learningEngine)
	assert.NotNil(t, ps.riskAssessor)
}

func TestRobustnessConfig_DeepCopy(t *testing.T) {
	t.Parallel()

	cfg1 := RobustnessConfig{
		EnableErrorHandling:  true,
		EnableSelfHealing:    true,
		EnableFaultInjection: true,
		ErrorHandlingConfig: ErrorHandlingConfig{
			CircuitBreakerConfig: CircuitBreakerConfig{
				FailureThreshold: 10,
				SuccessThreshold: 3,
				Timeout:          30 * time.Second,
			},
		},
	}

	cfg2 := cfg1

	assert.Equal(t, cfg1.EnableErrorHandling, cfg2.EnableErrorHandling)
	assert.Equal(t, cfg1.ErrorHandlingConfig.CircuitBreakerConfig.FailureThreshold, cfg2.ErrorHandlingConfig.CircuitBreakerConfig.FailureThreshold)

	cfg1.EnableErrorHandling = false
	assert.NotEqual(t, cfg1.EnableErrorHandling, cfg2.EnableErrorHandling)
}
