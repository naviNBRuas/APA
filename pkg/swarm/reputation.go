package swarm

import (
	"log/slog"
	"sync"
	"time"
)

// ReputationSystem manages peer reputation scores
type ReputationSystem struct {
	logger  *slog.Logger
	scores  map[string]*PeerScore // peerID -> score
	mu      sync.RWMutex
}

// PeerScore represents a peer's reputation score
type PeerScore struct {
	PeerID       string    `json:"peer_id"`
	Score        float64   `json:"score"`
	LastUpdated  time.Time `json:"last_updated"`
	InteractionCount int    `json:"interaction_count"`
	SuccessCount int      `json:"success_count"`
	FailureCount int      `json:"failure_count"`
	LastFailure  time.Time `json:"last_failure"`
}

// InteractionType represents the type of interaction with a peer
type InteractionType int

const (
	ModuleTransfer InteractionType = iota
	ControllerCommunication
	NetworkConnection
	ModuleExecution
)

// InteractionResult represents the result of an interaction with a peer
type InteractionResult int

const (
	Success InteractionResult = iota
	Failure
	Timeout
)

// NewReputationSystem creates a new reputation system
func NewReputationSystem(logger *slog.Logger) *ReputationSystem {
	return &ReputationSystem{
		logger: logger,
		scores: make(map[string]*PeerScore),
	}
}

// RecordInteraction records an interaction with a peer and updates their reputation score
func (rs *ReputationSystem) RecordInteraction(peerID string, interactionType InteractionType, result InteractionResult) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	// Get or create the peer score
	score, exists := rs.scores[peerID]
	if !exists {
		score = &PeerScore{
			PeerID:      peerID,
			Score:       50.0, // Start with a neutral score
			LastUpdated: time.Now(),
		}
		rs.scores[peerID] = score
	}

	// Update interaction counts
	score.InteractionCount++
	if result == Success {
		score.SuccessCount++
	} else {
		score.FailureCount++
		score.LastFailure = time.Now()
	}

	// Calculate the new score based on the interaction
	newScore := rs.calculateScore(score, interactionType, result)
	score.Score = newScore
	score.LastUpdated = time.Now()

	rs.logger.Debug("Updated peer reputation score", 
		"peer_id", peerID, 
		"score", score.Score, 
		"interaction_type", interactionType, 
		"result", result)
}

// calculateScore calculates the new reputation score based on the interaction
func (rs *ReputationSystem) calculateScore(score *PeerScore, interactionType InteractionType, result InteractionResult) float64 {
	// Base score adjustment based on result
	var adjustment float64
	switch result {
	case Success:
		adjustment = 2.0
	case Failure:
		adjustment = -5.0
	case Timeout:
		adjustment = -3.0
	}

	// Adjust based on interaction type
	switch interactionType {
	case ModuleTransfer:
		adjustment *= 1.5 // Module transfers are more important
	case ControllerCommunication:
		adjustment *= 1.2 // Controller communication is important
	case NetworkConnection:
		adjustment *= 1.0 // Base weight
	case ModuleExecution:
		adjustment *= 1.3 // Module execution is important
	}

	// Apply time decay - older interactions matter less
	timeSinceLastUpdate := time.Since(score.LastUpdated)
	if timeSinceLastUpdate > 24*time.Hour {
		// Reduce the impact of old interactions
		adjustment *= 0.8
	}

	// Calculate new score with bounds checking
	newScore := score.Score + adjustment
	if newScore > 100.0 {
		newScore = 100.0
	}
	if newScore < 0.0 {
		newScore = 0.0
	}

	return newScore
}

// GetScore returns the reputation score for a peer
func (rs *ReputationSystem) GetScore(peerID string) float64 {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	score, exists := rs.scores[peerID]
	if !exists {
		return 50.0 // Return neutral score for unknown peers
	}

	return score.Score
}

// GetPeerScore returns the complete peer score information
func (rs *ReputationSystem) GetPeerScore(peerID string) *PeerScore {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	score, exists := rs.scores[peerID]
	if !exists {
		return &PeerScore{
			PeerID:      peerID,
			Score:       50.0,
			LastUpdated: time.Now(),
		}
	}

	// Return a copy to prevent external modification
	return &PeerScore{
		PeerID:          score.PeerID,
		Score:           score.Score,
		LastUpdated:     score.LastUpdated,
		InteractionCount: score.InteractionCount,
		SuccessCount:    score.SuccessCount,
		FailureCount:    score.FailureCount,
		LastFailure:     score.LastFailure,
	}
}

// GetAllScores returns all peer scores
func (rs *ReputationSystem) GetAllScores() map[string]*PeerScore {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	// Return copies to prevent external modification
	scores := make(map[string]*PeerScore)
	for peerID, score := range rs.scores {
		scores[peerID] = &PeerScore{
			PeerID:          score.PeerID,
			Score:           score.Score,
			LastUpdated:     score.LastUpdated,
			InteractionCount: score.InteractionCount,
			SuccessCount:    score.SuccessCount,
			FailureCount:    score.FailureCount,
			LastFailure:     score.LastFailure,
		}
	}

	return scores
}

// IsTrustedPeer returns whether a peer is considered trusted based on their reputation score
func (rs *ReputationSystem) IsTrustedPeer(peerID string, threshold float64) bool {
	score := rs.GetScore(peerID)
	return score >= threshold
}

// GetTrustedPeers returns a list of trusted peers based on a score threshold
func (rs *ReputationSystem) GetTrustedPeers(threshold float64) []string {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	var trustedPeers []string
	for peerID, score := range rs.scores {
		if score.Score >= threshold {
			trustedPeers = append(trustedPeers, peerID)
		}
	}

	return trustedPeers
}

// DecayScores applies time decay to all peer scores
func (rs *ReputationSystem) DecayScores() {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	now := time.Now()
	for _, score := range rs.scores {
		// Apply decay based on time since last update
		timeSinceUpdate := now.Sub(score.LastUpdated)
		if timeSinceUpdate > 24*time.Hour {
			// Decay score by 1% per day
			days := timeSinceUpdate.Hours() / 24.0
			decayFactor := 1.0 - (0.01 * days)
			if decayFactor < 0.5 {
				decayFactor = 0.5 // Don't decay below 50% of current score
			}
			score.Score *= decayFactor
			score.LastUpdated = now
		}
	}

	rs.logger.Debug("Applied time decay to peer reputation scores")
}