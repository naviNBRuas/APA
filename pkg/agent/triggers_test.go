package agent

import (
	"testing"
	"time"
)

func TestTriggerConditions(t *testing.T) {
	now := time.Now()
	cond := TriggerConditions{After: now.Add(-time.Minute), Before: now.Add(time.Minute), MinPeers: 1, RequiredInteractions: 1, RequireNetworkIdle: true, RequireExternal: true, Cooldown: time.Minute}
	state := ActivationState{Now: now, PeerCount: 2, UserInteractions: 1, NetworkIdle: true, ExternalSignal: true, LastExecution: now.Add(-2 * time.Minute)}
	if !cond.ShouldActivate(state) {
		t.Fatalf("expected activation")
	}
	state.PeerCount = 0
	if cond.ShouldActivate(state) {
		t.Fatalf("expected peer gate to block")
	}
}
