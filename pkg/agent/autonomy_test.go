package agent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAutonomousStateMachineBudget(t *testing.T) {
	sm := NewAutonomousStateMachine(TaskBudget{MaxPerInterval: 2, Interval: time.Second})
	now := time.Now()
	require.True(t, sm.Tick(now, 0), "expected first tick to allow")
	require.False(t, sm.Tick(now, 3), "budget should block when executed exceeds limit")
}
