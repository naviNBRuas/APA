package selfhealing

import (
	"context"
	"time"
)

// HealingStrategy defines the interface for self-healing strategies
type HealingStrategy interface {
	Name() string

	Description() string

	CanHandle(issue *HealthIssue) bool

	Apply(ctx context.Context, issue *HealthIssue) (*HealingResult, error)

	Priority() int

	Configure(config map[string]interface{}) error
}

// HealthIssue represents a health problem detected in the system
type HealthIssue struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Description string                 `json:"description"`
	Component   string                 `json:"component"`
	Timestamp   time.Time              `json:"timestamp"`
	Metrics     map[string]interface{} `json:"metrics"`
	Context     map[string]interface{} `json:"context"`
}

// HealingResult represents the result of applying a healing strategy
type HealingResult struct {
	Success     bool                   `json:"success"`
	ActionTaken string                 `json:"action_taken"`
	Message     string                 `json:"message"`
	Duration    time.Duration          `json:"duration"`
	Metrics     map[string]interface{} `json:"metrics"`
	RetryNeeded bool                   `json:"retry_needed"`
}
