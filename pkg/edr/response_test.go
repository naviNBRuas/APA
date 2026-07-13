package edr

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponseManager(t *testing.T) {
	logger := slog.Default()
	responseManager := NewResponseManager(logger)

	require.NotNil(t, responseManager, "Failed to create response manager")

	assert.NotNil(t, responseManager.actions, "Actions map not initialized")
	assert.NotNil(t, responseManager.responseRules, "Response rules map not initialized")
}

func TestAddRemoveAction(t *testing.T) {
	logger := slog.Default()
	responseManager := NewResponseManager(logger)

	action := &ResponseAction{
		ID:          "test-action-001",
		Name:        "Test Action",
		Description: "A test response action",
		ActionType:  "quarantine",
		Severity:    "high",
		Enabled:     true,
		CreatedAt:   time.Now(),
	}

	err := responseManager.AddAction(action)
	assert.NoError(t, err, "Failed to add action")
	assert.Equal(t, 1, len(responseManager.actions))

	err = responseManager.RemoveAction(action.ID)
	assert.NoError(t, err, "Failed to remove action")
	assert.Equal(t, 0, len(responseManager.actions))
}

func TestEnableDisableAction(t *testing.T) {
	logger := slog.Default()
	responseManager := NewResponseManager(logger)

	action := &ResponseAction{
		ID:          "test-action-002",
		Name:        "Test Action 2",
		Description: "Another test response action",
		ActionType:  "terminate",
		Severity:    "critical",
		Enabled:     false,
		CreatedAt:   time.Now(),
	}

	if err := responseManager.AddAction(action); err != nil {
		assert.NoError(t, err, "Failed to add action")
	}

	err := responseManager.EnableAction(action.ID)
	assert.NoError(t, err, "Failed to enable action")
	assert.True(t, responseManager.actions[action.ID].Enabled, "Action should be enabled")

	err = responseManager.DisableAction(action.ID)
	assert.NoError(t, err, "Failed to disable action")
	assert.False(t, responseManager.actions[action.ID].Enabled, "Action should be disabled")
}

func TestResponseRules(t *testing.T) {
	logger := slog.Default()
	responseManager := NewResponseManager(logger)

	actionIDs := []string{"action-001", "action-002", "action-003"}
	err := responseManager.AddResponseRule("high", actionIDs)
	assert.NoError(t, err, "Failed to add response rule")

	rules := responseManager.GetResponseRules()
	assert.Equal(t, 1, len(rules))
	assert.Equal(t, 3, len(rules["high"]))
}

func TestExecuteResponse(t *testing.T) {
	logger := slog.Default()
	responseManager := NewResponseManager(logger)

	quarantineAction := &ResponseAction{
		ID:          "quarantine-action",
		Name:        "Quarantine Node",
		Description: "Quarantine the current node",
		ActionType:  "quarantine",
		Severity:    "high",
		Enabled:     true,
		CreatedAt:   time.Now(),
	}

	terminateAction := &ResponseAction{
		ID:          "terminate-action",
		Name:        "Terminate Process",
		Description: "Terminate a suspicious process",
		ActionType:  "terminate",
		Severity:    "critical",
		Enabled:     true,
		CreatedAt:   time.Now(),
	}

	if err := responseManager.AddAction(quarantineAction); err != nil {
		assert.NoError(t, err, "Failed to add quarantine action")
	}
	if err := responseManager.AddAction(terminateAction); err != nil {
		assert.NoError(t, err, "Failed to add terminate action")
	}

	if err := responseManager.AddResponseRule("high", []string{"quarantine-action"}); err != nil {
		assert.NoError(t, err, "Failed to add high response rule")
	}
	if err := responseManager.AddResponseRule("critical", []string{"terminate-action"}); err != nil {
		assert.NoError(t, err, "Failed to add critical response rule")
	}

	highEvent := &Event{
		ID:        "high-event-001",
		Type:      "process",
		Timestamp: time.Now(),
		Source:    "suspicious_process.exe",
		Details:   "High severity event",
		Severity:  "high",
	}

	criticalEvent := &Event{
		ID:        "critical-event-001",
		Type:      "file",
		Timestamp: time.Now(),
		Source:    "/etc/passwd",
		Details:   "Critical severity event",
		Severity:  "critical",
	}

	ctx := context.Background()
	err := responseManager.ExecuteResponse(ctx, highEvent)
	assert.NoError(t, err, "Failed to execute response for high severity event")

	err = responseManager.ExecuteResponse(ctx, criticalEvent)
	assert.NoError(t, err, "Failed to execute response for critical severity event")
}

func TestGetAvailableActions(t *testing.T) {
	logger := slog.Default()
	responseManager := NewResponseManager(logger)

	action1 := &ResponseAction{
		ID:          "action-001",
		Name:        "Action 1",
		Description: "Test action 1",
		ActionType:  "quarantine",
		Severity:    "high",
		Enabled:     true,
		CreatedAt:   time.Now(),
	}

	action2 := &ResponseAction{
		ID:          "action-002",
		Name:        "Action 2",
		Description: "Test action 2",
		ActionType:  "terminate",
		Severity:    "critical",
		Enabled:     false,
		CreatedAt:   time.Now(),
	}

	responseManager.AddAction(action1) //nolint:errcheck
	responseManager.AddAction(action2) //nolint:errcheck

	actions := responseManager.GetAvailableActions()
	assert.Equal(t, 2, len(actions))
}
