package consensus

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

// TestLeaderElection tests the basic functionality of the leader election consensus
func TestLeaderElection(t *testing.T) {
	// Create a logger
	logger := slog.Default()
	
	// Create a consensus config
	config := &Config{
		NodeID:     "test-node",
		PeerIDs:    []string{},
		Algorithm:  "leader-election",
		ListenAddr: ":8080",
	}
	
	// Create a leader election consensus
	le := NewLeaderElection(logger, config)
	
	// Test that we can create a leader election consensus
	if le == nil {
		t.Error("Failed to create leader election consensus")
	}
	
	// Test the IsLeader method
	if le.IsLeader() {
		t.Error("Node should not be leader initially")
	}
	
	// Test the GetLeaderID method
	if le.GetLeaderID() != "" {
		t.Error("Leader ID should be empty initially")
	}
	
	// Test the Start method
	ctx := context.Background()
	if err := le.Start(ctx); err != nil {
		t.Errorf("Failed to start leader election: %v", err)
	}
	
	// Give some time for the election to run
	time.Sleep(100 * time.Millisecond)
	
	// Test that we can propose a value
	key := "test-key"
	value := "test-value"
	if err := le.ProposeValue(ctx, key, value); err != nil {
		t.Errorf("Failed to propose value: %v", err)
	}
	
	// Test that we can get a value
	retrievedValue, err := le.GetValue(ctx, key)
	if err != nil {
		t.Errorf("Failed to get value: %v", err)
	}
	
	// In our simple implementation, non-leader nodes don't propose values
	// so the retrieved value should be nil
	if retrievedValue != nil {
		t.Errorf("Expected nil value, got %v", retrievedValue)
	}
	
	// Test the Stop method
	if err := le.Stop(); err != nil {
		t.Errorf("Failed to stop leader election: %v", err)
	}
}