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

func TestInjectionIntegration(t *testing.T) {
	logger := slog.Default()
	agentPath := "/test/agentd"
	peerID := peer.ID("test-peer")

	// Create process injector
	processInjector := NewProcessInjector(logger, agentPath, peerID)
	require.NotNil(t, processInjector)

	// Create library embedder
	libraryEmbedder := NewLibraryEmbedder(logger, agentPath, peerID)
	require.NotNil(t, libraryEmbedder)

	// Test that both components were created correctly
	assert.Equal(t, agentPath, processInjector.agentPath)
	assert.Equal(t, peerID, processInjector.peerID)
	assert.Equal(t, agentPath, libraryEmbedder.agentPath)
	assert.Equal(t, peerID, libraryEmbedder.peerID)

	// Start both components
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	processInjector.Start(ctx)
	libraryEmbedder.Start(ctx)

	// Give them a moment to start
	time.Sleep(100 * time.Millisecond)

	// Request injection
	processInjector.RequestInjection("test-process")

	// Request embedding
	libraryEmbedder.RequestEmbedding("test-library")

	// Give them a moment to process the requests
	time.Sleep(50 * time.Millisecond)

	// Test injecting with payload
	payloadFile, err := os.CreateTemp("", "payload-*")
	require.NoError(t, err)
	defer os.Remove(payloadFile.Name())

	err = processInjector.InjectIntoProcessWithPayload("test-process", payloadFile.Name())
	assert.NoError(t, err)

	// Test embedding with payload
	err = libraryEmbedder.EmbedIntoLibraryWithPayload("test-library", payloadFile.Name())
	assert.NoError(t, err)
}

func TestInjectionWithNonexistentPayload(t *testing.T) {
	logger := slog.Default()
	agentPath := "/test/agentd"
	peerID := peer.ID("test-peer")

	// Create process injector
	processInjector := NewProcessInjector(logger, agentPath, peerID)
	require.NotNil(t, processInjector)

	// Create library embedder
	libraryEmbedder := NewLibraryEmbedder(logger, agentPath, peerID)
	require.NotNil(t, libraryEmbedder)

	// Test injecting with nonexistent payload
	err := processInjector.InjectIntoProcessWithPayload("test-process", "/nonexistent/payload")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "payload not found")

	// Test embedding with nonexistent payload
	err = libraryEmbedder.EmbedIntoLibraryWithPayload("test-library", "/nonexistent/payload")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "payload not found")
}