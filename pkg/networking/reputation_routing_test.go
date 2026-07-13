package networking

import (
	"log/slog"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewReputationRoutingManager(t *testing.T) {
	logger := slog.Default()
	rrm := NewReputationRoutingManager(logger)
	require.NotNil(t, rrm)
	assert.NotNil(t, rrm.reputation)
	assert.NotNil(t, rrm.networkStats)
}

func TestNewPeerReputationManager(t *testing.T) {
	logger := slog.Default()
	prm := NewPeerReputationManager(logger)
	require.NotNil(t, prm)
	assert.NotNil(t, prm.scores)
	assert.Empty(t, prm.scores)
}

func TestNewNetworkStatsManager(t *testing.T) {
	logger := slog.Default()
	nsm := NewNetworkStatsManager(logger)
	require.NotNil(t, nsm)
	assert.NotNil(t, nsm.stats)
	assert.Empty(t, nsm.stats)
}

func TestRecordInteraction_NewPeer(t *testing.T) {
	prm := NewPeerReputationManager(slog.Default())
	pid := peer.ID("test-peer")

	prm.RecordInteraction(pid, ModuleTransfer, Success)

	score, exists := prm.scores[pid]
	require.True(t, exists)
	assert.Equal(t, 52.0, score.Score)
	assert.Equal(t, 1, score.InteractionCount)
}

func TestRecordInteraction_ExistingPeer(t *testing.T) {
	prm := NewPeerReputationManager(slog.Default())
	pid := peer.ID("test-peer")

	prm.RecordInteraction(pid, ModuleTransfer, Success)
	prm.RecordInteraction(pid, MessageExchange, Failure)
	prm.RecordInteraction(pid, DataSync, Timeout)

	score, exists := prm.scores[pid]
	require.True(t, exists)
	assert.Equal(t, 44.0, score.Score)
	assert.Equal(t, 3, score.InteractionCount)
}

func TestRecordInteraction_ScoreBounds(t *testing.T) {
	prm := NewPeerReputationManager(slog.Default())
	pid := peer.ID("test-peer")

	for i := 0; i < 30; i++ {
		prm.RecordInteraction(pid, ModuleTransfer, Success)
	}
	score, exists := prm.scores[pid]
	require.True(t, exists)
	assert.Equal(t, 100.0, score.Score)
	assert.Equal(t, 30, score.InteractionCount)

	for i := 0; i < 30; i++ {
		prm.RecordInteraction(pid, ModuleTransfer, Failure)
	}
	score, exists = prm.scores[pid]
	require.True(t, exists)
	assert.Equal(t, 0.0, score.Score)
}

func TestGetReputationScore_UnknownPeer(t *testing.T) {
	prm := NewPeerReputationManager(slog.Default())
	pid := peer.ID("unknown-peer")

	score := prm.GetReputationScore(pid)
	assert.Equal(t, 50.0, score)
}

func TestGetReputationScore_ExistingPeer(t *testing.T) {
	prm := NewPeerReputationManager(slog.Default())
	pid := peer.ID("known-peer")

	prm.scores[pid] = &PeerScore{
		PeerID:           pid,
		Score:            52.0,
		LastUpdated:      time.Now(),
		InteractionCount: 1,
	}
	score := prm.GetReputationScore(pid)
	assert.InDelta(t, 52.0, score, 0.01)
}

func TestGetReputationScore_Decay(t *testing.T) {
	prm := NewPeerReputationManager(slog.Default())
	pid := peer.ID("decaying-peer")

	prm.scores[pid] = &PeerScore{
		PeerID:           pid,
		Score:            80.0,
		LastUpdated:      time.Now().Add(-48 * time.Hour),
		InteractionCount: 5,
	}

	score := prm.GetReputationScore(pid)
	expectedDecay := 80.0 * (1.0 - (48.0/24.0)*0.01)
	assert.InDelta(t, expectedDecay, score, 0.01)
}

func TestGetReputationScore_DecayClampsToZero(t *testing.T) {
	prm := NewPeerReputationManager(slog.Default())
	pid := peer.ID("very-old-peer")

	prm.scores[pid] = &PeerScore{
		PeerID:           pid,
		Score:            5.0,
		LastUpdated:      time.Now().Add(-2400 * time.Hour), // 100 days
		InteractionCount: 1,
	}

	score := prm.GetReputationScore(pid)
	assert.Equal(t, 0.0, score)
}

func TestUpdateNetworkStats_NewPeer(t *testing.T) {
	nsm := NewNetworkStatsManager(slog.Default())
	pid := peer.ID("test-peer")

	nsm.UpdateNetworkStats(pid, 100*time.Millisecond, 50.0)

	stats, exists := nsm.stats[pid]
	require.True(t, exists)
	assert.Equal(t, 100*time.Millisecond, stats.Latency)
	assert.Equal(t, 50.0, stats.Bandwidth)
	assert.Equal(t, 1, stats.ConnectionCount)
}

func TestUpdateNetworkStats_ExistingPeer(t *testing.T) {
	nsm := NewNetworkStatsManager(slog.Default())
	pid := peer.ID("test-peer")

	nsm.UpdateNetworkStats(pid, 100*time.Millisecond, 50.0)
	nsm.UpdateNetworkStats(pid, 200*time.Millisecond, 75.0)

	stats, exists := nsm.stats[pid]
	require.True(t, exists)
	assert.Equal(t, 200*time.Millisecond, stats.Latency)
	assert.Equal(t, 75.0, stats.Bandwidth)
	assert.Equal(t, 2, stats.ConnectionCount)
}

func TestGetBestPeers_Empty(t *testing.T) {
	rrm := NewReputationRoutingManager(slog.Default())

	peers := rrm.GetBestPeers(5)
	assert.Empty(t, peers)
}

func TestGetBestPeers_Single(t *testing.T) {
	rrm := NewReputationRoutingManager(slog.Default())
	pid := peer.ID("best-peer")

	rrm.reputation.scores[pid] = &PeerScore{PeerID: pid, Score: 90.0, InteractionCount: 10}

	peers := rrm.GetBestPeers(5)
	assert.Len(t, peers, 1)
	assert.Equal(t, pid, peers[0])
}

func TestGetBestPeers_OrderedByScore(t *testing.T) {
	rrm := NewReputationRoutingManager(slog.Default())
	pid1 := peer.ID("low")
	pid2 := peer.ID("medium")
	pid3 := peer.ID("high")

	rrm.reputation.scores[pid1] = &PeerScore{PeerID: pid1, Score: 30.0, InteractionCount: 1}
	rrm.reputation.scores[pid2] = &PeerScore{PeerID: pid2, Score: 60.0, InteractionCount: 1}
	rrm.reputation.scores[pid3] = &PeerScore{PeerID: pid3, Score: 90.0, InteractionCount: 1}

	peers := rrm.GetBestPeers(5)
	assert.Len(t, peers, 3)
	assert.Equal(t, pid3, peers[0])
	assert.Equal(t, pid2, peers[1])
	assert.Equal(t, pid1, peers[2])
}

func TestGetBestPeers_LimitedCount(t *testing.T) {
	rrm := NewReputationRoutingManager(slog.Default())
	pid1 := peer.ID("peer-1")
	pid2 := peer.ID("peer-2")
	pid3 := peer.ID("peer-3")

	rrm.reputation.scores[pid1] = &PeerScore{PeerID: pid1, Score: 90.0, InteractionCount: 1}
	rrm.reputation.scores[pid2] = &PeerScore{PeerID: pid2, Score: 80.0, InteractionCount: 1}
	rrm.reputation.scores[pid3] = &PeerScore{PeerID: pid3, Score: 70.0, InteractionCount: 1}

	peers := rrm.GetBestPeers(2)
	assert.Len(t, peers, 2)
	assert.Equal(t, pid1, peers[0])
	assert.Equal(t, pid2, peers[1])
}

func TestSelectOptimalPeer_NoPeers(t *testing.T) {
	rrm := NewReputationRoutingManager(slog.Default())

	pid := rrm.SelectOptimalPeer(ModuleTransfer, nil)
	assert.Empty(t, pid)
}

func TestSelectOptimalPeer_SelectsBest(t *testing.T) {
	rrm := NewReputationRoutingManager(slog.Default())
	pid1 := peer.ID("low")
	pid2 := peer.ID("high")

	rrm.reputation.scores[pid1] = &PeerScore{PeerID: pid1, Score: 30.0, InteractionCount: 1}
	rrm.reputation.scores[pid2] = &PeerScore{PeerID: pid2, Score: 90.0, InteractionCount: 1}

	pid := rrm.SelectOptimalPeer(MessageExchange, nil)
	assert.Equal(t, pid2, pid)
}

func TestSelectOptimalPeer_ExcludesSpecified(t *testing.T) {
	rrm := NewReputationRoutingManager(slog.Default())
	pid1 := peer.ID("excluded")
	pid2 := peer.ID("selected")

	rrm.reputation.scores[pid1] = &PeerScore{PeerID: pid1, Score: 90.0, InteractionCount: 1}
	rrm.reputation.scores[pid2] = &PeerScore{PeerID: pid2, Score: 80.0, InteractionCount: 1}

	pid := rrm.SelectOptimalPeer(DataSync, []peer.ID{pid1})
	assert.Equal(t, pid2, pid)
}

func TestSelectOptimalPeer_AllExcludedFallsBack(t *testing.T) {
	rrm := NewReputationRoutingManager(slog.Default())
	pid1 := peer.ID("peer-1")
	pid2 := peer.ID("peer-2")

	rrm.reputation.scores[pid1] = &PeerScore{PeerID: pid1, Score: 90.0, InteractionCount: 1}
	rrm.reputation.scores[pid2] = &PeerScore{PeerID: pid2, Score: 80.0, InteractionCount: 1}

	pid := rrm.SelectOptimalPeer(ModuleTransfer, []peer.ID{pid1, pid2})
	assert.Contains(t, []peer.ID{pid1, pid2}, pid)
}
