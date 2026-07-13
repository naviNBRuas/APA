package networking

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
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
	require.False(t, sf.AllowForward("peer", 100), "expected forward to be denied for low reputation")
}

func TestSelectiveForwarderRespectsNetworkStats(t *testing.T) {
	stats := &NetworkStats{Latency: 800 * time.Millisecond, Bandwidth: 10, PacketLoss: 0.1}
	sf := NewSelectiveForwarder(ForwardPolicy{MaxLatency: 500 * time.Millisecond}, nil, stubNet{stats: stats})
	require.False(t, sf.AllowForward("peer", 100), "expected forward to be denied due to latency")
}

func TestSelectiveForwarderTokenBucket(t *testing.T) {
	sf := NewSelectiveForwarder(ForwardPolicy{BucketBytes: 100, RefillBytesPerSec: 0}, nil, nil)
	require.True(t, sf.AllowForward("peer", 60), "expected first forward to pass")
	require.False(t, sf.AllowForward("peer", 60), "expected second forward to fail due to budget")
}

func TestSelectiveForwarderCooldown(t *testing.T) {
	sf := NewSelectiveForwarder(ForwardPolicy{MinReputation: 80, Cooldown: time.Second}, stubRep{score: 50}, nil)
	require.False(t, sf.AllowForward("peer", 10), "expected deny")
	require.False(t, sf.AllowForward("peer", 10), "expected deny during cooldown")
}
