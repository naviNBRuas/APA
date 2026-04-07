package agent

import (
	"testing"
	"time"
)

func TestAutonomousStateMachineBudget(t *testing.T) {
	sm := NewAutonomousStateMachine(TaskBudget{MaxPerInterval: 2, Interval: time.Second})
	now := time.Now()
	if !sm.Tick(now, 0) {
		t.Fatalf("expected first tick to allow")
	}
	if sm.Tick(now, 3) {
		t.Fatalf("budget should block when executed exceeds limit")
	}
}
