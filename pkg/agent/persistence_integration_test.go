package agent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPersistencePlanner(t *testing.T) {
	planner := PersistencePlanner{}
	plan := planner.Plan("apa-agent", 15)
	require.NotEmpty(t, plan.SystemdUnit, "expected persistence artifacts")
	require.NotEmpty(t, plan.CronSpec, "expected persistence artifacts")
}
