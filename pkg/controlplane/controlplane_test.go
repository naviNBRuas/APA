package controlplane

import (
	"context"
	"sync"
	"testing"
	"time"

	"log/slog"
)

type mockTransport struct {
	id  string
	bus *topicBus
}

func (m *mockTransport) Publish(ctx context.Context, topic string, payload []byte) error {
	return m.bus.publish(topic, payload)
}

func (m *mockTransport) Subscribe(ctx context.Context, topic string) (<-chan []byte, error) {
	return m.bus.subscribe(topic), nil
}

func (m *mockTransport) LocalID() string { return m.id }

type topicBus struct {
	mu   sync.RWMutex
	subs map[string][]chan []byte
}

func newTopicBus() *topicBus {
	return &topicBus{subs: make(map[string][]chan []byte)}
}

func (b *topicBus) publish(topic string, payload []byte) error {
	b.mu.RLock()
	chans := append([]chan []byte(nil), b.subs[topic]...)
	b.mu.RUnlock()
	for _, ch := range chans {
		// best-effort non-blocking send
		select {
		case ch <- payload:
		default:
		}
	}
	return nil
}

func (b *topicBus) subscribe(topic string) <-chan []byte {
	ch := make(chan []byte, 8)
	b.mu.Lock()
	b.subs[topic] = append(b.subs[topic], ch)
	b.mu.Unlock()
	return ch
}

func TestLeaderlessReplication(t *testing.T) {
	bus := newTopicBus()
	logger := slog.Default()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t1 := &mockTransport{id: "nodeA", bus: bus}
	t2 := &mockTransport{id: "nodeB", bus: bus}

	cfg := Config{Mode: "leaderless", PartialStateLimit: 16, EntryTTL: time.Second}
	cp1 := New(logger, t1, cfg)
	cp2 := New(logger, t2, cfg)

	if err := cp1.Start(ctx); err != nil {
		t.Fatalf("start cp1: %v", err)
	}
	if err := cp2.Start(ctx); err != nil {
		t.Fatalf("start cp2: %v", err)
	}

	if err := cp1.Set(ctx, "alpha", []byte("v1"), time.Second); err != nil {
		t.Fatalf("set: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if val, ok := cp2.Get("alpha"); ok {
			if string(val) != "v1" {
				t.Fatalf("unexpected value: %s", string(val))
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("replication did not arrive")
}

func TestTTLExpiry(t *testing.T) {
	bus := newTopicBus()
	logger := slog.Default()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cp := New(logger, &mockTransport{id: "nodeA", bus: bus}, Config{EntryTTL: 100 * time.Millisecond})
	if err := cp.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}

	if err := cp.Set(ctx, "temp", []byte("x"), 50*time.Millisecond); err != nil {
		t.Fatalf("set: %v", err)
	}
	if _, ok := cp.Get("temp"); !ok {
		t.Fatalf("expected value present")
	}
	time.Sleep(200 * time.Millisecond)
	if _, ok := cp.Get("temp"); ok {
		t.Fatalf("expected value expired")
	}
}

func TestElectedModeRoutesThroughLeader(t *testing.T) {
	bus := newTopicBus()
	logger := slog.Default()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := Config{Mode: "elected", EntryTTL: time.Second, SyncInterval: 100 * time.Millisecond}

	leader := New(logger, &mockTransport{id: "leader", bus: bus}, cfg)
	leader.SetLeaderRank(10)
	follower := New(logger, &mockTransport{id: "follower", bus: bus}, cfg)
	follower.SetLeaderRank(1)

	if err := leader.Start(ctx); err != nil {
		t.Fatalf("leader start: %v", err)
	}
	if err := follower.Start(ctx); err != nil {
		t.Fatalf("follower start: %v", err)
	}

	// wait for election
	ticker := time.NewTicker(50 * time.Millisecond)
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		l := leader.IsLeader()
		f := follower.IsLeader()
		if l && !f {
			break
		}
		// print status
		t.Logf("election status leader=%v follower=%v", l, f)
		<-ticker.C
	}
	ticker.Stop()
	if !leader.IsLeader() {
		t.Fatalf("leader should win election")
	}
	if follower.IsLeader() {
		t.Fatalf("follower should not be leader")
	}

	if err := follower.Set(ctx, "k", []byte("v"), time.Second); err != nil {
		t.Fatalf("follower set: %v", err)
	}

	deadline2 := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline2) {
		if val, ok := leader.Get("k"); ok {
			if string(val) != "v" {
				t.Fatalf("unexpected value %s", string(val))
			}
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("leader did not apply follower update")
}
