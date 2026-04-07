package swarm

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
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
	if sr.Observe(sig) {
		t.Fatalf("should not evict with single source")
	}

	sig2 := SuspicionSignal{Peer: pid, Source: "health-monitor", Reason: "rate-spike", At: sig.At}
	if !sr.Observe(sig2) {
		t.Fatalf("expected eviction after quorum reached")
	}

	if rekeyCount != 1 {
		t.Fatalf("expected rekey trigger, got %d", rekeyCount)
	}
	if peers := tm.GetConnectedPeers(); len(peers) != 0 {
		t.Fatalf("expected peer removed, still have %v", peers)
	}
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

	// After window passes, previous signal should not count toward quorum.
	time.Sleep(1100 * time.Millisecond)
	second := SuspicionSignal{Peer: pid, Source: "health", Reason: "follow-up", At: time.Now()}
	if sr.Observe(second) {
		t.Fatalf("expected no eviction because first signal expired")
	}
	if peers := tm.GetConnectedPeers(); len(peers) != 1 {
		t.Fatalf("peer should remain since quorum not reached; peers=%v", peers)
	}
}
