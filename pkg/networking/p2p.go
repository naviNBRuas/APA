package networking

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/naviNBRuas/APA/pkg/module"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	discovery "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/naviNBRuas/APA/pkg/policy"
)

const (
	HeartbeatTopic    = "apa/heartbeat/1.0.0"
	ModuleTopic       = "apa/modules/1.0.0"
	ModuleFetchProtocol = "/apa/fetch-module/1.0.0"
)

// P2P manages the libp2p host, peer discovery, and pubsub.
type P2P struct {
	logger               *slog.Logger
	Host                 host.Host
	pubsub               *pubsub.PubSub
	heartbeatTopic         *pubsub.Topic
	heartbeatSub           *pubsub.Subscription
	moduleTopic            *pubsub.Topic
	moduleSub              *pubsub.Subscription
	dht                  *dht.IpfsDHT
	routingDiscovery     *discovery.RoutingDiscovery
	peerstore            peerstore.Peerstore
	OnModuleAnnouncement func(announcement ModuleAnnouncementMessage)
	FetchModuleHandler   func(name, version string) (*module.Manifest, []byte, error)
	policyEnforcer       policy.PolicyEnforcer
	serviceTag           string
}

// Config holds the configuration for the P2P network.
type Config struct {
	ListenAddrs      []string      `yaml:"listen_addresses"`
	BootstrapPeers   []string      `yaml:"bootstrap_peers"`
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`
	ServiceTag       string        `yaml:"service_tag"`
}

// HeartbeatMessage is the message broadcast by agents to announce their presence.
type HeartbeatMessage struct {
	PeerID    string    `json:"peer_id"`
	Timestamp time.Time `json:"timestamp"`
}

// ModuleAnnouncementMessage is broadcast when an agent loads a new module.
type ModuleAnnouncementMessage struct {
	AnnouncerPeerID peer.ID         `json:"announcer_peer_id"`
	Manifest        module.Manifest `json:"manifest"`
}

// ModuleFetchRequest is sent to a peer to request a module.
type ModuleFetchRequest struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ModuleFetchResponse is the response to a module fetch request.
// It contains either the module data or an error message.

type ModuleFetchResponse struct {
	Manifest  *module.Manifest `json:"manifest,omitempty"`
	WasmBytes []byte           `json:"wasm_bytes,omitempty"`
	Error     string           `json:"error,omitempty"`
}

// NewP2P creates and initializes a new libp2p host.
func NewP2P(ctx context.Context, logger *slog.Logger, cfg Config, id peer.ID, privKey crypto.PrivKey, policyEnforcer policy.PolicyEnforcer) (*P2P, error) {
	h, err := libp2p.New(
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings(cfg.ListenAddrs...),
		libp2p.EnableNATService(), // Enable NAT traversal
		libp2p.EnableRelayService(), // Enable circuit relay service
		libp2p.EnableAutoRelayWithStaticRelays(nil), // Enable automatic relay usage (no static relays for now)
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub service: %w", err)
	}

	p := &P2P{
		logger:     logger,
		Host:       h,
		pubsub:     ps,
		peerstore:  h.Peerstore(),
		policyEnforcer: policyEnforcer,
		serviceTag: cfg.ServiceTag,
	}

	// Create a new Kademlia DHT for peer discovery.
	kadDHT, err := dht.New(ctx, h, dht.Mode(dht.ModeServer))
	if err != nil {
		return nil, fmt.Errorf("failed to create Kademlia DHT: %w", err)
	}
	p.dht = kadDHT

	// Create a routing discovery service using the DHT.
	p.routingDiscovery = discovery.NewRoutingDiscovery(kadDHT)

	h.SetStreamHandler(ModuleFetchProtocol, p.handleModuleFetchStream)

	logger.Info("P2P host created", "id", h.ID(), "addrs", h.Addrs())

	// Connect to bootstrap peers
	p.connectToBootstrapPeers(ctx, cfg.BootstrapPeers)

	// Connect to bootstrap peers
	p.connectToBootstrapPeers(ctx, cfg.BootstrapPeers)

	return p, nil
}

// Shutdown gracefully closes the libp2p host.
func (p *P2P) Shutdown() error {
	p.logger.Info("Shutting down P2P host")
	if p.heartbeatSub != nil {
		p.heartbeatSub.Cancel()
	}
	if p.heartbeatTopic != nil {
		p.heartbeatTopic.Close()
	}
	if p.moduleSub != nil {
		p.moduleSub.Cancel()
	}
	if p.moduleTopic != nil {
		p.moduleTopic.Close()
	}
	return p.Host.Close()
}

// ClosePeer closes the connection to a specific peer.
func (p *P2P) ClosePeer(peerID peer.ID) error {
	return p.Host.Network().ClosePeer(peerID)
}

// connectToBootstrapPeers connects to the initial set of bootstrap peers.
func (p *P2P) connectToBootstrapPeers(ctx context.Context, bootstrapPeers []string) {
	p.logger.Info("Connecting to bootstrap peers...")
	var pis []peer.AddrInfo
	for _, addr := range bootstrapPeers {
		peerInfo, err := peer.AddrInfoFromString(addr)
		if err != nil {
			p.logger.Error("Failed to parse bootstrap peer address", "address", addr, "error", err)
			continue
		}
		p.peerstore.AddAddrs(peerInfo.ID, peerInfo.Addrs, peerstore.PermanentAddrTTL)
		pis = append(pis, *peerInfo)
	}

	if len(pis) > 0 {
		if err := p.dht.Bootstrap(ctx); err != nil {
			p.logger.Error("Failed to bootstrap DHT with provided peers", "error", err)
		}
	}

	// Connect to all known peers in the peerstore
	for _, peerID := range p.peerstore.Peers() {
		if peerID == p.Host.ID() {
			continue
		}
		addrInfo := p.peerstore.PeerInfo(peerID)
		if len(addrInfo.Addrs) > 0 {
			p.logger.Info("Connecting to known peer", "id", peerID)
			if err := p.Host.Connect(ctx, addrInfo); err != nil {
				p.logger.Error("Failed to connect to known peer", "peer", peerID, "error", err)
			}
		}
	}
}

// StartDiscovery initializes peer discovery mechanisms.
func (p *P2P) StartDiscovery(ctx context.Context) {
	// Bootstrap the DHT
	if err := p.dht.Bootstrap(ctx); err != nil {
		p.logger.Error("Failed to bootstrap DHT", "error", err)
		return
	}

	// Start mDNS discovery
	go p.setupMDNS(ctx)

	// Periodically find peers via DHT
	go p.findPeersPeriodically(ctx)
}

func (p *P2P) findPeersPeriodically(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second) // Find peers every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
							p.logger.Info("Searching for peers via DHT...")
						peers, err := p.routingDiscovery.FindPeers(ctx, "apa-agent") // Use a common tag for discovery
						if err != nil {
							p.logger.Error("Failed to find peers via DHT", "error", err)
							continue
						}
						for peerInfo := range peers {
							if peerInfo.ID == p.Host.ID() {
								continue
							}
							p.logger.Info("Discovered peer via DHT", "id", peerInfo.ID)
							p.peerstore.AddAddrs(peerInfo.ID, peerInfo.Addrs, peerstore.PermanentAddrTTL) // Add to peerstore
							if err := p.Host.Connect(ctx, peerInfo); err != nil {
								p.logger.Error("Failed to connect to DHT peer", "peer", peerInfo.ID, "error", err)
							}
						}		}
	}
}

// JoinHeartbeatTopic joins the heartbeat topic and starts processing incoming messages.
func (p *P2P) JoinHeartbeatTopic(ctx context.Context) error {	topic, err := p.pubsub.Join(HeartbeatTopic)
	if err != nil {
		return fmt.Errorf("failed to join heartbeat topic: %w", err)
	}
	p.heartbeatTopic = topic

	sub, err := topic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to heartbeat topic: %w", err)
	}
	p.heartbeatSub = sub

	go p.heartbeatReadLoop(ctx)
	return nil
}

// JoinModuleTopic joins the module announcement topic.
func (p *P2P) JoinModuleTopic(ctx context.Context) error {
	topic, err := p.pubsub.Join(ModuleTopic)
	if err != nil {
		return fmt.Errorf("failed to join module topic: %w", err)
	}
	p.moduleTopic = topic

	sub, err := topic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to module topic: %w", err)
	}
	p.moduleSub = sub

	go p.moduleReadLoop(ctx)
	return nil
}

// AnnounceModule broadcasts a module's manifest to the network.
func (p *P2P) AnnounceModule(ctx context.Context, manifest module.Manifest) error {
	msg := ModuleAnnouncementMessage{
		AnnouncerPeerID: p.Host.ID(),
		Manifest:        manifest,
	}
	bytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal module announcement: %w", err)
	}

	p.logger.Info("Announcing module", "name", manifest.Name, "version", manifest.Version)
	return p.moduleTopic.Publish(ctx, bytes)
}

// FetchModule connects to a peer and downloads the requested module.
func (p *P2P) FetchModule(ctx context.Context, peerID peer.ID, name, version string) (*module.Manifest, []byte, error) {
	p.logger.Info("Fetching module from peer", "peer", peerID, "name", name, "version", version)
	stream, err := p.Host.NewStream(ctx, peerID, ModuleFetchProtocol)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open stream to peer %s: %w", peerID, err)
	}
	defer stream.Close()

	// 1. Send request
	req := ModuleFetchRequest{Name: name, Version: version}
	if err := json.NewEncoder(stream).Encode(req); err != nil {
		return nil, nil, fmt.Errorf("failed to send fetch request: %w", err)
	}

	// 2. Read response
	var resp ModuleFetchResponse
	if err := json.NewDecoder(stream).Decode(&resp); err != nil {
		return nil, nil, fmt.Errorf("failed to decode fetch response: %w", err)
	}

	if resp.Error != "" {
		return nil, nil, fmt.Errorf("peer returned an error: %s", resp.Error)
	}

	p.logger.Info("Successfully fetched module", "name", name, "version", version, "size", len(resp.WasmBytes))
	return resp.Manifest, resp.WasmBytes, nil
}

// handleModuleFetchStream handles incoming requests for modules.
func (p *P2P) handleModuleFetchStream(s network.Stream) {
	p.logger.Info("Received new module fetch stream", "from", s.Conn().RemotePeer())
	defer s.Close()

	if p.FetchModuleHandler == nil {
		p.logger.Error("FetchModuleHandler is not set, cannot serve module")
		return
	}

	// 1. Read request
	var req ModuleFetchRequest
	if err := json.NewDecoder(s).Decode(&req); err != nil {
		p.logger.Error("Failed to decode fetch request", "error", err)
		return
	}

	// 2. Use the handler to get the module data
	manifest, wasmBytes, err := p.FetchModuleHandler(req.Name, req.Version)
	response := &ModuleFetchResponse{}
	if err != nil {
		p.logger.Error("Failed to handle fetch request", "module_name", req.Name, "error", err)
		response.Error = err.Error()
	} else {
		response.Manifest = manifest
		response.WasmBytes = wasmBytes
	}

	// 3. Send response
	if err := json.NewEncoder(s).Encode(response); err != nil {
		p.logger.Error("Failed to send fetch response", "error", err)
		return
	}
}

// StartHeartbeat starts a periodic broadcast of heartbeat messages.
func (p *P2P) StartHeartbeat(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			msg := HeartbeatMessage{
				PeerID:    p.Host.ID().String(),
				Timestamp: time.Now(),
			}
			bytes, err := json.Marshal(msg)
			if err != nil {
				p.logger.Error("Failed to marshal heartbeat message", "error", err)
				continue
			}

			if err := p.heartbeatTopic.Publish(ctx, bytes); err != nil {
				p.logger.Error("Failed to publish heartbeat message", "error", err)
			}
		}
	}
}

// heartbeatReadLoop processes messages received on the heartbeat topic.
func (p *P2P) heartbeatReadLoop(ctx context.Context) {
	for {
		msg, err := p.heartbeatSub.Next(ctx)
		if err != nil {
			return // Topic has been closed
		}
		if msg.ReceivedFrom == p.Host.ID() {
			continue
		}
		var hb HeartbeatMessage
		if err := json.Unmarshal(msg.Data, &hb); err != nil {
			p.logger.Error("Failed to unmarshal heartbeat message", "from", msg.ReceivedFrom)
			continue
		}
		p.logger.Info("Received heartbeat", "from", hb.PeerID)
	}
}

// moduleReadLoop processes messages received on the module topic.
func (p *P2P) moduleReadLoop(ctx context.Context) {
	for {
		msg, err := p.moduleSub.Next(ctx)
		if err != nil {
			return // Topic has been closed
		}
		if msg.ReceivedFrom == p.Host.ID() {
			continue
		}
		var announcement ModuleAnnouncementMessage
		if err := json.Unmarshal(msg.Data, &announcement); err != nil {
			p.logger.Error("Failed to unmarshal module announcement", "from", msg.ReceivedFrom)
			continue
		}
		announcement.AnnouncerPeerID = msg.ReceivedFrom

		if p.OnModuleAnnouncement != nil {
			p.OnModuleAnnouncement(announcement)
		}
	}
}


type discoveryNotifee struct {
	host   host.Host
	logger *slog.Logger
}

// HandlePeerFound is called when a new peer is discovered.
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	if pi.ID == n.host.ID() {
		return
	}
	n.logger.Info("Discovered a new peer", "id", pi.ID)
}

// setupMDNS initializes the mDNS discovery service.
func (p *P2P) setupMDNS(ctx context.Context) {
	disc := mdns.NewMdnsService(p.Host, p.serviceTag, &discoveryNotifee{host: p.Host, logger: p.logger})
	if err := disc.Start(); err != nil {
		p.logger.Error("Failed to start mDNS service", "error", err)
	}
	<-ctx.Done()
	disc.Close()
}
