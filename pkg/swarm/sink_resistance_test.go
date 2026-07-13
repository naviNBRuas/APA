package swarm

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func srLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestSinkResistanceQuorumEvictsAndRekeys(t *testing.T) {
	rep := NewReputationSystem(srLogger())
	routing := NewRoutingManager(srLogger(), rep)
	tm := NewTopologyManager(srLogger(), rep, routing)
	pid := peer.ID("suspicious")
	tm.UpdatePeerConnection(pid, true, []string{"na"}, nil)

	rekeyCount := 0
	sr := NewSinkResistance(tm, time.Minute, 2, func() { rekeyCount++ })

	sig := SuspicionSignal{Peer: pid, Source: "edr", Reason: "port-scan", At: time.Now()}
	assert.False(t, sr.Observe(sig), "should not evict with single source")

	sig2 := SuspicionSignal{Peer: pid, Source: "health-monitor", Reason: "rate-spike", At: sig.At}
	assert.True(t, sr.Observe(sig2), "expected eviction after quorum reached")
	assert.Equal(t, 1, rekeyCount, "expected rekey trigger")

	peers := tm.GetConnectedPeers()
	require.Empty(t, peers, "expected peer removed after eviction")
}

func TestSinkResistancePrunesOldSignals(t *testing.T) {
	rep := NewReputationSystem(srLogger())
	routing := NewRoutingManager(srLogger(), rep)
	tm := NewTopologyManager(srLogger(), rep, routing)
	pid := peer.ID("slow-burn")
	tm.UpdatePeerConnection(pid, true, nil, nil)

	sr := NewSinkResistance(tm, time.Second, 2, nil)

	first := SuspicionSignal{Peer: pid, Source: "edr", Reason: "minor", At: time.Now()}
	sr.Observe(first)

	time.Sleep(1100 * time.Millisecond)
	second := SuspicionSignal{Peer: pid, Source: "health", Reason: "follow-up", At: time.Now()}
	assert.False(t, sr.Observe(second), "expected no eviction because first signal expired")

	peers := tm.GetConnectedPeers()
	require.Len(t, peers, 1, "peer should remain since quorum not reached")
}

func TestSinkResistanceEmptyPeer(t *testing.T) {
	sr := NewSinkResistance(nil, time.Minute, 2, nil)
	sig := SuspicionSignal{Peer: "", Source: "edr", Reason: "test", At: time.Now()}
	assert.False(t, sr.Observe(sig), "empty peer should not trigger")
}

func TestSinkResistanceEmptySource(t *testing.T) {
	rep := NewReputationSystem(srLogger())
	routing := NewRoutingManager(srLogger(), rep)
	tm := NewTopologyManager(srLogger(), rep, routing)
	pid := peer.ID("test-peer")
	tm.UpdatePeerConnection(pid, true, nil, nil)

	sr := NewSinkResistance(tm, time.Minute, 2, nil)
	sig := SuspicionSignal{Peer: pid, Source: "", Reason: "test", At: time.Now()}
	assert.False(t, sr.Observe(sig), "empty source should not trigger")
}
