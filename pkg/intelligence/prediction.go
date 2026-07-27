package intelligence

import (
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"sync"
	"time"
)

type PredictiveAnalyticsEngine struct {
	logger            *slog.Logger
	config            PredictiveConfig
	forecastingModels map[ForecastType]*ForecastingModel
	timeSeriesEngine  *TimeSeriesEngine
	patternRecognizer *PatternRecognitionEngine
	confidenceEngine  *ConfidenceAssessmentEngine
	scenarioGenerator *ScenarioGenerator

	mu               sync.RWMutex
	predictions      []*Prediction
	forecastAccuracy map[ForecastType]*AccuracyMetrics
}

type ForecastingModel struct {
	Type         ForecastType     `json:"type"`
	Model        *TimeSeriesModel `json:"model"`
	Features     []string         `json:"features"`
	Horizon      int              `json:"horizon"`
	Confidence   float64          `json:"confidence"`
	LastForecast time.Time        `json:"last_forecast"`
	Accuracy     *AccuracyMetrics `json:"accuracy"`
}

type TimeSeriesEngine struct {
	logger              *slog.Logger
	decompositionEngine *TimeSeriesDecompositionEngine
	stationarityEngine  *StationarityEngine
	forecastingEngine   *ForecastingEngine
	anomalyDetection    *TimeSeriesAnomalyDetector

	mu sync.RWMutex
}

type PatternRecognitionEngine struct {
	logger           *slog.Logger
	patternMatchers  map[PatternType]*PatternMatcher
	sequenceAnalyzer *SequenceAnalyzer
	similarityEngine *SimilarityEngine
	clusteringEngine *PatternClusteringEngine

	mu sync.RWMutex
}

type ConfidenceAssessmentEngine struct{}

type ScenarioGenerator struct{}

type Prediction struct {
	ID         string          `json:"id"`
	Timestamp  time.Time       `json:"timestamp"`
	Type       ForecastType    `json:"type"`
	Target     string          `json:"target"`
	Forecast   []ForecastValue `json:"forecast"`
	Confidence float64         `json:"confidence"`
	Method     string          `json:"method"`
	Horizon    int             `json:"horizon"`
	Actual     interface{}     `json:"actual,omitempty"`
	Error      float64         `json:"error,omitempty"`
}

type AccuracyMetrics struct {
	MAE            float64   `json:"mae"`
	RMSE           float64   `json:"rmse"`
	MAPE           float64   `json:"mape"`
	RSquared       float64   `json:"r_squared"`
	LastCalculated time.Time `json:"last_calculated"`
	SampleSize     int       `json:"sample_size"`
	Confidence     float64   `json:"confidence"`
}

func NewPredictiveAnalyticsEngine(logger *slog.Logger, config PredictiveConfig) *PredictiveAnalyticsEngine {
	return &PredictiveAnalyticsEngine{
		logger:            logger,
		config:            config,
		forecastingModels: make(map[ForecastType]*ForecastingModel),
		timeSeriesEngine:  NewTimeSeriesEngine(logger),
		patternRecognizer: NewPatternRecognitionEngine(logger),
		confidenceEngine:  NewConfidenceAssessmentEngine(logger),
		scenarioGenerator: NewScenarioGenerator(logger),
		predictions:       make([]*Prediction, 0),
		forecastAccuracy:  make(map[ForecastType]*AccuracyMetrics),
	}
}

func NewTimeSeriesEngine(logger *slog.Logger) *TimeSeriesEngine {
	return &TimeSeriesEngine{logger: logger}
}

func NewPatternRecognitionEngine(logger *slog.Logger) *PatternRecognitionEngine {
	return &PatternRecognitionEngine{logger: logger, patternMatchers: make(map[PatternType]*PatternMatcher)}
}

func NewConfidenceAssessmentEngine(logger *slog.Logger) *ConfidenceAssessmentEngine {
	return &ConfidenceAssessmentEngine{}
}

func NewScenarioGenerator(logger *slog.Logger) *ScenarioGenerator {
	return &ScenarioGenerator{}
}

func (pae *PredictiveAnalyticsEngine) Shutdown() {
	pae.mu.Lock()
	defer pae.mu.Unlock()
	pae.predictions = nil
	pae.forecastAccuracy = nil
	pae.forecastingModels = nil
}

func (pae *PredictiveAnalyticsEngine) GenerateForecast(fType ForecastType, data interface{}) *Prediction {
	pae.mu.Lock()
	defer pae.mu.Unlock()

	values := toFloatSlice(data)
	n := len(values)

	prediction := &Prediction{
		ID:        fmt.Sprintf("pred-%d", time.Now().UnixNano()),
		Timestamp: time.Now().UTC(),
		Type:      fType,
		Target:    string(fType),
		Horizon:   10,
		Method:    "exponential_smoothing",
		Forecast:  make([]ForecastValue, 0, 10),
	}

	if n == 0 {
		prediction.Confidence = 0
		pae.predictions = append(pae.predictions, prediction)
		return prediction
	}

	alpha := 0.3
	var smoothed float64
	if n > 0 {
		smoothed = values[0]
	}
	for i := 1; i < n; i++ {
		smoothed = alpha*values[i] + (1-alpha)*smoothed
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

	trend := 0.0
	if n >= 2 {
		trend = (values[n-1] - values[0]) / float64(n)
	}

	now := time.Now().UTC()
	for i := 0; i < 10; i++ {
		ts := now.Add(time.Duration(i+1) * time.Hour)
		base := smoothed + trend*float64(i+1)
		upper := base + 1.96*stddev*math.Sqrt(float64(i+1))
		lower := base - 1.96*stddev*math.Sqrt(float64(i+1))

		prediction.Forecast = append(prediction.Forecast, ForecastValue{
			Timestamp:  ts,
			Value:      base,
			LowerBound: lower,
			UpperBound: upper,
			Confidence: 0.95 - float64(i)*0.02,
		})
	}

	sampleWeight := math.Min(float64(n)/100.0, 1.0)
	varConfidence := 1.0 - math.Min(stddev/(math.Abs(mean)+0.001), 0.5)
	prediction.Confidence = sampleWeight * varConfidence

	pae.predictions = append(pae.predictions, prediction)

	model, ok := pae.forecastingModels[fType]
	if !ok {
		model = &ForecastingModel{
			Type:       fType,
			Confidence: prediction.Confidence,
			Horizon:    10,
			Features:   []string{"historical_values"},
		}
		pae.forecastingModels[fType] = model
	}
	model.LastForecast = now
	model.Confidence = prediction.Confidence

	metrics := &AccuracyMetrics{
		LastCalculated: now,
		SampleSize:     n,
		Confidence:     prediction.Confidence,
	}
	if n >= 2 {
		sse := 0.0
		sst := 0.0
		for i := 1; i < n; i++ {
			err := values[i] - smoothed
			sse += err * err
			sst += (values[i] - mean) * (values[i] - mean)
		}
		metrics.MAE = math.Sqrt(sse / float64(n-1))
		metrics.RMSE = math.Sqrt(sse / float64(n-1))
		if sst > 0 {
			metrics.RSquared = 1.0 - sse/sst
		}
	}
	pae.forecastAccuracy[fType] = metrics

	return prediction
}

func (pae *PredictiveAnalyticsEngine) GenerateScenario(fType ForecastType, data interface{}, scenarios []string) map[string]*Prediction {
	base := pae.GenerateForecast(fType, data)
	if base == nil {
		return nil
	}

	result := make(map[string]*Prediction, len(scenarios))
	for _, name := range scenarios {
		p := *base
		p.ID = fmt.Sprintf("%s-%s", base.ID, name)
		p.Method = fmt.Sprintf("scenario_%s", name)

		var multiplier float64
		switch name {
		case "optimistic", "best_case", "aggressive":
			multiplier = 1.2
		case "pessimistic", "worst_case", "conservative":
			multiplier = 0.8
		default:
			multiplier = 1.0
		}

		p.Forecast = make([]ForecastValue, len(base.Forecast))
		for i, fv := range base.Forecast {
			val := toFloat64(fv.Value) * multiplier
			lb := toFloat64(fv.LowerBound) * multiplier
			ub := toFloat64(fv.UpperBound) * multiplier
			p.Forecast[i] = ForecastValue{
				Timestamp:  fv.Timestamp,
				Value:      val,
				LowerBound: lb,
				UpperBound: ub,
				Confidence: fv.Confidence * (1.0 - math.Abs(multiplier-1.0)*0.5),
			}
		}
		result[name] = &p
	}
	return result
}

func (pae *PredictiveAnalyticsEngine) RecordActual(fType ForecastType, actual interface{}) {
	pae.mu.Lock()
	defer pae.mu.Unlock()

	actualVal := toFloat64(actual)
	var bestPrediction *Prediction
	for _, p := range pae.predictions {
		if p.Type == fType && p.Actual == nil {
			bestPrediction = p
			break
		}
	}
	if bestPrediction == nil {
		return
	}

	bestPrediction.Actual = actualVal
	if len(bestPrediction.Forecast) > 0 {
		forecastVal := toFloat64(bestPrediction.Forecast[0].Value)
		bestPrediction.Error = math.Abs(forecastVal - actualVal)

		metrics := pae.forecastAccuracy[fType]
		if metrics == nil {
			metrics = &AccuracyMetrics{}
			pae.forecastAccuracy[fType] = metrics
		}
		metrics.SampleSize++
		metrics.MAE = (metrics.MAE*float64(metrics.SampleSize-1) + bestPrediction.Error) / float64(metrics.SampleSize)
		metrics.RMSE = math.Sqrt((metrics.RMSE*metrics.RMSE*float64(metrics.SampleSize-1) + bestPrediction.Error*bestPrediction.Error) / float64(metrics.SampleSize))
		metrics.LastCalculated = time.Now().UTC()
		if metrics.SampleSize >= 3 {
			pae.forecastingModels[fType].Accuracy = metrics
		}
	}
}

func (pae *PredictiveAnalyticsEngine) GetForecastAccuracy(fType ForecastType) *AccuracyMetrics {
	pae.mu.RLock()
	defer pae.mu.RUnlock()
	metrics, ok := pae.forecastAccuracy[fType]
	if !ok {
		return nil
	}
	m := *metrics
	return &m
}

func (pae *PredictiveAnalyticsEngine) GetModel(fType ForecastType) *ForecastingModel {
	pae.mu.RLock()
	defer pae.mu.RUnlock()
	model, ok := pae.forecastingModels[fType]
	if !ok {
		return nil
	}
	return model
}

func toFloatSlice(data interface{}) []float64 {
	switch v := data.(type) {
	case []float64:
		return v
	case []int:
		vals := make([]float64, len(v))
		for i, x := range v {
			vals[i] = float64(x)
		}
		return vals
	case []float32:
		vals := make([]float64, len(v))
		for i, x := range v {
			vals[i] = float64(x)
		}
		return vals
	default:
		return nil
	}
}

func toFloat64(val interface{}) float64 {
	switch v := val.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	}
	return 0
}

func (pae *PredictiveAnalyticsEngine) DetectTrend(data interface{}) string {
	values := toFloatSlice(data)
	if len(values) < 2 {
		return "insufficient_data"
	}

	first := values[0]
	last := values[len(values)-1]
	slope := (last - first) / float64(len(values))

	variance := 0.0
	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values))
	stddev := math.Sqrt(variance)

	if stddev == 0 {
		return "stable"
	}

	if slope > 0.1*stddev {
		return "increasing"
	} else if slope < -0.1*stddev {
		return "decreasing"
	}
	return "stable"
}

func NewTimeSeriesAnomalyDetector(logger *slog.Logger) *TimeSeriesAnomalyDetector {
	return &TimeSeriesAnomalyDetector{}
}

func (ts *TimeSeriesEngine) DetectAnomalies(data interface{}) []float64 {
	values := toFloatSlice(data)
	if len(values) < 3 {
		return nil
	}

	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))

	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values))
	stddev := math.Sqrt(variance)

	var anomalies []float64
	for _, v := range values {
		if stddev > 0 && math.Abs(v-mean)/stddev > 2.0 {
			anomalies = append(anomalies, v)
		}
	}
	return anomalies
}

func (pr *PatternRecognitionEngine) FindPatterns(data interface{}, patternType PatternType) []interface{} {
	values := toFloatSlice(data)
	if len(values) < 2 {
		return nil
	}

	pr.mu.Lock()
	defer pr.mu.Unlock()

	var patterns []interface{}
	switch patternType {
	case PatternSequential:
		for i := 0; i < len(values)-1; i++ {
			if values[i] < values[i+1] {
				patterns = append(patterns, fmt.Sprintf("increase at %d->%d: %.2f->%.2f", i, i+1, values[i], values[i+1]))
			} else if values[i] > values[i+1] {
				patterns = append(patterns, fmt.Sprintf("decrease at %d->%d: %.2f->%.2f", i, i+1, values[i], values[i+1]))
			}
		}
	case PatternRecurring:
		if len(values) >= 4 {
			for period := 2; period <= len(values)/2; period++ {
				match := true
				for i := 0; i < len(values)-period; i++ {
					if math.Abs(values[i]-values[i+period]) > 0.01*math.Max(math.Abs(values[i]), 0.001) {
						match = false
						break
					}
				}
				if match {
					patterns = append(patterns, fmt.Sprintf("periodic_pattern period=%d", period))
				}
			}
		}
	default:
		pat := &BehaviorPattern{
			Type:     patternType,
			Sequence: []interface{}{patternType},
		}
		patterns = append(patterns, pat)
	}
	return patterns
}

func (pa *PatternRecognitionEngine) MatchPattern(sequence []interface{}, patternType PatternType) float64 {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	if len(sequence) < 2 {
		return 0
	}
	return float64(len(sequence)) / float64(len(sequence)+5) * rand.Float64()
}

func DetectSeasonality(data interface{}) (int, float64) {
	values := toFloatSlice(data)
	if len(values) < 4 {
		return 1, 0
	}

	bestPeriod := 1
	bestScore := 0.0

	for period := 2; period <= len(values)/2; period++ {
		score := 0.0
		count := 0
		for i := 0; i < len(values)-period; i++ {
			diff := math.Abs(values[i] - values[i+period])
			maxAbs := math.Max(math.Abs(values[i]), 0.001)
			score += 1.0 - math.Min(diff/maxAbs, 1.0)
			count++
		}
		if count > 0 {
			avg := score / float64(count)
			if avg > bestScore {
				bestScore = avg
				bestPeriod = period
			}
		}
	}
	if bestScore < 0.5 {
		return 1, 0
	}
	return bestPeriod, bestScore
}
