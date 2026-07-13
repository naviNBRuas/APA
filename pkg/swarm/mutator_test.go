package swarm

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mutLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestMutatorPrunesWorstLatency(t *testing.T) {
	rep := NewReputationSystem(mutLogger())
	routing := NewRoutingManager(mutLogger(), rep)
	tm := NewTopologyManager(mutLogger(), rep, routing)
	local := peer.ID("local")

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

	require.Len(t, res.Prune, 1, "expected one peer to be pruned")
	assert.Equal(t, c, res.Prune[0], "expected worst latency peer c to be pruned")
}

func TestMutatorEmptyConnections(t *testing.T) {
	rep := NewReputationSystem(mutLogger())
	routing := NewRoutingManager(mutLogger(), rep)
	tm := NewTopologyManager(mutLogger(), rep, routing)
	local := peer.ID("local")

	mut := NewTopologyMutator(tm, MutationPolicy{TargetDegree: 2}, local)
	res := mut.Mutate(context.Background())
	require.NotNil(t, res)
	assert.Empty(t, res.Prune)
}

func TestMutatorBelowTargetDoesNotPrune(t *testing.T) {
	rep := NewReputationSystem(mutLogger())
	routing := NewRoutingManager(mutLogger(), rep)
	tm := NewTopologyManager(mutLogger(), rep, routing)
	local := peer.ID("local")

	a := peer.ID("a")
	tm.UpdatePeerConnection(a, true, nil, nil)
	tm.UpdateEdge(local, a, 50*time.Millisecond, 100, 0.9)

	mut := NewTopologyMutator(tm, MutationPolicy{TargetDegree: 3}, local)
	res := mut.Mutate(context.Background())
	assert.Empty(t, res.Prune, "should not prune when below target degree")
}
