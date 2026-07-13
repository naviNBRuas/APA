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

func repLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestReputationRecordInteraction(t *testing.T) {
	rs := NewReputationSystem(repLogger())
	pid := peer.ID("test-peer")

	rs.RecordInteraction(string(pid), ModuleTransfer, Success)
	score := rs.GetScore(string(pid))
	assert.Greater(t, score, 50.0, "score should increase after success")

	rs.RecordInteraction(string(pid), ModuleTransfer, Failure)
	newScore := rs.GetScore(string(pid))
	assert.Less(t, newScore, score, "score should decrease after failure")
}

func TestReputationPeerScore(t *testing.T) {
	rs := NewReputationSystem(repLogger())
	pid := peer.ID("test-peer")
	rs.RecordInteraction(string(pid), ModuleTransfer, Success)

	ps := rs.GetPeerScore(string(pid))
	require.NotNil(t, ps)
	assert.Equal(t, string(pid), ps.PeerID)
	assert.Equal(t, 1, ps.InteractionCount)
	assert.Equal(t, 1, ps.SuccessCount)
	assert.Equal(t, 0, ps.FailureCount)
}

func TestReputationGetScoreForUnknownPeer(t *testing.T) {
	rs := NewReputationSystem(repLogger())
	score := rs.GetScore("unknown")
	assert.Equal(t, 50.0, score, "unknown peers should get neutral score")
}

func TestReputationTrustedPeer(t *testing.T) {
	rs := NewReputationSystem(repLogger())
	pid := peer.ID("trusted-peer")
	rs.RecordInteraction(string(pid), ModuleTransfer, Success)

	assert.True(t, rs.IsTrustedPeer(string(pid), 40.0), "peer should be trusted with threshold 40.0")
	assert.False(t, rs.IsTrustedPeer(string(pid), 100.0), "peer should not be trusted with threshold 100.0")
}

func TestReputationGetTrustedPeers(t *testing.T) {
	rs := NewReputationSystem(repLogger())
	pid := peer.ID("peer-1")
	rs.RecordInteraction(string(pid), ModuleTransfer, Success)

	trusted := rs.GetTrustedPeers(40.0)
	require.Len(t, trusted, 1)
	assert.Equal(t, string(pid), trusted[0])

	trusted = rs.GetTrustedPeers(100.0)
	require.Empty(t, trusted)
}

func TestReputationGetAllScores(t *testing.T) {
	rs := NewReputationSystem(repLogger())
	rs.RecordInteraction("peer-a", ModuleTransfer, Success)
	rs.RecordInteraction("peer-b", ModuleTransfer, Failure)

	scores := rs.GetAllScores()
	assert.Len(t, scores, 2)
}

func TestReputationDecay(t *testing.T) {
	rs := NewReputationSystem(repLogger())
	pid := peer.ID("decay-peer")
	rs.RecordInteraction(string(pid), ModuleTransfer, Success)
	rs.RecordInteraction(string(pid), ModuleTransfer, Success)
	rs.RecordInteraction(string(pid), ModuleTransfer, Success)

	initialScore := rs.GetScore(string(pid))

	rs.mu.Lock()
	if score, exists := rs.scores[string(pid)]; exists {
		score.LastUpdated = time.Now().Add(-48 * time.Hour)
	}
	rs.mu.Unlock()

	rs.DecayScores()
	decayedScore := rs.GetScore(string(pid))
	assert.Less(t, decayedScore, initialScore, "score should have decayed")
	assert.GreaterOrEqual(t, decayedScore, 0.0, "decayed score should not be negative")
}

func TestReputationAllInteractionTypes(t *testing.T) {
	rs := NewReputationSystem(repLogger())
	pid := "multi-peer"

	for _, it := range []InteractionType{ModuleTransfer, ControllerCommunication, NetworkConnection, ModuleExecution} {
		rs.RecordInteraction(pid, it, Success)
	}
	score := rs.GetScore(pid)
	assert.Greater(t, score, 50.0)

	for _, it := range []InteractionType{ModuleTransfer, ControllerCommunication, NetworkConnection, ModuleExecution} {
		rs.RecordInteraction(pid, it, Failure)
	}
	score = rs.GetScore(pid)
	assert.Less(t, score, 50.0, "score should drop below neutral after failures")
}
