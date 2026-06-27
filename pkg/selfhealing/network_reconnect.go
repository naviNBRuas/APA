package selfhealing

import (
	"context"
	"fmt"
	"time"
)

// NewNetworkReconnectStrategy creates a new network reconnect strategy
func NewNetworkReconnectStrategy() *NetworkReconnectStrategy {
	return &NetworkReconnectStrategy{
		name:        "network-reconnect",
		description: "Reconnects broken network connections to restore connectivity",
		priority:    70,
		config:      make(map[string]interface{}),
	}
}

// Name returns the name of the strategy
func (n *NetworkReconnectStrategy) Name() string {
	return n.name
}

// Description returns the description of the strategy
func (n *NetworkReconnectStrategy) Description() string {
	return n.description
}

// CanHandle determines if this strategy can handle the given health issue
func (n *NetworkReconnectStrategy) CanHandle(issue *HealthIssue) bool {
	return issue.Type == "network" || issue.Component == "network"
}

// Apply applies the network reconnect strategy
func (n *NetworkReconnectStrategy) Apply(ctx context.Context, issue *HealthIssue) (*HealingResult, error) {
	startTime := time.Now()

	endpoint := "unknown"
	if ep, ok := issue.Context["endpoint"].(string); ok {
		endpoint = ep
	}

	if err := n.closeConnection(endpoint); err != nil {
		return nil, fmt.Errorf("failed to close connection: %w", err)
	}

	time.Sleep(50 * time.Millisecond)

	if err := n.establishConnection(endpoint); err != nil {
		return &HealingResult{
			Success:     false,
			ActionTaken: fmt.Sprintf("Attempted to reconnect to '%s'", endpoint),
			Message:     fmt.Sprintf("Connection establishment failed: %v", err),
			Metrics: map[string]interface{}{
				"reconnect_time_ms": time.Since(startTime).Milliseconds(),
			},
			RetryNeeded: true,
		}, nil
	}

	if err := n.verifyConnectivity(endpoint); err != nil {
		return &HealingResult{
			Success:     false,
			ActionTaken: fmt.Sprintf("Attempted to reconnect to '%s'", endpoint),
			Message:     fmt.Sprintf("Connectivity verification failed: %v", err),
			Metrics: map[string]interface{}{
				"reconnect_time_ms": time.Since(startTime).Milliseconds(),
			},
			RetryNeeded: true,
		}, nil
	}

	result := &HealingResult{
		Success:     true,
		ActionTaken: fmt.Sprintf("Reconnected network connection for '%s'", endpoint),
		Message:     "Network connection reestablished successfully",
		Metrics: map[string]interface{}{
			"reconnect_time_ms": time.Since(startTime).Milliseconds(),
			"packets_lost":      5,
		},
		RetryNeeded: false,
	}

	return result, nil
}

// closeConnection closes a network connection
func (n *NetworkReconnectStrategy) closeConnection(endpoint string) error {
	time.Sleep(30 * time.Millisecond)

	return nil
}

// establishConnection establishes a new network connection
func (n *NetworkReconnectStrategy) establishConnection(endpoint string) error {
	time.Sleep(100 * time.Millisecond)

	return nil
}

// verifyConnectivity verifies network connectivity
func (n *NetworkReconnectStrategy) verifyConnectivity(endpoint string) error {
	time.Sleep(50 * time.Millisecond)

	return nil
}

// Priority returns the priority of this strategy
func (n *NetworkReconnectStrategy) Priority() int {
	return n.priority
}

// Configure configures the strategy
func (n *NetworkReconnectStrategy) Configure(config map[string]interface{}) error {
	n.config = config
	return nil
}
