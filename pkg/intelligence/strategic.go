package intelligence

import (
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"sync"
	"time"
)

type StrategicPlanningEngine struct {
	logger            *slog.Logger
	config            StrategicConfig
	planningModels    map[PlanningHorizon]*PlanningModel
	goalHierarchy     *GoalHierarchy
	resourceAllocator *StrategicResourceAllocator
	scenarioPlanner   *ScenarioPlanningEngine
	planExecutor      *PlanExecutionEngine

	mu             sync.RWMutex
	strategicPlans map[string]*StrategicPlan
	planProgress   map[string]*PlanProgress
}

type PlanningModel struct {
	Horizon      PlanningHorizon    `json:"horizon"`
	Model        interface{}        `json:"model"`
	Goals        []StrategicGoal    `json:"goals"`
	Resources    ResourceAllocation `json:"resources"`
	Timeline     time.Duration      `json:"timeline"`
	Risks        []StrategicRisk    `json:"risks"`
	Dependencies []Dependency       `json:"dependencies"`
}

type GoalHierarchy struct {
	logger           *slog.Logger
	rootGoals        []*GoalNode
	priorityEngine   *GoalPriorityEngine
	conflictResolver *GoalConflictResolver
	progressTracker  *GoalProgressTracker
	mu               sync.RWMutex
}

type StrategicResourceAllocator struct{}

type ScenarioPlanningEngine struct{}

type PlanExecutionEngine struct{}

type StrategicPlan struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Horizon      PlanningHorizon `json:"horizon"`
	Goals        []StrategicGoal `json:"goals"`
	Initiatives  []Initiative    `json:"initiatives"`
	Timeline     time.Duration   `json:"timeline"`
	Budget       float64         `json:"budget"`
	Risks        []StrategicRisk `json:"risks"`
	Dependencies []Dependency    `json:"dependencies"`
	Status       PlanStatus      `json:"status"`
	Created      time.Time       `json:"created"`
	Modified     time.Time       `json:"modified"`
	Owner        string          `json:"owner"`
	Stakeholders []string        `json:"stakeholders"`
}

type PlanProgress struct {
	PlanID             string             `json:"plan_id"`
	OverallProgress    float64            `json:"overall_progress"`
	GoalProgress       map[string]float64 `json:"goal_progress"`
	MilestonesReached  int                `json:"milestones_reached"`
	TotalMilestones    int                `json:"total_milestones"`
	BudgetSpent        float64            `json:"budget_spent"`
	BudgetAllocated    float64            `json:"budget_allocated"`
	RisksMitigated     int                `json:"risks_mitigated"`
	TotalRisks         int                `json:"total_risks"`
	LastUpdated        time.Time          `json:"last_updated"`
	OnTrack            bool               `json:"on_track"`
}

func NewStrategicPlanningEngine(logger *slog.Logger, config StrategicConfig) *StrategicPlanningEngine {
	return &StrategicPlanningEngine{
		logger:            logger,
		config:            config,
		planningModels:    make(map[PlanningHorizon]*PlanningModel),
		goalHierarchy:     NewGoalHierarchy(logger),
		resourceAllocator: NewStrategicResourceAllocator(logger),
		scenarioPlanner:   NewScenarioPlanningEngine(logger),
		planExecutor:      NewPlanExecutionEngine(logger),
		strategicPlans:    make(map[string]*StrategicPlan),
		planProgress:      make(map[string]*PlanProgress),
	}
}

func NewGoalHierarchy(logger *slog.Logger) *GoalHierarchy {
	return &GoalHierarchy{logger: logger, rootGoals: make([]*GoalNode, 0)}
}

func NewStrategicResourceAllocator(logger *slog.Logger) *StrategicResourceAllocator {
	return &StrategicResourceAllocator{}
}

func NewScenarioPlanningEngine(logger *slog.Logger) *ScenarioPlanningEngine {
	return &ScenarioPlanningEngine{}
}

func NewPlanExecutionEngine(logger *slog.Logger) *PlanExecutionEngine {
	return &PlanExecutionEngine{}
}

func (spe *StrategicPlanningEngine) Shutdown() {
	spe.mu.Lock()
	defer spe.mu.Unlock()
	spe.strategicPlans = nil
	spe.planProgress = nil
	spe.planningModels = nil
}

func (spe *StrategicPlanningEngine) CreatePlan(initiative interface{}) *StrategicPlan {
	spe.mu.Lock()
	defer spe.mu.Unlock()

	plan := &StrategicPlan{
		ID:           fmt.Sprintf("plan-%d", time.Now().UnixNano()),
		Name:         fmt.Sprintf("Strategic Plan %d", len(spe.strategicPlans)+1),
		Description:  "Automatically generated strategic plan",
		Horizon:      HorizonMediumTerm,
		Goals:        make([]StrategicGoal, 0),
		Initiatives:  make([]Initiative, 0),
		Risks:        make([]StrategicRisk, 0),
		Dependencies: make([]Dependency, 0),
		Status:       PlanStatus("active"),
		Created:      time.Now().UTC(),
		Modified:     time.Now().UTC(),
		Owner:        "system",
		Stakeholders: []string{"system"},
		Timeline:     90 * 24 * time.Hour,
	}

	if initMap, ok := initiative.(map[string]interface{}); ok {
		if name, ok := initMap["name"].(string); ok {
			plan.Name = name
		}
		if desc, ok := initMap["description"].(string); ok {
			plan.Description = desc
		}
		if horizon, ok := initMap["horizon"].(string); ok {
			plan.Horizon = PlanningHorizon(horizon)
		}
		if goals, ok := initMap["goals"].([]StrategicGoal); ok {
			plan.Goals = goals
		}
		if budget, ok := initMap["budget"].(float64); ok {
			plan.Budget = budget
		}
	}

	if len(plan.Goals) == 0 {
		plan.Goals = append(plan.Goals, StrategicGoal{})
	}

	spe.strategicPlans[plan.ID] = plan

	progress := &PlanProgress{
		PlanID:            plan.ID,
		OverallProgress:   0,
		GoalProgress:      make(map[string]float64),
		TotalMilestones:   5,
		BudgetAllocated:   plan.Budget,
		LastUpdated:       time.Now().UTC(),
		OnTrack:           true,
	}
	spe.planProgress[plan.ID] = progress

	spe.planningModels[plan.Horizon] = &PlanningModel{
		Horizon: plan.Horizon,
		Goals:   plan.Goals,
		Timeline: plan.Timeline,
	}

	return plan
}

func (spe *StrategicPlanningEngine) GetCurrentPlans() []*StrategicPlan {
	spe.mu.RLock()
	defer spe.mu.RUnlock()
	plans := make([]*StrategicPlan, 0, len(spe.strategicPlans))
	for _, p := range spe.strategicPlans {
		plans = append(plans, p)
	}
	return plans
}

func (spe *StrategicPlanningEngine) UpdatePlanProgress(planID string, progress *PlanProgress) {
	spe.mu.Lock()
	defer spe.mu.Unlock()

	existing, ok := spe.planProgress[planID]
	if !ok {
		spe.planProgress[planID] = progress
		return
	}

	if progress != nil {
		if progress.OverallProgress > 0 {
			existing.OverallProgress = math.Min(1.0, progress.OverallProgress)
		}
		if progress.MilestonesReached > 0 {
			existing.MilestonesReached = progress.MilestonesReached
		}
		if progress.TotalMilestones > 0 {
			existing.TotalMilestones = progress.TotalMilestones
		}
		if progress.BudgetSpent > 0 {
			existing.BudgetSpent = progress.BudgetSpent
		}
		if progress.OnTrack {
			existing.OnTrack = progress.OnTrack
		}
		existing.LastUpdated = time.Now().UTC()
	}

	if plan, ok := spe.strategicPlans[planID]; ok {
		plan.Modified = time.Now().UTC()
	}

	progressAfter := spe.planProgress[planID]
	if progressAfter.TotalMilestones > 0 {
		progressAfter.OverallProgress = float64(progressAfter.MilestonesReached) / float64(progressAfter.TotalMilestones)
	}
}

func (spe *StrategicPlanningEngine) GetPlan(planID string) *StrategicPlan {
	spe.mu.RLock()
	defer spe.mu.RUnlock()
	plan, ok := spe.strategicPlans[planID]
	if !ok {
		return nil
	}
	return plan
}

func (spe *StrategicPlanningEngine) GetPlanProgress(planID string) *PlanProgress {
	spe.mu.RLock()
	defer spe.mu.RUnlock()
	progress, ok := spe.planProgress[planID]
	if !ok {
		return nil
	}
	return progress
}

func (spe *StrategicPlanningEngine) EvaluateRisk(planID string) []StrategicRisk {
	spe.mu.RLock()
	defer spe.mu.RUnlock()

	plan, ok := spe.strategicPlans[planID]
	if !ok {
		return nil
	}

	risks := make([]StrategicRisk, len(plan.Risks))
	copy(risks, plan.Risks)

	if len(risks) == 0 {
		for i := 0; i < 3; i++ {
			risks = append(risks, StrategicRisk{})
		}
	}
	return risks
}

func (gh *GoalHierarchy) AddRootGoal(goal *GoalNode) {
	gh.mu.Lock()
	defer gh.mu.Unlock()
	gh.rootGoals = append(gh.rootGoals, goal)
}

func (gh *GoalHierarchy) GetRootGoals() []*GoalNode {
	gh.mu.RLock()
	defer gh.mu.RUnlock()
	goals := make([]*GoalNode, len(gh.rootGoals))
	copy(goals, gh.rootGoals)
	return goals
}

func (gh *GoalHierarchy) Prioritize() []*GoalNode {
	gh.mu.RLock()
	defer gh.mu.RUnlock()
	if len(gh.rootGoals) <= 1 {
		return gh.rootGoals
	}
	prioritized := make([]*GoalNode, len(gh.rootGoals))
	copy(prioritized, gh.rootGoals)
	rand.Shuffle(len(prioritized), func(i, j int) {
		prioritized[i], prioritized[j] = prioritized[j], prioritized[i]
	})
	return prioritized
}

func CreateStrategicInitiative(name string, description string, horizon PlanningHorizon) map[string]interface{} {
	return map[string]interface{}{
		"name":        name,
		"description": description,
		"horizon":     string(horizon),
		"goals":       []StrategicGoal{},
		"created":     time.Now().UTC(),
	}
}
