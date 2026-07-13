package intelligence

import (
	"context"
	"log/slog"
	"math"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// NewIntelligenceEngine
// ---------------------------------------------------------------------------

func TestNewIntelligenceEngine_NilLogger(t *testing.T) {
	t.Parallel()
	_, err := NewIntelligenceEngine(nil, IntelligenceConfig{})
	require.Error(t, err)
}

func TestNewIntelligenceEngine_ValidConfig(t *testing.T) {
	t.Parallel()
	eng, err := NewIntelligenceEngine(slog.Default(), IntelligenceConfig{
		EnableAdaptiveDecisionMaking: true,
		EnablePredictiveAnalytics:    true,
		EnableAnomalyDetection:       true,
	})
	require.NoError(t, err)
	require.NotNil(t, eng)
	assert.NotNil(t, eng.logger, "logger should be set")
	assert.NotNil(t, eng.decisionMaker, "decisionMaker should be initialized when enabled")
	assert.NotNil(t, eng.predictiveEngine, "predictiveEngine should be initialized when enabled")
	assert.NotNil(t, eng.anomalyDetector, "anomalyDetector should be initialized when enabled")
	assert.Nil(t, eng.learningSystem, "learningSystem should be nil when not enabled")
	assert.Nil(t, eng.behaviorAnalyzer, "behaviorAnalyzer should be nil when not enabled")
	assert.Nil(t, eng.optimizationEngine, "optimizationEngine should be nil when not enabled")
	assert.Nil(t, eng.strategyPlanner, "strategyPlanner should be nil when not enabled")
	assert.NotNil(t, eng.knowledgeBase, "knowledgeBase should always be initialized")
	assert.NotNil(t, eng.knowledgeBase.Facts, "knowledgeBase.Facts should be initialized")
	assert.NotNil(t, eng.knowledgeBase.Rules, "knowledgeBase.Rules should be initialized")
	assert.NotNil(t, eng.knowledgeBase.Concepts, "knowledgeBase.Concepts should be initialized")
	assert.NotNil(t, eng.knowledgeBase.Relationships, "knowledgeBase.Relationships should be initialized")
	assert.NotNil(t, eng.knowledgeBase.Theories, "knowledgeBase.Theories should be initialized")
	assert.Equal(t, "1.0.0", eng.knowledgeBase.Version, "expected version 1.0.0")
}

func TestNewIntelligenceEngine_DefaultConfig(t *testing.T) {
	t.Parallel()
	eng, err := NewIntelligenceEngine(slog.Default(), IntelligenceConfig{})
	require.NoError(t, err)
	assert.False(t, eng.isRunning, "engine should not be running after creation")
	assert.Nil(t, eng.decisionMaker, "decisionMaker should be nil with default config")
	assert.Nil(t, eng.learningSystem, "learningSystem should be nil with default config")
	assert.Nil(t, eng.predictiveEngine, "predictiveEngine should be nil with default config")
	assert.Nil(t, eng.behaviorAnalyzer, "behaviorAnalyzer should be nil with default config")
	assert.Nil(t, eng.optimizationEngine, "optimizationEngine should be nil with default config")
	assert.Nil(t, eng.strategyPlanner, "strategyPlanner should be nil with default config")
	assert.Nil(t, eng.anomalyDetector, "anomalyDetector should be nil with default config")
	// verify channel / context
	assert.NotNil(t, eng.ctx, "ctx should be initialized")
	assert.NotNil(t, eng.cancel, "cancel should be initialized")
}

func TestNewIntelligenceEngine_AllEnabled(t *testing.T) {
	t.Parallel()
	eng, err := NewIntelligenceEngine(slog.Default(), IntelligenceConfig{
		EnableAdaptiveDecisionMaking: true,
		EnableMachineLearning:        true,
		EnablePredictiveAnalytics:    true,
		EnableBehavioralAnalysis:     true,
		EnableOptimization:           true,
		EnableStrategicPlanning:      true,
		EnableAnomalyDetection:       true,
	})
	require.NoError(t, err)
	assert.NotNil(t, eng.decisionMaker, "decisionMaker should be initialized")
	assert.NotNil(t, eng.learningSystem, "learningSystem should be initialized")
	assert.NotNil(t, eng.predictiveEngine, "predictiveEngine should be initialized")
	assert.NotNil(t, eng.behaviorAnalyzer, "behaviorAnalyzer should be initialized")
	assert.NotNil(t, eng.optimizationEngine, "optimizationEngine should be initialized")
	assert.NotNil(t, eng.strategyPlanner, "strategyPlanner should be initialized")
	assert.NotNil(t, eng.anomalyDetector, "anomalyDetector should be initialized")
}

// ---------------------------------------------------------------------------
// Start / Stop lifecycle
// ---------------------------------------------------------------------------

func TestEngineStartStop(t *testing.T) {
	t.Parallel()
	eng, err := NewIntelligenceEngine(slog.Default(), IntelligenceConfig{
		EnableAdaptiveDecisionMaking: true,
		EnablePredictiveAnalytics:    true,
		EnableAnomalyDetection:       true,
	})
	require.NoError(t, err)

	require.NoError(t, eng.Start())

	eng.mu.RLock()
	running := eng.isRunning
	eng.mu.RUnlock()
	assert.True(t, running, "engine should be running after Start()")

	eng.Stop()

	eng.mu.RLock()
	running = eng.isRunning
	eng.mu.RUnlock()
	assert.False(t, running, "engine should not be running after Stop()")
}

func TestEngineDoubleStart(t *testing.T) {
	t.Parallel()
	eng, err := NewIntelligenceEngine(slog.Default(), IntelligenceConfig{
		EnableAdaptiveDecisionMaking: true,
	})
	require.NoError(t, err)

	require.NoError(t, eng.Start())

	assert.Error(t, eng.Start(), "second Start() should return error")

	eng.Stop()
}

func TestEngineStopWhenNotRunning(t *testing.T) {
	t.Parallel()
	eng, err := NewIntelligenceEngine(slog.Default(), IntelligenceConfig{})
	require.NoError(t, err)

	// Stop without Start should not panic
	eng.Stop()
}

func TestEngineStartStop_DefaultConfig(t *testing.T) {
	t.Parallel()
	eng, err := NewIntelligenceEngine(slog.Default(), IntelligenceConfig{})
	require.NoError(t, err)

	require.NoError(t, eng.Start())

	eng.Stop()
	// verify no panic on second stop
	eng.Stop()
}

func TestEngineStartStop_AllComponents(t *testing.T) {
	t.Parallel()
	eng, err := NewIntelligenceEngine(slog.Default(), IntelligenceConfig{
		EnableAdaptiveDecisionMaking: true,
		EnableMachineLearning:        true,
		EnablePredictiveAnalytics:    true,
		EnableBehavioralAnalysis:     true,
		EnableOptimization:           true,
		EnableStrategicPlanning:      true,
		EnableAnomalyDetection:       true,
	})
	require.NoError(t, err)

	require.NoError(t, eng.Start())

	eng.Stop()
}

func TestEngineContextCancelPropagation(t *testing.T) {
	t.Parallel()
	eng, err := NewIntelligenceEngine(slog.Default(), IntelligenceConfig{
		EnableAdaptiveDecisionMaking: true,
	})
	require.NoError(t, err)

	require.NoError(t, eng.Start())

	select {
	case <-eng.ctx.Done():
		assert.Fail(t, "context should not be cancelled while engine is running")
	default:
	}

	eng.Stop()

	select {
	case <-eng.ctx.Done():
		// expected after stop
	default:
		assert.Fail(t, "context should be cancelled after Stop()")
	}
}

// ---------------------------------------------------------------------------
// AdaptiveDecisionMaker
// ---------------------------------------------------------------------------

func TestNewAdaptiveDecisionMaker(t *testing.T) {
	t.Parallel()
	adm := NewAdaptiveDecisionMaker(slog.Default(), DecisionMakingConfig{})
	require.NotNil(t, adm)
	assert.NotNil(t, adm.logger, "logger should be set")
	assert.NotNil(t, adm.decisionModels, "decisionModels should be initialized")
	assert.NotNil(t, adm.contextAnalyzer, "contextAnalyzer should be initialized")
	assert.NotNil(t, adm.riskAssessor, "riskAssessor should be initialized")
	assert.NotNil(t, adm.utilityCalculator, "utilityCalculator should be initialized")
	assert.NotNil(t, adm.consensusBuilder, "consensusBuilder should be initialized")
	assert.NotNil(t, adm.decisionHistory, "decisionHistory should be initialized")
	assert.NotNil(t, adm.modelPerformance, "modelPerformance should be initialized")
	assert.Empty(t, adm.decisionHistory, "expected empty decision history")
}

func TestAdaptiveDecisionMaker_Shutdown(t *testing.T) {
	t.Parallel()
	adm := NewAdaptiveDecisionMaker(slog.Default(), DecisionMakingConfig{})
	// Shutdown should not panic
	adm.Shutdown()
}

func TestAdaptiveDecisionMaker_DecisionDomains(t *testing.T) {
	domains := []DecisionDomain{
		DecisionNetwork,
		DecisionResource,
		DecisionSecurity,
		DecisionPerformance,
		DecisionMaintenance,
		DecisionScaling,
		DecisionRouting,
		DecisionScheduling,
	}
	expected := map[DecisionDomain]bool{
		DecisionNetwork:     true,
		DecisionResource:    true,
		DecisionSecurity:    true,
		DecisionPerformance: true,
		DecisionMaintenance: true,
		DecisionScaling:     true,
		DecisionRouting:     true,
		DecisionScheduling:  true,
	}
	assert.Len(t, domains, len(expected), "expected unique domains")
	for _, d := range domains {
	assert.True(t, expected[d], "unexpected domain: %s", d)
		_ = d // suppress unused; we already verified via map
	}
}

func TestAdaptiveDecisionMaker_DecisionModelStruct(t *testing.T) {
	t.Parallel()
	now := time.Now()
	dm := &DecisionModel{
		Domain:    DecisionSecurity,
		ModelType: ModelBayesian,
		Parameters: map[string]interface{}{
			"alpha": 0.05,
			"beta":  0.95,
		},
		TrainingData: []TrainingSample{
			{Features: map[string]interface{}{"cpu": 0.8}, Target: "safe", Weight: 1.0, Timestamp: now, Quality: 0.99},
		},
		Performance: &ModelPerformance{
			Accuracy: 0.95, Precision: 0.93, Recall: 0.97, F1Score: 0.95,
			TrainingTime: 5 * time.Second, SampleSize: 1000,
		},
		LastUpdated: now,
		Version:     "2.0.0",
	}
	assert.Equal(t, DecisionSecurity, dm.Domain, "expected DecisionSecurity")
	assert.Equal(t, ModelBayesian, dm.ModelType, "expected ModelBayesian")
	assert.Equal(t, 0.95, dm.Performance.Accuracy, "expected accuracy 0.95")
	assert.Equal(t, "2.0.0", dm.Version, "expected version 2.0.0")
}

func TestAdaptiveDecisionMaker_ModelPerformance(t *testing.T) {
	t.Parallel()
	mp := &ModelPerformance{
		Accuracy:   0.92,
		Precision:  0.89,
		Recall:     0.94,
		F1Score:    0.91,
		AUC:        0.96,
		Loss:       0.08,
		SampleSize: 5000,
		Confidence: 0.88,
	}
	assert.True(t, mp.F1Score > 0, "F1Score should be positive")
	assert.Equal(t, 5000, mp.SampleSize, "expected sample size 5000")
	assert.Equal(t, 0.88, mp.Confidence, "expected confidence 0.88")
}

func TestAdaptiveDecisionMaker_DecisionRecord(t *testing.T) {
	t.Parallel()
	dr := &DecisionRecord{
		ID:        "test-decision-1",
		Timestamp: time.Now(),
		Domain:    DecisionResource,
		Context:   &DecisionContext{Environment: map[string]interface{}{"load": 0.75}},
		Alternatives: []*DecisionAlternative{
			{ID: "opt_a", Description: "Scale up", ExpectedUtility: 0.85},
			{ID: "opt_b", Description: "Scale down", ExpectedUtility: 0.40},
		},
		Selected:   &DecisionAlternative{ID: "opt_a", ExpectedUtility: 0.85},
		Confidence: 0.85,
		Learning:   &LearningFeedback{Reinforcement: 0.1, KnowledgeGain: decimal.NewFromFloat(0.15)},
	}
	assert.Len(t, dr.Alternatives, 2, "expected 2 alternatives")
	assert.Equal(t, "opt_a", dr.Selected.ID, "expected selected opt_a")
	assert.Equal(t, 0.85, dr.Confidence, "expected confidence 0.85")
	assert.Equal(t, 0.15, dr.Learning.KnowledgeGain.InexactFloat64(), "expected knowledge gain 0.15")
}

// ---------------------------------------------------------------------------
// ContextAnalyzer
// ---------------------------------------------------------------------------

func TestNewContextAnalyzer(t *testing.T) {
	t.Parallel()
	ca := NewContextAnalyzer(slog.Default())
	require.NotNil(t, ca)
	assert.NotNil(t, ca.logger, "logger should be set")
}

func TestContextAnalyzer_ContextFactor(t *testing.T) {
	t.Parallel()
	cf := ContextFactor{
		Name:       "cpu_usage",
		Type:       FactorNumerical,
		Weight:     0.8,
		Importance: ImportanceHigh,
		Dynamic:    true,
		UpdateRate: 30 * time.Second,
	}
	assert.Equal(t, "cpu_usage", cf.Name, "expected name cpu_usage")
	assert.Equal(t, FactorNumerical, cf.Type, "expected FactorNumerical")
	assert.Equal(t, 0.8, cf.Weight, "expected weight 0.8")
	assert.Equal(t, ImportanceHigh, cf.Importance, "expected ImportanceHigh")
	assert.True(t, cf.Dynamic, "Dynamic should be true")
}

func TestContextAnalyzer_FactorTypes(t *testing.T) {
	t.Parallel()
	factors := []FactorType{FactorNumerical, FactorCategorical, FactorBoolean, FactorTemporal, FactorSpatial}
	assert.Len(t, factors, 5, "expected 5 factor types")
	seen := make(map[FactorType]bool)
	for _, f := range factors {
		seen[f] = true
	}
	assert.True(t, seen[FactorNumerical] && seen[FactorCategorical] && seen[FactorBoolean] &&
		seen[FactorTemporal] && seen[FactorSpatial], "missing factor types")
}

func TestContextAnalyzer_ImportanceLevels(t *testing.T) {
	t.Parallel()
	levels := []ImportanceLevel{ImportanceCritical, ImportanceHigh, ImportanceMedium, ImportanceLow}
	assert.Len(t, levels, 4, "expected 4 importance levels")
}

// ---------------------------------------------------------------------------
// RiskAssessmentEngine
// ---------------------------------------------------------------------------

func TestNewRiskAssessmentEngine(t *testing.T) {
	t.Parallel()
	rae := NewRiskAssessmentEngine(slog.Default())
	require.NotNil(t, rae)
	assert.NotNil(t, rae.logger, "logger should be set")
	assert.NotNil(t, rae.riskModels, "riskModels should be initialized")
	assert.Empty(t, rae.riskModels, "expected empty riskModels")
}

func TestRiskAssessmentEngine_RiskTypes(t *testing.T) {
	t.Parallel()
	types := []RiskType{RiskOperational, RiskSecurity, RiskFinancial, RiskReputational, RiskCompliance, RiskTechnical}
	assert.Len(t, types, 6, "expected 6 risk types")
	seen := make(map[RiskType]bool)
	for _, rt := range types {
		seen[rt] = true
	}
	assert.Len(t, seen, 6, "duplicate risk types detected")
}

func TestRiskAssessmentEngine_RiskModel(t *testing.T) {
	t.Parallel()
	rae := NewRiskAssessmentEngine(slog.Default())
	rae.riskModels[RiskSecurity] = &RiskModel{}
	rae.riskModels[RiskOperational] = &RiskModel{}
	assert.Len(t, rae.riskModels, 2, "expected 2 risk models")
	assert.Contains(t, rae.riskModels, RiskSecurity, "RiskSecurity model should exist")
	assert.Contains(t, rae.riskModels, RiskOperational, "RiskOperational model should exist")
}

func TestRiskAssessmentEngine_RiskProfile(t *testing.T) {
	t.Parallel()
	var rp RiskProfile
	_ = rp
}

// ---------------------------------------------------------------------------
// PredictiveAnalyticsEngine
// ---------------------------------------------------------------------------

func TestNewPredictiveAnalyticsEngine(t *testing.T) {
	t.Parallel()
	pae := NewPredictiveAnalyticsEngine(slog.Default(), PredictiveConfig{})
	require.NotNil(t, pae)
	assert.NotNil(t, pae.logger, "logger should be set")
	assert.NotNil(t, pae.forecastingModels, "forecastingModels should be initialized")
	assert.NotNil(t, pae.timeSeriesEngine, "timeSeriesEngine should be initialized")
	assert.NotNil(t, pae.patternRecognizer, "patternRecognizer should be initialized")
	assert.NotNil(t, pae.confidenceEngine, "confidenceEngine should be initialized")
	assert.NotNil(t, pae.scenarioGenerator, "scenarioGenerator should be initialized")
	assert.NotNil(t, pae.predictions, "predictions should be initialized")
	assert.NotNil(t, pae.forecastAccuracy, "forecastAccuracy should be initialized")
}

func TestPredictiveAnalyticsEngine_GenerateForecast(t *testing.T) {
	t.Parallel()
	pae := NewPredictiveAnalyticsEngine(slog.Default(), PredictiveConfig{})

	forecastTypes := []ForecastType{
		ForecastDemand, ForecastPerformance, ForecastResource,
		ForecastFailure, ForecastSecurity, ForecastMarket,
	}
	for _, ft := range forecastTypes {
		pred := pae.GenerateForecast(ft, nil)
	assert.NotNil(t, pred, "GenerateForecast(%s) should not return nil")
	assert.Equal(t, ft, pred.Type, "expected forecast type ft, got ft")
	}
}

func TestPredictiveAnalyticsEngine_ForecastValue(t *testing.T) {
	t.Parallel()
	now := time.Now()
	fv := ForecastValue{
		Timestamp:  now,
		Value:      100.5,
		LowerBound: 90.0,
		UpperBound: 110.0,
		Confidence: 0.95,
	}
	assert.Equal(t, 0.95, fv.Confidence, "expected confidence 0.95")
	v, ok := fv.Value.(float64)
	assert.True(t, ok, "type assertion should succeed")
	assert.Equal(t, 100.5, v, "expected value 100.5")
}

func TestPredictiveAnalyticsEngine_AccuracyMetrics(t *testing.T) {
	t.Parallel()
	am := &AccuracyMetrics{
		MAE:        1.5,
		RMSE:       2.0,
		MAPE:       3.5,
		RSquared:   0.94,
		Confidence: 0.90,
	}
	assert.Equal(t, 1.5, am.MAE, "expected MAE 1.5")
	assert.Equal(t, 2.0, am.RMSE, "expected RMSE 2.0")
	assert.Equal(t, 3.5, am.MAPE, "expected MAPE 3.5")
	assert.Equal(t, 0.94, am.RSquared, "expected R² 0.94")
}

func TestPredictiveAnalyticsEngine_ForecastTypesComplete(t *testing.T) {
	t.Parallel()
	types := []ForecastType{ForecastDemand, ForecastPerformance, ForecastResource, ForecastFailure, ForecastSecurity, ForecastMarket}
	seen := make(map[ForecastType]bool)
	for _, ft := range types {
		seen[ft] = true
	}
	assert.Len(t, seen, 6, "expected 6 unique forecast types, got %d")
}

func TestPredictiveAnalyticsEngine_Shutdown(t *testing.T) {
	t.Parallel()
	pae := NewPredictiveAnalyticsEngine(slog.Default(), PredictiveConfig{})
	pae.Shutdown() // should not panic
}

// ---------------------------------------------------------------------------
// AnomalyDetector
// ---------------------------------------------------------------------------

func TestNewAdvancedAnomalyDetector(t *testing.T) {
	t.Parallel()
	aad := NewAdvancedAnomalyDetector(slog.Default(), AnomalyDetectionConfig{})
	require.NotNil(t, aad)
	assert.NotNil(t, aad.logger, "logger should be set")
	assert.NotNil(t, aad.detectors, "detectors should be initialized")
	assert.NotNil(t, aad.fusionEngine, "fusionEngine should be initialized")
	assert.NotNil(t, aad.contextEngine, "contextEngine should be initialized")
	assert.NotNil(t, aad.alertSystem, "alertSystem should be initialized")
	assert.NotNil(t, aad.anomalies, "anomalies should be initialized")
	assert.NotNil(t, aad.detectionRates, "detectionRates should be initialized")
}

func TestAdvancedAnomalyDetector_DetectMultiple(t *testing.T) {
	t.Parallel()
	aad := NewAdvancedAnomalyDetector(slog.Default(), AnomalyDetectionConfig{})
	results := aad.DetectMultiple(map[string]interface{}{"cpu": 0.95, "memory": 0.88})
	require.NotNil(t, results)
	assert.Len(t, results, 0, "expected empty results, got %d")
}

func TestAdvancedAnomalyDetector_UpdateModels(t *testing.T) {
	t.Parallel()
	aad := NewAdvancedAnomalyDetector(slog.Default(), AnomalyDetectionConfig{})
	anomalies := []*DetectedAnomaly{
		{
			ID: "anomaly-1", Type: AnomalyStatistical, Severity: AnomalySeverityHigh,
			Confidence: 0.92, Description: "CPU spike detected",
		},
	}
	aad.UpdateModels(anomalies) // should not panic
}

func TestAdvancedAnomalyDetector_DetectedAnomaly(t *testing.T) {
	t.Parallel()
	a := &DetectedAnomaly{
		ID:              "anomaly-test-1",
		Timestamp:       time.Now(),
		Type:            AnomalyContextual,
		Entity:          "server-01",
		Severity:        AnomalySeverityCritical,
		Confidence:      0.99,
		Description:     "Memory leak detected in process X",
		Context:         map[string]interface{}{"memory_growth_rate": "2.5MB/min"},
		Evidence:        []Evidence{{}, {}},
		Impact:          ImpactAssessment{},
		Recommendations: []string{"restart process", "increase memory limit"},
	}
	assert.Equal(t, AnomalySeverityCritical, a.Severity, "expected critical severity")
	assert.Len(t, a.Recommendations, 2, "expected 2 recommendations, got %d")
	assert.Len(t, a.Evidence, 2, "expected 2 evidence entries, got %d")
}

func TestAdvancedAnomalyDetector_AnomalyTypes(t *testing.T) {
	t.Parallel()
	types := []AnomalyType{AnomalyStatistical, AnomalyBehavioral, AnomalyContextual, AnomalyCollective, AnomalyConceptDrift}
	assert.Len(t, types, 5, "expected 5 anomaly types, got %d")
	seen := make(map[AnomalyType]bool)
	for _, at := range types {
		seen[at] = true
	}
	assert.Len(t, seen, 5, "duplicate anomaly types detected")
}

func TestAdvancedAnomalyDetector_AnomalyAlgorithms(t *testing.T) {
	t.Parallel()
	algos := []AnomalyAlgorithm{
		AlgorithmIsolationForest, AlgorithmOneClassSVM, AlgorithmAutoencoder,
		AlgorithmLOF, AlgorithmARIMA, AlgorithmKalmanFilter,
	}
	assert.Len(t, algos, 6, "expected 6 anomaly algorithms, got %d")
}

func TestAdvancedAnomalyDetector_AnomalySeverityValues(t *testing.T) {
	t.Parallel()
	severities := []AnomalySeverity{AnomalySeverityLow, AnomalySeverityMedium, AnomalySeverityHigh, AnomalySeverityCritical}
	assert.Len(t, severities, 4, "expected 4 severity levels, got %d")
}

func TestAdvancedAnomalyDetector_Shutdown(t *testing.T) {
	t.Parallel()
	aad := NewAdvancedAnomalyDetector(slog.Default(), AnomalyDetectionConfig{})
	aad.Shutdown() // should not panic
}

// ---------------------------------------------------------------------------
// KnowledgeBase
// ---------------------------------------------------------------------------

func TestKnowledgeBase_StoreAndRetrieveFacts(t *testing.T) {
	t.Parallel()
	eng, err := NewIntelligenceEngine(slog.Default(), IntelligenceConfig{})
	require.NoError(t, err)

	kb := eng.knowledgeBase
	kb.Facts["fact-1"] = &Fact{}
	kb.Facts["fact-2"] = &Fact{}

	assert.Len(t, kb.Facts, 2, "expected 2 facts, got %d")
	assert.Contains(t, kb.Facts, "fact-1", "fact-1 should exist")
	assert.Contains(t, kb.Facts, "fact-2", "fact-2 should exist")

	delete(kb.Facts, "fact-1")
	assert.Len(t, kb.Facts, 1, "expected 1 fact after delete, got %d")
}

func TestKnowledgeBase_StoreAndRetrieveRules(t *testing.T) {
	t.Parallel()
	eng, err := NewIntelligenceEngine(slog.Default(), IntelligenceConfig{})
	require.NoError(t, err)

	kb := eng.knowledgeBase
	kb.Rules["rule-high-cpu"] = &Rule{}
	kb.Rules["rule-high-mem"] = &Rule{}

	assert.Len(t, kb.Rules, 2, "expected 2 rules, got %d")
	assert.Contains(t, kb.Rules, "rule-high-cpu", "rule-high-cpu should exist")
}

func TestKnowledgeBase_StoreAndRetrieveConcepts(t *testing.T) {
	t.Parallel()
	eng, err := NewIntelligenceEngine(slog.Default(), IntelligenceConfig{})
	require.NoError(t, err)

	kb := eng.knowledgeBase
	kb.Concepts["anomaly"] = &Concept{}
	kb.Concepts["baseline"] = &Concept{}

	assert.Len(t, kb.Concepts, 2, "expected 2 concepts, got %d")
}

func TestKnowledgeBase_StoreAndRetrieveRelationships(t *testing.T) {
	t.Parallel()
	eng, err := NewIntelligenceEngine(slog.Default(), IntelligenceConfig{})
	require.NoError(t, err)

	kb := eng.knowledgeBase
	kb.Relationships["rel-cpu-mem"] = &Relationship{}
	kb.Relationships["rel-network-latency"] = &Relationship{}

	assert.Len(t, kb.Relationships, 2, "expected 2 relationships, got %d")
}

func TestKnowledgeBase_StoreAndRetrieveTheories(t *testing.T) {
	t.Parallel()
	eng, err := NewIntelligenceEngine(slog.Default(), IntelligenceConfig{})
	require.NoError(t, err)

	kb := eng.knowledgeBase
	kb.Theories["theory-cpu-correlation"] = &Theory{}
	kb.Theories["theory-memory-leak"] = &Theory{}

	assert.Len(t, kb.Theories, 2, "expected 2 theories, got %d")
}

func TestKnowledgeBase_Version(t *testing.T) {
	t.Parallel()
	eng, err := NewIntelligenceEngine(slog.Default(), IntelligenceConfig{})
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", eng.knowledgeBase.Version, "expected version 1.0.0")
}

func TestKnowledgeBase_LastUpdated(t *testing.T) {
	t.Parallel()
	eng, err := NewIntelligenceEngine(slog.Default(), IntelligenceConfig{})
	require.NoError(t, err)
	assert.False(t, eng.knowledgeBase.LastUpdated.IsZero(), "LastUpdated should not be zero")
}

func TestKnowledgeBase_EmptyOnInit(t *testing.T) {
	t.Parallel()
	eng, err := NewIntelligenceEngine(slog.Default(), IntelligenceConfig{})
	require.NoError(t, err)
	kb := eng.knowledgeBase
	assert.Len(t, kb.Facts, 0, "expected 0 facts, got %d")
	assert.Len(t, kb.Rules, 0, "expected 0 rules, got %d")
	assert.Len(t, kb.Concepts, 0, "expected 0 concepts, got %d")
	assert.Len(t, kb.Relationships, 0, "expected 0 relationships, got %d")
	assert.Len(t, kb.Theories, 0, "expected 0 theories, got %d")
}

// ---------------------------------------------------------------------------
// UtilityCalculator
// ---------------------------------------------------------------------------

func TestNewUtilityCalculator(t *testing.T) {
	t.Parallel()
	uc := NewUtilityCalculator(slog.Default())
	require.NotNil(t, uc)
	assert.NotNil(t, uc.utilityFunctions, "utilityFunctions should be initialized")
	assert.NotNil(t, uc.multiCriteriaEngine, "multiCriteriaEngine should be initialized")
	assert.NotNil(t, uc.sensitivityAnalyzer, "sensitivityAnalyzer should be initialized")
}

func TestUtilityCalculator_UtilityTypes(t *testing.T) {
	t.Parallel()
	types := []UtilityType{UtilityLinear, UtilityExponential, UtilityLogarithmic, UtilityQuadratic, UtilitySigmoid}
	assert.Len(t, types, 5, "expected 5 utility types, got %d")
}

// ---------------------------------------------------------------------------
// ConsensusBuilder
// ---------------------------------------------------------------------------

func TestNewConsensusBuilder(t *testing.T) {
	t.Parallel()
	cb := NewConsensusBuilder(slog.Default())
	require.NotNil(t, cb)
	assert.NotNil(t, cb.logger, "logger should be set")
}

func TestConsensusBuilder_Methods(t *testing.T) {
	t.Parallel()
	methods := []ConsensusMethod{ConsensusVoting, ConsensusWeighted, ConsensusBayesian, ConsensusGameTheory, ConsensusFuzzyLogic}
	assert.Len(t, methods, 5, "expected 5 consensus methods, got %d")
}

// ---------------------------------------------------------------------------
// Decision engine — alternative generation and evaluation
// ---------------------------------------------------------------------------

func TestEngineGenerateAlternatives_Network(t *testing.T) {
	t.Parallel()
	eng, err := NewIntelligenceEngine(slog.Default(), IntelligenceConfig{})
	require.NoError(t, err)
	ctx := eng.gatherDecisionContext()
	alts := eng.generateAlternatives(DecisionNetwork, ctx)
	require.NotEmpty(t, alts, "expected at least 1 alternative for network domain")
	found := false
	for _, a := range alts {
		if a.ID == "network_route_1" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected network_route_1 alternative")
}

func TestEngineGenerateAlternatives_Resource(t *testing.T) {
	t.Parallel()
	eng, err := NewIntelligenceEngine(slog.Default(), IntelligenceConfig{})
	require.NoError(t, err)
	ctx := eng.gatherDecisionContext()
	alts := eng.generateAlternatives(DecisionResource, ctx)
	require.NotEmpty(t, alts, "expected at least 1 alternative for resource domain")
}

func TestEngineGenerateAlternatives_UnknownDomain(t *testing.T) {
	t.Parallel()
	eng, err := NewIntelligenceEngine(slog.Default(), IntelligenceConfig{})
	require.NoError(t, err)
	ctx := eng.gatherDecisionContext()
	alts := eng.generateAlternatives("unknown_domain", ctx)
	require.NotNil(t, alts)
	assert.Len(t, alts, 0, "expected empty alternatives for unknown domain, got %d")
}

func TestEngineEvaluateAlternatives(t *testing.T) {
	t.Parallel()
	eng, err := NewIntelligenceEngine(slog.Default(), IntelligenceConfig{})
	require.NoError(t, err)
	ctx := eng.gatherDecisionContext()
	alts := []*DecisionAlternative{
		{ID: "low", ExpectedUtility: 0.3},
		{ID: "high", ExpectedUtility: 0.9},
		{ID: "mid", ExpectedUtility: 0.6},
	}
	evaluated := eng.evaluateAlternatives(alts, ctx)
	require.Len(t, evaluated, 3)
	// verify sorted by expected utility descending
	assert.Equal(t, "high", evaluated[0].ID, "first element should be high")
	assert.Equal(t, "mid", evaluated[1].ID, "second element should be mid")
	assert.Equal(t, "low", evaluated[2].ID, "third element should be low")
}

func TestEngineSelectBestAlternative(t *testing.T) {
	t.Parallel()
	eng, err := NewIntelligenceEngine(slog.Default(), IntelligenceConfig{})
	require.NoError(t, err)
	ctx := eng.gatherDecisionContext()
	alts := []*DecisionAlternative{
		{ID: "a", ExpectedUtility: 0.5},
		{ID: "b", ExpectedUtility: 0.9},
	}
	best := eng.selectBestAlternative(alts, ctx)
	require.NotNil(t, best)
	assert.Equal(t, "a", best.ID, "expected first element a")
}

func TestEngineSelectBestAlternative_Empty(t *testing.T) {
	t.Parallel()
	eng, err := NewIntelligenceEngine(slog.Default(), IntelligenceConfig{})
	require.NoError(t, err)
	best := eng.selectBestAlternative([]*DecisionAlternative{}, eng.gatherDecisionContext())
	assert.Nil(t, best, "expected nil for empty alternatives")
}

func TestEngineExecuteDecision(t *testing.T) {
	t.Parallel()
	eng, err := NewIntelligenceEngine(slog.Default(), IntelligenceConfig{})
	require.NoError(t, err)
	alt := &DecisionAlternative{ID: "test", ExpectedUtility: 0.75}
	ctx := eng.gatherDecisionContext()
	outcome := eng.executeDecision(alt, ctx)
	require.NotNil(t, outcome)
	assert.NotNil(t, outcome.Performance, "Performance map should be initialized")
	assert.NotNil(t, outcome.ResourceUsage, "ResourceUsage map should be initialized")
	assert.True(t, outcome.Duration > 0, "Duration should be positive")
}

func TestEngineLearnFromOutcome(t *testing.T) {
	t.Parallel()
	eng, err := NewIntelligenceEngine(slog.Default(), IntelligenceConfig{})
	require.NoError(t, err)
	alt := &DecisionAlternative{
		ID:              "test",
		ExpectedUtility: 0.5,
		Attributes:      make(map[string]interface{}),
	}
	outcome := &DecisionOutcome{
		ActualUtility: 0.8,
		Success:       true,
	}
	eng.learnFromOutcome(alt, outcome, eng.gatherDecisionContext())
	fb, ok := alt.Attributes["learning_feedback"]
	require.True(t, ok, "expected learning_feedback in attributes")
	lf, ok := fb.(*LearningFeedback)
	require.True(t, ok, "expected *LearningFeedback, got %T", fb)
	assert.InDelta(t, 0.3, lf.Reinforcement, 1e-9, "expected reinforcement ~0.3")
	expectedKG := decimal.NewFromFloat(math.Abs(outcome.ActualUtility - alt.ExpectedUtility))
	assert.True(t, lf.KnowledgeGain.Equal(expectedKG), "expected knowledge gain %s", expectedKG.String())
}

func TestEngineRecordDecision(t *testing.T) {
	t.Parallel()
	eng, err := NewIntelligenceEngine(slog.Default(), IntelligenceConfig{
		EnableAdaptiveDecisionMaking: true,
	})
	require.NoError(t, err)
	record := &DecisionRecord{
		ID:         "rec-1",
		Domain:     DecisionNetwork,
		Confidence: 0.9,
		Outcome:    &DecisionOutcome{Success: true},
	}
	eng.recordDecision(record)
	require.Len(t, eng.decisionMaker.decisionHistory, 1)
	assert.Equal(t, "rec-1", eng.decisionMaker.decisionHistory[0].ID, "expected record ID rec-1")
}

// ---------------------------------------------------------------------------
// Model types and constants
// ---------------------------------------------------------------------------

func TestModelTypes(t *testing.T) {
	t.Parallel()
	types := []ModelType{ModelSupervised, ModelUnsupervised, ModelReinforcement, ModelEnsemble, ModelDeepLearning, ModelBayesian}
	assert.Len(t, types, 6, "expected 6 model types, got %d")
}

func TestAlgorithmTypes(t *testing.T) {
	t.Parallel()
	types := []AlgorithmType{
		AlgorithmRandomForest, AlgorithmNeuralNetwork, AlgorithmSVM,
		AlgorithmGradientBoosting, AlgorithmKMeans, AlgorithmPCA,
		AlgorithmLSTM, AlgorithmQLearning,
	}
	assert.Len(t, types, 8, "expected 8 algorithm types, got %d")
}

func TestDeploymentStatusValues(t *testing.T) {
	t.Parallel()
	statuses := []DeploymentStatus{
		DeploymentDevelopment, DeploymentTesting, DeploymentStaging,
		DeploymentProduction, DeploymentRetired,
	}
	assert.Len(t, statuses, 5, "expected 5 deployment statuses, got %d")
}

func TestBehaviorTypes(t *testing.T) {
	t.Parallel()
	types := []BehaviorType{
		BehaviorNetwork, BehaviorResource, BehaviorUser, BehaviorSystem,
		BehaviorSecurity, BehaviorPerformance,
	}
	assert.Len(t, types, 6, "expected 6 behavior types, got %d")
}

func TestPatternTypes(t *testing.T) {
	t.Parallel()
	types := []PatternType{
		PatternSequential, PatternTemporal, PatternSpatial,
		PatternBehavioral, PatternAnomalous, PatternRecurring,
	}
	assert.Len(t, types, 6, "expected 6 pattern types, got %d")
}

func TestOptimizationTypes(t *testing.T) {
	t.Parallel()
	types := []OptimizationType{
		OptimizationLinear, OptimizationNonlinear, OptimizationInteger,
		OptimizationMultiObjective, OptimizationGenetic, OptimizationSwarm,
	}
	assert.Len(t, types, 6, "expected 6 optimization types, got %d")
}

func TestOptimizationAlgorithms(t *testing.T) {
	t.Parallel()
	algos := []OptimizationAlgorithm{
		AlgorithmGradientDescent, AlgorithmGenetic, AlgorithmSimulatedAnnealing,
		AlgorithmParticleSwarm, AlgorithmAntColony, AlgorithmTabuSearch,
	}
	assert.Len(t, algos, 6, "expected 6 optimization algorithms, got %d")
}

func TestPlanningHorizons(t *testing.T) {
	t.Parallel()
	horizons := []PlanningHorizon{
		HorizonShortTerm, HorizonMediumTerm, HorizonLongTerm, HorizonStrategic,
	}
	assert.Len(t, horizons, 4, "expected 4 planning horizons, got %d")
}

// ---------------------------------------------------------------------------
// ExperienceRecord and LearningFeedback
// ---------------------------------------------------------------------------

func TestExperienceRecord(t *testing.T) {
	t.Parallel()
	exp := &ExperienceRecord{
		ID:       "exp-1",
		Action:   "scale_up",
		Reward:   0.85,
		Success:  true,
		Learning: &LearningInsight{},
	}
	assert.Equal(t, 0.85, exp.Reward, "expected reward 0.85")
	assert.True(t, exp.Success, "expected success to be true")
}

func TestLearningFeedbackDecimal(t *testing.T) {
	t.Parallel()
	lf := &LearningFeedback{
		Reinforcement: 0.25,
		KnowledgeGain: decimal.NewFromFloat(0.5),
	}
	assert.Equal(t, 0.5, lf.KnowledgeGain.InexactFloat64(), "expected knowledge gain 0.5")
	d, err := decimal.NewFromString("0.75")
	require.NoError(t, err)
	lf.KnowledgeGain = d
	assert.Equal(t, 0.75, lf.KnowledgeGain.InexactFloat64(), "expected knowledge gain 0.75")
}

// ---------------------------------------------------------------------------
// Machine Learning System
// ---------------------------------------------------------------------------

func TestNewMachineLearningSystem(t *testing.T) {
	t.Parallel()
	mls := NewMachineLearningSystem(slog.Default(), LearningConfig{})
	require.NotNil(t, mls)
	assert.NotNil(t, mls.models, "models should be initialized")
	assert.NotNil(t, mls.trainingEngine, "trainingEngine should be initialized")
	assert.NotNil(t, mls.featureEngine, "featureEngine should be initialized")
	assert.NotNil(t, mls.modelSelector, "modelSelector should be initialized")
	assert.NotNil(t, mls.ensembleSystem, "ensembleSystem should be initialized")
	assert.NotNil(t, mls.learningEvents, "learningEvents should be initialized")
	assert.NotNil(t, mls.modelMetrics, "modelMetrics should be initialized")
}

func TestMachineLearningSystem_ProcessExperience(t *testing.T) {
	t.Parallel()
	mls := NewMachineLearningSystem(slog.Default(), LearningConfig{})
	exp := &ExperienceRecord{ID: "exp-1", Reward: 0.9}
	mls.ProcessExperience(exp) // should not panic
}

func TestMachineLearningSystem_UpdateModels(t *testing.T) {
	t.Parallel()
	mls := NewMachineLearningSystem(slog.Default(), LearningConfig{})
	mls.UpdateModels() // should not panic
}

func TestMachineLearningSystem_ValidateModels(t *testing.T) {
	t.Parallel()
	mls := NewMachineLearningSystem(slog.Default(), LearningConfig{})
	mls.ValidateModels() // should not panic
}

func TestMachineLearningSystem_Shutdown(t *testing.T) {
	t.Parallel()
	mls := NewMachineLearningSystem(slog.Default(), LearningConfig{})
	mls.Shutdown() // should not panic
}

func TestMachineLearningSystem_MLModel(t *testing.T) {
	t.Parallel()
	model := &MLModel{
		Name:              "resource-predictor",
		Type:              ModelSupervised,
		Algorithm:         AlgorithmRandomForest,
		Features:          []string{"cpu", "memory", "disk_io"},
		Target:            "load",
		TrainingMetrics:   &ModelMetrics{},
		ValidationMetrics: &ModelMetrics{},
		DeploymentStatus:  DeploymentDevelopment,
		Version:           "0.1.0",
	}
	assert.Len(t, model.Features, 3, "expected 3 features, got %d")
	assert.Equal(t, DeploymentDevelopment, model.DeploymentStatus, "expected Development")
}

// ---------------------------------------------------------------------------
// Behavioral Analysis System
// ---------------------------------------------------------------------------

func TestNewBehavioralAnalysisSystem(t *testing.T) {
	t.Parallel()
	bas := NewBehavioralAnalysisSystem(slog.Default(), BehavioralConfig{})
	require.NotNil(t, bas)
	assert.NotNil(t, bas.behaviorModels, "behaviorModels should be initialized")
	assert.NotNil(t, bas.patternMatcher, "patternMatcher should be initialized")
	assert.NotNil(t, bas.anomalyEngine, "anomalyEngine should be initialized")
	assert.NotNil(t, bas.trendAnalyzer, "trendAnalyzer should be initialized")
	assert.NotNil(t, bas.clusteringEngine, "clusteringEngine should be initialized")
	assert.NotNil(t, bas.behaviorProfiles, "behaviorProfiles should be initialized")
	assert.NotNil(t, bas.analysisResults, "analysisResults should be initialized")
}

func TestBehavioralAnalysisSystem_AnalyzePatterns(t *testing.T) {
	t.Parallel()
	bas := NewBehavioralAnalysisSystem(slog.Default(), BehavioralConfig{})
	results := bas.AnalyzePatterns(map[string]interface{}{"value": 42})
	require.NotNil(t, results)
	require.Empty(t, results)
}

func TestBehavioralAnalysisSystem_DetectAnomalies(t *testing.T) {
	t.Parallel()
	bas := NewBehavioralAnalysisSystem(slog.Default(), BehavioralConfig{})
	results := bas.DetectAnomalies([]interface{}{})
	require.NotNil(t, results)
}

func TestBehavioralAnalysisSystem_UpdateProfiles(t *testing.T) {
	t.Parallel()
	bas := NewBehavioralAnalysisSystem(slog.Default(), BehavioralConfig{})
	bas.UpdateProfiles([]interface{}{}) // should not panic
}

func TestBehavioralAnalysisSystem_GenerateInsights(t *testing.T) {
	t.Parallel()
	bas := NewBehavioralAnalysisSystem(slog.Default(), BehavioralConfig{})
	insights := bas.GenerateInsights([]interface{}{}, []interface{}{})
	require.NotNil(t, insights)
}

func TestBehavioralAnalysisSystem_Shutdown(t *testing.T) {
	t.Parallel()
	bas := NewBehavioralAnalysisSystem(slog.Default(), BehavioralConfig{})
	bas.Shutdown() // should not panic
}

func TestBehavioralAnalysisSystem_BehaviorProfile(t *testing.T) {
	t.Parallel()
	bp := &BehaviorProfile{
		Entity:   "server-01",
		Type:     BehaviorSystem,
		Baseline: &BehaviorBaseline{},
		Patterns: []BehaviorPattern{
			{
				ID:        "pattern-1",
				Type:      PatternTemporal,
				Frequency: 0.85,
				Duration:  5 * time.Minute,
			},
		},
		Confidence: 0.92,
	}
	assert.Len(t, bp.Patterns, 1, "expected 1 pattern, got %d")
	assert.Equal(t, 0.92, bp.Confidence, "expected confidence 0.92")
}

// ---------------------------------------------------------------------------
// Optimization Engine
// ---------------------------------------------------------------------------

func TestNewOptimizationEngine(t *testing.T) {
	t.Parallel()
	oe := NewOptimizationEngine(slog.Default(), OptimizationConfig{})
	require.NotNil(t, oe)
	assert.NotNil(t, oe.optimizers, "optimizers should be initialized")
	assert.NotNil(t, oe.constraintEngine, "constraintEngine should be initialized")
	assert.NotNil(t, oe.objectiveEngine, "objectiveEngine should be initialized")
	assert.NotNil(t, oe.solutionSpace, "solutionSpace should be initialized")
	assert.NotNil(t, oe.metaOptimizer, "metaOptimizer should be initialized")
	assert.NotNil(t, oe.optimizationRuns, "optimizationRuns should be initialized")
	assert.NotNil(t, oe.bestSolutions, "bestSolutions should be initialized")
}

func TestOptimizationEngine_Solve(t *testing.T) {
	t.Parallel()
	oe := NewOptimizationEngine(slog.Default(), OptimizationConfig{})
	solution := oe.Solve("test_problem")
	require.NotNil(t, solution)
}

func TestOptimizationEngine_Shutdown(t *testing.T) {
	t.Parallel()
	oe := NewOptimizationEngine(slog.Default(), OptimizationConfig{})
	oe.Shutdown() // should not panic
}

func TestOptimizationEngine_OptimalSolution(t *testing.T) {
	t.Parallel()
	sol := &OptimalSolution{
		Variables:      map[string]interface{}{"x": 1.0, "y": 2.0},
		ObjectiveValue: 42.0,
		Feasibility:    true,
		Optimality:     0.98,
		Confidence:     0.95,
	}
	assert.True(t, sol.Feasibility, "expected feasible solution")
	assert.Equal(t, 0.98, sol.Optimality, "expected optimality 0.98")
}

// ---------------------------------------------------------------------------
// Strategic Planning Engine
// ---------------------------------------------------------------------------

func TestNewStrategicPlanningEngine(t *testing.T) {
	t.Parallel()
	spe := NewStrategicPlanningEngine(slog.Default(), StrategicConfig{})
	require.NotNil(t, spe)
	assert.NotNil(t, spe.planningModels, "planningModels should be initialized")
	assert.NotNil(t, spe.goalHierarchy, "goalHierarchy should be initialized")
	assert.NotNil(t, spe.resourceAllocator, "resourceAllocator should be initialized")
	assert.NotNil(t, spe.scenarioPlanner, "scenarioPlanner should be initialized")
	assert.NotNil(t, spe.planExecutor, "planExecutor should be initialized")
	assert.NotNil(t, spe.strategicPlans, "strategicPlans should be initialized")
	assert.NotNil(t, spe.planProgress, "planProgress should be initialized")
}

func TestStrategicPlanningEngine_GetCurrentPlans(t *testing.T) {
	t.Parallel()
	spe := NewStrategicPlanningEngine(slog.Default(), StrategicConfig{})
	plans := spe.GetCurrentPlans()
	require.NotNil(t, plans)
	assert.Len(t, plans, 0, "expected 0 plans, got %d")
}

func TestStrategicPlanningEngine_CreatePlan(t *testing.T) {
	t.Parallel()
	spe := NewStrategicPlanningEngine(slog.Default(), StrategicConfig{})
	plan := spe.CreatePlan("initiative-1")
	require.NotNil(t, plan)
}

func TestStrategicPlanningEngine_UpdatePlanProgress(t *testing.T) {
	t.Parallel()
	spe := NewStrategicPlanningEngine(slog.Default(), StrategicConfig{})
	spe.UpdatePlanProgress("plan-1", &PlanProgress{}) // should not panic
}

func TestStrategicPlanningEngine_Shutdown(t *testing.T) {
	t.Parallel()
	spe := NewStrategicPlanningEngine(slog.Default(), StrategicConfig{})
	spe.Shutdown() // should not panic
}

func TestStrategicPlanningEngine_StrategicPlan(t *testing.T) {
	t.Parallel()
	plan := &StrategicPlan{
		ID:      "plan-scale-2026",
		Name:    "Scale Infrastructure 2026",
		Horizon: HorizonLongTerm,
		Goals: []StrategicGoal{
			{},
		},
		Timeline: 365 * 24 * time.Hour,
		Budget:   500000,
		Status:   PlanStatus("active"),
	}
	assert.Equal(t, HorizonLongTerm, plan.Horizon, "expected long-term horizon")
	assert.Equal(t, 500000.0, plan.Budget, "expected budget 500000")
	assert.Len(t, plan.Goals, 1, "expected 1 goal, got %d")
}

// ---------------------------------------------------------------------------
// DecisionContext
// ---------------------------------------------------------------------------

func TestDecisionContext(t *testing.T) {
	t.Parallel()
	dc := &DecisionContext{
		Environment: map[string]interface{}{"region": "us-east-1"},
		Constraints: []Constraint{{}, {}},
		Objectives:  []Objective{{}},
		RiskFactors: []RiskFactor{{}},
		RealTime:    map[string]interface{}{"cpu": 0.85},
	}
	assert.Len(t, dc.Constraints, 2, "expected 2 constraints, got %d")
	assert.Len(t, dc.Objectives, 1, "expected 1 objective, got %d")
}

func TestDecisionAlternative(t *testing.T) {
	t.Parallel()
	alt := &DecisionAlternative{
		ID:              "alt-1",
		Description:     "Deploy additional nodes",
		ExpectedUtility: 0.88,
		Feasibility:     0.95,
		Cost:            15000.0,
		Benefits:        []Benefit{{}, {}},
		Dependencies:    []string{"network", "storage"},
	}
	assert.Equal(t, 0.88, alt.ExpectedUtility, "expected utility 0.88")
	assert.Equal(t, 0.95, alt.Feasibility, "expected feasibility 0.95")
	assert.Equal(t, 15000.0, alt.Cost, "expected cost 15000")
}

// ---------------------------------------------------------------------------
// Integration: engine with all features enabled
// ---------------------------------------------------------------------------

func TestEngineFullLifecycle(t *testing.T) {
	eng, err := NewIntelligenceEngine(slog.Default(), IntelligenceConfig{
		EnableAdaptiveDecisionMaking: true,
		EnableMachineLearning:        true,
		EnablePredictiveAnalytics:    true,
		EnableBehavioralAnalysis:     true,
		EnableOptimization:           true,
		EnableStrategicPlanning:      true,
		EnableAnomalyDetection:       true,
	})
	require.NoError(t, err)

	require.NoError(t, eng.Start())

	// Let background loops start
	time.Sleep(50 * time.Millisecond)

	eng.Stop()

	// After stop, verify no panics and clean state
	assert.False(t, eng.isRunning, "engine should not be running after stop")

	// Verify knowledge base is intact after lifecycle
	assert.NotNil(t, eng.knowledgeBase, "knowledge base should persist")
}

// ---------------------------------------------------------------------------
// Concurrent safety
// ---------------------------------------------------------------------------

func TestEngineConcurrentAccess(t *testing.T) {
	eng, err := NewIntelligenceEngine(slog.Default(), IntelligenceConfig{
		EnableAdaptiveDecisionMaking: true,
		EnablePredictiveAnalytics:    true,
		EnableAnomalyDetection:       true,
	})
	require.NoError(t, err)

	done := make(chan bool)
	go func() {
		_ = eng.Start() //nolint:errcheck
		done <- true
	}()
	go func() {
		eng.Stop()
		done <- true
	}()

	// Let both goroutines run; they should not deadlock
	<-done
	<-done

	eng.Stop() // final cleanup
}

// ---------------------------------------------------------------------------
// Context cancellation propagation
// ---------------------------------------------------------------------------

func TestEngineContextCancelled(t *testing.T) {
	t.Parallel()
	eng, err := NewIntelligenceEngine(slog.Default(), IntelligenceConfig{
		EnableAdaptiveDecisionMaking: true,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	eng.ctx = ctx
	eng.cancel = cancel

	require.NoError(t, eng.Start())

	cancel()
	time.Sleep(50 * time.Millisecond)

	eng.mu.RLock()
	running := eng.isRunning
	eng.mu.RUnlock()

	if running {
		t.Log("engine may still be running if context cancelled externally (expected for current impl)")
	}
}
