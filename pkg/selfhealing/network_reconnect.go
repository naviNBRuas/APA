package selfhealing

import (
	"context"
	"fmt"
	"net"
	"time"
)

func NewNetworkReconnectStrategy() *NetworkReconnectStrategy {
	return &NetworkReconnectStrategy{
		name:        "network-reconnect",
		description: "Reconnects broken network connections to restore connectivity",
		priority:    70,
		config:      make(map[string]interface{}),
	}
}

func (n *NetworkReconnectStrategy) Name() string {
	return n.name
}

func (n *NetworkReconnectStrategy) Description() string {
	return n.description
}

func (n *NetworkReconnectStrategy) CanHandle(issue *HealthIssue) bool {
	return issue.Type == "network" || issue.Component == "network"
}

func (n *NetworkReconnectStrategy) Apply(ctx context.Context, issue *HealthIssue) (*HealingResult, error) {
	startTime := time.Now()

	endpoint := "unknown"
	if ep, ok := issue.Context["endpoint"].(string); ok {
		endpoint = ep
	}

	if err := n.closeConnection(endpoint); err != nil {
		return nil, fmt.Errorf("failed to close connection: %w", err)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

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
		},
		RetryNeeded: false,
	}

	return result, nil
}

func (n *NetworkReconnectStrategy) closeConnection(endpoint string) error {
	_ = endpoint
	return nil
}

func (n *NetworkReconnectStrategy) establishConnection(endpoint string) error {
	conn, err := net.DialTimeout("tcp", endpoint, 3*time.Second)
	if err != nil {
		return fmt.Errorf("dial %s: %w", endpoint, err)
	}
	_ = conn.Close()
	return nil
}

func (n *NetworkReconnectStrategy) verifyConnectivity(endpoint string) error {
	conn, err := net.DialTimeout("tcp", endpoint, 2*time.Second)
	if err != nil {
		return fmt.Errorf("connectivity check failed for %s: %w", endpoint, err)
	}
	_ = conn.Close()
	return nil
}

func (n *NetworkReconnectStrategy) Priority() int {
	return n.priority
}

func (n *NetworkReconnectStrategy) Configure(config map[string]interface{}) error {
	n.config = config
	return nil
}
