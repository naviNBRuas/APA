// Package networking provides P2P networking capabilities for the APA agent.
package networking

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
	"reflect"

	"github.com/naviNBRuas/APA/pkg/module"
	"github.com/naviNBRuas/APA/pkg/update"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/naviNBRuas/APA/pkg/policy"
)

const (
	HeartbeatTopic        = "apa/heartbeat/1.0.0"
	ModuleTopic           = "apa/modules/1.0.0"
	ControllerCommTopic   = "apa/controller-comm/1.0.0"
	LeaderElectionTopic   = "apa/leader-election/1.0.0"
	ModuleFetchProtocol   = "/apa/fetch-module/1.0.0"
	UpdateFetchProtocol   = "/apa/fetch-update/1.0.0"
)

// P2P manages the libp2p host and networking components for the agent.
type P2P struct {
	logger                *slog.Logger
	host                  host.Host
	dht                   *dht.IpfsDHT
	pubsub                *pubsub.PubSub
	heartbeatTopic        *pubsub.Topic
	moduleTopic           *pubsub.Topic
	controllerCommTopic   *pubsub.Topic
	leaderElectionTopic   *pubsub.Topic
	advancedDiscovery     *AdvancedDiscovery
	FetchModuleHandler    func(name, version string) (*module.Manifest, []byte, error)
	FetchUpdateHandler    func(version string) (*update.ReleaseInfo, []byte, error)
	OnModuleAnnouncement  func(ModuleAnnouncementMessage)
}

// Config holds the configuration for the P2P networking.
type Config struct {
	ListenAddresses     []string      `yaml:"listen_addresses"`
	BootstrapPeers      []string      `yaml:"bootstrap_peers"`
	HeartbeatInterval   time.Duration `yaml:"heartbeat_interval"`
	ServiceTag          string        `yaml:"service_tag"`
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
	CandidateID string    `json:"candidate_id"`
	Rank        int       `json:"rank"`
	IsLeader    bool      `json:"is_leader"`
	Timestamp   time.Time `json:"timestamp"`
}

// NewP2P creates a new P2P networking instance.
func NewP2P(ctx context.Context, logger *slog.Logger, config Config, peerID peer.ID, privKey crypto.PrivKey, policyEnforcer policy.PolicyEnforcer) (*P2P, error) {
	// Create the libp2p host
	host, err := libp2p.New(
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings(config.ListenAddresses...),
		libp2p.EnableRelay(),           // Enable Circuit Relay v2 (client and server)
		libp2p.EnableHolePunching(),    // Enable Hole Punching
		libp2p.EnableNATService(),      // Enable NAT service
		libp2p.NATPortMap(),            // Enable NAT port mapping
	)
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
		logger: logger,
		host:   host,
		dht:    dht,
		pubsub: ps,
	}

	// Create advanced discovery
	p2p.advancedDiscovery = NewAdvancedDiscovery(logger, host, dht, config.ServiceTag)

	// Set up the update protocol handler
	p2p.setupUpdateProtocol()

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
		} else {
			p.logger.Info("Connected to bootstrap peer", "peer", peerInfo.ID)
		}
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

	// Publish the message
	if err := p.moduleTopic.Publish(ctx, msgBytes); err != nil {
		return fmt.Errorf("failed to publish module announcement: %w", err)
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

// PublishControllerMessage publishes a controller message to the network.
func (p *P2P) PublishControllerMessage(ctx context.Context, msgBytes []byte) error {
	if p.controllerCommTopic == nil {
		return fmt.Errorf("controller communication topic not joined")
	}

	// Publish the message
	if err := p.controllerCommTopic.Publish(ctx, msgBytes); err != nil {
		return fmt.Errorf("failed to publish controller message: %w", err)
	}

	return nil
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

		// Use reflection to access the unexported ch field
		subValue := reflect.ValueOf(sub)
		chField := subValue.Elem().FieldByName("ch")
		if !chField.IsValid() {
			p.logger.Error("Failed to access subscription channel")
			return
		}

		ch := chField.Interface().(<-chan *pubsub.Message)

		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok || msg == nil {
					continue
				}

				// Unmarshal the message
				var ctrlMsg ControllerMessage
				if err := json.Unmarshal(msg.Data, &ctrlMsg); err != nil {
					p.logger.Error("Failed to unmarshal controller message", "error", err)
					continue
				}

				// Set the sender peer ID
				peerID, err := peer.IDFromBytes(msg.From)
				if err != nil {
					p.logger.Error("Failed to decode peer ID from message", "error", err)
					continue
				}
				ctrlMsg.SenderPeerID = peerID.String()

				// Send the message to the channel
				select {
				case msgCh <- &ctrlMsg:
				default:
					p.logger.Warn("Controller message channel full, dropping message")
				}
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

		// Use reflection to access the unexported ch field
		subValue := reflect.ValueOf(sub)
		chField := subValue.Elem().FieldByName("ch")
		if !chField.IsValid() {
			p.logger.Error("Failed to access subscription channel")
			return
		}

		ch := chField.Interface().(<-chan *pubsub.Message)

		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok || msg == nil {
					continue
				}

				// Unmarshal the message
				var leMsg LeaderElectionMessage
				if err := json.Unmarshal(msg.Data, &leMsg); err != nil {
					p.logger.Error("Failed to unmarshal leader election message", "error", err)
					continue
				}

				// Set the sender peer ID
				peerID, err := peer.IDFromBytes(msg.From)
				if err != nil {
					p.logger.Error("Failed to decode peer ID from message", "error", err)
					continue
				}
				leMsg.CandidateID = peerID.String()
				leMsg.Timestamp = time.Now()

				// Send the message to the channel
				select {
				case msgCh <- &leMsg:
				default:
					p.logger.Warn("Leader election message channel full, dropping message")
				}
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

	// Publish the message
	if err := p.leaderElectionTopic.Publish(ctx, msgBytes); err != nil {
		return fmt.Errorf("failed to publish leader election message: %w", err)
	}

	return nil
}

// GetConnectedPeers returns the list of currently connected peers.
func (p *P2P) GetConnectedPeers() []peer.ID {
	if p.advancedDiscovery != nil {
		return p.advancedDiscovery.GetConnectedPeers()
	}
	
	// Fallback to basic peer listing
	return p.host.Network().Peers()
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

// Shutdown gracefully shuts down the P2P networking.
func (p *P2P) Shutdown() error {
	p.logger.Info("Shutting down P2P networking")

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