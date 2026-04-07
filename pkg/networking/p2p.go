// Package networking provides P2P networking capabilities for the APA agent.
package networking

import (
	"context"
	stdcrypto "crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/naviNBRuas/APA/pkg/module"
	"github.com/naviNBRuas/APA/pkg/policy"
	"github.com/naviNBRuas/APA/pkg/update"
)

const (
	HeartbeatTopic      = "apa/heartbeat/1.0.0"
	ModuleTopic         = "apa/modules/1.0.0"
	ControllerCommTopic = "apa/controller-comm/1.0.0"
	LeaderElectionTopic = "apa/leader-election/1.0.0"
	ModuleFetchProtocol = "/apa/fetch-module/1.0.0"
	UpdateFetchProtocol = "/apa/fetch-update/1.0.0"
	PropagationProtocol = "/apa/propagate/1.0.0"
)

// PropagationPayload is exchanged over the propagation protocol to deliver
// signed agent binaries between peers.
type PropagationPayload struct {
	FileName  string `json:"file_name"`
	Hash      string `json:"hash"`
	Signature []byte `json:"signature"`
	PublicKey []byte `json:"public_key"`
	Payload   []byte `json:"payload"`
}

// P2P manages the libp2p host and networking components for the agent.
type P2P struct {
	AdmittedPeers        map[peer.ID]bool // Explicit peer admission opt-in (exported)
	logger               *slog.Logger
	host                 host.Host
	dht                  *dht.IpfsDHT
	pubsub               *pubsub.PubSub
	heartbeatTopic       *pubsub.Topic
	moduleTopic          *pubsub.Topic
	controllerCommTopic  *pubsub.Topic
	leaderElectionTopic  *pubsub.Topic
	advancedDiscovery    *AdvancedDiscovery
	forwardDecider       ForwardDecider
	FetchModuleHandler   func(name, version string) (*module.Manifest, []byte, error)
	FetchUpdateHandler   func(version string) (*update.ReleaseInfo, []byte, error)
	OnModuleAnnouncement func(ModuleAnnouncementMessage)
	resilienceCancel     context.CancelFunc
	config               Config
	propagationHandler   func(context.Context, peer.ID, PropagationPayload) error
	privKey              crypto.PrivKey
}

// HostID returns the local host peer ID as a string.
func (p *P2P) HostID() string {
	if p == nil || p.host == nil {
		return ""
	}
	return p.host.ID().String()
}

// PeerCount returns the number of connected peers.
func (p *P2P) PeerCount() int {
	if p == nil || p.host == nil {
		return 0
	}
	return len(p.host.Network().Peers())
}

// SetForwardDecider installs a forward decider used to gate hop-by-hop tasks.
func (p *P2P) SetForwardDecider(decider ForwardDecider) {
	p.forwardDecider = decider
}

// Config holds the configuration for the P2P networking.
type Config struct {
	ListenAddresses            []string      `yaml:"listen_addresses"`
	BootstrapPeers             []string      `yaml:"bootstrap_peers"`
	HeartbeatInterval          time.Duration `yaml:"heartbeat_interval"`
	ServiceTag                 string        `yaml:"service_tag"`
	IsolationBootstrapInterval time.Duration `yaml:"isolation_bootstrap_interval"`
	TopicRejoinInterval        time.Duration `yaml:"topic_rejoin_interval"`
	ReconnectBackoffMin        time.Duration `yaml:"reconnect_backoff_min"`
	ReconnectBackoffMax        time.Duration `yaml:"reconnect_backoff_max"`
	EnableRelayService         bool          `yaml:"enable_relay_service"`
}

// ModuleAnnouncementMessage represents a message announcing a module.
type ModuleAnnouncementMessage struct {
	Manifest        module.Manifest `json:"manifest"`
	AnnouncerPeerID string          `json:"announcer_peer_id"`
}

// ControllerMessage represents a message between controllers.
type ControllerMessage struct {
	Type         string          `json:"type"`
	Data         json.RawMessage `json:"data"`
	SenderPeerID string          `json:"sender_peer_id"`
}

// LeaderElectionMessage represents a leader election message.
type LeaderElectionMessage struct {
	CandidateID  string    `json:"candidate_id"`
	SenderPeerID string    `json:"sender_peer_id"`
	Rank         int       `json:"rank"`
	IsLeader     bool      `json:"is_leader"`
	Timestamp    time.Time `json:"timestamp"`
}

// NewP2P creates a new P2P networking instance.
func NewP2P(ctx context.Context, logger *slog.Logger, config Config, peerID peer.ID, privKey crypto.PrivKey, policyEnforcer policy.PolicyEnforcer) (*P2P, error) {
	// ...existing code...
	// ...existing code...
	// Provide resilient defaults if none supplied
	if len(config.ListenAddresses) == 0 {
		config.ListenAddresses = []string{
			"/ip4/0.0.0.0/tcp/4001",
			"/ip4/0.0.0.0/udp/4001/quic-v1",
			"/ip4/0.0.0.0/tcp/4002/ws",
		}
	}

	// Resilience defaults
	if config.IsolationBootstrapInterval <= 0 {
		config.IsolationBootstrapInterval = 90 * time.Second
	}
	if config.TopicRejoinInterval <= 0 {
		config.TopicRejoinInterval = 2 * time.Minute
	}
	if config.ReconnectBackoffMin <= 0 {
		config.ReconnectBackoffMin = 200 * time.Millisecond
	}
	if config.ReconnectBackoffMax <= 0 {
		config.ReconnectBackoffMax = 1200 * time.Millisecond
	}
	if config.ReconnectBackoffMax < config.ReconnectBackoffMin {
		config.ReconnectBackoffMax = config.ReconnectBackoffMin * 2
	}

	// Create the libp2p host
	hostOpts := []libp2p.Option{
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings(config.ListenAddresses...),
		libp2p.EnableHolePunching(), // Enable Hole Punching
		libp2p.EnableNATService(),   // Enable NAT service
		libp2p.NATPortMap(),         // Enable NAT port mapping
	}

	// If acting as a relay server, expose the circuit relay v2 service. Otherwise,
	// enable relay client support so this node can dial via relays.
	if config.EnableRelayService {
		hostOpts = append(hostOpts, libp2p.EnableRelayService())
	} else {
		hostOpts = append(hostOpts, libp2p.EnableRelay())
	}

	host, err := libp2p.New(hostOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	// Create the DHT
	// Use ModeAuto to automatically switch between client and server mode
	// Use Public IPFS DHT protocol if we want to join the global network, or a custom one for private.
	// For "autonomous" discovery without a central server, joining the public DHT is the best bet.
	dhtOpts := []dht.Option{
		dht.Mode(dht.ModeAuto),
	}

	dht, err := dht.New(ctx, host, dhtOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create DHT: %w", err)
	}

	// Create the pubsub system
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub: %w", err)
	}

	// Create the P2P instance
	p2p := &P2P{
		logger:  logger,
		host:    host,
		dht:     dht,
		pubsub:  ps,
		config:  config,
		privKey: privKey,
	}
	p2p.AdmittedPeers = make(map[peer.ID]bool)

	// Create advanced discovery
	p2p.advancedDiscovery = NewAdvancedDiscovery(logger, host, dht, config.ServiceTag)

	// Register reconnect notifee to aggressively recover dropped peers
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	host.Network().Notify(&reconnectNotifee{
		host:       host,
		logger:     logger,
		backoffMin: config.ReconnectBackoffMin,
		backoffMax: config.ReconnectBackoffMax,
		rng:        rng,
	})

	// Set up the update protocol handler
	p2p.setupUpdateProtocol()

	// Register propagation protocol with a placeholder handler; callers can
	// override via RegisterPropagationHandler.
	p2p.setupPropagationProtocol()

	// Connect to bootstrap peers
	if err := p2p.connectToBootstrapPeers(ctx, config.BootstrapPeers); err != nil {
		logger.Warn("Failed to connect to bootstrap peers", "error", err)
	}

	return p2p, nil
}

// connectToBootstrapPeers connects to the configured bootstrap peers.
func (p *P2P) connectToBootstrapPeers(ctx context.Context, bootstrapPeers []string) error {
	var peers []ma.Multiaddr

	if len(bootstrapPeers) == 0 {
		p.logger.Info("No bootstrap peers configured, using default IPFS bootstrap peers")
		peers = dht.DefaultBootstrapPeers
	} else {
		for _, addrStr := range bootstrapPeers {
			addr, err := ma.NewMultiaddr(addrStr)
			if err != nil {
				p.logger.Error("Failed to parse bootstrap peer address", "address", addrStr, "error", err)
				continue
			}
			peers = append(peers, addr)
		}
	}

	for _, addr := range peers {
		// Extract the peer ID from the multiaddress
		peerInfo, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			p.logger.Error("Failed to extract peer info from address", "address", addr.String(), "error", err)
			continue
		}

		// Add the peer to the peerstore
		p.host.Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, peerstore.PermanentAddrTTL)

		// Connect to the peer
		if err := p.host.Connect(ctx, *peerInfo); err != nil {
			p.logger.Debug("Failed to connect to bootstrap peer", "peer", peerInfo.ID, "error", err)
			// Retry with backoff once
			ctxRetry, cancel := context.WithTimeout(ctx, 5*time.Second)
			retryErr := p.host.Connect(ctxRetry, *peerInfo)
			cancel()
			if retryErr != nil {
				p.logger.Debug("Retry connect to bootstrap peer failed", "peer", peerInfo.ID, "error", retryErr)
				continue
			}
		}
		p.logger.Info("Connected to bootstrap peer", "peer", peerInfo.ID)
	}

	return nil
}

// StartDiscovery starts the peer discovery process.
func (p *P2P) StartDiscovery(ctx context.Context) {
	p.logger.Info("Starting peer discovery")

	// Bootstrap the DHT
	if err := p.dht.Bootstrap(ctx); err != nil {
		p.logger.Error("Failed to bootstrap DHT", "error", err)
	}

	// Start advanced discovery
	if p.advancedDiscovery != nil {
		p.advancedDiscovery.Start(ctx)
	}

	// Start resilience loops to keep discovery alive and reconnect when isolated
	p.startResilienceLoops(ctx)
}

// JoinHeartbeatTopic joins the heartbeat topic.
func (p *P2P) JoinHeartbeatTopic(ctx context.Context) error {
	topic, err := p.pubsub.Join(HeartbeatTopic)
	if err != nil {
		return fmt.Errorf("failed to join heartbeat topic: %w", err)
	}

	p.heartbeatTopic = topic
	return nil
}

// IsHeartbeatJoined reports whether the heartbeat topic is active.
func (p *P2P) IsHeartbeatJoined() bool {
	return p != nil && p.heartbeatTopic != nil
}

// StartHeartbeat starts broadcasting heartbeats.
func (p *P2P) StartHeartbeat(ctx context.Context, interval time.Duration) {
	if p.heartbeatTopic == nil {
		p.logger.Error("Heartbeat topic not joined")
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("Stopping heartbeat")
			return
		case <-ticker.C:
			// Create heartbeat message
			msg := map[string]interface{}{
				"peer_id": p.host.ID().String(),
				"time":    time.Now().Unix(),
			}

			// Marshal to JSON
			msgBytes, err := json.Marshal(msg)
			if err != nil {
				p.logger.Error("Failed to marshal heartbeat message", "error", err)
				continue
			}

			// Publish the message
			if err := p.heartbeatTopic.Publish(ctx, msgBytes); err != nil {
				p.logger.Error("Failed to publish heartbeat", "error", err)
			}
		}
	}
}

// JoinModuleTopic joins the module announcement topic.
func (p *P2P) JoinModuleTopic(ctx context.Context) error {
	topic, err := p.pubsub.Join(ModuleTopic)
	if err != nil {
		return fmt.Errorf("failed to join module topic: %w", err)
	}

	p.moduleTopic = topic
	return nil
}

// AnnounceModule announces a module to the network.
func (p *P2P) AnnounceModule(ctx context.Context, manifest module.Manifest) error {
	if p.moduleTopic == nil {
		return fmt.Errorf("module topic not joined")
	}

	// Create announcement message
	msg := ModuleAnnouncementMessage{
		Manifest:        manifest,
		AnnouncerPeerID: p.host.ID().String(),
	}

	// Marshal to JSON
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal module announcement: %w", err)
	}

	if err := publishWithRetry(ctx, p.logger, p.moduleTopic, msgBytes, "module announcement"); err != nil {
		return err
	}

	p.logger.Info("Announced module", "name", manifest.Name, "version", manifest.Version)
	return nil
}

// JoinControllerCommTopic joins the controller communication topic.
func (p *P2P) JoinControllerCommTopic(ctx context.Context) error {
	topic, err := p.pubsub.Join(ControllerCommTopic)
	if err != nil {
		return fmt.Errorf("failed to join controller communication topic: %w", err)
	}

	p.controllerCommTopic = topic
	return nil
}

// IsControllerJoined reports whether the controller communication topic is active.
func (p *P2P) IsControllerJoined() bool {
	return p != nil && p.controllerCommTopic != nil
}

// PublishControllerMessage publishes a controller message to the network.
func (p *P2P) PublishControllerMessage(ctx context.Context, msgBytes []byte) error {
	if p.controllerCommTopic == nil {
		return fmt.Errorf("controller communication topic not joined")
	}

	return publishWithRetry(ctx, p.logger, p.controllerCommTopic, msgBytes, "controller message")
}

// SubscribeControllerMessages subscribes to controller messages.
func (p *P2P) SubscribeControllerMessages(ctx context.Context) (<-chan *ControllerMessage, error) {
	if p.controllerCommTopic == nil {
		return nil, fmt.Errorf("controller communication topic not joined")
	}

	sub, err := p.controllerCommTopic.Subscribe()
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to controller messages: %w", err)
	}

	msgCh := make(chan *ControllerMessage, 10)

	go func() {
		defer close(msgCh)
		defer sub.Cancel()

		for {
			msg, err := sub.Next(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, pubsub.ErrSubscriptionCancelled) {
					return
				}
				p.logger.Error("Failed to read controller message", "error", err)
				time.Sleep(200 * time.Millisecond)
				continue
			}

			if msg == nil {
				continue
			}

			var ctrlMsg ControllerMessage
			if err := json.Unmarshal(msg.Data, &ctrlMsg); err != nil {
				p.logger.Error("Failed to unmarshal controller message", "error", err)
				continue
			}

			peerID, err := peer.IDFromBytes(msg.From)
			if err != nil {
				p.logger.Error("Failed to decode peer ID from message", "error", err)
				continue
			}
			ctrlMsg.SenderPeerID = peerID.String()

			select {
			case msgCh <- &ctrlMsg:
			default:
				p.logger.Warn("Controller message channel full, dropping message")
			}
		}
	}()

	return msgCh, nil
}

// JoinLeaderElectionTopic joins the leader election topic.
func (p *P2P) JoinLeaderElectionTopic(ctx context.Context) error {
	topic, err := p.pubsub.Join(LeaderElectionTopic)
	if err != nil {
		return fmt.Errorf("failed to join leader election topic: %w", err)
	}

	p.leaderElectionTopic = topic
	return nil
}

// IsLeaderElectionJoined reports whether the leader election topic is active.
func (p *P2P) IsLeaderElectionJoined() bool {
	return p != nil && p.leaderElectionTopic != nil
}

// SubscribeLeaderElectionMessages subscribes to leader election messages.
func (p *P2P) SubscribeLeaderElectionMessages(ctx context.Context) (<-chan *LeaderElectionMessage, error) {
	if p.leaderElectionTopic == nil {
		return nil, fmt.Errorf("leader election topic not joined")
	}

	sub, err := p.leaderElectionTopic.Subscribe()
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to leader election messages: %w", err)
	}

	msgCh := make(chan *LeaderElectionMessage, 10)

	go func() {
		defer close(msgCh)
		defer sub.Cancel()

		for {
			msg, err := sub.Next(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, pubsub.ErrSubscriptionCancelled) {
					return
				}
				p.logger.Error("Failed to read leader election message", "error", err)
				time.Sleep(200 * time.Millisecond)
				continue
			}

			if msg == nil {
				continue
			}

			var leMsg LeaderElectionMessage
			if err := json.Unmarshal(msg.Data, &leMsg); err != nil {
				p.logger.Error("Failed to unmarshal leader election message", "error", err)
				continue
			}

			peerID, err := peer.IDFromBytes(msg.From)
			if err != nil {
				p.logger.Error("Failed to decode peer ID from message", "error", err)
				continue
			}
			leMsg.CandidateID = peerID.String()
			leMsg.Timestamp = time.Now()

			select {
			case msgCh <- &leMsg:
			default:
				p.logger.Warn("Leader election message channel full, dropping message")
			}
		}
	}()

	return msgCh, nil
}

// PublishLeaderElectionMessage publishes a leader election message.
func (p *P2P) PublishLeaderElectionMessage(ctx context.Context, msg LeaderElectionMessage) error {
	if p.leaderElectionTopic == nil {
		return fmt.Errorf("leader election topic not joined")
	}

	// Marshal to JSON
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal leader election message: %w", err)
	}

	return publishWithRetry(ctx, p.logger, p.leaderElectionTopic, msgBytes, "leader election message")
}

// GetConnectedPeers returns the list of currently connected peers.
func (p *P2P) GetConnectedPeers() []peer.ID {
	if p.advancedDiscovery != nil {
		return p.advancedDiscovery.GetConnectedPeers()
	}

	// Fallback to basic peer listing
	return p.host.Network().Peers()
}

// AdmitPeer explicitly opts-in a peer for trusted communication.
func (p *P2P) AdmitPeer(id peer.ID) {
	if p == nil {
		return
	}
	if p.AdmittedPeers == nil {
		p.AdmittedPeers = make(map[peer.ID]bool)
	}
	p.AdmittedPeers[id] = true
}

// IsPeerAdmitted checks if a peer is explicitly admitted.
func (p *P2P) IsPeerAdmitted(id peer.ID) bool {
	if p == nil || p.AdmittedPeers == nil {
		return false
	}
	return p.AdmittedPeers[id]
}

// GetTopicHealth reports the join status of all critical topics.
func (p *P2P) GetTopicHealth() map[string]bool {
	return map[string]bool{
		"heartbeat":  p.IsHeartbeatJoined(),
		"controller": p.IsControllerJoined(),
		"leader":     p.IsLeaderElectionJoined(),
		"module":     p.moduleTopic != nil,
	}
}

// HostAddrs returns the multiaddrs the host is listening on.
func (p *P2P) HostAddrs() []ma.Multiaddr {
	if p == nil || p.host == nil {
		return nil
	}
	return p.host.Addrs()
}

// ConnectViaRelay dials a target peer using a relay multiaddr of the form
// /ip4/host/tcp/port/p2p/<relayID>. It will first connect to the relay, then
// to the target via /p2p-circuit.
func (p *P2P) ConnectViaRelay(ctx context.Context, relayAddr string, target peer.ID) error {
	if p == nil || p.host == nil {
		return fmt.Errorf("host not initialized")
	}

	relayMa, err := ma.NewMultiaddr(relayAddr)
	if err != nil {
		return fmt.Errorf("invalid relay multiaddr: %w", err)
	}

	relayInfo, err := peer.AddrInfoFromP2pAddr(relayMa)
	if err != nil {
		return fmt.Errorf("failed to parse relay addr info: %w", err)
	}

	if err := p.host.Connect(ctx, *relayInfo); err != nil {
		return fmt.Errorf("failed to connect to relay: %w", err)
	}

	circuitAddr, err := ma.NewMultiaddr(fmt.Sprintf("%s/p2p-circuit/p2p/%s", relayAddr, target))
	if err != nil {
		return fmt.Errorf("failed to build circuit addr: %w", err)
	}

	targetInfo := peer.AddrInfo{ID: target, Addrs: []ma.Multiaddr{circuitAddr}}
	if err := p.host.Connect(ctx, targetInfo); err != nil {
		return fmt.Errorf("failed to connect to target via relay: %w", err)
	}

	return nil
}

// ConnectToAddrInfo dials a peer using the provided address info.
func (p *P2P) ConnectToAddrInfo(ctx context.Context, info peer.AddrInfo) error {
	if p == nil || p.host == nil {
		return fmt.Errorf("host not initialized")
	}
	return p.host.Connect(ctx, info)
}

// PutDHTValue stores a key/value pair in the DHT for integration harnessing.
func (p *P2P) PutDHTValue(ctx context.Context, key string, val []byte) error {
	if p == nil || p.dht == nil {
		return fmt.Errorf("dht not initialized")
	}
	return p.dht.PutValue(ctx, key, val)
}

// GetDHTValue retrieves a value from the DHT.
func (p *P2P) GetDHTValue(ctx context.Context, key string) ([]byte, error) {
	if p == nil || p.dht == nil {
		return nil, fmt.Errorf("dht not initialized")
	}
	return p.dht.GetValue(ctx, key)
}

// setupUpdateProtocol sets up the update protocol handler
func (p *P2P) setupUpdateProtocol() {
	// Set up the update protocol handler
	p.host.SetStreamHandler(UpdateFetchProtocol, p.handleUpdateFetchRequest)
}

// handleUpdateFetchRequest handles incoming update fetch requests
func (p *P2P) handleUpdateFetchRequest(stream network.Stream) {
	defer stream.Close()

	// Read the version from the stream
	decoder := json.NewDecoder(stream)
	var request struct {
		Version string `json:"version"`
	}

	if err := decoder.Decode(&request); err != nil {
		p.logger.Error("Failed to decode update fetch request", "error", err)
		return
	}

	// Call the handler if it's set
	if p.FetchUpdateHandler != nil {
		release, data, err := p.FetchUpdateHandler(request.Version)
		if err != nil {
			p.logger.Error("Failed to fetch update", "error", err)
			return
		}

		// Send the response
		response := struct {
			Release *update.ReleaseInfo `json:"release"`
			Data    []byte              `json:"data"`
		}{
			Release: release,
			Data:    data,
		}

		encoder := json.NewEncoder(stream)
		if err := encoder.Encode(response); err != nil {
			p.logger.Error("Failed to encode update response", "error", err)
			return
		}
	} else {
		p.logger.Warn("No update handler set")
	}
}

// FetchUpdateFromPeer fetches an update from a specific peer
func (p *P2P) FetchUpdateFromPeer(ctx context.Context, peerID peer.ID, version string) (*update.ReleaseInfo, []byte, error) {
	// Create a stream to the peer
	stream, err := p.host.NewStream(ctx, peerID, UpdateFetchProtocol)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create stream: %w", err)
	}
	defer stream.Close()

	// Send the request
	request := struct {
		Version string `json:"version"`
	}{
		Version: version,
	}

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(request); err != nil {
		return nil, nil, fmt.Errorf("failed to encode request: %w", err)
	}

	// Read the response
	decoder := json.NewDecoder(stream)
	var response struct {
		Release *update.ReleaseInfo `json:"release"`
		Data    []byte              `json:"data"`
	}

	if err := decoder.Decode(&response); err != nil {
		return nil, nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Release, response.Data, nil
}

// RegisterPropagationHandler registers a callback for incoming propagation payloads.
func (p *P2P) RegisterPropagationHandler(handler func(context.Context, peer.ID, PropagationPayload) error) {
	p.propagationHandler = handler
}

// setupPropagationProtocol wires the libp2p stream handler for propagation.
func (p *P2P) setupPropagationProtocol() {
	p.host.SetStreamHandler(PropagationProtocol, p.handlePropagationRequest)
}

// handlePropagationRequest handles incoming propagation payloads with verification.
func (p *P2P) handlePropagationRequest(stream network.Stream) {
	defer stream.Close()

	ctx := context.Background()
	remotePeer := stream.Conn().RemotePeer()

	decoder := json.NewDecoder(stream)
	var payload PropagationPayload
	if err := decoder.Decode(&payload); err != nil {
		p.logger.Error("Failed to decode propagation payload", "peer", remotePeer, "error", err)
		return
	}

	// Hash verification
	sum := sha256.Sum256(payload.Payload)
	computed := hex.EncodeToString(sum[:])
	if payload.Hash != "" && payload.Hash != computed {
		p.logger.Warn("Propagation payload hash mismatch", "peer", remotePeer, "expected", payload.Hash, "computed", computed)
		return
	}

	// Signature verification (optional)
	if len(payload.Signature) > 0 && len(payload.PublicKey) > 0 {
		pub, err := x509.ParsePKIXPublicKey(payload.PublicKey)
		if err != nil {
			p.logger.Warn("Failed to parse propagation public key", "peer", remotePeer, "error", err)
			return
		}
		rsaPub, ok := pub.(*rsa.PublicKey)
		if !ok {
			p.logger.Warn("Propagation public key is not RSA", "peer", remotePeer)
			return
		}
		if err := rsa.VerifyPSS(rsaPub, stdcrypto.SHA256, sum[:], payload.Signature, nil); err != nil {
			p.logger.Warn("Propagation signature verification failed", "peer", remotePeer, "error", err)
			return
		}
	}

	// Invoke registered handler to persist the payload
	if p.propagationHandler != nil {
		if err := p.propagationHandler(ctx, remotePeer, payload); err != nil {
			p.logger.Error("Propagation handler failed", "peer", remotePeer, "error", err)
			_ = json.NewEncoder(stream).Encode(map[string]string{"status": "error", "message": err.Error()})
			return
		}
	}

	if err := json.NewEncoder(stream).Encode(map[string]string{"status": "ok", "message": "received"}); err != nil {
		p.logger.Warn("Failed to send propagation ack", "peer", remotePeer, "error", err)
	}
}

// SendPropagationPayload sends a signed payload to a peer over the propagation protocol.
func (p *P2P) SendPropagationPayload(ctx context.Context, peerID peer.ID, payload PropagationPayload) error {
	if p.forwardDecider != nil && !p.forwardDecider.AllowForward(peerID, len(payload.Payload)) {
		return fmt.Errorf("forward vetoed by policy for peer %s", peerID)
	}
	stream, err := p.host.NewStream(ctx, peerID, PropagationProtocol)
	if err != nil {
		return fmt.Errorf("failed to create propagation stream: %w", err)
	}
	defer stream.Close()

	if err := json.NewEncoder(stream).Encode(payload); err != nil {
		return fmt.Errorf("failed to encode propagation payload: %w", err)
	}

	// Wait for ack
	var ack struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(stream).Decode(&ack); err != nil {
		return fmt.Errorf("failed to decode propagation ack: %w", err)
	}

	if ack.Status != "ok" {
		return fmt.Errorf("peer returned error: %s", ack.Message)
	}

	return nil
}

// FetchModule requests a module (manifest + wasm bytes) from a peer.
// This is a simplified placeholder that relies on the remote peer exposing the
// ModuleFetchProtocol handler. If the handler is not available, an error is returned.
func (p *P2P) FetchModule(ctx context.Context, peerID peer.ID, name, version string) (*module.Manifest, []byte, error) {
	stream, err := p.host.NewStream(ctx, peerID, ModuleFetchProtocol)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create module fetch stream: %w", err)
	}
	defer stream.Close()

	request := struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}{Name: name, Version: version}

	if err := json.NewEncoder(stream).Encode(request); err != nil {
		return nil, nil, fmt.Errorf("failed to encode module fetch request: %w", err)
	}

	var response struct {
		Manifest *module.Manifest `json:"manifest"`
		Wasm     []byte           `json:"wasm"`
	}

	if err := json.NewDecoder(stream).Decode(&response); err != nil {
		return nil, nil, fmt.Errorf("failed to decode module fetch response: %w", err)
	}

	return response.Manifest, response.Wasm, nil
}

// ClosePeer closes any connections to the given peer.
func (p *P2P) ClosePeer(peerID peer.ID) error {
	return p.host.Network().ClosePeer(peerID)
}

// startResilienceLoops continuously re-validates connectivity, discovery, and topic health.
func (p *P2P) startResilienceLoops(ctx context.Context) {
	if p.resilienceCancel != nil {
		return // already running
	}

	loopCtx, cancel := context.WithCancel(ctx)
	p.resilienceCancel = cancel

	// Periodically ensure we have peers and re-bootstrap if isolated.
	go func() {
		ticker := time.NewTicker(p.config.IsolationBootstrapInterval)
		defer ticker.Stop()
		for {
			select {
			case <-loopCtx.Done():
				return
			case <-ticker.C:
				if len(p.GetConnectedPeers()) == 0 {
					p.logger.Warn("No connected peers detected, re-bootstrapping")
					if err := p.connectToBootstrapPeers(loopCtx, p.config.BootstrapPeers); err != nil {
						p.logger.Debug("Re-bootstrap attempt failed", "error", err)
					}
					if err := p.dht.Bootstrap(loopCtx); err != nil {
						p.logger.Debug("DHT re-bootstrap failed", "error", err)
					}
				}
			}
		}
	}()

	// Periodically re-join topics in case of transient failures.
	go func() {
		ticker := time.NewTicker(p.config.TopicRejoinInterval)
		defer ticker.Stop()
		for {
			select {
			case <-loopCtx.Done():
				return
			case <-ticker.C:
				p.ensureTopics(loopCtx)
			}
		}
	}()
}

func (p *P2P) ensureTopics(ctx context.Context) {
	if p.heartbeatTopic == nil {
		_ = p.JoinHeartbeatTopic(ctx)
	}
	if p.moduleTopic == nil {
		_ = p.JoinModuleTopic(ctx)
	}
	if p.controllerCommTopic == nil {
		_ = p.JoinControllerCommTopic(ctx)
	}
	if p.leaderElectionTopic == nil {
		_ = p.JoinLeaderElectionTopic(ctx)
	}
}

// publishWithRetry provides limited backoff and retry for pubsub publishes.
func publishWithRetry(ctx context.Context, logger *slog.Logger, topic *pubsub.Topic, msg []byte, label string) error {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if err := topic.Publish(ctx, msg); err != nil {
			lastErr = err
			logger.Warn("Pubsub publish failed, will retry", "label", label, "attempt", attempt+1, "error", err)
			time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
			continue
		}
		return nil
	}
	return fmt.Errorf("failed to publish %s after retries: %w", label, lastErr)
}

// reconnectNotifee attempts to reconnect to peers on disconnect events.
type reconnectNotifee struct {
	host       host.Host
	logger     *slog.Logger
	backoffMin time.Duration
	backoffMax time.Duration
	rng        *rand.Rand
	mu         sync.Mutex
}

func (n *reconnectNotifee) jitterDelay() time.Duration {
	if n.rng == nil || n.backoffMin <= 0 {
		return 0
	}
	spread := n.backoffMax - n.backoffMin
	if spread <= 0 {
		return n.backoffMin
	}

	n.mu.Lock()
	delay := n.backoffMin + time.Duration(n.rng.Int63n(int64(spread)))
	n.mu.Unlock()
	return delay
}

func (n *reconnectNotifee) Disconnected(_ network.Network, c network.Conn) {
	peerID := c.RemotePeer()
	addrs := n.host.Peerstore().Addrs(peerID)
	if len(addrs) == 0 {
		return
	}
	go func() {
		delay := n.jitterDelay()
		if delay > 0 {
			time.Sleep(delay)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := n.host.Connect(ctx, peer.AddrInfo{ID: peerID, Addrs: addrs}); err != nil {
			n.logger.Debug("Reconnect attempt failed", "peer", peerID, "error", err)
		} else {
			n.logger.Info("Reconnected to peer", "peer", peerID, "delay", delay)
		}
	}()
}

func (n *reconnectNotifee) Connected(net network.Network, c network.Conn)      {}
func (n *reconnectNotifee) OpenedStream(net network.Network, s network.Stream) {}
func (n *reconnectNotifee) ClosedStream(net network.Network, s network.Stream) {}
func (n *reconnectNotifee) Listen(net network.Network, addr ma.Multiaddr)      {}
func (n *reconnectNotifee) ListenClose(net network.Network, addr ma.Multiaddr) {}

// GetReputationScore returns the reputation score for a peer if the advanced
// discovery system is available. Falls back to a neutral score otherwise.
func (p *P2P) GetReputationScore(id peer.ID) float64 {
	if p == nil || p.advancedDiscovery == nil || p.advancedDiscovery.reputationRouting == nil {
		return 50.0
	}
	return p.advancedDiscovery.reputationRouting.reputation.GetReputationScore(id)
}

// IsPeerConnected reports whether the given peer is currently connected.
func (p *P2P) IsPeerConnected(id peer.ID) bool {
	if p == nil {
		return false
	}
	for _, connected := range p.GetConnectedPeers() {
		if connected == id {
			return true
		}
	}
	return false
}

// Shutdown gracefully shuts down the P2P networking.
func (p *P2P) Shutdown() error {
	p.logger.Info("Shutting down P2P networking")

	if p.resilienceCancel != nil {
		p.resilienceCancel()
	}

	// Close topics
	if p.heartbeatTopic != nil {
		p.heartbeatTopic.Close()
	}
	if p.moduleTopic != nil {
		p.moduleTopic.Close()
	}
	if p.controllerCommTopic != nil {
		p.controllerCommTopic.Close()
	}
	if p.leaderElectionTopic != nil {
		p.leaderElectionTopic.Close()
	}

	// Close the DHT
	if err := p.dht.Close(); err != nil {
		p.logger.Error("Failed to close DHT", "error", err)
	}

	// Close the host
	if err := p.host.Close(); err != nil {
		return fmt.Errorf("failed to close libp2p host: %w", err)
	}

	return nil
}
