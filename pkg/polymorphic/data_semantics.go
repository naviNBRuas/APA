package polymorphic

import (
	"errors"
	"sync"
)

// DataEvent encodes an instruction as a data object.
type DataEvent struct {
	Type string
	Data map[string]interface{}
}

// Reducer applies a data event to state.
type Reducer interface {
	Apply(state map[string]interface{}, ev DataEvent) error
}

// ReducerFunc helper.
type ReducerFunc func(map[string]interface{}, DataEvent) error

func (f ReducerFunc) Apply(state map[string]interface{}, ev DataEvent) error { return f(state, ev) }

// DataBus ingests events and drives state transitions without explicit control verbs.
type DataBus struct {
	mu       sync.Mutex
	state    map[string]interface{}
	reducers map[string]Reducer
	history  []DataEvent
}

// NewDataBus creates a bus with empty state.
func NewDataBus() *DataBus {
	return &DataBus{state: make(map[string]interface{}), reducers: make(map[string]Reducer)}
}

// RegisterReducer registers a reducer for an event type.
func (b *DataBus) RegisterReducer(eventType string, r Reducer) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.reducers[eventType] = r
}

// Ingest applies a data event, updating state via the registered reducer.
func (b *DataBus) Ingest(ev DataEvent) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	r, ok := b.reducers[ev.Type]
	if !ok {
		return errors.New("no reducer for event type")
	}

	if err := r.Apply(b.state, ev); err != nil {
		return err
	}
	b.history = append(b.history, ev)
	return nil
}

// Snapshot returns a copy of the current state.
func (b *DataBus) Snapshot() map[string]interface{} {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make(map[string]interface{}, len(b.state))
	for k, v := range b.state {
		out[k] = v
	}
	return out
}

// History returns a copy of events applied.
func (b *DataBus) History() []DataEvent {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]DataEvent, len(b.history))
	copy(out, b.history)
	return out
}
