package polymorphic

import "testing"

func TestDataBusAppliesEvents(t *testing.T) {
	bus := NewDataBus()
	bus.RegisterReducer("inc", ReducerFunc(func(state map[string]interface{}, ev DataEvent) error {
		cur, _ := state["counter"].(int)
		state["counter"] = cur + 1
		return nil
	}))

	if err := bus.Ingest(DataEvent{Type: "inc"}); err != nil {
		t.Fatalf("ingest failed: %v", err)
	}
	if err := bus.Ingest(DataEvent{Type: "inc"}); err != nil {
		t.Fatalf("ingest failed: %v", err)
	}

	snap := bus.Snapshot()
	if snap["counter"].(int) != 2 {
		t.Fatalf("expected counter 2, got %v", snap["counter"])
	}
}

func TestDataBusUnknownReducer(t *testing.T) {
	bus := NewDataBus()
	if err := bus.Ingest(DataEvent{Type: "missing"}); err == nil {
		t.Fatalf("expected error for unknown reducer")
	}
}
