// Package networking provides reputation-based routing capabilities for the APA agent.
package networking

import (
	"log/slog"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// ReputationRoutingManager manages reputation-based routing decisions
type ReputationRoutingManager struct {
	logger      *slog.Logger
	reputation  *PeerReputationManager
	networkStats *NetworkStatsManager
	mu          sync.RWMutex
}

// PeerReputationManager manages peer reputation scores
type PeerReputationManager struct {
	logger  *slog.Logger
	scores  map[peer.ID]*PeerScore
	mu      sync.RWMutex
}

// NetworkStatsManager manages network statistics for peers
type NetworkStatsManager struct {
	logger *slog.Logger
	stats  map[peer.ID]*NetworkStats
	mu     sync.RWMutex
}

// PeerScore represents a peer's reputation score
type PeerScore struct {
	PeerID          peer.ID   `json:"peer_id"`
	Score           float64   `json:"score"`
	LastUpdated     time.Time `json:"last_updated"`
	InteractionCount int       `json:"interaction_count"`
}

// NetworkStats represents network statistics for a peer
type NetworkStats struct {
	PeerID         peer.ID       `json:"peer_id"`
	Latency        time.Duration `json:"latency"`
	Bandwidth      float64       `json:"bandwidth"` // Mbps
	LastUpdated    time.Time     `json:"last_updated"`
	ConnectionCount int          `json:"connection_count"`
}

// InteractionType represents the type of interaction with a peer
type InteractionType int

const (
	// ModuleTransfer represents a module transfer interaction
	ModuleTransfer InteractionType = iota
	// MessageExchange represents a message exchange interaction
	MessageExchange
	// DataSync represents a data synchronization interaction
	DataSync
)

// InteractionResult represents the result of an interaction with a peer
type InteractionResult int

const (
	// Success represents a successful interaction
	Success InteractionResult = iota
	// Failure represents a failed interaction
	Failure
	// Timeout represents a timed out interaction
	Timeout
)

// NewReputationRoutingManager creates a new ReputationRoutingManager
func NewReputationRoutingManager(logger *slog.Logger) *ReputationRoutingManager {
	return &ReputationRoutingManager{
		logger:      logger,
		reputation:  NewPeerReputationManager(logger),
		networkStats: NewNetworkStatsManager(logger),
	}
}

// NewPeerReputationManager creates a new PeerReputationManager
func NewPeerReputationManager(logger *slog.Logger) *PeerReputationManager {
	return &PeerReputationManager{
		logger: logger,
		scores: make(map[peer.ID]*PeerScore),
	}
}

// NewNetworkStatsManager creates a new NetworkStatsManager
func NewNetworkStatsManager(logger *slog.Logger) *NetworkStatsManager {
	return &NetworkStatsManager{
		logger: logger,
		stats:  make(map[peer.ID]*NetworkStats),
	}
}

// RecordInteraction records an interaction with a peer
func (prm *PeerReputationManager) RecordInteraction(peerID peer.ID, interactionType InteractionType, result InteractionResult) {
	prm.mu.Lock()
	defer prm.mu.Unlock()

	// Get or create the peer score
	score, exists := prm.scores[peerID]
	if !exists {
		score = &PeerScore{
			PeerID:          peerID,
			Score:           50.0, // Initial neutral score
			LastUpdated:     time.Now(),
			InteractionCount: 0,
		}
		prm.scores[peerID] = score
	}

	// Update the score based on the interaction result
	switch result {
	case Success:
		score.Score += 2.0
	case Failure:
		score.Score -= 5.0
	case Timeout:
		score.Score -= 3.0
	}

	// Apply bounds to the score
	if score.Score > 100.0 {
		score.Score = 100.0
	}
	if score.Score < 0.0 {
		score.Score = 0.0
	}

	// Update interaction count and timestamp
	score.InteractionCount++
	score.LastUpdated = time.Now()

	prm.logger.Debug("Updated peer reputation score", "peer", peerID, "score", score.Score)
}

// GetReputationScore gets the reputation score for a peer
func (prm *PeerReputationManager) GetReputationScore(peerID peer.ID) float64 {
	prm.mu.RLock()
	defer prm.mu.RUnlock()

	score, exists := prm.scores[peerID]
	if !exists {
		return 50.0 // Return neutral score for unknown peers
	}

	// Apply reputation decay over time
	decayFactor := time.Since(score.LastUpdated).Hours() / 24.0 // 1% decay per day
	decayedScore := score.Score * (1.0 - (decayFactor * 0.01))

	// Ensure score stays within bounds
	if decayedScore > 100.0 {
		decayedScore = 100.0
	}
	if decayedScore < 0.0 {
		decayedScore = 0.0
	}

	return decayedScore
}

// UpdateNetworkStats updates network statistics for a peer
func (nsm *NetworkStatsManager) UpdateNetworkStats(peerID peer.ID, latency time.Duration, bandwidth float64) {
	nsm.mu.Lock()
	defer nsm.mu.Unlock()

	// Get or create the network stats
	stats, exists := nsm.stats[peerID]
	if !exists {
		stats = &NetworkStats{
			PeerID:         peerID,
			Latency:        latency,
			Bandwidth:      bandwidth,
			LastUpdated:    time.Now(),
			ConnectionCount: 0,
		}
		nsm.stats[peerID] = stats
	}

	// Update the stats
	stats.Latency = latency
	stats.Bandwidth = bandwidth
	stats.ConnectionCount++
	stats.LastUpdated = time.Now()

	nsm.logger.Debug("Updated network stats", "peer", peerID, "latency", latency, "bandwidth", bandwidth)
}

// GetBestPeers returns the best peers based on reputation and network stats
func (rrm *ReputationRoutingManager) GetBestPeers(count int) []peer.ID {
	rrm.mu.RLock()
	defer rrm.mu.RUnlock()

	// Get all peer scores
	peerScores := make(map[peer.ID]float64)
	rrm.reputation.mu.RLock()
	for peerID, score := range rrm.reputation.scores {
		peerScores[peerID] = score.Score
	}
	rrm.reputation.mu.RUnlock()

	// Convert to slice for sorting
	type PeerScorePair struct {
		PeerID peer.ID
		Score  float64
	}
	
	var pairs []PeerScorePair
	for peerID, score := range peerScores {
		pairs = append(pairs, PeerScorePair{PeerID: peerID, Score: score})
	}

	// Sort by score (descending)
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Score > pairs[j].Score
	})

	// Return top N peers
	result := make([]peer.ID, 0, count)
	for i := 0; i < len(pairs) && i < count; i++ {
		result = append(result, pairs[i].PeerID)
	}

	return result
}

// SelectOptimalPeer selects the optimal peer for a specific task based on reputation and network stats
func (rrm *ReputationRoutingManager) SelectOptimalPeer(taskType InteractionType, excludePeers []peer.ID) peer.ID {
	rrm.mu.RLock()
	defer rrm.mu.RUnlock()

	// Get best peers
	bestPeers := rrm.GetBestPeers(10)

	// Filter out excluded peers
	filteredPeers := make([]peer.ID, 0)
	excludeMap := make(map[peer.ID]bool)
	for _, peerID := range excludePeers {
		excludeMap[peerID] = true
	}

	for _, peerID := range bestPeers {
		if !excludeMap[peerID] {
			filteredPeers = append(filteredPeers, peerID)
		}
	}

	// If no peers left, return a random peer
	if len(filteredPeers) == 0 {
		if len(bestPeers) > 0 {
			return bestPeers[rand.Intn(len(bestPeers))]
		}
		return ""
	}

	// For now, just return the first peer in the filtered list
	// In a real implementation, this would consider network stats as well
	return filteredPeers[0]
}