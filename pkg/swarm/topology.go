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
	peerConnections map[peer.ID]*PeerConnection
	topology        *TopologyGraph
	policy          TierPolicy
	mu              sync.RWMutex
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
