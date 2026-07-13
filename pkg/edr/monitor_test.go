package edr

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMonitor(t *testing.T) {
	logger := slog.Default()
	monitor := NewMonitor(logger)

	require.NotNil(t, monitor, "Failed to create monitor")

	assert.NotNil(t, monitor.eventChannel, "Event channel not initialized")
	assert.NotNil(t, monitor.stopChannel, "Stop channel not initialized")
}

func TestEventHandling(t *testing.T) {
	logger := slog.Default()
	monitor := NewMonitor(logger)

	event := &Event{
		ID:        "test-event-001",
		Type:      "process",
		Timestamp: time.Now(),
		Source:    "test_process.exe",
		Details:   "Test event for testing purposes",
		Severity:  "medium",
	}

	monitor.handleEvent(event)
}

func TestMonitorLifecycle(t *testing.T) {
	logger := slog.Default()
	monitor := NewMonitor(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	monitor.StartMonitoring(ctx)

	time.Sleep(100 * time.Millisecond)

	monitor.StopMonitoring()

	time.Sleep(100 * time.Millisecond)
}
