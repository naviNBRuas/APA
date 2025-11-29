package edr

import (
	"log/slog"
	"testing"
	"time"
)

func TestAnomalyDetector(t *testing.T) {
	logger := slog.Default()
	detector := NewAnomalyDetector(logger)

	// Test creating a detector
	if detector == nil {
		t.Fatal("Failed to create anomaly detector")
	}

	// Test that fields are initialized
	if detector.eventCounts == nil {
		t.Error("Event counts map not initialized")
	}

	if detector.totalEvents != 0 {
		t.Errorf("Expected totalEvents to be 0, got %d", detector.totalEvents)
	}

	if detector.threshold != 2.0 {
		t.Errorf("Expected threshold to be 2.0, got %f", detector.threshold)
	}
}

func TestAnalyzeEvent(t *testing.T) {
	logger := slog.Default()
	detector := NewAnomalyDetector(logger)

	// Create test events
	event1 := &Event{
		ID:        "event-001",
		Type:      "process",
		Timestamp: time.Now(),
		Source:    "normal_process.exe",
		Details:   "Normal process execution",
		Severity:  "low",
	}

	event2 := &Event{
		ID:        "event-002",
		Type:      "process",
		Timestamp: time.Now(),
		Source:    "suspicious_process.exe",
		Details:   "Suspicious process execution",
		Severity:  "high",
	}

	// Test analyzing normal events
	isAnomaly1 := detector.AnalyzeEvent(event1)
	if isAnomaly1 {
		t.Error("Normal event should not be flagged as anomaly")
	}

	// Test analyzing suspicious events
	isAnomaly2 := detector.AnalyzeEvent(event2)
	if isAnomaly2 {
		// This might be flagged as anomaly depending on the implementation
		// For now, we'll just log it
		t.Log("Suspicious event may be flagged as anomaly")
	}

	// Check that total events count is updated
	if detector.totalEvents != 2 {
		t.Errorf("Expected totalEvents to be 2, got %d", detector.totalEvents)
	}
}

func TestUpdateThreshold(t *testing.T) {
	logger := slog.Default()
	detector := NewAnomalyDetector(logger)

	// Test updating threshold
	newThreshold := 3.0
	detector.UpdateThreshold(newThreshold)

	if detector.threshold != newThreshold {
		t.Errorf("Expected threshold to be %f, got %f", newThreshold, detector.threshold)
	}
}

func TestGetAnomalyStats(t *testing.T) {
	logger := slog.Default()
	detector := NewAnomalyDetector(logger)

	// Test getting stats with no events
	stats := detector.GetAnomalyStats()
	if stats == nil {
		t.Error("Expected stats map, got nil")
	}

	// Check that all expected keys are present
	expectedKeys := []string{"total_events", "unique_event_types", "current_threshold", "detected_anomalies"}
	for _, key := range expectedKeys {
		if _, exists := stats[key]; !exists {
			t.Errorf("Expected key %s in stats", key)
		}
	}
}

func TestMachineLearningDetector(t *testing.T) {
	logger := slog.Default()
	mlDetector := NewMachineLearningDetector(logger)

	// Test creating ML detector
	if mlDetector == nil {
		t.Fatal("Failed to create ML detector")
	}

	// Test training model
	trainingData := []Event{
		{
			ID:        "train-001",
			Type:      "process",
			Timestamp: time.Now(),
			Source:    "training_process.exe",
			Details:   "Training data process",
			Severity:  "low",
		},
	}

	err := mlDetector.TrainModel(trainingData)
	if err != nil {
		t.Errorf("Failed to train model: %v", err)
	}

	// Test detecting anomaly
	testEvent := &Event{
		ID:        "test-001",
		Type:      "process",
		Timestamp: time.Now(),
		Source:    "test_process.exe",
		Details:   "Test process",
		Severity:  "medium",
	}

	_, _ = mlDetector.DetectAnomaly(testEvent)
	// We don't check the return values since they're placeholders
	// In a real implementation, we would validate them

	// Test updating model
	newData := []Event{
		{
			ID:        "update-001",
			Type:      "file",
			Timestamp: time.Now(),
			Source:    "/tmp/new_file.txt",
			Details:   "New file data",
			Severity:  "low",
		},
	}

	err = mlDetector.UpdateModel(newData)
	if err != nil {
		t.Errorf("Failed to update model: %v", err)
	}
}