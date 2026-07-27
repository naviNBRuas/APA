package intelligence

import (
	"fmt"
	"log/slog"
	"math"
	"strings"
	"sync"
	"time"
)

type BehavioralAnalysisSystem struct {
	logger           *slog.Logger
	config           BehavioralConfig
	behaviorModels   map[BehaviorType]*BehaviorModel
	patternMatcher   *PatternMatchingEngine
	anomalyEngine    *BehavioralAnomalyEngine
	trendAnalyzer    *TrendAnalysisEngine
	clusteringEngine *ClusteringEngine

	mu               sync.RWMutex
	behaviorProfiles map[string]*BehaviorProfile
	analysisResults  []*BehaviorAnalysis
}

type BehaviorModel struct {
	Type        BehaviorType       `json:"type"`
	Model       interface{}        `json:"model"`
	Features    []string           `json:"features"`
	Baseline    *BehaviorBaseline  `json:"baseline"`
	Thresholds  BehaviorThresholds `json:"thresholds"`
	LastUpdated time.Time          `json:"last_updated"`
	Accuracy    float64            `json:"accuracy"`
}

type PatternMatchingEngine struct {
	logger           *slog.Logger
	patternLibrary   map[string]*PatternTemplate
	matchingEngine   *PatternMatchingAlgorithm
	similarityEngine *PatternSimilarityEngine
	evolutionTracker *PatternEvolutionTracker
	mu               sync.RWMutex
}

type BehavioralAnomalyEngine struct{}

type TrendAnalysisEngine struct{}

type ClusteringEngine struct{}

type BehaviorProfile struct {
	Entity      string            `json:"entity"`
	Type        BehaviorType      `json:"type"`
	Patterns    []BehaviorPattern `json:"patterns"`
	Baseline    *BehaviorBaseline `json:"baseline"`
	Deviations  []Deviation       `json:"deviations"`
	LastUpdated time.Time         `json:"last_updated"`
	Confidence  float64           `json:"confidence"`
	Anomalies   []AnomalyFlag     `json:"anomalies"`
}

type BehaviorAnalysis struct {
	ID              string            `json:"id"`
	Timestamp       time.Time         `json:"timestamp"`
	PatternsFound   int               `json:"patterns_found"`
	AnomaliesCount  int               `json:"anomalies_count"`
	ProfilesUpdated int               `json:"profiles_updated"`
	Insights        []string          `json:"insights"`
	Summary         string            `json:"summary"`
	Metrics         map[string]float64 `json:"metrics"`
}

func NewBehavioralAnalysisSystem(logger *slog.Logger, config BehavioralConfig) *BehavioralAnalysisSystem {
	return &BehavioralAnalysisSystem{
		logger:           logger,
		config:           config,
		behaviorModels:   make(map[BehaviorType]*BehaviorModel),
		patternMatcher:   NewPatternMatchingEngine(logger),
		anomalyEngine:    NewBehavioralAnomalyEngine(logger),
		trendAnalyzer:    NewTrendAnalysisEngine(logger),
		clusteringEngine: NewClusteringEngine(logger),
		behaviorProfiles: make(map[string]*BehaviorProfile),
		analysisResults:  make([]*BehaviorAnalysis, 0),
	}
}

func NewPatternMatchingEngine(logger *slog.Logger) *PatternMatchingEngine {
	return &PatternMatchingEngine{logger: logger, patternLibrary: make(map[string]*PatternTemplate)}
}

func NewBehavioralAnomalyEngine(logger *slog.Logger) *BehavioralAnomalyEngine {
	return &BehavioralAnomalyEngine{}
}

func NewTrendAnalysisEngine(logger *slog.Logger) *TrendAnalysisEngine {
	return &TrendAnalysisEngine{}
}

func NewClusteringEngine(logger *slog.Logger) *ClusteringEngine {
	return &ClusteringEngine{}
}

func (bas *BehavioralAnalysisSystem) Shutdown() {
	bas.mu.Lock()
	defer bas.mu.Unlock()
	bas.behaviorModels = nil
	bas.behaviorProfiles = nil
	bas.analysisResults = nil
}

func (bas *BehavioralAnalysisSystem) AnalyzePatterns(data interface{}) []interface{} {
	values := toFloatSlice(data)
	if len(values) < 2 {
		return make([]interface{}, 0)
	}

	bas.mu.Lock()
	defer bas.mu.Unlock()

	var patterns []interface{}

	for i := 1; i < len(values); i++ {
		diff := values[i] - values[i-1]
		if math.Abs(diff) > 0.001 {
			direction := "up"
			if diff < 0 {
				direction = "down"
			}
			magnitude := "small"
			ratio := math.Abs(diff) / (math.Abs(values[i-1]) + 0.001)
			switch {
			case ratio > 0.1:
				magnitude = "large"
			case ratio > 0.03:
				magnitude = "medium"
			}
			pattern := BehaviorPattern{
				ID:        fmt.Sprintf("pat-%d", i),
				Type:      PatternSequential,
				Sequence:  []interface{}{direction, magnitude, values[i-1], values[i]},
				Frequency: 1.0 / float64(len(values)),
				Importance: ratio,
			}
			patterns = append(patterns, pattern)
		}
	}

	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))

	recurring := make([]float64, 0)
	for i := 2; i <= len(values)/2 && i <= 10; i++ {
		matches := 0
		for j := 0; j < len(values)-i; j++ {
			if math.Abs(values[j]-values[j+i]) < 0.05*math.Max(math.Abs(values[j]), 0.001) {
				matches++
			}
		}
		if matches >= len(values)/4 {
			recurring = append(recurring, float64(i))
		}
	}
	if len(recurring) > 0 {
		patterns = append(patterns, BehaviorPattern{
			ID:        "recurring",
			Type:      PatternRecurring,
			Sequence:  []interface{}{recurring},
			Importance: float64(len(recurring)) / float64(len(values)/2),
		})
	}

	globalMean := mean
	anomalous := make([]float64, 0)
	for _, v := range values {
		if math.Abs(v-globalMean) > 2*globalMean && globalMean > 0 {
			anomalous = append(anomalous, v)
		}
	}
	if len(anomalous) > 0 {
		patterns = append(patterns, BehaviorPattern{
			ID:        "anomalous_values",
			Type:      PatternAnomalous,
			Sequence:  []interface{}{anomalous},
			Importance: float64(len(anomalous)) / float64(len(values)),
		})
	}

	analysis := &BehaviorAnalysis{
		ID:            fmt.Sprintf("ba-%d", time.Now().UnixNano()),
		Timestamp:     time.Now().UTC(),
		PatternsFound: len(patterns),
		Metrics: map[string]float64{
			"mean":     mean,
			"count":    float64(len(values)),
			"patterns": float64(len(patterns)),
		},
	}
	bas.analysisResults = append(bas.analysisResults, analysis)

	return patterns
}

func (bas *BehavioralAnalysisSystem) DetectAnomalies(patterns []interface{}) []interface{} {
	bas.mu.Lock()
	defer bas.mu.Unlock()

	anomalies := make([]interface{}, 0)
	threshold := 0.15

	for _, p := range patterns {
		if bp, ok := p.(BehaviorPattern); ok {
			if bp.Importance > threshold || bp.Type == PatternAnomalous {
				anomalies = append(anomalies, map[string]interface{}{
					"pattern_id": bp.ID,
					"type":       string(bp.Type),
					"importance": bp.Importance,
					"detected":   time.Now().UTC(),
				})
			}
		}
	}

	return anomalies
}

func (bas *BehavioralAnalysisSystem) UpdateProfiles(patterns []interface{}) {
	bas.mu.Lock()
	defer bas.mu.Unlock()

	for _, p := range patterns {
		if bp, ok := p.(BehaviorPattern); ok {
			entityKey := string(bp.Type)
			prof, found := bas.behaviorProfiles[entityKey]
			if !found {
				bType := BehaviorType(entityKey)
				prof = &BehaviorProfile{
					Entity:     entityKey,
					Type:       bType,
					Patterns:   make([]BehaviorPattern, 0),
					Confidence: 0.5,
				}
				bas.behaviorProfiles[entityKey] = prof
			}

			prof.Patterns = append(prof.Patterns, bp)
			prof.LastUpdated = time.Now().UTC()
			prof.Confidence = math.Min(1.0, prof.Confidence+0.05)

			if bp.Importance > 0.3 {
				prof.Deviations = append(prof.Deviations, Deviation{})
			}
		}
	}
}

func (bas *BehavioralAnalysisSystem) GenerateInsights(patterns []interface{}, anomalies []interface{}) []interface{} {
	bas.mu.RLock()
	defer bas.mu.RUnlock()

	var insights []interface{}

	patternCount := len(patterns)
	anomalyCount := len(anomalies)

	insights = append(insights, map[string]interface{}{
		"type":    "summary",
		"message": fmt.Sprintf("Analyzed %d patterns, detected %d anomalies", patternCount, anomalyCount),
	})

	if anomalyCount > 0 {
		insights = append(insights, map[string]interface{}{
			"type":    "alert",
			"message": fmt.Sprintf("Found %d anomalous patterns requiring attention", anomalyCount),
			"severity": "medium",
		})
	}

	for entity, profile := range bas.behaviorProfiles {
		if len(profile.Patterns) > 5 && float64(len(profile.Deviations))/float64(len(profile.Patterns)) > 0.2 {
			insights = append(insights, map[string]interface{}{
				"type":    "trend",
				"entity":  entity,
				"message": fmt.Sprintf("High deviation rate (%.0f%%) in %s behaviors", float64(len(profile.Deviations))/float64(len(profile.Patterns))*100, entity),
			})
		}
	}

	return insights
}

func (bas *BehavioralAnalysisSystem) GetProfile(entity string) *BehaviorProfile {
	bas.mu.RLock()
	defer bas.mu.RUnlock()
	profile, ok := bas.behaviorProfiles[entity]
	if !ok {
		return nil
	}
	return profile
}

func (bas *BehavioralAnalysisSystem) GetAnalysisHistory(limit int) []*BehaviorAnalysis {
	bas.mu.RLock()
	defer bas.mu.RUnlock()
	if limit <= 0 || limit > len(bas.analysisResults) {
		limit = len(bas.analysisResults)
	}
	result := make([]*BehaviorAnalysis, limit)
	for i := 0; i < limit; i++ {
		result[i] = bas.analysisResults[len(bas.analysisResults)-1-i]
	}
	return result
}

func (pm *PatternMatchingEngine) RegisterPattern(name string, template *PatternTemplate) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.patternLibrary[name] = template
}

func (pm *PatternMatchingEngine) Match(input []float64) []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	if len(input) < 2 {
		return nil
	}
	matches := make([]string, 0)
	trend := input[len(input)-1] - input[0]
	for name := range pm.patternLibrary {
		lower := strings.ToLower(name)
		switch {
		case strings.Contains(lower, "up") || strings.Contains(lower, "increas"):
			if trend > 0 {
				matches = append(matches, name)
			}
		case strings.Contains(lower, "down") || strings.Contains(lower, "decreas"):
			if trend < 0 {
				matches = append(matches, name)
			}
		default:
			matches = append(matches, name)
		}
	}
	return matches
}
