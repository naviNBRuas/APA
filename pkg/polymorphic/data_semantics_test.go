package polymorphic

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDataBusAppliesEvents(t *testing.T) {
	bus := NewDataBus()
	bus.RegisterReducer("inc", ReducerFunc(func(state map[string]interface{}, ev DataEvent) error {
		cur, _ := state["counter"].(int) //nolint:errcheck
		state["counter"] = cur + 1
		return nil
	}))

	require.NoError(t, bus.Ingest(DataEvent{Type: "inc"}), "ingest failed")
	require.NoError(t, bus.Ingest(DataEvent{Type: "inc"}), "ingest failed")

	snap := bus.Snapshot()
	require.Equal(t, 2, snap["counter"])
}

func TestDataBusUnknownReducer(t *testing.T) {
	bus := NewDataBus()
	err := bus.Ingest(DataEvent{Type: "missing"})
	require.Error(t, err, "expected error for unknown reducer")
}
