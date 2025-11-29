package edr

import (
	"log/slog"
	"math"
)

// AnomalyDetector detects anomalies in system events using statistical analysis
type AnomalyDetector struct {
	logger      *slog.Logger
	eventCounts map[string]int // Track event frequencies
	totalEvents int            // Total number of events
	threshold   float64        // Threshold for anomaly detection
}

// NewAnomalyDetector creates a new anomaly detector
func NewAnomalyDetector(logger *slog.Logger) *AnomalyDetector {
	return &AnomalyDetector{
		logger:      logger,
		eventCounts: make(map[string]int),
		totalEvents: 0,
		threshold:   2.0, // 2 standard deviations from mean
	}
}

// AnalyzeEvent analyzes an event for anomalies
func (ad *AnomalyDetector) AnalyzeEvent(event *Event) bool {
	ad.totalEvents++
	
	// Update event frequency tracking
	key := ad.getEventKey(event)
	ad.eventCounts[key]++
	
	// Calculate probability of this event
	probability := float64(ad.eventCounts[key]) / float64(ad.totalEvents)
	
	// Calculate expected probability (uniform distribution)
	expectedProbability := 1.0 / float64(len(ad.eventCounts))
	
	// Calculate z-score
	zScore := math.Abs(probability-expectedProbability) / math.Sqrt(expectedProbability*(1-expectedProbability)/float64(ad.totalEvents))
	
	// Check if this is an anomaly
	isAnomaly := zScore > ad.threshold
	
	if isAnomaly {
		ad.logger.Warn("Anomaly detected", 
			"event_id", event.ID,
			"event_type", event.Type,
			"source", event.Source,
			"z_score", zScore,
			"probability", probability,
			"expected_probability", expectedProbability)
	}
	
	return isAnomaly
}

// getEventKey creates a unique key for an event for tracking purposes
func (ad *AnomalyDetector) getEventKey(event *Event) string {
	// Create a composite key based on event characteristics
	return event.Type + ":" + event.Source
}

// UpdateThreshold updates the anomaly detection threshold
func (ad *AnomalyDetector) UpdateThreshold(newThreshold float64) {
	ad.threshold = newThreshold
	ad.logger.Info("Updated anomaly detection threshold", "new_threshold", newThreshold)
}

// GetAnomalyStats returns statistics about detected anomalies
func (ad *AnomalyDetector) GetAnomalyStats() map[string]interface{} {
	stats := make(map[string]interface{})
	stats["total_events"] = ad.totalEvents
	stats["unique_event_types"] = len(ad.eventCounts)
	stats["current_threshold"] = ad.threshold
	
	// Count anomalies (simplified)
	anomalyCount := 0
	for _, count := range ad.eventCounts {
		if float64(count)/float64(ad.totalEvents) < 0.01 { // Less than 1% frequency
			anomalyCount++
		}
	}
	stats["detected_anomalies"] = anomalyCount
	
	return stats
}

// MachineLearningDetector represents a more advanced ML-based anomaly detector
type MachineLearningDetector struct {
	logger *slog.Logger
	// In a real implementation, this would contain:
	// - Trained ML models
	// - Feature extraction functions
	// - Model evaluation metrics
}

// NewMachineLearningDetector creates a new ML-based anomaly detector
func NewMachineLearningDetector(logger *slog.Logger) *MachineLearningDetector {
	return &MachineLearningDetector{
		logger: logger,
	}
}

// TrainModel trains the ML model on historical data
func (ml *MachineLearningDetector) TrainModel(trainingData []Event) error {
	// In a real implementation, this would:
	// 1. Extract features from the training data
	// 2. Train an ML model (e.g., isolation forest, autoencoder, etc.)
	// 3. Validate the model performance
	// 4. Save the trained model
	
	ml.logger.Info("Training ML model on historical data", "training_samples", len(trainingData))
	
	// For now, we'll just log the action
	ml.logger.Info("Would train ML model on provided data")
	
	return nil
}

// DetectAnomaly uses the trained ML model to detect anomalies
func (ml *MachineLearningDetector) DetectAnomaly(event *Event) (bool, float64) {
	// In a real implementation, this would:
	// 1. Extract features from the event
	// 2. Use the trained model to predict if it's an anomaly
	// 3. Return the prediction and confidence score
	
	// For now, we'll just return a random result for demonstration
	isAnomaly := false
	confidence := 0.0
	
	ml.logger.Debug("Using ML model to detect anomaly", 
		"event_id", event.ID,
		"is_anomaly", isAnomaly,
		"confidence", confidence)
	
	return isAnomaly, confidence
}

// UpdateModel updates the ML model with new data
func (ml *MachineLearningDetector) UpdateModel(newData []Event) error {
	// In a real implementation, this would:
	// 1. Incorporate new data into the model
	// 2. Retrain or fine-tune the model
	// 3. Evaluate updated model performance
	
	ml.logger.Info("Updating ML model with new data", "new_samples", len(newData))
	
	// For now, we'll just log the action
	ml.logger.Info("Would update ML model with new data")
	
	return nil
}