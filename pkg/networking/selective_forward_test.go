package networking

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

type stubRep struct{ score float64 }

func (s stubRep) GetScore(id string) float64 { return s.score }

type stubNet struct{ stats *NetworkStats }

func (s stubNet) GetNetworkStats(id peer.ID) *NetworkStats { return s.stats }

func sfwdLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestSelectiveForwarderRespectsReputation(t *testing.T) {
	sf := NewSelectiveForwarder(ForwardPolicy{MinReputation: 60}, stubRep{score: 50}, nil)
	if sf.AllowForward("peer", 100) {
		t.Fatalf("expected forward to be denied for low reputation")
	}
}

func TestSelectiveForwarderRespectsNetworkStats(t *testing.T) {
	stats := &NetworkStats{Latency: 800 * time.Millisecond, Bandwidth: 10, PacketLoss: 0.1}
	sf := NewSelectiveForwarder(ForwardPolicy{MaxLatency: 500 * time.Millisecond}, nil, stubNet{stats: stats})
	if sf.AllowForward("peer", 100) {
		t.Fatalf("expected forward to be denied due to latency")
	}
}

func TestSelectiveForwarderTokenBucket(t *testing.T) {
	sf := NewSelectiveForwarder(ForwardPolicy{BucketBytes: 100, RefillBytesPerSec: 0}, nil, nil)
	if !sf.AllowForward("peer", 60) {
		t.Fatalf("expected first forward to pass")
	}
	if sf.AllowForward("peer", 60) {
		t.Fatalf("expected second forward to fail due to budget")
	}
}

func TestSelectiveForwarderCooldown(t *testing.T) {
	sf := NewSelectiveForwarder(ForwardPolicy{MinReputation: 80, Cooldown: time.Second}, stubRep{score: 50}, nil)
	if sf.AllowForward("peer", 10) {
		t.Fatalf("expected deny")
	}
	if sf.AllowForward("peer", 10) {
		t.Fatalf("expected deny during cooldown")
	}
}
