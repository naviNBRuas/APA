package intelligence

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

type AdaptiveDecisionMaker struct {
	logger            *slog.Logger
	config            DecisionMakingConfig
	decisionModels    map[DecisionDomain]*DecisionModel
	contextAnalyzer   *ContextAnalyzer
	riskAssessor      *RiskAssessmentEngine
	utilityCalculator *UtilityCalculator
	consensusBuilder  *ConsensusBuilder

	mu               sync.RWMutex
	decisionHistory  []*DecisionRecord
	modelPerformance map[string]*ModelPerformance
}

type DecisionModel struct {
	Domain       DecisionDomain         `json:"domain"`
	ModelType    ModelType              `json:"model_type"`
	Parameters   map[string]interface{} `json:"parameters"`
	TrainingData []TrainingSample       `json:"training_data"`
	Performance  *ModelPerformance      `json:"performance"`
	LastUpdated  time.Time              `json:"last_updated"`
	Version      string                 `json:"version"`
}

type DecisionRecord struct {
	ID           string                 `json:"id"`
	Timestamp    time.Time              `json:"timestamp"`
	Domain       DecisionDomain         `json:"domain"`
	Context      *DecisionContext       `json:"context"`
	Alternatives []*DecisionAlternative `json:"alternatives"`
	Selected     *DecisionAlternative   `json:"selected"`
	Outcome      *DecisionOutcome       `json:"outcome"`
	Confidence   float64                `json:"confidence"`
	Learning     *LearningFeedback      `json:"learning"`
}

type DecisionContext struct {
	Environment map[string]interface{} `json:"environment"`
	Constraints []Constraint           `json:"constraints"`
	Objectives  []Objective            `json:"objectives"`
	RiskFactors []RiskFactor           `json:"risk_factors"`
	Historical  []HistoricalDecision   `json:"historical"`
	RealTime    map[string]interface{} `json:"real_time"`
}

type DecisionAlternative struct {
	ID              string                 `json:"id"`
	Description     string                 `json:"description"`
	Attributes      map[string]interface{} `json:"attributes"`
	ExpectedUtility float64                `json:"expected_utility"`
	RiskProfile     *RiskProfile           `json:"risk_profile"`
	Feasibility     float64                `json:"feasibility"`
	Cost            float64                `json:"cost"`
	Benefits        []Benefit              `json:"benefits"`
	Dependencies    []string               `json:"dependencies"`
}

type DecisionOutcome struct {
	ActualUtility float64            `json:"actual_utility"`
	Performance   map[string]float64 `json:"performance"`
	Duration      time.Duration      `json:"duration"`
	ResourceUsage map[string]float64 `json:"resource_usage"`
	Unintended    []UnintendedEffect `json:"unintended_effects"`
	Success       bool               `json:"success"`
	Feedback      *OutcomeFeedback   `json:"feedback"`
}

type TrainingSample struct {
	Features  map[string]interface{} `json:"features"`
	Target    interface{}            `json:"target"`
	Weight    float64                `json:"weight"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	Quality   float64                `json:"quality"`
}

type ModelPerformance struct {
	Accuracy      float64       `json:"accuracy"`
	Precision     float64       `json:"precision"`
	Recall        float64       `json:"recall"`
	F1Score       float64       `json:"f1_score"`
	AUC           float64       `json:"auc"`
	Loss          float64       `json:"loss"`
	TrainingTime  time.Duration `json:"training_time"`
	LastEvaluated time.Time     `json:"last_evaluated"`
	SampleSize    int           `json:"sample_size"`
	Confidence    float64       `json:"confidence"`
}

func NewAdaptiveDecisionMaker(logger *slog.Logger, config DecisionMakingConfig) *AdaptiveDecisionMaker {
	return &AdaptiveDecisionMaker{
		logger:            logger,
		config:            config,
		decisionModels:    make(map[DecisionDomain]*DecisionModel),
		contextAnalyzer:   NewContextAnalyzer(logger),
		riskAssessor:      NewRiskAssessmentEngine(logger),
		utilityCalculator: NewUtilityCalculator(logger),
		consensusBuilder:  NewConsensusBuilder(logger),
		decisionHistory:   make([]*DecisionRecord, 0),
		modelPerformance:  make(map[string]*ModelPerformance),
	}
}

func (adm *AdaptiveDecisionMaker) MakeDecision(domain DecisionDomain, ctx map[string]interface{}) *DecisionAlternative {
	adm.mu.Lock()
	defer adm.mu.Unlock()

	decisionCtx := adm.contextAnalyzer.Analyze(ctx)

	alternatives := adm.generateAlternatives(domain, decisionCtx)
	if len(alternatives) == 0 {
		adm.logger.Warn("no alternatives generated for decision", "domain", domain)
		return nil
	}

	for _, alt := range alternatives {
		alt.RiskProfile = adm.riskAssessor.AssessRisk(alt, decisionCtx)
	}

	for _, alt := range alternatives {
		alt.ExpectedUtility = adm.utilityCalculator.Calculate(alt, decisionCtx)
	}

	selected := adm.consensusBuilder.BuildConsensus(alternatives)
	if selected == nil {
		selected = alternatives[0]
	}

	model := adm.getOrCreateModel(domain)
	model.LastUpdated = time.Now()

	record := &DecisionRecord{
		ID:           fmt.Sprintf("dec-%s-%d", domain, time.Now().UnixNano()),
		Timestamp:    time.Now(),
		Domain:       domain,
		Context:      decisionCtx,
		Alternatives: alternatives,
		Selected:     selected,
		Outcome:      nil,
		Confidence:   selected.ExpectedUtility,
	}

	adm.decisionHistory = append(adm.decisionHistory, record)
	adm.logger.Debug("decision made", "domain", domain, "selected", selected.ID, "utility", selected.ExpectedUtility)
	return selected
}

func (adm *AdaptiveDecisionMaker) RecordOutcome(decisionID string, outcome *DecisionOutcome) {
	adm.mu.Lock()
	defer adm.mu.Unlock()

	for _, record := range adm.decisionHistory {
		if record.ID == decisionID {
			record.Outcome = outcome
			learning := &LearningFeedback{}
			if outcome.Success {
				learning.Reinforcement = outcome.ActualUtility
			} else {
				learning.Reinforcement = -outcome.ActualUtility
			}
			record.Learning = learning
			adm.updateModelPerformance(record.Domain, outcome)
			adm.logger.Debug("outcome recorded", "decision", decisionID, "success", outcome.Success)
			return
		}
	}
}

func (adm *AdaptiveDecisionMaker) GetDecisionHistory(domain DecisionDomain) []*DecisionRecord {
	adm.mu.RLock()
	defer adm.mu.RUnlock()

	if domain == "" {
		result := make([]*DecisionRecord, len(adm.decisionHistory))
		copy(result, adm.decisionHistory)
		return result
	}

	var filtered []*DecisionRecord
	for _, r := range adm.decisionHistory {
		if r.Domain == domain {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func (adm *AdaptiveDecisionMaker) GetModelPerformance(domain DecisionDomain) *ModelPerformance {
	adm.mu.RLock()
	defer adm.mu.RUnlock()

	key := string(domain)
	if perf, ok := adm.modelPerformance[key]; ok {
		return perf
	}
	return &ModelPerformance{Confidence: 0.5}
}

func (adm *AdaptiveDecisionMaker) LearnFromOutcome(record *DecisionRecord) {
	adm.mu.Lock()
	defer adm.mu.Unlock()

	model := adm.getOrCreateModel(record.Domain)
	if record.Outcome != nil {
		sample := TrainingSample{
			Features:  map[string]interface{}{"domain": record.Domain, "confidence": record.Confidence},
			Target:    record.Outcome.Success,
			Weight:    1.0,
			Timestamp: time.Now(),
			Quality:   record.Confidence,
		}
		model.TrainingData = append(model.TrainingData, sample)
		model.Performance.SampleSize = len(model.TrainingData)

		successes := 0
		for _, s := range model.TrainingData {
			if s.Target == true {
				successes++
			}
		}
		model.Performance.Accuracy = float64(successes) / float64(len(model.TrainingData))
		model.LastUpdated = time.Now()
	}
}

func (adm *AdaptiveDecisionMaker) Shutdown() {
	adm.mu.Lock()
	defer adm.mu.Unlock()

	adm.decisionHistory = nil
	adm.decisionModels = nil
	adm.modelPerformance = nil
	adm.logger.Debug("adaptive decision maker shut down")
}

func (adm *AdaptiveDecisionMaker) generateAlternatives(domain DecisionDomain, ctx *DecisionContext) []*DecisionAlternative {
	model := adm.getOrCreateModel(domain)
	alternatives := []*DecisionAlternative{
		{
			ID:          fmt.Sprintf("alt-%s-proactive", domain),
			Description: "Proactive action based on current context",
			Attributes:  map[string]interface{}{"strategy": "proactive"},
			Feasibility: 0.8,
			Cost:        10.0,
		},
		{
			ID:          fmt.Sprintf("alt-%s-reactive", domain),
			Description: "Reactive response to observed conditions",
			Attributes:  map[string]interface{}{"strategy": "reactive"},
			Feasibility: 0.9,
			Cost:        5.0,
		},
		{
			ID:          fmt.Sprintf("alt-%s-balanced", domain),
			Description: "Balanced approach considering all objectives",
			Attributes:  map[string]interface{}{"strategy": "balanced"},
			Feasibility: 0.85,
			Cost:        7.5,
		},
	}

	if model.Performance != nil && model.Performance.Accuracy > 0.5 {
		alternatives = append(alternatives, &DecisionAlternative{
			ID:          fmt.Sprintf("alt-%s-learned", domain),
			Description: "Learned strategy from historical performance",
			Attributes:  map[string]interface{}{"strategy": "learned", "model_accuracy": model.Performance.Accuracy},
			Feasibility: model.Performance.Accuracy,
			Cost:        6.0,
		})
	}
	return alternatives
}

func (adm *AdaptiveDecisionMaker) getOrCreateModel(domain DecisionDomain) *DecisionModel {
	model, exists := adm.decisionModels[domain]
	if !exists {
		model = &DecisionModel{
			Domain:      domain,
			ModelType:   ModelBayesian,
			Parameters:  map[string]interface{}{"learning_rate": 0.1, "exploration_rate": 0.2},
			Performance: &ModelPerformance{Confidence: 0.5},
			Version:     "1.0.0",
		}
		adm.decisionModels[domain] = model
	}
	return model
}

func (adm *AdaptiveDecisionMaker) updateModelPerformance(domain DecisionDomain, outcome *DecisionOutcome) {
	key := string(domain)
	perf, exists := adm.modelPerformance[key]
	if !exists {
		perf = &ModelPerformance{Confidence: 0.5}
		adm.modelPerformance[key] = perf
	}
	if outcome.Success {
		perf.Accuracy = (perf.Accuracy*float64(perf.SampleSize) + 1.0) / float64(perf.SampleSize+1)
	} else {
		perf.Accuracy = (perf.Accuracy*float64(perf.SampleSize)) / float64(perf.SampleSize+1)
	}
	perf.SampleSize++
	perf.LastEvaluated = time.Now()
	perf.Confidence = 1.0 - 1.0/(2.0+float64(perf.SampleSize))
}
