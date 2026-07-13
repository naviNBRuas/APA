package swarm

import (
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func elLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

func TestElasticityScalesUpToDemand(t *testing.T) {
	initial := map[CapacityClass]int{CapacityCloud: 2, CapacityEdge: 1, CapacityResidential: 1}
	em := NewElasticityManager(elLogger(), initial, 0.7)
	actions := em.ObserveDemand(10)
	require.Len(t, actions, 1)
	assert.Greater(t, actions[0].Delta, 0, "expected positive delta for scale-up")
	assert.Equal(t, CapacityCloud, actions[0].Class, "expected cloud capacity class")

	em.Apply(actions)
	snap := em.Snapshot()
	assert.Greater(t, snap[CapacityCloud], initial[CapacityCloud], "expected cloud capacity to increase")
}

func TestElasticityScalesDownWhenIdle(t *testing.T) {
	initial := map[CapacityClass]int{CapacityCloud: 4, CapacityEdge: 2, CapacityResidential: 1}
	em := NewElasticityManager(elLogger(), initial, 0.7)
	actions := em.ObserveDemand(1)
	require.NotEmpty(t, actions, "expected scale-down actions when idle")

	em.Apply(actions)
	snap := em.Snapshot()
	assert.Less(t, snap[CapacityCloud], initial[CapacityCloud], "expected cloud capacity to decrease")
}

func TestElasticityZeroDemand(t *testing.T) {
	em := NewElasticityManager(elLogger(), map[CapacityClass]int{CapacityCloud: 2}, 0.7)
	actions := em.ObserveDemand(0)
	require.NotEmpty(t, actions, "expected scale-down actions at zero demand")
}

func TestElasticityEmptyInitialCapacity(t *testing.T) {
	em := NewElasticityManager(elLogger(), map[CapacityClass]int{}, 0.7)
	actions := em.ObserveDemand(5)
	require.NotEmpty(t, actions, "expected actions even with empty initial capacity")
}
