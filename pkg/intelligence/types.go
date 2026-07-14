// Package intelligence provides advanced adaptive algorithms and decision-making capabilities.
package intelligence

import (
	"time"

	"github.com/shopspring/decimal"
)

// DecisionDomain represents different areas of decision-making.
type DecisionDomain string

const (
	DecisionNetwork     DecisionDomain = "network"
	DecisionResource    DecisionDomain = "resource"
	DecisionSecurity    DecisionDomain = "security"
	DecisionPerformance DecisionDomain = "performance"
	DecisionMaintenance DecisionDomain = "maintenance"
	DecisionScaling     DecisionDomain = "scaling"
	DecisionRouting     DecisionDomain = "routing"
	DecisionScheduling  DecisionDomain = "scheduling"
)

type ModelType string

const (
	ModelSupervised    ModelType = "supervised"
	ModelUnsupervised  ModelType = "unsupervised"
	ModelReinforcement ModelType = "reinforcement"
	ModelEnsemble      ModelType = "ensemble"
	ModelDeepLearning  ModelType = "deep_learning"
	ModelBayesian      ModelType = "bayesian"
)

type AlgorithmType string

const (
	AlgorithmRandomForest     AlgorithmType = "random_forest"
	AlgorithmNeuralNetwork    AlgorithmType = "neural_network"
	AlgorithmSVM              AlgorithmType = "svm"
	AlgorithmGradientBoosting AlgorithmType = "gradient_boosting"
	AlgorithmKMeans           AlgorithmType = "k_means"
	AlgorithmPCA              AlgorithmType = "pca"
	AlgorithmLSTM             AlgorithmType = "lstm"
	AlgorithmQLearning        AlgorithmType = "q_learning"
)

type FactorType string

const (
	FactorNumerical   FactorType = "numerical"
	FactorCategorical FactorType = "categorical"
	FactorBoolean     FactorType = "boolean"
	FactorTemporal    FactorType = "temporal"
	FactorSpatial     FactorType = "spatial"
)

type ImportanceLevel string

const (
	ImportanceCritical ImportanceLevel = "critical"
	ImportanceHigh     ImportanceLevel = "high"
	ImportanceMedium   ImportanceLevel = "medium"
	ImportanceLow      ImportanceLevel = "low"
)

type RiskType string

const (
	RiskOperational  RiskType = "operational"
	RiskSecurity     RiskType = "security"
	RiskFinancial    RiskType = "financial"
	RiskReputational RiskType = "reputational"
	RiskCompliance   RiskType = "compliance"
	RiskTechnical    RiskType = "technical"
)

type UtilityType string

const (
	UtilityLinear      UtilityType = "linear"
	UtilityExponential UtilityType = "exponential"
	UtilityLogarithmic UtilityType = "logarithmic"
	UtilityQuadratic   UtilityType = "quadratic"
	UtilitySigmoid     UtilityType = "sigmoid"
)

type ConsensusMethod string

const (
	ConsensusVoting     ConsensusMethod = "voting"
	ConsensusWeighted   ConsensusMethod = "weighted"
	ConsensusBayesian   ConsensusMethod = "bayesian"
	ConsensusGameTheory ConsensusMethod = "game_theory"
	ConsensusFuzzyLogic ConsensusMethod = "fuzzy_logic"
)

type DeploymentStatus string

const (
	DeploymentDevelopment DeploymentStatus = "development"
	DeploymentTesting     DeploymentStatus = "testing"
	DeploymentStaging     DeploymentStatus = "staging"
	DeploymentProduction  DeploymentStatus = "production"
	DeploymentRetired     DeploymentStatus = "retired"
)

type ForecastType string

const (
	ForecastDemand      ForecastType = "demand"
	ForecastPerformance ForecastType = "performance"
	ForecastResource    ForecastType = "resource"
	ForecastFailure     ForecastType = "failure"
	ForecastSecurity    ForecastType = "security"
	ForecastMarket      ForecastType = "market"
)

type BehaviorType string

const (
	BehaviorNetwork     BehaviorType = "network"
	BehaviorResource    BehaviorType = "resource"
	BehaviorUser        BehaviorType = "user"
	BehaviorSystem      BehaviorType = "system"
	BehaviorSecurity    BehaviorType = "security"
	BehaviorPerformance BehaviorType = "performance"
)

type PatternType string

const (
	PatternSequential PatternType = "sequential"
	PatternTemporal   PatternType = "temporal"
	PatternSpatial    PatternType = "spatial"
	PatternBehavioral PatternType = "behavioral"
	PatternAnomalous  PatternType = "anomalous"
	PatternRecurring  PatternType = "recurring"
)

type OptimizationType string

const (
	OptimizationLinear         OptimizationType = "linear"
	OptimizationNonlinear      OptimizationType = "nonlinear"
	OptimizationInteger        OptimizationType = "integer"
	OptimizationMultiObjective OptimizationType = "multi_objective"
	OptimizationGenetic        OptimizationType = "genetic"
	OptimizationSwarm          OptimizationType = "swarm"
)

type OptimizationAlgorithm string

const (
	AlgorithmGradientDescent    OptimizationAlgorithm = "gradient_descent"
	AlgorithmGenetic            OptimizationAlgorithm = "genetic_algorithm"
	AlgorithmSimulatedAnnealing OptimizationAlgorithm = "simulated_annealing"
	AlgorithmParticleSwarm      OptimizationAlgorithm = "particle_swarm"
	AlgorithmAntColony          OptimizationAlgorithm = "ant_colony"
	AlgorithmTabuSearch         OptimizationAlgorithm = "tabu_search"
)

type PlanningHorizon string

const (
	HorizonShortTerm  PlanningHorizon = "short_term"
	HorizonMediumTerm PlanningHorizon = "medium_term"
	HorizonLongTerm   PlanningHorizon = "long_term"
	HorizonStrategic  PlanningHorizon = "strategic"
)

type AnomalyType string

const (
	AnomalyStatistical  AnomalyType = "statistical"
	AnomalyBehavioral   AnomalyType = "behavioral"
	AnomalyContextual   AnomalyType = "contextual"
	AnomalyCollective   AnomalyType = "collective"
	AnomalyConceptDrift AnomalyType = "concept_drift"
)

type AnomalyAlgorithm string

const (
	AlgorithmIsolationForest AnomalyAlgorithm = "isolation_forest"
	AlgorithmOneClassSVM     AnomalyAlgorithm = "one_class_svm"
	AlgorithmAutoencoder     AnomalyAlgorithm = "autoencoder"
	AlgorithmLOF             AnomalyAlgorithm = "lof"
	AlgorithmARIMA           AnomalyAlgorithm = "arima"
	AlgorithmKalmanFilter    AnomalyAlgorithm = "kalman_filter"
)

type LearningEventType string

const (
	LearningTraining   LearningEventType = "training"
	LearningValidation LearningEventType = "validation"
	LearningDeployment LearningEventType = "deployment"
	LearningRetraining LearningEventType = "retraining"
	LearningEvaluation LearningEventType = "evaluation"
)

type PlanStatus string

type AnomalySeverity string

const (
	AnomalySeverityLow      AnomalySeverity = "low"
	AnomalySeverityMedium   AnomalySeverity = "medium"
	AnomalySeverityHigh     AnomalySeverity = "high"
	AnomalySeverityCritical AnomalySeverity = "critical"
)

type LearningFeedback struct {
	Reinforcement   float64         `json:"reinforcement"`
	ModelUpdates    []string        `json:"model_updates"`
	StrategyChanges []string        `json:"strategy_changes"`
	KnowledgeGain   decimal.Decimal `json:"knowledge_gain"`
}

type ForecastValue struct {
	Timestamp  time.Time   `json:"timestamp"`
	Value      interface{} `json:"value"`
	LowerBound interface{} `json:"lower_bound"`
	UpperBound interface{} `json:"upper_bound"`
	Confidence float64     `json:"confidence"`
}

type BehaviorPattern struct {
	ID           string                 `json:"id"`
	Type         PatternType            `json:"type"`
	Sequence     []interface{}          `json:"sequence"`
	Frequency    float64                `json:"frequency"`
	Duration     time.Duration          `json:"duration"`
	Context      map[string]interface{} `json:"context"`
	Importance   float64                `json:"importance"`
	LastObserved time.Time              `json:"last_observed"`
}

type KnowledgeBase struct {
	Facts         map[string]*Fact         `json:"facts"`
	Rules         map[string]*Rule         `json:"rules"`
	Concepts      map[string]*Concept      `json:"concepts"`
	Relationships map[string]*Relationship `json:"relationships"`
	Theories      map[string]*Theory       `json:"theories"`
	LastUpdated   time.Time                `json:"last_updated"`
	Version       string                   `json:"version"`
}

type ExperienceRecord struct {
	ID         string                 `json:"id"`
	Timestamp  time.Time              `json:"timestamp"`
	Context    map[string]interface{} `json:"context"`
	Action     string                 `json:"action"`
	Outcome    interface{}            `json:"outcome"`
	Reward     float64                `json:"reward"`
	Learning   *LearningInsight       `json:"learning"`
	Success    bool                   `json:"success"`
	Confidence float64                `json:"confidence"`
}

type DecisionMakingConfig struct{}
type LearningConfig struct{}
type PredictiveConfig struct{}
type BehavioralConfig struct{}
type OptimizationConfig struct{}
type StrategicConfig struct{}
type AnomalyDetectionConfig struct{}
type FactorRange struct{}
type TrainingJob struct {
	ID        string `json:"id"`
	ModelRef  string `json:"model_ref"`
	Status    string `json:"status"`
	Progress  float64 `json:"progress"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Error     string `json:"error,omitempty"`
}
type FeaturePipeline struct{}
type FeatureEngineeringRule struct {
	Name   string                 `json:"name"`
	Params map[string]interface{} `json:"params"`
}
type TimeSeriesModel struct{}
type PatternTemplate struct{}
type BehaviorBaseline struct{}
type BehaviorThresholds struct{}
type ConstraintDefinition struct{}
type ObjectiveFunction struct{}
type Constraint struct{}
type GoalNode struct{}
type Dependency struct{}
type StrategicGoal struct{}
type ResourceAllocation struct{}
type StrategicRisk struct{}
type AnomalySample struct{}
type FusionMethod struct{}
type Benefit struct{}
type UnintendedEffect struct{}
type OutcomeFeedback struct{}
type RiskFactor struct {
	Name  string  `json:"name"`
	Score float64 `json:"score"`
}
type HistoricalDecision struct{}
type Objective struct{}
type Variable struct{}
type Tradeoff struct{}
type Initiative struct{}
type Evidence struct{}
type ImpactAssessment struct{}
type Resolution struct{}
type Fact struct{}
type Rule struct{}
type Concept struct{}
type Relationship struct{}
type Theory struct{}
type ContextWeightEngine struct{}
type ContextNormalizer struct{}
type RiskModel struct{}
type RiskCorrelationEngine struct{}
type RiskMitigationEngine struct{}
type RiskPortfolioEngine struct{}
type UtilityFunction struct{}
type MultiCriteriaDecisionEngine struct{}
type SensitivityAnalyzer struct{}
type ConsensusWeightEngine struct{}
type ModelSelectionEngine struct{}
type EnsembleLearningSystem struct{}
type ConflictResolutionEngine struct{}
type TrainingScheduler struct{}
type TrainingResourceManager struct{}
type ModelValidationEngine struct{}
type FeatureSelectionEngine struct{}
type FeatureTransformationEngine struct{}
type TimeSeriesDecompositionEngine struct{}
type LearningResult struct{}
type StationarityEngine struct{}
type ForecastingEngine struct{}
type TimeSeriesAnomalyDetector struct{}
type PatternMatcher struct{}
type SequenceAnalyzer struct{}
type SimilarityEngine struct{}
type PatternClusteringEngine struct{}
type PatternMatchingAlgorithm struct{}
type PatternSimilarityEngine struct{}
type PatternEvolutionTracker struct{}
type OptimizationPerformance struct{}
type ConstraintValidationEngine struct{}
type Deviation struct{}
type AnomalyFlag struct{}
type ConstraintRelaxationEngine struct{}
type ConstraintPrioritizationEngine struct{}
type GoalPriorityEngine struct{}
type GoalConflictResolver struct{}
type GoalProgressTracker struct{}
type FusionWeightEngine struct{}
type FusionConfidenceEngine struct{}
type FusionDecisionEngine struct{}
type LearningInsight struct{}
