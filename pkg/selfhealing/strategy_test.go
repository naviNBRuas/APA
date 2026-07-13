package selfhealing

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRestartProcessStrategy_Basics(t *testing.T) {
	s := NewRestartProcessStrategy()
	require.NotNil(t, s)

	assert.Equal(t, "restart-process", s.Name())
	assert.Equal(t, "Restarts failed processes to restore functionality", s.Description())
	assert.Equal(t, 80, s.Priority())
}

func TestRestartProcessStrategy_CanHandle(t *testing.T) {
	s := NewRestartProcessStrategy()

	tests := []struct {
		name  string
		issue *HealthIssue
		want  bool
	}{
		{"process type", &HealthIssue{Type: "process"}, true},
		{"process component", &HealthIssue{Component: "process"}, true},
		{"other type", &HealthIssue{Type: "memory"}, false},
		{"other component", &HealthIssue{Component: "network"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, s.CanHandle(tt.issue))
		})
	}
}

func TestRestartProcessStrategy_Apply_NonExistentProcess(t *testing.T) {
	s := NewRestartProcessStrategy()
	issue := &HealthIssue{
		Type:      "process",
		Component: "test-does-not-exist-12345",
	}
	_, err := s.Apply(context.Background(), issue)
	assert.Error(t, err)
}

func TestRestartProcessStrategy_ApplyWithContextName(t *testing.T) {
	s := NewRestartProcessStrategy()
	issue := &HealthIssue{
		Type:      "process",
		Component: "fallback-process",
		Context: map[string]interface{}{
			"process_name": "also-not-real-99999",
		},
	}
	_, err := s.Apply(context.Background(), issue)
	assert.Error(t, err)
}

func TestRestartProcessStrategy_Configure(t *testing.T) {
	s := NewRestartProcessStrategy()
	config := map[string]interface{}{"timeout": 30, "retries": 3}
	err := s.Configure(config)
	assert.NoError(t, err)
}

func TestRebuildModuleStrategy_Basics(t *testing.T) {
	s := NewRebuildModuleStrategy()
	require.NotNil(t, s)

	assert.Equal(t, "rebuild-module", s.Name())
	assert.Equal(t, "Rebuilds corrupted or missing modules from trusted sources", s.Description())
	assert.Equal(t, 90, s.Priority())
}

func TestRebuildModuleStrategy_CanHandle(t *testing.T) {
	s := NewRebuildModuleStrategy()

	tests := []struct {
		name  string
		issue *HealthIssue
		want  bool
	}{
		{"module type", &HealthIssue{Type: "module"}, true},
		{"module component", &HealthIssue{Component: "module"}, true},
		{"other type", &HealthIssue{Type: "process"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, s.CanHandle(tt.issue))
		})
	}
}

func TestRebuildModuleStrategy_Apply_Success(t *testing.T) {
	s := NewRebuildModuleStrategy()
	issue := &HealthIssue{
		Type:      "module",
		Component: "test-module",
		Context: map[string]interface{}{
			"module_name": "test-module",
		},
	}
	ctx := context.Background()
	result, err := s.Apply(ctx, issue)
	assert.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Contains(t, result.ActionTaken, "test-module")
	assert.Contains(t, result.Message, "rebuilt")
}

func TestRebuildModuleStrategy_Apply_DefaultComponent(t *testing.T) {
	s := NewRebuildModuleStrategy()
	issue := &HealthIssue{
		Type:      "module",
		Component: "default-module",
	}
	ctx := context.Background()
	result, err := s.Apply(ctx, issue)
	assert.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
}

func TestRebuildModuleStrategy_VerifyIntegrity(t *testing.T) {
	s := NewRebuildModuleStrategy()

	err := s.verifyModuleIntegrity([]byte{}, "empty")
	assert.Error(t, err)

	err = s.verifyModuleIntegrity([]byte("short"), "short")
	assert.Error(t, err)

	err = s.verifyModuleIntegrity([]byte("long enough data"), "valid")
	assert.NoError(t, err)
}

func TestRebuildModuleStrategy_RequestFromPeers_CancelledCtx(t *testing.T) {
	s := NewRebuildModuleStrategy()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := s.requestModuleFromPeers(ctx, "test")
	assert.Error(t, err)
}

func TestRebuildModuleStrategy_Configure(t *testing.T) {
	s := NewRebuildModuleStrategy()
	config := map[string]interface{}{"source": "peer", "timeout": 10}
	err := s.Configure(config)
	assert.NoError(t, err)
}

func TestNetworkReconnectStrategy_Basics(t *testing.T) {
	s := NewNetworkReconnectStrategy()
	require.NotNil(t, s)

	assert.Equal(t, "network-reconnect", s.Name())
	assert.Equal(t, "Reconnects broken network connections to restore connectivity", s.Description())
	assert.Equal(t, 70, s.Priority())
}

func TestNetworkReconnectStrategy_CanHandle(t *testing.T) {
	s := NewNetworkReconnectStrategy()

	tests := []struct {
		name  string
		issue *HealthIssue
		want  bool
	}{
		{"network type", &HealthIssue{Type: "network"}, true},
		{"network component", &HealthIssue{Component: "network"}, true},
		{"other type", &HealthIssue{Type: "process"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, s.CanHandle(tt.issue))
		})
	}
}

func TestNetworkReconnectStrategy_Apply_UnknownEndpoint(t *testing.T) {
	s := NewNetworkReconnectStrategy()
	issue := &HealthIssue{
		Type:      "network",
		Component: "network",
	}
	ctx := context.Background()
	result, err := s.Apply(ctx, issue)
	assert.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, result.ActionTaken, "unknown")
}

func TestNetworkReconnectStrategy_Apply_CancelledCtx(t *testing.T) {
	s := NewNetworkReconnectStrategy()
	issue := &HealthIssue{
		Type:      "network",
		Component: "network",
		Context: map[string]interface{}{
			"endpoint": "127.0.0.1:1",
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := s.Apply(ctx, issue)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestNetworkReconnectStrategy_Configure(t *testing.T) {
	s := NewNetworkReconnectStrategy()
	config := map[string]interface{}{"timeout": 5}
	err := s.Configure(config)
	assert.NoError(t, err)
}

func TestMemoryOptimizationStrategy_Basics(t *testing.T) {
	s := NewMemoryOptimizationStrategy()
	require.NotNil(t, s)

	assert.Equal(t, "memory-optimization", s.Name())
	assert.Equal(t, "Optimizes memory usage to prevent out-of-memory conditions", s.Description())
	assert.Equal(t, 60, s.Priority())
}

func TestMemoryOptimizationStrategy_CanHandle(t *testing.T) {
	s := NewMemoryOptimizationStrategy()

	tests := []struct {
		name  string
		issue *HealthIssue
		want  bool
	}{
		{"memory type", &HealthIssue{Type: "memory"}, true},
		{"memory component", &HealthIssue{Component: "memory"}, true},
		{"other type", &HealthIssue{Type: "process"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, s.CanHandle(tt.issue))
		})
	}
}

func TestMemoryOptimizationStrategy_Apply(t *testing.T) {
	s := NewMemoryOptimizationStrategy()
	issue := &HealthIssue{Type: "memory", Component: "memory"}
	ctx := context.Background()
	result, err := s.Apply(ctx, issue)
	assert.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Contains(t, result.ActionTaken, "Optimized")
	assert.Contains(t, result.Message, "successfully")
	assert.Contains(t, result.Metrics, "memory_freed_mb")
}

func TestMemoryOptimizationStrategy_Configure(t *testing.T) {
	s := NewMemoryOptimizationStrategy()
	config := map[string]interface{}{"gc_threshold": "256MB"}
	err := s.Configure(config)
	assert.NoError(t, err)
}

func TestQuarantineNodeStrategy_Basics(t *testing.T) {
	s := NewQuarantineNodeStrategy()
	require.NotNil(t, s)

	assert.Equal(t, "quarantine-node", s.Name())
	assert.Equal(t, "Quarantines compromised nodes to prevent spread of issues", s.Description())
	assert.Equal(t, 100, s.Priority())
}

func TestQuarantineNodeStrategy_CanHandle(t *testing.T) {
	s := NewQuarantineNodeStrategy()

	tests := []struct {
		name  string
		issue *HealthIssue
		want  bool
	}{
		{"critical severity", &HealthIssue{Severity: "critical"}, true},
		{"security type", &HealthIssue{Type: "security"}, true},
		{"high severity non-security", &HealthIssue{Severity: "high", Type: "process"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, s.CanHandle(tt.issue))
		})
	}
}

func TestQuarantineNodeStrategy_Apply(t *testing.T) {
	s := NewQuarantineNodeStrategy()
	issue := &HealthIssue{
		Type:        "security",
		Severity:    "critical",
		Description: "Security breach detected",
	}
	ctx := context.Background()
	result, err := s.Apply(ctx, issue)
	assert.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Contains(t, result.ActionTaken, "Security breach detected")
}

func TestQuarantineNodeStrategy_Configure(t *testing.T) {
	s := NewQuarantineNodeStrategy()
	config := map[string]interface{}{"block_duration": "30m"}
	err := s.Configure(config)
	assert.NoError(t, err)
}

func TestMultipleStrategiesIntegration(t *testing.T) {
	logger := slog.Default()
	mockHealthChecker := new(MockHealthChecker)
	mockEventHandler := new(MockEventHandler)
	mockEventHandler.On("OnHealingSuccess", mock.Anything, mock.Anything, mock.Anything).Return()
	mockEventHandler.On("OnHealingFailure", mock.Anything, mock.Anything, mock.Anything).Return()
	mockEventHandler.On("OnHealingAttempt", mock.Anything, mock.Anything, mock.Anything).Return()

	framework := NewHealingFramework(logger, mockHealthChecker, mockEventHandler)

	framework.RegisterStrategy(NewRestartProcessStrategy())    //nolint:errcheck
	framework.RegisterStrategy(NewRebuildModuleStrategy())    //nolint:errcheck
	framework.RegisterStrategy(NewNetworkReconnectStrategy()) //nolint:errcheck
	framework.RegisterStrategy(NewMemoryOptimizationStrategy()) //nolint:errcheck
	framework.RegisterStrategy(NewQuarantineNodeStrategy())   //nolint:errcheck

	strategies := framework.ListStrategies()
	assert.Len(t, strategies, 5)
}

func TestApplyHealingStrategies_AllStrategiesFail(t *testing.T) {
	logger := slog.Default()
	mockHealthChecker := new(MockHealthChecker)
	mockEventHandler := new(MockEventHandler)
	mockEventHandler.On("OnHealingFailure", mock.Anything, mock.Anything, mock.Anything).Return()
	mockEventHandler.On("OnHealingAttempt", mock.Anything, mock.Anything, mock.Anything).Return()

	framework := NewHealingFramework(logger, mockHealthChecker, mockEventHandler)

	failStrategy := &MockStrategy{
		name:     "always-fail",
		priority: 50,
		failApply: true,
	}
	framework.RegisterStrategy(failStrategy) //nolint:errcheck

	issue := &HealthIssue{ID: "test-fail", Type: "test", Severity: "high"}
	ctx := context.Background()
	err := framework.applyHealingStrategies(ctx, issue)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "all healing strategies failed")
}
