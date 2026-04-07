package swarm

import (
	"io"
	"log/slog"
	"testing"
)

func elLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

func TestElasticityScalesUpToDemand(t *testing.T) {
	initial := map[CapacityClass]int{CapacityCloud: 2, CapacityEdge: 1, CapacityResidential: 1}
	em := NewElasticityManager(elLogger(), initial, 0.7)
	actions := em.ObserveDemand(10)
	if len(actions) != 1 || actions[0].Delta <= 0 || actions[0].Class != CapacityCloud {
		t.Fatalf("expected cloud scale-up action, got %v", actions)
	}
	em.Apply(actions)
	snap := em.Snapshot()
	if snap[CapacityCloud] <= initial[CapacityCloud] {
		t.Fatalf("expected cloud capacity to increase, snapshot=%v", snap)
	}
}

func TestElasticityScalesDownWhenIdle(t *testing.T) {
	initial := map[CapacityClass]int{CapacityCloud: 4, CapacityEdge: 2, CapacityResidential: 1}
	em := NewElasticityManager(elLogger(), initial, 0.7)
	actions := em.ObserveDemand(1)
	if len(actions) == 0 {
		t.Fatalf("expected scale-down actions when idle")
	}
	em.Apply(actions)
	snap := em.Snapshot()
	if snap[CapacityCloud] >= initial[CapacityCloud] {
		t.Fatalf("expected cloud capacity to decrease, snapshot=%v", snap)
	}
}
