package edr

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

func TestMonitor(t *testing.T) {
	logger := slog.Default()
	monitor := NewMonitor(logger)

	// Test creating a monitor
	if monitor == nil {
		t.Fatal("Failed to create monitor")
	}

	// Test that channels are initialized
	if monitor.eventChannel == nil {
		t.Error("Event channel not initialized")
	}

	if monitor.stopChannel == nil {
		t.Error("Stop channel not initialized")
	}
}

func TestEventHandling(t *testing.T) {
	logger := slog.Default()
	monitor := NewMonitor(logger)

	// Create a test event
	event := &Event{
		ID:        "test-event-001",
		Type:      "process",
		Timestamp: time.Now(),
		Source:    "test_process.exe",
		Details:   "Test event for testing purposes",
		Severity:  "medium",
	}

	// Test handling an event
	monitor.handleEvent(event)
	
	// Since handleEvent just logs, we can't easily test its effects
	// In a real implementation, we might mock the logger or add callbacks
}

func TestMonitorLifecycle(t *testing.T) {
	logger := slog.Default()
	monitor := NewMonitor(logger)

	// Create a context with timeout for testing
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Test starting monitoring
	monitor.StartMonitoring(ctx)

	// Give some time for monitoring to start
	time.Sleep(100 * time.Millisecond)

	// Test stopping monitoring
	monitor.StopMonitoring()

	// Give some time for monitoring to stop
	time.Sleep(100 * time.Millisecond)
}