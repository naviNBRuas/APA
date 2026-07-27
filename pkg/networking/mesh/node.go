package mesh

import (
	"crypto/ed25519"
	"encoding/json"
	"net"
	"time"
)

const Version = "1.0.0"

// CapabilitySet describes the features a mesh node supports.
type CapabilitySet struct {
	SupportsCodeSync  bool `json:"supports_code_sync"`
	SupportsStateSync bool `json:"supports_state_sync"`
	SupportsRelay     bool `json:"supports_relay"`
	SupportsDHT       bool `json:"supports_dht"`
	IsController      bool `json:"is_controller"`
}

// ServiceEndpoint describes a reachable service on a mesh node.
type ServiceEndpoint struct {
	Protocol string `json:"protocol"`
	Address  string `json:"address"`
	Port     int    `json:"port"`
}

// MeshNode represents a single peer on the APA mesh network.
type MeshNode struct {
	ID           string            `json:"id"`
	PublicKey    ed25519.PublicKey `json:"-"`
	VirtualIP    net.IP            `json:"virtual_ip"`
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Status       NodeStatus        `json:"status"`
	FirstSeen    time.Time         `json:"first_seen"`
	LastSeen     time.Time         `json:"last_seen"`
	Capabilities CapabilitySet     `json:"capabilities"`
	Services     map[string]string `json:"services"`
	Addresses    []string          `json:"addresses"`
	Latency      time.Duration     `json:"latency"`
	SignalStrength float64         `json:"signal_strength"`
}

// Clone returns a deep copy of the MeshNode.
func (n *MeshNode) Clone() *MeshNode {
	if n == nil {
		return nil
	}
	svcs := make(map[string]string, len(n.Services))
	for k, v := range n.Services {
		svcs[k] = v
	}
	addrs := make([]string, len(n.Addresses))
	copy(addrs, n.Addresses)

	pubKey := make(ed25519.PublicKey, len(n.PublicKey))
	copy(pubKey, n.PublicKey)

	vip := make(net.IP, len(n.VirtualIP))
	copy(vip, n.VirtualIP)

	return &MeshNode{
		ID:           n.ID,
		PublicKey:    pubKey,
		VirtualIP:    vip,
		Name:         n.Name,
		Version:      n.Version,
		Status:       n.Status,
		FirstSeen:    n.FirstSeen,
		LastSeen:     n.LastSeen,
		Capabilities: n.Capabilities,
		Services:     svcs,
		Addresses:    addrs,
		Latency:      n.Latency,
	}
}

// NodeInfo is the public identity exchanged during connection setup.
type NodeInfo struct {
	ID           string            `json:"id"`
	PublicKey    []byte            `json:"public_key"`
	VirtualIP    net.IP            `json:"virtual_ip"`
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Capabilities CapabilitySet     `json:"capabilities"`
	Services     map[string]string `json:"services"`
	Signature    []byte            `json:"signature"`
}

// HeartbeatMessage is broadcast periodically to announce presence.
type HeartbeatMessage struct {
	NodeID    string `json:"node_id"`
	Name      string `json:"name"`
	VirtualIP net.IP `json:"virtual_ip"`
	Version   string `json:"version"`
	Timestamp int64  `json:"timestamp"`
}

// MarshalJSON implements json.Marshaler for MeshNode.
func (n *MeshNode) MarshalJSON() ([]byte, error) {
	type alias MeshNode
	return json.Marshal((*alias)(n))
}

// UnmarshalJSON implements json.Unmarshaler for MeshNode.
func (n *MeshNode) UnmarshalJSON(data []byte) error {
	type alias MeshNode
	return json.Unmarshal(data, (*alias)(n))
}
