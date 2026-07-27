package intelligence

import (
	"fmt"
	"log/slog"
	"math"
	"sort"
	"sync"
	"time"
)

type AdvancedAnomalyDetector struct {
	logger        *slog.Logger
	config        AnomalyDetectionConfig
	detectors     map[AnomalyType]*AnomalyDetector
	fusionEngine  *AnomalyFusionEngine
	contextEngine *AnomalyContextEngine
	alertSystem   *AnomalyAlertSystem

	mu             sync.RWMutex
	anomalies      []*DetectedAnomaly
	detectionRates map[AnomalyType]*DetectionMetrics
}

type AnomalyDetector struct {
	Type         AnomalyType            `json:"type"`
	Algorithm    AnomalyAlgorithm       `json:"algorithm"`
	Parameters   map[string]interface{} `json:"parameters"`
	Threshold    float64                `json:"threshold"`
	TrainingData []AnomalySample        `json:"training_data"`
	Performance  *DetectionMetrics      `json:"performance"`
	LastUpdated  time.Time              `json:"last_updated"`
}

type AnomalyFusionEngine struct {
	logger           *slog.Logger
	fusionMethods    []FusionMethod
	weightEngine     *FusionWeightEngine
	confidenceEngine *FusionConfidenceEngine
	decisionEngine   *FusionDecisionEngine
	mu               sync.RWMutex
}

type AnomalyContextEngine struct{}

type AnomalyAlertSystem struct{}

type DetectedAnomaly struct {
	ID              string                 `json:"id"`
	Timestamp       time.Time              `json:"timestamp"`
	Type            AnomalyType            `json:"type"`
	Entity          string                 `json:"entity"`
	Severity        AnomalySeverity        `json:"severity"`
	Confidence      float64                `json:"confidence"`
	Description     string                 `json:"description"`
	Context         map[string]interface{} `json:"context"`
	Evidence        []Evidence             `json:"evidence"`
	Impact          ImpactAssessment       `json:"impact"`
	Recommendations []string               `json:"recommendations"`
	Resolved        bool                   `json:"resolved"`
	Resolution      *Resolution            `json:"resolution,omitempty"`
}

type DetectionMetrics struct {
	TotalDetections   int       `json:"total_detections"`
	TruePositives     int       `json:"true_positives"`
	FalsePositives    int       `json:"false_positives"`
	Precision         float64   `json:"precision"`
	Recall            float64   `json:"recall"`
	F1Score           float64   `json:"f1_score"`
	AvgDetectionTime  float64   `json:"avg_detection_time_ms"`
	LastUpdated       time.Time `json:"last_updated"`
}

func NewAdvancedAnomalyDetector(logger *slog.Logger, config AnomalyDetectionConfig) *AdvancedAnomalyDetector {
	return &AdvancedAnomalyDetector{
		logger:         logger,
		config:         config,
		detectors:      make(map[AnomalyType]*AnomalyDetector),
		fusionEngine:   NewAnomalyFusionEngine(logger),
		contextEngine:  NewAnomalyContextEngine(logger),
		alertSystem:    NewAnomalyAlertSystem(logger),
		anomalies:      make([]*DetectedAnomaly, 0),
		detectionRates: make(map[AnomalyType]*DetectionMetrics),
	}
}

func NewAnomalyFusionEngine(logger *slog.Logger) *AnomalyFusionEngine {
	return &AnomalyFusionEngine{logger: logger}
}

func NewAnomalyContextEngine(logger *slog.Logger) *AnomalyContextEngine {
	return &AnomalyContextEngine{}
}

func NewAnomalyAlertSystem(logger *slog.Logger) *AnomalyAlertSystem {
	return &AnomalyAlertSystem{}
}

func (aad *AdvancedAnomalyDetector) Shutdown() {
	aad.mu.Lock()
	defer aad.mu.Unlock()
	aad.detectors = nil
	aad.anomalies = nil
	aad.detectionRates = nil
}

func (aad *AdvancedAnomalyDetector) DetectMultiple(data interface{}) []*DetectedAnomaly {
	aad.mu.Lock()
	defer aad.mu.Unlock()

	values := toFloatSlice(data)
	if len(values) < 3 {
		return make([]*DetectedAnomaly, 0)
	}

	anomalies := make([]*DetectedAnomaly, 0)

	zScoreAnomalies := aad.detectZScore(values)
	iqrAnomalies := aad.detectIQR(values)
	trendAnomalies := aad.detectTrendShift(values)

	anomalies = append(anomalies, zScoreAnomalies...)
	anomalies = append(anomalies, iqrAnomalies...)
	anomalies = append(anomalies, trendAnomalies...)

	anomalies = aad.fusionEngine.Fuse(anomalies)

	aad.anomalies = append(aad.anomalies, anomalies...)

	metrics := aad.detectionRates[AnomalyStatistical]
	if metrics == nil {
		metrics = &DetectionMetrics{}
		aad.detectionRates[AnomalyStatistical] = metrics
	}
	metrics.TotalDetections += len(anomalies)
	metrics.LastUpdated = time.Now().UTC()
	if len(aad.anomalies) > 0 {
		metrics.Precision = float64(metrics.TruePositives) / float64(metrics.TotalDetections)
	}

	return anomalies
}

func (aad *AdvancedAnomalyDetector) detectZScore(values []float64) []*DetectedAnomaly {
	n := len(values)
	if n < 3 {
		return nil
	}

	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(n)

	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(n)
	stddev := math.Sqrt(variance)

	if stddev == 0 {
		return nil
	}

	var anomalies []*DetectedAnomaly
	for i, v := range values {
		zScore := math.Abs(v-mean) / stddev
		if zScore > 2.0 {
			severity := AnomalySeverityLow
			switch {
			case zScore > 4.0:
				severity = AnomalySeverityCritical
			case zScore > 3.0:
				severity = AnomalySeverityHigh
			case zScore > 2.5:
				severity = AnomalySeverityMedium
			}

			anomalies = append(anomalies, &DetectedAnomaly{
				ID:          fmt.Sprintf("zscore-%s-%d", time.Now().UTC().Format("150405"), i),
				Timestamp:   time.Now().UTC(),
				Type:        AnomalyStatistical,
				Entity:      "timeseries",
				Severity:    severity,
				Confidence:  math.Min(1.0, (zScore-2.0)/3.0),
				Description: fmt.Sprintf("Z-score anomaly: value=%.4f, mean=%.4f, stddev=%.4f, z=%.2f", v, mean, stddev, zScore),
				Context: map[string]interface{}{
					"index":   i,
					"value":   v,
					"z_score": zScore,
					"method":  "z_score",
				},
				Recommendations: []string{
					"Investigate data point for root cause",
					"Verify sensor or data source integrity",
				},
			})
		}
	}
	return anomalies
}

func (aad *AdvancedAnomalyDetector) detectIQR(values []float64) []*DetectedAnomaly {
	n := len(values)
	if n < 4 {
		return nil
	}

	sorted := make([]float64, n)
	copy(sorted, values)
	sort.Float64s(sorted)

	q1 := sorted[n/4]
	q3 := sorted[3*n/4]
	iqr := q3 - q1

	lower := q1 - 1.5*iqr
	upper := q3 + 1.5*iqr

	var anomalies []*DetectedAnomaly
	for i, v := range values {
		if v < lower || v > upper {
			anomalies = append(anomalies, &DetectedAnomaly{
				ID:          fmt.Sprintf("iqr-%s-%d", time.Now().UTC().Format("150405"), i),
				Timestamp:   time.Now().UTC(),
				Type:        AnomalyStatistical,
				Entity:      "timeseries",
				Severity:    AnomalySeverityMedium,
				Confidence:  0.8,
				Description: fmt.Sprintf("IQR outlier: value=%.4f outside [%.4f, %.4f]", v, lower, upper),
				Context: map[string]interface{}{
					"index": i,
					"value": v,
					"q1":    q1,
					"q3":    q3,
					"iqr":   iqr,
					"lower": lower,
					"upper": upper,
				},
				Recommendations: []string{
					"Review data point in context of peers",
				},
			})
		}
	}
	return anomalies
}

func (aad *AdvancedAnomalyDetector) detectTrendShift(values []float64) []*DetectedAnomaly {
	n := len(values)
	if n < 6 {
		return nil
	}

	half := n / 2
	firstHalf := values[:half]
	secondHalf := values[half:]

	mean1 := 0.0
	for _, v := range firstHalf {
		mean1 += v
	}
	mean1 /= float64(len(firstHalf))

	mean2 := 0.0
	for _, v := range secondHalf {
		mean2 += v
	}
	mean2 /= float64(len(secondHalf))

	diff := math.Abs(mean2 - mean1)
	scale := math.Max(math.Abs(mean1), 0.001)
	if diff/scale > 0.2 {
		direction := "upward"
		if mean2 < mean1 {
			direction = "downward"
		}
		return []*DetectedAnomaly{
			{
				ID:          fmt.Sprintf("trend-%s", time.Now().UTC().Format("150405")),
				Timestamp:   time.Now().UTC(),
				Type:        AnomalyContextual,
				Entity:      "timeseries",
				Severity:    AnomalySeverityHigh,
				Confidence:  math.Min(1.0, diff/scale),
				Description: fmt.Sprintf("Trend shift detected: %s shift of %.2f%%", direction, diff/scale*100),
				Context: map[string]interface{}{
					"first_half_mean":  mean1,
					"second_half_mean": mean2,
					"shift_pct":        diff / scale * 100,
					"direction":        direction,
				},
				Recommendations: []string{
					"Recalibrate models for new regime",
					"Review external factors for change cause",
				},
			},
		}
	}
	return nil
}

func (aad *AdvancedAnomalyDetector) UpdateModels(anomalies []*DetectedAnomaly) {
	aad.mu.Lock()
	defer aad.mu.Unlock()

	for _, anomaly := range anomalies {
		detector, ok := aad.detectors[anomaly.Type]
		if !ok {
			detector = &AnomalyDetector{
				Type:      anomaly.Type,
				Threshold: 0.5,
				Parameters: map[string]interface{}{
					"confidence": anomaly.Confidence,
					"severity":   string(anomaly.Severity),
				},
			}
			aad.detectors[anomaly.Type] = detector
		}
		detector.LastUpdated = time.Now().UTC()

		metrics := aad.detectionRates[anomaly.Type]
		if metrics == nil {
			metrics = &DetectionMetrics{}
			aad.detectionRates[anomaly.Type] = metrics
		}
		metrics.TotalDetections++
		metrics.LastUpdated = time.Now().UTC()
	}
}

func (aad *AdvancedAnomalyDetector) GetAnomalyHistory(anomalyType AnomalyType, limit int) []*DetectedAnomaly {
	aad.mu.RLock()
	defer aad.mu.RUnlock()

	var filtered []*DetectedAnomaly
	for _, a := range aad.anomalies {
		if a.Type == anomalyType {
			filtered = append(filtered, a)
		}
	}

	if limit <= 0 || limit > len(filtered) {
		limit = len(filtered)
	}
	return filtered[len(filtered)-limit:]
}

func (aad *AdvancedAnomalyDetector) GetMetrics(anomalyType AnomalyType) *DetectionMetrics {
	aad.mu.RLock()
	defer aad.mu.RUnlock()
	metrics, ok := aad.detectionRates[anomalyType]
	if !ok {
		return nil
	}
	return metrics
}

func (aad *AdvancedAnomalyDetector) ResolveAnomaly(anomalyID string) {
	aad.mu.Lock()
	defer aad.mu.Unlock()
	for _, a := range aad.anomalies {
		if a.ID == anomalyID {
			a.Resolved = true
			a.Resolution = &Resolution{}
			break
		}
	}
}

func (afe *AnomalyFusionEngine) Fuse(anomalies []*DetectedAnomaly) []*DetectedAnomaly {
	afe.mu.Lock()
	defer afe.mu.Unlock()

	if len(anomalies) <= 1 {
		return anomalies
	}

	type anomalyKey struct {
		entity string
		desc   string
	}
	merged := make(map[anomalyKey]*DetectedAnomaly)
	order := make([]anomalyKey, 0)

	for _, a := range anomalies {
		key := anomalyKey{entity: a.Entity, desc: a.Description}
		existing, ok := merged[key]
		if !ok {
			merged[key] = a
			order = append(order, key)
		} else {
			if a.Confidence > existing.Confidence {
				existing.Confidence = a.Confidence
			}
			if severityScore(a.Severity) > severityScore(existing.Severity) {
				existing.Severity = a.Severity
			}
			existing.Evidence = append(existing.Evidence, a.Evidence...)
		}
	}

	result := make([]*DetectedAnomaly, 0, len(order))
	for _, key := range order {
		result = append(result, merged[key])
	}
	return result
}

func (afe *AnomalyFusionEngine) AddFusionMethod(method FusionMethod) {
	afe.mu.Lock()
	defer afe.mu.Unlock()
	afe.fusionMethods = append(afe.fusionMethods, method)
}

func severityScore(s AnomalySeverity) int {
	switch s {
	case AnomalySeverityCritical:
		return 4
	case AnomalySeverityHigh:
		return 3
	case AnomalySeverityMedium:
		return 2
	case AnomalySeverityLow:
		return 1
	default:
		return 0
	}
}
