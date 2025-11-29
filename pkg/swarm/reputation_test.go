package swarm

import (
	"log/slog"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// TestReputationSystem tests the reputation system functionality
func TestReputationSystem(t *testing.T) {
	// Create a logger
	logger := slog.Default()

	// Create a reputation system
	rs := NewReputationSystem(logger)

	// Test that we can create a reputation system
	if rs == nil {
		t.Error("Failed to create reputation system")
	}

	// Test recording interactions
	peerID := peer.ID("test-peer")

	// Record a successful interaction
	rs.RecordInteraction(string(peerID), ModuleTransfer, Success)

	// Check the score
	score := rs.GetScore(string(peerID))
	if score <= 50.0 {
		t.Errorf("Expected score > 50.0, got %f", score)
	}

	// Record a failed interaction
	rs.RecordInteraction(string(peerID), ModuleTransfer, Failure)

	// Check the score again
	newScore := rs.GetScore(string(peerID))
	if newScore >= score {
		t.Errorf("Expected score to decrease after failure, got %f (was %f)", newScore, score)
	}

	// Test getting peer score
	peerScore := rs.GetPeerScore(string(peerID))
	if peerScore == nil {
		t.Error("Failed to get peer score")
	}

	if peerScore.PeerID != string(peerID) {
		t.Errorf("Expected peer ID %s, got %s", peerID, peerScore.PeerID)
	}

	// Test trusted peer functionality
	if !rs.IsTrustedPeer(string(peerID), 40.0) {
		t.Error("Peer should be trusted with threshold 40.0")
	}

	if rs.IsTrustedPeer(string(peerID), 100.0) {
		t.Error("Peer should not be trusted with threshold 100.0")
	}

	// Test getting trusted peers
	trustedPeers := rs.GetTrustedPeers(40.0)
	if len(trustedPeers) != 1 {
		t.Errorf("Expected 1 trusted peer, got %d", len(trustedPeers))
	}

	// Test getting all scores
	allScores := rs.GetAllScores()
	if len(allScores) != 1 {
		t.Errorf("Expected 1 score, got %d", len(allScores))
	}
}

// TestReputationDecay tests the time decay functionality
func TestReputationDecay(t *testing.T) {
	// Create a logger
	logger := slog.Default()

	// Create a reputation system
	rs := NewReputationSystem(logger)

	// Add a peer with a high score
	peerID := peer.ID("test-peer")
	rs.RecordInteraction(string(peerID), ModuleTransfer, Success)
	rs.RecordInteraction(string(peerID), ModuleTransfer, Success)
	rs.RecordInteraction(string(peerID), ModuleTransfer, Success)

	initialScore := rs.GetScore(string(peerID))

	// Manually update the score's LastUpdated time in the reputation system
	rs.mu.Lock()
	if score, exists := rs.scores[string(peerID)]; exists {
		score.LastUpdated = time.Now().Add(-48 * time.Hour) // 2 days ago
	}
	rs.mu.Unlock()

	// Apply decay
	rs.DecayScores()

	decayedScore := rs.GetScore(string(peerID))
	if decayedScore >= initialScore {
		t.Errorf("Score should have decayed, initial: %f, decayed: %f", initialScore, decayedScore)
	}

	if decayedScore < 0.0 {
		t.Error("Decayed score should not be negative")
	}
}