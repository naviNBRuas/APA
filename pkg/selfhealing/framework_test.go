package selfhealing

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/naviNBRuas/APA/pkg/health"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockHealthChecker is a mock implementation of the HealthChecker interface
type MockHealthChecker struct {
	mock.Mock
}

func (m *MockHealthChecker) CheckHealth(ctx context.Context) ([]*health.CheckResult, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*health.CheckResult), args.Error(1)
}

// MockEventHandler is a mock implementation of the EventHandler interface
type MockEventHandler struct {
	mock.Mock
}

func (m *MockEventHandler) OnHealingAttempt(issue *HealthIssue, strategy HealingStrategy, result *HealingResult) {
	m.Called(issue, strategy, result)
}

func (m *MockEventHandler) OnHealingFailure(issue *HealthIssue, strategy HealingStrategy, err error) {
	m.Called(issue, strategy, err)
}

func (m *MockEventHandler) OnHealingSuccess(issue *HealthIssue, strategy HealingStrategy, result *HealingResult) {
	m.Called(issue, strategy, result)
}

func TestNewHealingFramework(t *testing.T) {
	logger := slog.Default()
	mockHealthChecker := new(MockHealthChecker)
	mockEventHandler := new(MockEventHandler)

	framework := NewHealingFramework(logger, mockHealthChecker, mockEventHandler)

	assert.NotNil(t, framework)
	assert.Empty(t, framework.ListStrategies())
}

func TestRegisterAndUnregisterStrategy(t *testing.T) {
	logger := slog.Default()
	mockHealthChecker := new(MockHealthChecker)
	mockEventHandler := new(MockEventHandler)

	framework := NewHealingFramework(logger, mockHealthChecker, mockEventHandler)

	// Test registering a strategy
	strategy := NewRestartProcessStrategy()
	err := framework.RegisterStrategy(strategy)
	assert.NoError(t, err)

	// Verify strategy is registered
	strategies := framework.ListStrategies()
	assert.Len(t, strategies, 1)
	assert.Contains(t, strategies, "restart-process")

	// Test registering duplicate strategy
	err = framework.RegisterStrategy(strategy)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")

	// Test unregistering strategy
	err = framework.UnregisterStrategy("restart-process")
	assert.NoError(t, err)

	// Verify strategy is unregistered
	strategies = framework.ListStrategies()
	assert.Empty(t, strategies)

	// Test unregistering non-existent strategy
	err = framework.UnregisterStrategy("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestConfigureStrategy(t *testing.T) {
	logger := slog.Default()
	mockHealthChecker := new(MockHealthChecker)
	mockEventHandler := new(MockEventHandler)

	framework := NewHealingFramework(logger, mockHealthChecker, mockEventHandler)

	// Register a strategy
	strategy := NewRestartProcessStrategy()
	err := framework.RegisterStrategy(strategy)
	assert.NoError(t, err)

	// Test configuring the strategy
	config := map[string]interface{}{
		"timeout": 30,
		"retries": 3,
	}

	err = framework.ConfigureStrategy("restart-process", config)
	assert.NoError(t, err)

	// Test configuring non-existent strategy
	err = framework.ConfigureStrategy("non-existent", config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestConvertCheckResultsToIssues(t *testing.T) {
	logger := slog.Default()
	mockHealthChecker := new(MockHealthChecker)
	mockEventHandler := new(MockEventHandler)

	framework := NewHealingFramework(logger, mockHealthChecker, mockEventHandler)

	// Create test health check results
	results := []*health.CheckResult{
		{
			Component: "process",
			Status:    health.StatusHealthy,
			Message:   "Process is running normally",
			Metrics:   map[string]interface{}{"cpu": 25.5, "memory": 100},
		},
		{
			Component: "network",
			Status:    health.StatusFailed,
			Message:   "Network connection lost",
			Metrics:   map[string]interface{}{"packets_lost": 100, "latency": 500},
		},
		{
			Component: "disk",
			Status:    health.StatusWarning,
			Message:   "Disk space low",
			Metrics:   map[string]interface{}{"free_space_mb": 100, "total_space_gb": 500},
		},
	}

	// Convert results to issues
	issues := framework.convertCheckResultsToIssues(results)

	// Should only have issues for failed and warning statuses
	assert.Len(t, issues, 2)

	// Verify first issue (failed network)
	assert.Equal(t, "network", issues[0].Type)
	assert.Equal(t, "critical", issues[0].Severity)
	assert.Equal(t, "Network connection lost", issues[0].Description)

	// Verify second issue (warning disk)
	assert.Equal(t, "disk", issues[1].Type)
	assert.Equal(t, "high", issues[1].Severity)
	assert.Equal(t, "Disk space low", issues[1].Description)
}

func TestGetApplicableStrategies(t *testing.T) {
	logger := slog.Default()
	mockHealthChecker := new(MockHealthChecker)
	mockEventHandler := new(MockEventHandler)

	framework := NewHealingFramework(logger, mockHealthChecker, mockEventHandler)

	// Register multiple strategies
	restartStrategy := NewRestartProcessStrategy()
	rebuildStrategy := NewRebuildModuleStrategy()
	networkStrategy := NewNetworkReconnectStrategy()
	quarantineStrategy := NewQuarantineNodeStrategy()

	framework.RegisterStrategy(restartStrategy)
	framework.RegisterStrategy(rebuildStrategy)
	framework.RegisterStrategy(networkStrategy)
	framework.RegisterStrategy(quarantineStrategy)

	// Create a process-related issue
	processIssue := &HealthIssue{
		Type:      "process",
		Severity:  "high",
		Component: "test-process",
	}

	// Get applicable strategies for process issue
	applicable := framework.getApplicableStrategies(processIssue)

	// Should only get the restart strategy
	assert.Len(t, applicable, 1)
	assert.Equal(t, "restart-process", applicable[0].Name())

	// Create a critical security issue
	securityIssue := &HealthIssue{
		Type:      "security",
		Severity:  "critical",
		Component: "test-component",
	}

	// Get applicable strategies for security issue
	applicable = framework.getApplicableStrategies(securityIssue)

	// Should get the quarantine strategy (highest priority)
	assert.Len(t, applicable, 1)
	assert.Equal(t, "quarantine-node", applicable[0].Name())
}

func TestApplyHealingStrategies(t *testing.T) {
	logger := slog.Default()
	mockHealthChecker := new(MockHealthChecker)
	mockEventHandler := new(MockEventHandler)

	framework := NewHealingFramework(logger, mockHealthChecker, mockEventHandler)

	// Register a strategy
	strategy := NewRestartProcessStrategy()
	framework.RegisterStrategy(strategy)

	// Create a process issue
	processIssue := &HealthIssue{
		ID:        "test-issue-1",
		Type:      "process",
		Severity:  "high",
		Component: "test-process",
		Timestamp: time.Now(),
	}

	// Apply healing strategies
	ctx := context.Background()
	err := framework.applyHealingStrategies(ctx, processIssue)

	// Should succeed
	assert.NoError(t, err)
}

func TestApplyHealingStrategies_NoApplicableStrategies(t *testing.T) {
	logger := slog.Default()
	mockHealthChecker := new(MockHealthChecker)
	mockEventHandler := new(MockEventHandler)

	framework := NewHealingFramework(logger, mockHealthChecker, mockEventHandler)

	// Create an issue for which we have no strategies
	unknownIssue := &HealthIssue{
		ID:        "test-issue-1",
		Type:      "unknown-type",
		Severity:  "high",
		Component: "unknown-component",
		Timestamp: time.Now(),
	}

	// Apply healing strategies
	ctx := context.Background()
	err := framework.applyHealingStrategies(ctx, unknownIssue)

	// Should not error, but log that no strategies were found
	assert.NoError(t, err)
}

func TestStrategiesPriorityOrdering(t *testing.T) {
	logger := slog.Default()
	mockHealthChecker := new(MockHealthChecker)
	mockEventHandler := new(MockEventHandler)

	framework := NewHealingFramework(logger, mockHealthChecker, mockEventHandler)

	// Register strategies with different priorities
	lowPriority := &MockStrategy{name: "low", priority: 10}
	mediumPriority := &MockStrategy{name: "medium", priority: 50}
	highPriority := &MockStrategy{name: "high", priority: 90}

	framework.RegisterStrategy(lowPriority)
	framework.RegisterStrategy(mediumPriority)
	framework.RegisterStrategy(highPriority)

	// Create an issue that all strategies can handle
	issue := &HealthIssue{
		Type:      "test",
		Severity:  "high",
		Component: "test-component",
	}

	// Get applicable strategies
	applicable := framework.getApplicableStrategies(issue)

	// Should be ordered by priority (highest first)
	assert.Len(t, applicable, 3)
	assert.Equal(t, "high", applicable[0].Name())
	assert.Equal(t, "medium", applicable[1].Name())
	assert.Equal(t, "low", applicable[2].Name())
}

// MockStrategy is a mock implementation of HealingStrategy for testing
type MockStrategy struct {
	name     string
	priority int
}

func (m *MockStrategy) Name() string {
	return m.name
}

func (m *MockStrategy) Description() string {
	return "Mock strategy for testing"
}

func (m *MockStrategy) CanHandle(issue *HealthIssue) bool {
	// Can handle any issue for testing
	return true
}

func (m *MockStrategy) Apply(ctx context.Context, issue *HealthIssue) (*HealingResult, error) {
	return &HealingResult{
		Success:     true,
		ActionTaken: "Mock action",
		Message:     "Mock strategy applied successfully",
	}, nil
}

func (m *MockStrategy) Priority() int {
	return m.priority
}

func (m *MockStrategy) Configure(config map[string]interface{}) error {
	return nil
}