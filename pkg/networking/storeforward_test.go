package networking

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

func sfLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestEnqueueAndNextForPeerMarksSeen(t *testing.T) {
	sf := NewStoreAndForward(sfLogger(), 10, time.Minute)
	env := sf.Enqueue([]byte("hello"), 3)

	out := sf.NextForPeer(peer.ID("peer-1"), 5)
	if len(out) != 1 {
		t.Fatalf("expected one envelope, got %d", len(out))
	}
	if out[0].ID != env.ID {
		t.Fatalf("unexpected envelope id")
	}

	// second call should not return same envelope for same peer
	out2 := sf.NextForPeer(peer.ID("peer-1"), 5)
	if len(out2) != 0 {
		t.Fatalf("expected no envelopes after seen, got %d", len(out2))
	}
}

func TestExpireAndEvict(t *testing.T) {
	sf := NewStoreAndForward(sfLogger(), 2, 10*time.Millisecond)
	sf.Enqueue([]byte("a"), 2)
	sf.Enqueue([]byte("b"), 2)
	sf.Enqueue([]byte("c"), 2) // should evict oldest to stay within cache after prune attempt

	time.Sleep(15 * time.Millisecond)
	// trigger prune
	_ = sf.NextForPeer(peer.ID("p"), 10)
	if len(sf.tasks) > 2 {
		t.Fatalf("expected cache to stay within limit, got %d", len(sf.tasks))
	}
}

func TestMaxHopsRespected(t *testing.T) {
	sf := NewStoreAndForward(sfLogger(), 10, time.Minute)
	env := sf.Enqueue([]byte("x"), 1)

	out := sf.NextForPeer(peer.ID("peer-1"), 10)
	if len(out) != 1 {
		t.Fatalf("expected envelope available")
	}
	// push it beyond max hops
	sf.Receive(TaskEnvelope{ID: env.ID, Payload: env.Payload, ExpiresAt: time.Now().Add(time.Minute), Hops: 1, MaxHops: 1}, "peer-1")

	out2 := sf.NextForPeer(peer.ID("peer-2"), 10)
	if len(out2) != 0 {
		t.Fatalf("expected no forwarding when max hops reached")
	}
}

// gate allows only a fixed number of forwards.
type gate struct{ allow int }

func (g *gate) AllowForward(_ peer.ID, _ int) bool {
	if g.allow <= 0 {
		return false
	}
	g.allow--
	return true
}

func TestDeciderBlocksForwarding(t *testing.T) {
	sf := NewStoreAndForward(sfLogger(), 10, time.Minute)
	g := &gate{allow: 1}
	sf.SetDecider(g)

	sf.Enqueue([]byte("a"), 2)
	sf.Enqueue([]byte("b"), 2)

	out1 := sf.NextForPeer(peer.ID("p"), 10)
	if len(out1) != 1 {
		t.Fatalf("expected decider to allow only one, got %d", len(out1))
	}

	out2 := sf.NextForPeer(peer.ID("p2"), 10)
	if len(out2) != 0 {
		t.Fatalf("expected decider to block remaining after budget, got %d", len(out2))
	}
}
