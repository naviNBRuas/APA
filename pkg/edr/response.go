package edr

import (
	"context"
	"log/slog"
)

// ResponseManager handles automated response actions
type ResponseManager struct {
	logger        *slog.Logger
	actions       map[string]*ResponseAction
	responseRules map[string][]string // Map of severity to action IDs
}

// NewResponseManager creates a new response manager
func NewResponseManager(logger *slog.Logger) *ResponseManager {
	return &ResponseManager{
		logger:        logger,
		actions:       make(map[string]*ResponseAction),
		responseRules: make(map[string][]string),
	}
}

// AddAction adds a response action to the manager
func (rm *ResponseManager) AddAction(action *ResponseAction) error {
	rm.actions[action.ID] = action
	rm.logger.Info("Added response action", "id", action.ID, "name", action.Name, "type", action.ActionType)
	return nil
}

// RemoveAction removes a response action from the manager
func (rm *ResponseManager) RemoveAction(actionID string) error {
	if _, exists := rm.actions[actionID]; !exists {
		return nil // Already removed or doesn't exist
	}

	delete(rm.actions, actionID)
	rm.logger.Info("Removed response action", "id", actionID)
	return nil
}

// EnableAction enables a response action
func (rm *ResponseManager) EnableAction(actionID string) error {
	action, exists := rm.actions[actionID]
	if !exists {
		return nil // Action doesn't exist
	}

	action.Enabled = true
	rm.logger.Info("Enabled response action", "id", actionID)
	return nil
}

// DisableAction disables a response action
func (rm *ResponseManager) DisableAction(actionID string) error {
	action, exists := rm.actions[actionID]
	if !exists {
		return nil // Action doesn't exist
	}

	action.Enabled = false
	rm.logger.Info("Disabled response action", "id", actionID)
	return nil
}

// AddResponseRule adds a rule for automatic response based on event severity
func (rm *ResponseManager) AddResponseRule(severity string, actionIDs []string) error {
	rm.responseRules[severity] = actionIDs
	rm.logger.Info("Added response rule", "severity", severity, "action_count", len(actionIDs))
	return nil
}

// ExecuteResponse executes response actions for a given event
func (rm *ResponseManager) ExecuteResponse(ctx context.Context, event *Event) error {
	// Get actions for this event's severity
	actionIDs, exists := rm.responseRules[event.Severity]
	if !exists {
		rm.logger.Debug("No response rules for severity", "severity", event.Severity)
		return nil
	}

	rm.logger.Info("Executing response actions for event",
		"event_id", event.ID,
		"severity", event.Severity,
		"action_count", len(actionIDs))

	// Execute each action
	for _, actionID := range actionIDs {
		if err := rm.executeAction(ctx, actionID, event); err != nil {
			rm.logger.Error("Failed to execute response action", "action_id", actionID, "error", err)
		}
	}

	return nil
}

// executeAction executes a single response action
func (rm *ResponseManager) executeAction(ctx context.Context, actionID string, event *Event) error {
	action, exists := rm.actions[actionID]
	if !exists {
		return nil // Action doesn't exist
	}

	if !action.Enabled {
		rm.logger.Debug("Skipping disabled action", "action_id", actionID)
		return nil
	}

	rm.logger.Info("Executing response action",
		"action_id", actionID,
		"action_type", action.ActionType,
		"event_id", event.ID)

	// Execute the appropriate action based on type
	switch action.ActionType {
	case "quarantine":
		return rm.quarantineNode(ctx, event)
	case "terminate":
		return rm.terminateProcess(ctx, event)
	case "isolate":
		return rm.isolateNetwork(ctx, event)
	case "self-destruct":
		return rm.selfDestruct(ctx, event)
	default:
		rm.logger.Warn("Unknown action type", "action_type", action.ActionType)
		return nil
	}
}

// GetAvailableActions returns all available response actions
func (rm *ResponseManager) GetAvailableActions() []*ResponseAction {
	var actions []*ResponseAction

	for _, action := range rm.actions {
		actions = append(actions, action)
	}

	return actions
}

// GetResponseRules returns all response rules
func (rm *ResponseManager) GetResponseRules() map[string][]string {
	return rm.responseRules
}
