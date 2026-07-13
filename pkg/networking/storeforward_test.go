package networking

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

func sfLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestEnqueueAndNextForPeerMarksSeen(t *testing.T) {
	sf := NewStoreAndForward(sfLogger(), 10, time.Minute)
	env := sf.Enqueue([]byte("hello"), 3)

	out := sf.NextForPeer(peer.ID("peer-1"), 5)
	require.Len(t, out, 1, "expected one envelope, got %d", len(out))
	require.Equal(t, env.ID, out[0].ID, "unexpected envelope id")

	// second call should not return same envelope for same peer
	out2 := sf.NextForPeer(peer.ID("peer-1"), 5)
	require.Empty(t, out2, "expected no envelopes after seen, got %d", len(out2))
}

func TestExpireAndEvict(t *testing.T) {
	sf := NewStoreAndForward(sfLogger(), 2, 10*time.Millisecond)
	sf.Enqueue([]byte("a"), 2)
	sf.Enqueue([]byte("b"), 2)
	sf.Enqueue([]byte("c"), 2) // should evict oldest to stay within cache after prune attempt

	time.Sleep(15 * time.Millisecond)
	_ = sf.NextForPeer(peer.ID("p"), 10)
	require.LessOrEqual(t, len(sf.tasks), 2, "expected cache to stay within limit, got %d", len(sf.tasks))
}

func TestMaxHopsRespected(t *testing.T) {
	sf := NewStoreAndForward(sfLogger(), 10, time.Minute)
	env := sf.Enqueue([]byte("x"), 1)

	out := sf.NextForPeer(peer.ID("peer-1"), 10)
	require.Len(t, out, 1, "expected envelope available")
	sf.Receive(TaskEnvelope{ID: env.ID, Payload: env.Payload, ExpiresAt: time.Now().Add(time.Minute), Hops: 1, MaxHops: 1}, "peer-1")

	out2 := sf.NextForPeer(peer.ID("peer-2"), 10)
	require.Empty(t, out2, "expected no forwarding when max hops reached")
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
	require.Len(t, out1, 1, "expected decider to allow only one, got %d", len(out1))

	out2 := sf.NextForPeer(peer.ID("p2"), 10)
	require.Empty(t, out2, "expected decider to block remaining after budget, got %d", len(out2))
}
