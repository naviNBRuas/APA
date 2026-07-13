package consensus

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	require.NotNil(t, le, "Failed to create leader election consensus")
	assert.False(t, le.IsLeader(), "Node should not be leader initially")
	assert.Empty(t, le.GetLeaderID(), "Leader ID should be empty initially")

	ctx := context.Background()
	assert.NoError(t, le.Start(ctx), "Failed to start leader election")

	time.Sleep(100 * time.Millisecond)

	key := "test-key"
	value := "test-value"
	assert.NoError(t, le.ProposeValue(ctx, key, value), "Failed to propose value")

	retrievedValue, err := le.GetValue(ctx, key)
	assert.NoError(t, err, "Failed to get value")

	assert.Nil(t, retrievedValue, "Expected nil value, got %v", retrievedValue)

	assert.NoError(t, le.Stop(), "Failed to stop leader election")
}
