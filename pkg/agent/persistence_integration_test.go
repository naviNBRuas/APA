package agent

import "testing"

func TestPersistencePlanner(t *testing.T) {
	planner := PersistencePlanner{}
	plan := planner.Plan("apa-agent", 15)
	if plan.SystemdUnit == "" || plan.CronSpec == "" {
		t.Fatalf("expected persistence artifacts")
	}
}
