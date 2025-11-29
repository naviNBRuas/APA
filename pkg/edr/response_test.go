package edr

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

func TestResponseManager(t *testing.T) {
	logger := slog.Default()
	responseManager := NewResponseManager(logger)

	// Test creating a response manager
	if responseManager == nil {
		t.Fatal("Failed to create response manager")
	}

	// Test that fields are initialized
	if responseManager.actions == nil {
		t.Error("Actions map not initialized")
	}

	if responseManager.responseRules == nil {
		t.Error("Response rules map not initialized")
	}
}

func TestAddRemoveAction(t *testing.T) {
	logger := slog.Default()
	responseManager := NewResponseManager(logger)

	// Create a test action
	action := &ResponseAction{
		ID:          "test-action-001",
		Name:        "Test Action",
		Description: "A test response action",
		ActionType:  "quarantine",
		Severity:    "high",
		Enabled:     true,
		CreatedAt:   time.Now(),
	}

	// Test adding an action
	err := responseManager.AddAction(action)
	if err != nil {
		t.Errorf("Failed to add action: %v", err)
	}

	// Check that action was added
	if len(responseManager.actions) != 1 {
		t.Errorf("Expected 1 action, got %d", len(responseManager.actions))
	}

	// Test removing an action
	err = responseManager.RemoveAction(action.ID)
	if err != nil {
		t.Errorf("Failed to remove action: %v", err)
	}

	// Check that action was removed
	if len(responseManager.actions) != 0 {
		t.Errorf("Expected 0 actions, got %d", len(responseManager.actions))
	}
}

func TestEnableDisableAction(t *testing.T) {
	logger := slog.Default()
	responseManager := NewResponseManager(logger)

	// Create a test action (disabled by default)
	action := &ResponseAction{
		ID:          "test-action-002",
		Name:        "Test Action 2",
		Description: "Another test response action",
		ActionType:  "terminate",
		Severity:    "critical",
		Enabled:     false,
		CreatedAt:   time.Now(),
	}

	// Add the action
	responseManager.AddAction(action)

	// Test enabling an action
	err := responseManager.EnableAction(action.ID)
	if err != nil {
		t.Errorf("Failed to enable action: %v", err)
	}

	if !responseManager.actions[action.ID].Enabled {
		t.Error("Action should be enabled")
	}

	// Test disabling an action
	err = responseManager.DisableAction(action.ID)
	if err != nil {
		t.Errorf("Failed to disable action: %v", err)
	}

	if responseManager.actions[action.ID].Enabled {
		t.Error("Action should be disabled")
	}
}

func TestResponseRules(t *testing.T) {
	logger := slog.Default()
	responseManager := NewResponseManager(logger)

	// Test adding response rules
	actionIDs := []string{"action-001", "action-002", "action-003"}
	err := responseManager.AddResponseRule("high", actionIDs)
	if err != nil {
		t.Errorf("Failed to add response rule: %v", err)
	}

	// Check that rule was added
	rules := responseManager.GetResponseRules()
	if len(rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(rules))
	}

	if len(rules["high"]) != 3 {
		t.Errorf("Expected 3 action IDs, got %d", len(rules["high"]))
	}
}

func TestExecuteResponse(t *testing.T) {
	logger := slog.Default()
	responseManager := NewResponseManager(logger)

	// Create test actions
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

	// Add actions
	responseManager.AddAction(quarantineAction)
	responseManager.AddAction(terminateAction)

	// Add response rules
	responseManager.AddResponseRule("high", []string{"quarantine-action"})
	responseManager.AddResponseRule("critical", []string{"terminate-action"})

	// Create test events
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

	// Test executing response for high severity event
	ctx := context.Background()
	err := responseManager.ExecuteResponse(ctx, highEvent)
	if err != nil {
		t.Errorf("Failed to execute response for high severity event: %v", err)
	}

	// Test executing response for critical severity event
	err = responseManager.ExecuteResponse(ctx, criticalEvent)
	if err != nil {
		t.Errorf("Failed to execute response for critical severity event: %v", err)
	}
}

func TestGetAvailableActions(t *testing.T) {
	logger := slog.Default()
	responseManager := NewResponseManager(logger)

	// Create test actions
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

	// Add actions
	responseManager.AddAction(action1)
	responseManager.AddAction(action2)

	// Test getting available actions
	actions := responseManager.GetAvailableActions()
	if len(actions) != 2 {
		t.Errorf("Expected 2 actions, got %d", len(actions))
	}
}