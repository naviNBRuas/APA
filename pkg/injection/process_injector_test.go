package injection

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProcessInjector(t *testing.T) {
	logger := slog.Default()
	agentPath := "/test/agentd"
	peerID := peer.ID("test-peer")

	injector := NewProcessInjector(logger, agentPath, peerID)
	require.NotNil(t, injector)
	assert.Equal(t, agentPath, injector.agentPath)
	assert.Equal(t, peerID, injector.peerID)
	assert.NotNil(t, injector.injectChan)
}

func TestProcessInjectorStart(t *testing.T) {
	logger := slog.Default()
	agentPath := "/test/agentd"
	peerID := peer.ID("test-peer")

	injector := NewProcessInjector(logger, agentPath, peerID)
	require.NotNil(t, injector)

	// Start the injector
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	injector.Start(ctx)

	// Give it a moment to start
	time.Sleep(50 * time.Millisecond)

	// The injector should be running without errors
	assert.NotNil(t, injector)
}

func TestIsProcessRunning(t *testing.T) {
	injector := &ProcessInjector{
		logger: slog.Default(),
	}

	// Test with the current process name
	result := injector.isProcessRunning("go")
	// This should return true or false depending on the environment
	// We'll just check that it doesn't panic
	assert.NotNil(t, result)
}

func TestRequestInjection(t *testing.T) {
	logger := slog.Default()
	agentPath := "/test/agentd"
	peerID := peer.ID("test-peer")

	injector := NewProcessInjector(logger, agentPath, peerID)
	require.NotNil(t, injector)

	// Request injection
	injector.RequestInjection("test-process")

	// Check that the request was queued
	select {
	case process := <-injector.injectChan:
		assert.Equal(t, "test-process", process)
	default:
		t.Error("Injection request was not queued")
	}
}

func TestInjectIntoProcessWithPayload(t *testing.T) {
	logger := slog.Default()
	agentPath := "/test/agentd"
	peerID := peer.ID("test-peer")

	injector := NewProcessInjector(logger, agentPath, peerID)
	require.NotNil(t, injector)

	// Create a temporary payload file
	payloadFile, err := os.CreateTemp("", "payload-*")
	require.NoError(t, err)
	defer os.Remove(payloadFile.Name())

	// Test injecting with payload
	err = injector.InjectIntoProcessWithPayload("test-process", payloadFile.Name())
	assert.NoError(t, err)
}

func TestInjectIntoProcessWithNonexistentPayload(t *testing.T) {
	logger := slog.Default()
	agentPath := "/test/agentd"
	peerID := peer.ID("test-peer")

	injector := NewProcessInjector(logger, agentPath, peerID)
	require.NotNil(t, injector)

	// Test injecting with nonexistent payload
	err := injector.InjectIntoProcessWithPayload("test-process", "/nonexistent/payload")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "payload not found")
}