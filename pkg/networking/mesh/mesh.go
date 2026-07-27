package mesh

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"
)

// NodeStatus represents the current state of a mesh node.
type NodeStatus string

const (
	NodeStatusOnline     NodeStatus = "online"
	NodeStatusOffline    NodeStatus = "offline"
	NodeStatusConnecting NodeStatus = "connecting"
	NodeStatusDegraded   NodeStatus = "degraded"
)

// TransportType enumerates the available transport methods.
type TransportType string

const (
	TransportWebRTC   TransportType = "webrtc"
	TransportWebSocket TransportType = "websocket"
	TransportHTTPS    TransportType = "https"
	TransportTCP      TransportType = "tcp"
	TransportLAN      TransportType = "lan"
	TransportRelay    TransportType = "relay"
	TransportP2P      TransportType = "p2p"
)

// MeshConfig configures the mesh network node.
type MeshConfig struct {
	NodeName       string   `yaml:"node_name"`
	ListenPort     int      `yaml:"listen_port"`
	EnableRelay    bool     `yaml:"enable_relay"`
	EnableLAN      bool     `yaml:"enable_lan"`
	RelayAddresses []string `yaml:"relay_addresses"`
	BootstrapPeers []string `yaml:"bootstrap_peers"`
	SyncInterval   time.Duration `yaml:"sync_interval"`
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`
	Subnet         string   `yaml:"subnet"`
	MaxPeers       int      `yaml:"max_peers"`
}

func DefaultMeshConfig() MeshConfig {
	return MeshConfig{
		NodeName:          "",
		ListenPort:        0,
		EnableRelay:       true,
		EnableLAN:         true,
		RelayAddresses:    nil,
		BootstrapPeers:    nil,
		SyncInterval:      30 * time.Second,
		HeartbeatInterval: 10 * time.Second,
		Subnet:            "fd00:apa::/48",
		MaxPeers:          256,
	}
}

// applyMeshDefaults fills zero-valued fields so Start never panics on NewTicker(0).
func applyMeshDefaults(cfg MeshConfig) MeshConfig {
	defaults := DefaultMeshConfig()
	if cfg.SyncInterval <= 0 {
		cfg.SyncInterval = defaults.SyncInterval
	}
	if cfg.HeartbeatInterval <= 0 {
		cfg.HeartbeatInterval = defaults.HeartbeatInterval
	}
	if cfg.MaxPeers <= 0 {
		cfg.MaxPeers = defaults.MaxPeers
	}
	if cfg.Subnet == "" {
		cfg.Subnet = defaults.Subnet
	}
	return cfg
}

// MeshEvent represents events emitted by the mesh network.
type MeshEvent struct {
	Type    string
	NodeID  string
	Payload interface{}
}

const (
	EventPeerJoined    = "peer_joined"
	EventPeerLeft      = "peer_left"
	EventPeerUpdated   = "peer_updated"
	EventCodeReceived  = "code_received"
	EventSyncCompleted = "sync_completed"
)

// MeshNetwork is the central orchestrator for the APA mesh.
type MeshNetwork struct {
	mu       sync.RWMutex
	logger   *slog.Logger
	config   MeshConfig
	ctx      context.Context
	cancel   context.CancelFunc

	self     *MeshNode
	peers    map[string]*MeshNode

	transport *TransportChain
	discovery *DiscoveryEngine
	sync      *SyncEngine
	security  *SecurityManager

	eventCh  chan MeshEvent
	done     chan struct{}

	startTime   time.Time
	relayServer *RelayServer
}

// NewMeshNetwork creates and initialises a mesh network node.
func NewMeshNetwork(logger *slog.Logger, cfg MeshConfig, privKey ed25519.PrivateKey) (*MeshNetwork, error) {
	cfg = applyMeshDefaults(cfg)

	if privKey == nil {
		_, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("generate key: %w", err)
		}
		privKey = priv
	}

	pubKey := privKey.Public().(ed25519.PublicKey)
	nodeID := hex.EncodeToString(pubKey[:16])

	virtualIP := deriveVirtualIP(pubKey, cfg.Subnet)

	self := &MeshNode{
		ID:        nodeID,
		PublicKey: pubKey,
		VirtualIP: virtualIP,
		Name:      cfg.NodeName,
		Version:   Version,
		Status:    NodeStatusOnline,
		FirstSeen: time.Now().UTC(),
		LastSeen:  time.Now().UTC(),
		Capabilities: CapabilitySet{
			SupportsCodeSync:  true,
			SupportsStateSync: true,
			SupportsRelay:     cfg.EnableRelay,
		},
		Services: make(map[string]string),
	}

	ctx, cancel := context.WithCancel(context.Background())

	sec := NewSecurityManager(logger, privKey, pubKey)
	transport := NewTransportChain(logger, cfg)
	syncEng := NewSyncEngine(logger, nodeID, cfg.SyncInterval)
	disc := NewDiscoveryEngine(logger, cfg)

	var relaySrv *RelayServer
	if cfg.EnableRelay {
		relaySrv = NewRelayServer(logger, cfg.ListenPort+1)
	}

	mn := &MeshNetwork{
		logger:      logger,
		config:      cfg,
		ctx:         ctx,
		cancel:      cancel,
		self:        self,
		peers:       make(map[string]*MeshNode),
		transport:   transport,
		discovery:   disc,
		sync:        syncEng,
		security:    sec,
		eventCh:     make(chan MeshEvent, 256),
		done:        make(chan struct{}),
		startTime:   time.Now().UTC(),
		relayServer: relaySrv,
	}

	transport.SetAuthCallback(func(addr string, nodeInfo *NodeInfo) (*MeshNode, error) {
		return mn.handleIncomingConnection(addr, nodeInfo)
	})
	transport.SetNodeID(nodeID)

	return mn, nil
}

// Start launches the mesh network – discovery, heartbeat, and sync loops.
func (mn *MeshNetwork) Start() error {
	mn.logger.Info("Starting mesh network",
		"node_id", truncateID(mn.self.ID),
		"virtual_ip", mn.self.VirtualIP,
		"subnet", mn.config.Subnet,
	)

	if err := mn.transport.Start(mn.ctx); err != nil {
		return fmt.Errorf("transport start: %w", err)
	}
	if err := mn.discovery.Start(mn.ctx, mn.self, mn.transport); err != nil {
		return fmt.Errorf("discovery start: %w", err)
	}
	mn.sync.Start(mn.ctx)

	if mn.config.EnableRelay && mn.relayServer != nil {
		if err := mn.relayServer.Start(); err != nil {
			mn.logger.Warn("Relay server start failed", "error", err)
		} else {
			mn.logger.Info("Relay server started", "addr", mn.relayServer.Addr())
		}
	}

	go mn.heartbeatLoop()
	go mn.discoveryEventLoop()

	mn.logger.Info("Mesh network started successfully",
		"node_id", truncateID(mn.self.ID),
		"transports", mn.transport.Available(),
	)
	return nil
}

// Stop gracefully shuts down the mesh network.
func (mn *MeshNetwork) Stop() {
	mn.logger.Info("Stopping mesh network")
	mn.cancel()
	mn.sync.Stop()
	mn.discovery.Stop()
	mn.transport.Stop()
	if mn.relayServer != nil {
		mn.relayServer.Stop()
	}
	close(mn.done)
	mn.logger.Info("Mesh network stopped")
}

// Self returns this node's MeshNode identity.
func (mn *MeshNetwork) Self() *MeshNode {
	mn.mu.RLock()
	defer mn.mu.RUnlock()
	return mn.self.Clone()
}

// Peers returns a snapshot of all currently connected peers.
func (mn *MeshNetwork) Peers() []*MeshNode {
	mn.mu.RLock()
	defer mn.mu.RUnlock()
	out := make([]*MeshNode, 0, len(mn.peers))
	for _, p := range mn.peers {
		out = append(out, p.Clone())
	}
	return out
}

// PeerCount returns the number of connected peers.
func (mn *MeshNetwork) PeerCount() int {
	mn.mu.RLock()
	defer mn.mu.RUnlock()
	return len(mn.peers)
}

// GetPeer returns a specific peer by ID.
func (mn *MeshNetwork) GetPeer(id string) *MeshNode {
	mn.mu.RLock()
	defer mn.mu.RUnlock()
	p, ok := mn.peers[id]
	if !ok {
		return nil
	}
	return p.Clone()
}

// Events returns a channel of mesh events for consumers.
func (mn *MeshNetwork) Events() <-chan MeshEvent {
	return mn.eventCh
}

// SyncCode pushes code/data to the mesh for distribution.
func (mn *MeshNetwork) SyncCode(name string, version string, data []byte) error {
	return mn.sync.PublishCode(mn.ctx, name, version, data)
}

// SyncState broadcasts a state update to all peers.
func (mn *MeshNetwork) SyncState(key string, value []byte) error {
	return mn.sync.PublishState(mn.ctx, key, value)
}

// SendToPeer sends an arbitrary message to a specific peer.
func (mn *MeshNetwork) SendToPeer(peerID string, msgType string, payload []byte) error {
	return mn.transport.SendToPeer(peerID, msgType, payload)
}

// Broadcast sends a message to all connected peers.
func (mn *MeshNetwork) Broadcast(msgType string, payload []byte) error {
	return mn.transport.Broadcast(msgType, payload)
}

// handleIncomingConnection authenticates and registers a newly connected peer.
func (mn *MeshNetwork) handleIncomingConnection(addr string, nodeInfo *NodeInfo) (*MeshNode, error) {
	if nodeInfo == nil {
		return nil, fmt.Errorf("node info is nil")
	}

	if !mn.security.Authenticate(nodeInfo) {
		return nil, fmt.Errorf("authentication failed for %s", nodeInfo.ID)
	}

	now := time.Now().UTC()
	peer := &MeshNode{
		ID:        nodeInfo.ID,
		PublicKey: nodeInfo.PublicKey,
		VirtualIP: nodeInfo.VirtualIP,
		Name:      nodeInfo.Name,
		Version:   nodeInfo.Version,
		Status:    NodeStatusOnline,
		FirstSeen: now,
		LastSeen:  now,
		Capabilities: nodeInfo.Capabilities,
		Services: nodeInfo.Services,
		Addresses: []string{addr},
	}

	mn.mu.Lock()
	if existing, ok := mn.peers[peer.ID]; ok {
		existing.Status = NodeStatusOnline
		existing.LastSeen = now
		existing.Addresses = append(existing.Addresses, addr)
		if len(existing.Addresses) > 10 {
			existing.Addresses = existing.Addresses[len(existing.Addresses)-10:]
		}
		peer = existing
	} else {
		mn.peers[peer.ID] = peer
	}
	peerCount := len(mn.peers)
	mn.mu.Unlock()

	mn.logger.Info("Peer connected",
		"peer_id", truncateID(peer.ID),
		"name", peer.Name,
		"addr", addr,
		"total_peers", peerCount,
	)

	mn.emit(MeshEvent{Type: EventPeerJoined, NodeID: peer.ID, Payload: peer.Clone()})
	return peer, nil
}

// heartbeatLoop periodically announces presence and checks peer liveness.
func (mn *MeshNetwork) heartbeatLoop() {
	ticker := time.NewTicker(mn.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-mn.ctx.Done():
			return
		case <-ticker.C:
			mn.sendHeartbeat()
			mn.pruneStalePeers()
		}
	}
}

func (mn *MeshNetwork) sendHeartbeat() {
	hb := HeartbeatMessage{
		NodeID:    mn.self.ID,
		Name:      mn.self.Name,
		VirtualIP: mn.self.VirtualIP,
		Version:   Version,
		Timestamp: time.Now().UTC().UnixNano(),
	}
	if err := mn.transport.Broadcast("heartbeat", mustMarshal(hb)); err != nil {
		mn.logger.Debug("Heartbeat broadcast failed", "error", err)
	}
}

func (mn *MeshNetwork) pruneStalePeers() {
	cutoff := time.Now().UTC().Add(-3 * mn.config.HeartbeatInterval)

	mn.mu.Lock()
	for id, peer := range mn.peers {
		if peer.LastSeen.Before(cutoff) {
			delete(mn.peers, id)
			mn.logger.Debug("Pruned stale peer", "peer_id", truncateID(id))
			mn.emit(MeshEvent{Type: EventPeerLeft, NodeID: id, Payload: peer.Clone()})
		}
	}
	mn.mu.Unlock()
}

// discoveryEventLoop processes peer discovery results.
func (mn *MeshNetwork) discoveryEventLoop() {
	discEvents := mn.discovery.Events()
	for {
		select {
		case <-mn.ctx.Done():
			return
		case ev, ok := <-discEvents:
			if !ok {
				return
			}
			switch e := ev.Payload.(type) {
			case *MeshNode:
				mn.mu.RLock()
				_, exists := mn.peers[e.ID]
				mn.mu.RUnlock()
				if !exists {
					mn.logger.Info("Discovered new peer",
						"peer_id", truncateID(e.ID),
						"name", e.Name,
						"method", ev.Type,
					)
				}
			}
		}
	}
}

func (mn *MeshNetwork) emit(ev MeshEvent) {
	select {
	case mn.eventCh <- ev:
	default:
	}
}

// Info returns runtime statistics about the mesh network.
func (mn *MeshNetwork) Info() map[string]interface{} {
	mn.mu.RLock()
	peerCount := len(mn.peers)
	mn.mu.RUnlock()

	peers := mn.Peers()
	peerVersions := make(map[string]int)
	for _, p := range peers {
		peerVersions[p.Version]++
	}

	relayAddr := ""
	relayClients := 0
	if mn.relayServer != nil {
		relayAddr = mn.relayServer.Addr()
		relayClients = mn.relayServer.ClientCount()
	}

	return map[string]interface{}{
		"node_id":         truncateID(mn.self.ID),
		"node_name":       mn.self.Name,
		"virtual_ip":      mn.self.VirtualIP.String(),
		"peer_count":      peerCount,
		"subnet":          mn.config.Subnet,
		"transports":      mn.transport.Available(),
		"uptime":          time.Since(mn.startTime).String(),
		"sync_enabled":    true,
		"relay_enabled":   mn.config.EnableRelay,
		"relay_addr":      relayAddr,
		"relay_clients":   relayClients,
		"lan_enabled":     mn.config.EnableLAN,
		"peer_versions":   peerVersions,
	}
}

func (mn *MeshNetwork) InjectDiscoveredPeer(peerID string, name string, addrs []string) {
	mn.mu.Lock()
	defer mn.mu.Unlock()
	if _, exists := mn.peers[peerID]; !exists {
		node := &MeshNode{
			ID:        peerID,
			Name:      name,
			Addresses: addrs,
			Status:    NodeStatusOnline,
			LastSeen:  time.Now().UTC(),
			FirstSeen: time.Now().UTC(),
		}
		mn.peers[peerID] = node
		mn.logger.Info("P2P-discovered peer added to mesh", "peer_id", truncateID(peerID))
		mn.emit(MeshEvent{Type: EventPeerJoined, NodeID: peerID, Payload: node})
	}
}

func (mn *MeshNetwork) SendViaP2P(sendFn func(peerID string, data []byte) error) {
	mn.mu.RLock()
	peers := make([]*MeshNode, 0, len(mn.peers))
	for _, p := range mn.peers {
		peers = append(peers, p)
	}
	mn.mu.RUnlock()
	for _, p := range peers {
		env := Envelope{
			Type:    "p2p_mesh_sync",
			From:    mn.self.ID,
			To:      p.ID,
			Payload: mustMarshal(mn.self),
			SentAt:  time.Now().UTC().UnixNano(),
		}
		data, _ := json.Marshal(env)
		if err := sendFn(p.ID, data); err != nil {
			mn.logger.Debug("P2P send failed", "peer", truncateID(p.ID), "error", err)
		}
	}
}

func (mn *MeshNetwork) BanPeer(peerID string) {
	mn.security.BanPeer(peerID, "admin")
}

func (mn *MeshNetwork) AdmitPeer(peerID string) {
	mn.security.AdmitPeer(peerID)
}

func (mn *MeshNetwork) TriggerCodeSync() int {
	mn.mu.RLock()
	peers := make([]string, 0, len(mn.peers))
	for id := range mn.peers {
		peers = append(peers, id)
	}
	mn.mu.RUnlock()
	for _, id := range peers {
		mn.logger.Debug("Triggered code sync to peer", "peer_id", truncateID(id))
	}
	return len(peers)
}

func (mn *MeshNetwork) TriggerStateSync() int {
	return mn.TriggerCodeSync()
}

func deriveVirtualIP(pubKey ed25519.PublicKey, subnet string) net.IP {
	seed := make([]byte, 8)
	h := pubKey
	for i := 0; i < len(h) && i < 8; i++ {
		seed[i] = h[i]
	}
	ip := net.ParseIP("fd00:0:0:0:0:0:0:0")
	if ip == nil {
		ip = net.ParseIP("fd00::1")
	}
	copy(ip[8:], seed)
	return ip
}

func mustMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("mesh marshal: %v", err))
	}
	return data
}
