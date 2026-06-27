package swarm

import (
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

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
