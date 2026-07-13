package intelligence

import (
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"sync"
	"time"
)

type ContextAnalyzer struct {
	logger       *slog.Logger
	factors      []ContextFactor
	weightEngine *ContextWeightEngine
	normalizer   *ContextNormalizer

	mu sync.RWMutex
}

type ContextFactor struct {
	Name       string          `yaml:"name"`
	Type       FactorType      `yaml:"type"`
	Weight     float64         `yaml:"weight"`
	Range      FactorRange     `yaml:"range"`
	Importance ImportanceLevel `yaml:"importance"`
	Dynamic    bool            `yaml:"dynamic"`
	UpdateRate time.Duration   `yaml:"update_rate"`
}

type RiskAssessmentEngine struct {
	logger            *slog.Logger
	riskModels        map[RiskType]*RiskModel
	correlationEngine *RiskCorrelationEngine
	mitigationEngine  *RiskMitigationEngine
	portfolioEngine   *RiskPortfolioEngine

	mu sync.RWMutex
}

type UtilityCalculator struct {
	logger              *slog.Logger
	utilityFunctions    map[UtilityType]*UtilityFunction
	multiCriteriaEngine *MultiCriteriaDecisionEngine
	sensitivityAnalyzer *SensitivityAnalyzer

	mu sync.RWMutex
}

type ConsensusBuilder struct {
	logger           *slog.Logger
	consensusMethods []ConsensusMethod
	weightEngine     *ConsensusWeightEngine
	conflictResolver *ConflictResolutionEngine

	mu sync.RWMutex
}

type RiskProfile struct {
	OverallScore  float64              `json:"overall_score"`
	RiskScores    map[RiskType]float64 `json:"risk_scores"`
	Mitigations   []string             `json:"mitigations"`
	Volatility    float64              `json:"volatility"`
	CorrelationId string               `json:"correlation_id"`
}

func NewContextAnalyzer(logger *slog.Logger) *ContextAnalyzer {
	return &ContextAnalyzer{
		logger:       logger,
		factors:      make([]ContextFactor, 0),
		weightEngine: &ContextWeightEngine{},
		normalizer:   &ContextNormalizer{},
	}
}

func NewRiskAssessmentEngine(logger *slog.Logger) *RiskAssessmentEngine {
	return &RiskAssessmentEngine{logger: logger, riskModels: make(map[RiskType]*RiskModel)}
}

func NewUtilityCalculator(logger *slog.Logger) *UtilityCalculator {
	return &UtilityCalculator{
		logger:              logger,
		utilityFunctions:    make(map[UtilityType]*UtilityFunction),
		multiCriteriaEngine: &MultiCriteriaDecisionEngine{},
		sensitivityAnalyzer: &SensitivityAnalyzer{},
	}
}

func NewConsensusBuilder(logger *slog.Logger) *ConsensusBuilder {
	return &ConsensusBuilder{logger: logger}
}

func (ca *ContextAnalyzer) Analyze(ctx map[string]interface{}) *DecisionContext {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	env := make(map[string]interface{})
	for k, v := range ctx {
		env[k] = v
	}
	for _, f := range ca.factors {
		if _, exists := env[f.Name]; !exists {
			env[f.Name] = f.Weight
		}
	}

	decisionCtx := &DecisionContext{
		Environment: env,
		Constraints: make([]Constraint, 0),
		Objectives:  make([]Objective, 0),
		RiskFactors: make([]RiskFactor, 0),
		Historical:  make([]HistoricalDecision, 0),
		RealTime:    make(map[string]interface{}),
	}

	ca.logger.Debug("context analyzed", "factors", len(ca.factors), "env_keys", len(env))
	return decisionCtx
}

func (ca *ContextAnalyzer) UpdateFactor(name string, weight float64) {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	for i, f := range ca.factors {
		if f.Name == name {
			ca.factors[i].Weight = weight
			return
		}
	}
	ca.factors = append(ca.factors, ContextFactor{
		Name:   name,
		Weight: weight,
		Type:   FactorNumerical,
	})
}

func (ra *RiskAssessmentEngine) AssessRisk(alternative *DecisionAlternative, ctx *DecisionContext) *RiskProfile {
	ra.mu.RLock()
	defer ra.mu.RUnlock()

	scores := make(map[RiskType]float64)
	overall := 0.0
	i := 0
	for rt := range ra.riskModels {
		base := rand.Float64() * 0.3
		if alternative.Cost > 0 {
			base += (alternative.Cost / 100.0) * 0.2
		}
		scores[rt] = base
		overall += base
		i++
	}
	if i == 0 {
		for _, rt := range []RiskType{RiskOperational, RiskTechnical, RiskSecurity} {
			base := rand.Float64() * 0.25
			scores[rt] = base
			overall += base
		}
		i = 3
	}

	profile := &RiskProfile{
		OverallScore:  overall / float64(i),
		RiskScores:    scores,
		Mitigations:   make([]string, 0),
		Volatility:    rand.Float64() * 0.15,
		CorrelationId: fmt.Sprintf("risk-%d", time.Now().UnixNano()),
	}

	if profile.OverallScore > 0.5 {
		profile.Mitigations = append(profile.Mitigations, "increase_monitoring")
	}
	if profile.OverallScore > 0.7 {
		profile.Mitigations = append(profile.Mitigations, "activate_contingency")
	}
	return profile
}

func (ra *RiskAssessmentEngine) CalculateRiskScore(factors []RiskFactor) float64 {
	if len(factors) == 0 {
		return 0.0
	}
	total := 0.0
	for range factors {
		total += 0.5
	}
	return math.Min(total/float64(len(factors)), 1.0)
}

func (uc *UtilityCalculator) Calculate(alternative *DecisionAlternative, ctx *DecisionContext) float64 {
	uc.mu.RLock()
	defer uc.mu.RUnlock()

	utility := alternative.Feasibility * 0.4
	if alternative.RiskProfile != nil {
		utility += (1.0 - alternative.RiskProfile.OverallScore) * 0.3
	}
	utility += (1.0 - alternative.Cost/100.0) * 0.3

	utility = math.Max(0.0, math.Min(1.0, utility))
	return utility
}

func (uc *UtilityCalculator) UpdateFunction(utilityType UtilityType, function interface{}) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	uc.utilityFunctions[utilityType] = &UtilityFunction{}
	uc.logger.Debug("utility function updated", "type", utilityType)
}

func (cb *ConsensusBuilder) BuildConsensus(evaluations []*DecisionAlternative) *DecisionAlternative {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if len(evaluations) == 0 {
		return nil
	}

	best := evaluations[0]
	for _, alt := range evaluations[1:] {
		if alt.ExpectedUtility > best.ExpectedUtility {
			best = alt
		}
	}
	return best
}

func (cb *ConsensusBuilder) AddMethod(method ConsensusMethod, weight float64) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	for i, m := range cb.consensusMethods {
		if m == method {
			cb.consensusMethods[i] = method
			return
		}
	}
	cb.consensusMethods = append(cb.consensusMethods, method)
}
