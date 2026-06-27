package swarm

import (
	"context"
	"log/slog"
	"sort"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// NodeTier models a node's role in the multi-tier mesh.
type NodeTier string

const (
	NodeTierUnknown  NodeTier = "unknown"
	NodeTierEdge     NodeTier = "edge"
	NodeTierRelay    NodeTier = "relay"
	NodeTierBackbone NodeTier = "backbone"
)

// TierPolicy provides connection goals per tier.
type TierPolicy struct {
	TargetEdgeDegree     int
	TargetRelayDegree    int
	TargetBackboneDegree int
	RegionBias           float64
	QualityWeight        float64
	ReputationWeight     float64
}

func (p TierPolicy) withDefaults() TierPolicy {
	if p.TargetEdgeDegree == 0 {
		p.TargetEdgeDegree = 4
	}
	if p.TargetRelayDegree == 0 {
		p.TargetRelayDegree = 8
	}
	if p.TargetBackboneDegree == 0 {
		p.TargetBackboneDegree = 12
	}
	if p.RegionBias == 0 {
		p.RegionBias = 0.2
	}
	if p.QualityWeight == 0 {
		p.QualityWeight = 0.5
	}
	if p.ReputationWeight == 0 {
		p.ReputationWeight = 0.5
	}
	return p
}

// OptimizeTopology optimizes the network topology by suggesting new connections or disconnections
func (tm *TopologyManager) OptimizeTopology(ctx context.Context) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// This is a simplified optimization algorithm
	// In a real implementation, this would be much more sophisticated

	currentPeers := tm.GetConnectedPeers()

	// If we have too few connections, suggest new ones
	if len(currentPeers) < 5 {
		optimalPeers := tm.FindOptimalPeers(ctx, 5-len(currentPeers), nil)
		tm.logger.Info("Suggesting new connections to optimize topology",
			"suggested_peers", len(optimalPeers))

		// In a real implementation, this would trigger actual connection attempts
	}

	// If we have too many connections, suggest pruning low-value ones
	if len(currentPeers) > 20 {
		// Score current connections
		type ConnectionScore struct {
			peerID peer.ID
			score  float64
		}

		var connectionScores []ConnectionScore
		for _, peerID := range currentPeers {
			// Score based on reputation, recent activity, and value
			reputation := tm.reputation.GetScore(string(peerID))

			conn, exists := tm.peerConnections[peerID]
			if !exists {
				connectionScores = append(connectionScores, ConnectionScore{peerID: peerID, score: reputation})
				continue
			}

			// Factor in recent activity (more recent = higher score)
			timeSinceLastSeen := time.Since(conn.LastSeen)
			recencyFactor := 1.0
			if timeSinceLastSeen > time.Hour {
				recencyFactor = 0.5
			} else if timeSinceLastSeen > time.Minute {
				recencyFactor = 0.8
			}

			score := reputation * recencyFactor
			connectionScores = append(connectionScores, ConnectionScore{peerID: peerID, score: score})
		}

		// Sort by score (lower scores are candidates for removal)
		sort.Slice(connectionScores, func(i, j int) bool {
			return connectionScores[i].score < connectionScores[j].score
		})

		// Suggest removing lowest-scoring connections
		toRemove := len(currentPeers) - 15 // Target 15 connections
		if toRemove > 0 {
			tm.logger.Info("Suggesting connection pruning to optimize topology",
				"candidates_for_removal", toRemove)

			// In a real implementation, this would trigger actual disconnection attempts
		}
	}

	tm.logger.Debug("Topology optimization completed")
}

// SuggestTierAdjustments computes connection recommendations for a node tier.
// It returns suggested peers to connect to and, if over target, peers to disconnect from.
func (tm *TopologyManager) SuggestTierAdjustments(localPeer peer.ID, localTier NodeTier) (connect []peer.ID, disconnect []peer.ID) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	policy := tm.policy.withDefaults()
	target := tm.targetForTier(policy, localTier)
	if target == 0 {
		return nil, nil
	}

	connected := tm.connectedPeersUnsafe(localPeer)
	connectedSet := make(map[peer.ID]struct{}, len(connected))
	for _, pid := range connected {
		connectedSet[pid] = struct{}{}
	}

	if over := len(connected) - target; over > 0 {
		disconnect = tm.pickDisconnects(localPeer, localTier, policy, connected, over)
	}

	if needed := target - len(connected); needed > 0 {
		connect = tm.pickConnects(localPeer, localTier, policy, connectedSet, needed)
	}

	return connect, disconnect
}

func (tm *TopologyManager) targetForTier(policy TierPolicy, tier NodeTier) int {
	switch tier {
	case NodeTierEdge:
		return policy.TargetEdgeDegree
	case NodeTierRelay:
		return policy.TargetRelayDegree
	case NodeTierBackbone:
		return policy.TargetBackboneDegree
	default:
		return policy.TargetEdgeDegree
	}
}

func (tm *TopologyManager) pickConnects(localPeer peer.ID, localTier NodeTier, policy TierPolicy, connected map[peer.ID]struct{}, needed int) []peer.ID {
	type candidate struct {
		peerID peer.ID
		score  float64
	}

	var cands []candidate
	for pid, node := range tm.topology.Nodes {
		if pid == localPeer {
			continue
		}
		if _, exists := connected[pid]; exists {
			continue
		}
		if node.Tier == NodeTierUnknown {
			continue
		}

		score := tm.scorePeer(localPeer, localTier, node, policy)
		cands = append(cands, candidate{peerID: pid, score: score})
	}

	sort.Slice(cands, func(i, j int) bool {
		return cands[i].score > cands[j].score
	})

	if needed > len(cands) {
		needed = len(cands)
	}

	result := make([]peer.ID, 0, needed)
	for i := 0; i < needed; i++ {
		result = append(result, cands[i].peerID)
	}

	return result
}

func (tm *TopologyManager) pickDisconnects(localPeer peer.ID, localTier NodeTier, policy TierPolicy, connected []peer.ID, over int) []peer.ID {
	type candidate struct {
		peerID peer.ID
		score  float64
	}

	var cands []candidate
	for _, pid := range connected {
		node := tm.topology.Nodes[pid]
		if node == nil {
			continue
		}
		score := tm.scorePeer(localPeer, localTier, node, policy)
		cands = append(cands, candidate{peerID: pid, score: score})
	}

	sort.Slice(cands, func(i, j int) bool {
		return cands[i].score < cands[j].score
	})

	if over > len(cands) {
		over = len(cands)
	}

	result := make([]peer.ID, 0, over)
	for i := 0; i < over; i++ {
		result = append(result, cands[i].peerID)
	}

	return result
}

func (tm *TopologyManager) scorePeer(localPeer peer.ID, localTier NodeTier, node *NodeInfo, policy TierPolicy) float64 {
	score := 0.0

	switch localTier {
	case NodeTierEdge:
		switch node.Tier {
		case NodeTierRelay:
			score += 2.0
		case NodeTierBackbone:
			score += 1.0
		case NodeTierEdge:
			score -= 2.0
		}
	case NodeTierRelay:
		switch node.Tier {
		case NodeTierBackbone:
			score += 2.0
		case NodeTierEdge:
			score += 1.0
		case NodeTierRelay:
			score += 0.5
		}
	case NodeTierBackbone:
		switch node.Tier {
		case NodeTierBackbone:
			score += 1.5
		case NodeTierRelay:
			score += 1.0
		case NodeTierEdge:
			score -= 2.0
		}
	}

	localNode := tm.topology.Nodes[localPeer]
	if localNode != nil && localNode.Region != "" && node.Region != "" && localNode.Region == node.Region {
		score += policy.RegionBias
	}

	edge := tm.edgeInfo(localPeer, node.PeerID)
	if edge != nil {
		score += policy.QualityWeight * edge.Quality
	}

	if tm.reputation != nil {
		rep := tm.reputation.GetScore(string(node.PeerID)) / 100.0
		score += policy.ReputationWeight * rep
	}

	return score
}
