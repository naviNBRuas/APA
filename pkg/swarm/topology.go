package swarm

import (
	"context"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// TopologyManager manages the dynamic topology of the swarm
type TopologyManager struct {
	logger          *slog.Logger
	reputation      *ReputationSystem
	routing         *RoutingManager
	peerConnections map[peer.ID]*PeerConnection // peerID -> connection info
	topology        *TopologyGraph
	policy          TierPolicy
	mu              sync.RWMutex
}

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

// PeerConnection represents connection information for a peer
type PeerConnection struct {
	PeerID       peer.ID   `json:"peer_id"`
	Connected    bool      `json:"connected"`
	LastSeen     time.Time `json:"last_seen"`
	Connections  int       `json:"connections"`
	Regions      []string  `json:"regions"`
	Capabilities []string  `json:"capabilities"`
}

// TopologyGraph represents the network topology
type TopologyGraph struct {
	Nodes map[peer.ID]*NodeInfo `json:"nodes"`
	Edges map[string]*EdgeInfo  `json:"edges"` // edgeID -> edge info
}

// NodeInfo represents information about a node in the topology
type NodeInfo struct {
	PeerID       peer.ID    `json:"peer_id"`
	Position     [3]float64 `json:"position"` // x, y, z coordinates in virtual space
	Region       string     `json:"region"`
	Capabilities []string   `json:"capabilities"`
	Tier         NodeTier   `json:"tier"`
	LastUpdated  time.Time  `json:"last_updated"`
}

// EdgeInfo represents information about a connection between nodes
type EdgeInfo struct {
	Source      peer.ID       `json:"source"`
	Target      peer.ID       `json:"target"`
	Latency     time.Duration `json:"latency"`
	Bandwidth   float64       `json:"bandwidth"` // Mbps
	Quality     float64       `json:"quality"`   // 0.0 - 1.0
	LastUpdated time.Time     `json:"last_updated"`
}

// NewTopologyManager creates a new topology manager
func NewTopologyManager(logger *slog.Logger, reputation *ReputationSystem, routing *RoutingManager) *TopologyManager {
	return &TopologyManager{
		logger:          logger,
		reputation:      reputation,
		routing:         routing,
		peerConnections: make(map[peer.ID]*PeerConnection),
		topology: &TopologyGraph{
			Nodes: make(map[peer.ID]*NodeInfo),
			Edges: make(map[string]*EdgeInfo),
		},
		policy: TierPolicy{},
	}
}

// SetPolicy overrides the tier connection policy.
func (tm *TopologyManager) SetPolicy(policy TierPolicy) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.policy = policy
}

// SetPeerTier sets the tier role for a peer.
func (tm *TopologyManager) SetPeerTier(peerID peer.ID, tier NodeTier) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	node, ok := tm.topology.Nodes[peerID]
	if !ok {
		node = &NodeInfo{PeerID: peerID}
		tm.topology.Nodes[peerID] = node
	}

	node.Tier = tier
	node.LastUpdated = time.Now()
}

// UpdatePeerConnection updates connection information for a peer
func (tm *TopologyManager) UpdatePeerConnection(peerID peer.ID, connected bool, regions []string, capabilities []string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	conn, exists := tm.peerConnections[peerID]
	if !exists {
		conn = &PeerConnection{
			PeerID: peerID,
		}
		tm.peerConnections[peerID] = conn
	}

	conn.Connected = connected
	conn.LastSeen = time.Now()
	if connected {
		conn.Connections++
	}
	conn.Regions = regions
	conn.Capabilities = capabilities

	// Update topology graph
	node, nodeExists := tm.topology.Nodes[peerID]
	if !nodeExists {
		node = &NodeInfo{
			PeerID: peerID,
		}
		tm.topology.Nodes[peerID] = node
	}

	if len(regions) > 0 {
		node.Region = regions[0] // Use first region as primary
	}
	node.Capabilities = capabilities
	node.LastUpdated = time.Now()

	tm.logger.Debug("Updated peer connection",
		"peer_id", peerID,
		"connected", connected,
		"connections", conn.Connections)
}

// RemovePeer removes a peer from the topology
func (tm *TopologyManager) RemovePeer(peerID peer.ID) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	delete(tm.peerConnections, peerID)
	delete(tm.topology.Nodes, peerID)

	// Remove edges involving this peer
	for edgeID, edge := range tm.topology.Edges {
		if edge.Source == peerID || edge.Target == peerID {
			delete(tm.topology.Edges, edgeID)
		}
	}

	tm.logger.Debug("Removed peer from topology", "peer_id", peerID)
}

// UpdateEdge updates information about a connection between two peers
func (tm *TopologyManager) UpdateEdge(source, target peer.ID, latency time.Duration, bandwidth float64, quality float64) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Create edge ID (consistent ordering)
	edgeID := string(source) + "->" + string(target)
	if string(target) < string(source) {
		edgeID = string(target) + "->" + string(source)
	}

	edge, exists := tm.topology.Edges[edgeID]
	if !exists {
		edge = &EdgeInfo{
			Source: source,
			Target: target,
		}
		tm.topology.Edges[edgeID] = edge
	}

	edge.Latency = latency
	edge.Bandwidth = bandwidth
	edge.Quality = quality
	edge.LastUpdated = time.Now()

	tm.logger.Debug("Updated edge information",
		"edge_id", edgeID,
		"latency", latency,
		"bandwidth", bandwidth,
		"quality", quality)
}

// GetConnectedPeers returns a list of currently connected peers
func (tm *TopologyManager) GetConnectedPeers() []peer.ID {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var connectedPeers []peer.ID
	for peerID, conn := range tm.peerConnections {
		if conn.Connected {
			connectedPeers = append(connectedPeers, peerID)
		}
	}

	return connectedPeers
}

// RegionFor returns the recorded region for a peer if known.
func (tm *TopologyManager) RegionFor(peerID peer.ID) string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	if node, ok := tm.topology.Nodes[peerID]; ok {
		return node.Region
	}
	return ""
}

// GetPeerConnection returns connection information for a peer
func (tm *TopologyManager) GetPeerConnection(peerID peer.ID) *PeerConnection {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	conn, exists := tm.peerConnections[peerID]
	if !exists {
		return nil
	}

	// Return a copy to prevent external modification
	return &PeerConnection{
		PeerID:       conn.PeerID,
		Connected:    conn.Connected,
		LastSeen:     conn.LastSeen,
		Connections:  conn.Connections,
		Regions:      append([]string(nil), conn.Regions...),
		Capabilities: append([]string(nil), conn.Capabilities...),
	}
}

// GetTopology returns the current network topology
func (tm *TopologyManager) GetTopology() *TopologyGraph {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	// Return a copy of the topology
	topology := &TopologyGraph{
		Nodes: make(map[peer.ID]*NodeInfo),
		Edges: make(map[string]*EdgeInfo),
	}

	// Copy nodes
	for peerID, node := range tm.topology.Nodes {
		topology.Nodes[peerID] = &NodeInfo{
			PeerID:       node.PeerID,
			Position:     node.Position,
			Region:       node.Region,
			Capabilities: append([]string(nil), node.Capabilities...),
			Tier:         node.Tier,
			LastUpdated:  node.LastUpdated,
		}
	}

	// Copy edges
	for edgeID, edge := range tm.topology.Edges {
		topology.Edges[edgeID] = &EdgeInfo{
			Source:      edge.Source,
			Target:      edge.Target,
			Latency:     edge.Latency,
			Bandwidth:   edge.Bandwidth,
			Quality:     edge.Quality,
			LastUpdated: edge.LastUpdated,
		}
	}

	return topology
}

// FindOptimalPeers finds optimal peers to connect to based on topology and reputation
func (tm *TopologyManager) FindOptimalPeers(ctx context.Context, maxPeers int, requiredCapabilities []string) []peer.ID {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	// Get all known peers
	var candidates []struct {
		peerID   peer.ID
		score    float64
		distance float64
	}

	currentPeers := tm.GetConnectedPeers()

	// Score all known peers
	for peerID, node := range tm.topology.Nodes {
		// Skip already connected peers
		connected := false
		for _, connectedPeer := range currentPeers {
			if connectedPeer == peerID {
				connected = true
				break
			}
		}
		if connected {
			continue
		}

		// Check if peer has required capabilities
		if len(requiredCapabilities) > 0 {
			hasAllCapabilities := true
			for _, requiredCap := range requiredCapabilities {
				found := false
				for _, peerCap := range node.Capabilities {
					if peerCap == requiredCap {
						found = true
						break
					}
				}
				if !found {
					hasAllCapabilities = false
					break
				}
			}
			if !hasAllCapabilities {
				continue
			}
		}

		// Calculate score based on reputation and proximity
		reputationScore := tm.reputation.GetScore(string(peerID))

		// Calculate distance in virtual space (simplified)
		// In a real implementation, this would use actual coordinates
		distance := 1.0 // Placeholder

		// Composite score (weights can be adjusted)
		score := (reputationScore / 100.0 * 0.7) + (distance * 0.3)

		candidates = append(candidates, struct {
			peerID   peer.ID
			score    float64
			distance float64
		}{peerID: peerID, score: score, distance: distance})
	}

	// Sort candidates by score (higher is better)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	// Return top candidates
	var optimalPeers []peer.ID
	for i := 0; i < len(candidates) && i < maxPeers; i++ {
		optimalPeers = append(optimalPeers, candidates[i].peerID)
	}

	tm.logger.Debug("Found optimal peers for connection",
		"count", len(optimalPeers),
		"max_requested", maxPeers)

	return optimalPeers
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

	// Tier affinity
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

	// Region bias
	localNode := tm.topology.Nodes[localPeer]
	if localNode != nil && localNode.Region != "" && node.Region != "" && localNode.Region == node.Region {
		score += policy.RegionBias
	}

	// Edge quality (if known)
	edge := tm.edgeInfo(localPeer, node.PeerID)
	if edge != nil {
		score += policy.QualityWeight * edge.Quality
	}

	// Reputation contribution
	if tm.reputation != nil {
		rep := tm.reputation.GetScore(string(node.PeerID)) / 100.0
		score += policy.ReputationWeight * rep
	}

	return score
}

func (tm *TopologyManager) edgeInfo(a, b peer.ID) *EdgeInfo {
	id := tm.edgeKey(a, b)
	return tm.topology.Edges[id]
}

func (tm *TopologyManager) edgeKey(a, b peer.ID) string {
	if string(a) < string(b) {
		return string(a) + "->" + string(b)
	}
	return string(b) + "->" + string(a)
}

// EdgeInfoBetween returns edge information if present.
func (tm *TopologyManager) EdgeInfoBetween(a, b peer.ID) *EdgeInfo {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.edgeInfo(a, b)
}

func (tm *TopologyManager) connectedPeersUnsafe(exclude peer.ID) []peer.ID {
	var peers []peer.ID
	for pid, conn := range tm.peerConnections {
		if pid == exclude {
			continue
		}
		if conn.Connected {
			peers = append(peers, pid)
		}
	}
	return peers
}
