package agent

import "time"

// ActivationState reflects runtime signals used to gate execution.
type ActivationState struct {
	Now              time.Time
	PeerCount        int
	UserInteractions int
	NetworkIdle      bool
	ExternalSignal   bool
	LastExecution    time.Time
}

// TriggerConditions define when execution should proceed.
type TriggerConditions struct {
	After                time.Time
	Before               time.Time
	MinPeers             int
	RequiredInteractions int
	RequireNetworkIdle   bool
	RequireExternal      bool
	Cooldown             time.Duration
}

// ShouldActivate evaluates the conditions against the current state.
func (c TriggerConditions) ShouldActivate(state ActivationState) bool {
	now := state.Now
	if now.IsZero() {
		now = time.Now()
	}
	if !c.After.IsZero() && now.Before(c.After) {
		return false
	}
	if !c.Before.IsZero() && now.After(c.Before) {
		return false
	}
	if c.MinPeers > 0 && state.PeerCount < c.MinPeers {
		return false
	}
	if c.RequiredInteractions > 0 && state.UserInteractions < c.RequiredInteractions {
		return false
	}
	if c.RequireNetworkIdle && !state.NetworkIdle {
		return false
	}
	if c.RequireExternal && !state.ExternalSignal {
		return false
	}
	if c.Cooldown > 0 && !state.LastExecution.IsZero() && now.Sub(state.LastExecution) < c.Cooldown {
		return false
	}
	return true
}
