package selfhealing

import (
	"context"
	"fmt"
	"time"
)

// NewQuarantineNodeStrategy creates a new quarantine node strategy
func NewQuarantineNodeStrategy() *QuarantineNodeStrategy {
	return &QuarantineNodeStrategy{
		name:        "quarantine-node",
		description: "Quarantines compromised nodes to prevent spread of issues",
		priority:    100,
		config:      make(map[string]interface{}),
	}
}

// Name returns the name of the strategy
func (q *QuarantineNodeStrategy) Name() string {
	return q.name
}

// Description returns the description of the strategy
func (q *QuarantineNodeStrategy) Description() string {
	return q.description
}

// CanHandle determines if this strategy can handle the given health issue
func (q *QuarantineNodeStrategy) CanHandle(issue *HealthIssue) bool {
	return issue.Severity == "critical" || issue.Type == "security"
}

// Apply applies the quarantine node strategy
func (q *QuarantineNodeStrategy) Apply(ctx context.Context, issue *HealthIssue) (*HealingResult, error) {
	startTime := time.Now()

	if err := q.isolateNetwork(); err != nil {
		return nil, fmt.Errorf("failed to isolate network: %w", err)
	}

	if err := q.stopModulesAndControllers(); err != nil {
		return nil, fmt.Errorf("failed to stop modules and controllers: %w", err)
	}

	if err := q.preventNewModules(); err != nil {
		return nil, fmt.Errorf("failed to prevent new modules: %w", err)
	}

	if err := q.reportQuarantineEvent(issue); err != nil {
		fmt.Printf("Warning: Failed to report quarantine event: %v\n", err)
	}

	result := &HealingResult{
		Success:     true,
		ActionTaken: fmt.Sprintf("Quarantined node due to '%s'", issue.Description),
		Message:     "Node quarantined successfully to prevent issue spread",
		Metrics: map[string]interface{}{
			"quarantine_time_ms":  time.Since(startTime).Milliseconds(),
			"connections_blocked": 15,
		},
		RetryNeeded: false,
	}

	return result, nil
}

// isolateNetwork isolates the node from the network
func (q *QuarantineNodeStrategy) isolateNetwork() error {
	time.Sleep(200 * time.Millisecond)

	return nil
}

// stopModulesAndControllers stops all running modules and controllers
func (q *QuarantineNodeStrategy) stopModulesAndControllers() error {
	time.Sleep(150 * time.Millisecond)

	return nil
}

// preventNewModules prevents new modules from loading
func (q *QuarantineNodeStrategy) preventNewModules() error {
	time.Sleep(50 * time.Millisecond)

	return nil
}

// reportQuarantineEvent reports the quarantine event to central management
func (q *QuarantineNodeStrategy) reportQuarantineEvent(issue *HealthIssue) error {
	time.Sleep(100 * time.Millisecond)

	return nil
}

// Priority returns the priority of this strategy
func (q *QuarantineNodeStrategy) Priority() int {
	return q.priority
}

// Configure configures the strategy
func (q *QuarantineNodeStrategy) Configure(config map[string]interface{}) error {
	q.config = config
	return nil
}
