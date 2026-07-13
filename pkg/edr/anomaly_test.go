package edr

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnomalyDetector(t *testing.T) {
	logger := slog.Default()
	detector := NewAnomalyDetector(logger)

	require.NotNil(t, detector, "Failed to create anomaly detector")

	assert.NotNil(t, detector.eventCounts, "Event counts map not initialized")
	assert.Equal(t, 0, detector.totalEvents)
	assert.Equal(t, 2.0, detector.threshold)
}

func TestAnalyzeEvent(t *testing.T) {
	logger := slog.Default()
	detector := NewAnomalyDetector(logger)

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

	isAnomaly1 := detector.AnalyzeEvent(event1)
	assert.False(t, isAnomaly1, "Normal event should not be flagged as anomaly")

	isAnomaly2 := detector.AnalyzeEvent(event2)
	if isAnomaly2 {
		t.Log("Suspicious event may be flagged as anomaly")
	}

	assert.Equal(t, 2, detector.totalEvents)
}

func TestUpdateThreshold(t *testing.T) {
	logger := slog.Default()
	detector := NewAnomalyDetector(logger)

	newThreshold := 3.0
	detector.UpdateThreshold(newThreshold)

	assert.Equal(t, newThreshold, detector.threshold)
}

func TestGetAnomalyStats(t *testing.T) {
	logger := slog.Default()
	detector := NewAnomalyDetector(logger)

	stats := detector.GetAnomalyStats()
	assert.NotNil(t, stats, "Expected stats map, got nil")

	expectedKeys := []string{"total_events", "unique_event_types", "current_threshold", "detected_anomalies"}
	for _, key := range expectedKeys {
		assert.Contains(t, stats, key, "Expected key %s in stats", key)
	}
}

func TestMachineLearningDetector(t *testing.T) {
	logger := slog.Default()
	mlDetector := NewMachineLearningDetector(logger)

	require.NotNil(t, mlDetector, "Failed to create ML detector")

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
	assert.NoError(t, err, "Failed to train model")

	testEvent := &Event{
		ID:        "test-001",
		Type:      "process",
		Timestamp: time.Now(),
		Source:    "test_process.exe",
		Details:   "Test process",
		Severity:  "medium",
	}

	_, _ = mlDetector.DetectAnomaly(testEvent)

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
	assert.NoError(t, err, "Failed to update model")
}
