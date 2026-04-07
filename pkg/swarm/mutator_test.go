package swarm

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

func mutLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestMutatorPrunesWorstLatency(t *testing.T) {
	rep := NewReputationSystem(mutLogger())
	routing := NewRoutingManager(mutLogger(), rep)
	tm := NewTopologyManager(mutLogger(), rep, routing)
	local := peer.ID("local")

	// connect 3 peers with varying latency
	a := peer.ID("a")
	b := peer.ID("b")
	c := peer.ID("c")
	tm.UpdatePeerConnection(a, true, nil, nil)
	tm.UpdatePeerConnection(b, true, nil, nil)
	tm.UpdatePeerConnection(c, true, nil, nil)

	tm.UpdateEdge(local, a, 50*time.Millisecond, 100, 0.9)
	tm.UpdateEdge(local, b, 200*time.Millisecond, 50, 0.7)
	tm.UpdateEdge(local, c, 400*time.Millisecond, 10, 0.3)

	mut := NewTopologyMutator(tm, MutationPolicy{TargetDegree: 2}, local)
	res := mut.Mutate(context.Background())

	if len(res.Prune) != 1 || res.Prune[0] != c {
		t.Fatalf("expected worst latency peer c to be pruned, got %v", res.Prune)
	}
}
