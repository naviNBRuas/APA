package agent

import "time"

// TaskBudget controls resource usage for autonomous logic.
type TaskBudget struct {
	MaxPerInterval int
	Interval       time.Duration
}

// AutonomousStateMachine drives execution frequency and task selection without external input.
type AutonomousStateMachine struct {
	budget   TaskBudget
	lastTick time.Time
}

// NewAutonomousStateMachine builds a state machine with the given budget.
func NewAutonomousStateMachine(budget TaskBudget) *AutonomousStateMachine {
	if budget.Interval <= 0 {
		budget.Interval = time.Minute
	}
	if budget.MaxPerInterval == 0 {
		budget.MaxPerInterval = 1
	}
	return &AutonomousStateMachine{budget: budget}
}

// Tick decides whether another task should run based on time and budget.
func (sm *AutonomousStateMachine) Tick(now time.Time, executed int) bool {
	if now.IsZero() {
		now = time.Now()
	}
	if sm.lastTick.IsZero() || now.Sub(sm.lastTick) >= sm.budget.Interval {
		sm.lastTick = now
		return executed < sm.budget.MaxPerInterval
	}
	return executed < sm.budget.MaxPerInterval
}
