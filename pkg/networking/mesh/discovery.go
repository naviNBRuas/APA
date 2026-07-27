package mesh

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"
)

// DiscoveryMethod finds other mesh nodes on the network.
type DiscoveryMethod interface {
	Name() string
	Start(ctx context.Context, results chan<- DiscoveryResult) error
	Stop() error
}

// DiscoveryResult is emitted when a peer is discovered.
type DiscoveryResult struct {
	Method    string
	NodeID    string
	Addresses []string
	NodeInfo  *NodeInfo
}

// DiscoveryEngine runs multiple discovery methods in parallel.
type DiscoveryEngine struct {
	logger   *slog.Logger
	cfg      MeshConfig
	mu       sync.RWMutex
	methods  []DiscoveryMethod
	results  chan DiscoveryResult
	events   chan MeshEvent
	started  bool
}

// NewDiscoveryEngine creates a discovery engine.
func NewDiscoveryEngine(logger *slog.Logger, cfg MeshConfig) *DiscoveryEngine {
	return &DiscoveryEngine{
		logger:  logger,
		cfg:     cfg,
		results: make(chan DiscoveryResult, 128),
		events:  make(chan MeshEvent, 128),
	}
}

// Start begins all discovery methods.
func (de *DiscoveryEngine) Start(ctx context.Context, self *MeshNode, transport *TransportChain) error {
	de.mu.Lock()
	defer de.mu.Unlock()
	if de.started {
		return nil
	}
	de.started = true

	methods := []DiscoveryMethod{
		&localDiscovery{logger: de.logger, port: de.cfg.ListenPort},
		&gossipDiscovery{logger: de.logger, self: self, transport: transport},
		&dhtDiscovery{logger: de.logger, bootstrap: de.cfg.BootstrapPeers},
	}

	for _, disc := range methods {
		m := disc
		if err := m.Start(ctx, de.results); err != nil {
			de.logger.Warn("Discovery method failed to start", "method", m.Name(), "error", err)
			continue
		}
		de.methods = append(de.methods, m)
		de.logger.Info("Discovery method started", "method", m.Name())
	}

	go de.resultLoop(ctx)
	return nil
}

// Stop stops all discovery methods.
func (de *DiscoveryEngine) Stop() {
	de.mu.Lock()
	defer de.mu.Unlock()
	for _, m := range de.methods {
		_ = m.Stop()
	}
	de.methods = nil
	de.started = false
}

// Events returns a channel of mesh events from discovered peers.
func (de *DiscoveryEngine) Events() <-chan MeshEvent {
	return de.events
}

func (de *DiscoveryEngine) resultLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case result, ok := <-de.results:
			if !ok {
				return
			}
			de.logger.Debug("Peer discovered",
				"method", result.Method,
				"node_id", truncateID(result.NodeID),
				"addrs", result.Addresses,
			)
			if result.NodeInfo != nil {
				de.emit(MeshEvent{
					Type:   result.Method,
					NodeID: result.NodeID,
					Payload: &MeshNode{
						ID:        result.NodeID,
						Name:      result.NodeInfo.Name,
						Version:   result.NodeInfo.Version,
						Addresses: result.Addresses,
						Status:    NodeStatusOnline,
						LastSeen:  time.Now().UTC(),
					},
				})
			}
		}
	}
}

func (de *DiscoveryEngine) emit(ev MeshEvent) {
	select {
	case de.events <- ev:
	default:
	}
}

// ---------------------------------------------------------------------------
// Local network discovery (mDNS-like UDP broadcast)
// ---------------------------------------------------------------------------

type localDiscovery struct {
	logger  *slog.Logger
	port    int
	conn    *net.UDPConn
	started bool
	mu      sync.Mutex
}

const (
	discoveryAddr = "224.0.0.251"
	discoveryPort = 5353
	discoveryMsg  = "APA_MESH_DISCOVER"
)

func (ld *localDiscovery) Name() string { return "local" }

func (ld *localDiscovery) Start(ctx context.Context, results chan<- DiscoveryResult) error {
	ld.mu.Lock()
	defer ld.mu.Unlock()
	if ld.started {
		return nil
	}

	addr := &net.UDPAddr{IP: net.ParseIP(discoveryAddr), Port: discoveryPort}
	conn, err := net.ListenMulticastUDP("udp", nil, addr)
	if err != nil {
		conn, err = net.DialUDP("udp", nil, addr)
		if err != nil {
			return fmt.Errorf("local discovery: %w", err)
		}
	}
	ld.conn = conn
	ld.started = true

	go ld.discoveryLoop(ctx, results)
	return nil
}

func (ld *localDiscovery) discoveryLoop(ctx context.Context, results chan<- DiscoveryResult) {
	buf := make([]byte, 1024)

	ld.announcePresence()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			ld.conn.SetReadDeadline(time.Now().Add(3 * time.Second))
			n, _, err := ld.conn.ReadFromUDP(buf)
			if err != nil {
				continue
			}
			msg := strings.TrimSpace(string(buf[:n]))
			if strings.HasPrefix(msg, discoveryMsg) {
				parts := strings.SplitN(msg, "|", 3)
				if len(parts) >= 2 {
					_ = parts[1] // peer ID
					addrs := []string{}
					if len(parts) >= 3 && parts[2] != "" {
						addrs = append(addrs, parts[2])
					}
					select {
					case results <- DiscoveryResult{
						Method:    "local",
						NodeID:    parts[1],
						Addresses: addrs,
					}:
					default:
					}
				}
			}
		}
	}
}

func (ld *localDiscovery) announcePresence() {
	msg := []byte(fmt.Sprintf("%s|%s|", discoveryMsg, ld.localID()))
	if ld.conn != nil {
		ld.conn.WriteToUDP(msg, &net.UDPAddr{IP: net.ParseIP(discoveryAddr), Port: discoveryPort})
	}
}

func (ld *localDiscovery) localID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (ld *localDiscovery) Stop() error {
	ld.mu.Lock()
	defer ld.mu.Unlock()
	if ld.conn != nil {
		ld.conn.Close()
	}
	ld.started = false
	return nil
}

// ---------------------------------------------------------------------------
// Gossip-based peer exchange
// ---------------------------------------------------------------------------

type gossipDiscovery struct {
	logger    *slog.Logger
	self      *MeshNode
	transport *TransportChain
	known     map[string]time.Time
	mu        sync.RWMutex
}

func (gd *gossipDiscovery) Name() string { return "gossip" }

func (gd *gossipDiscovery) Start(ctx context.Context, results chan<- DiscoveryResult) error {
	gd.known = make(map[string]time.Time)
	selfInfo := &NodeInfo{
		ID:        gd.self.ID,
		PublicKey: gd.self.PublicKey,
		VirtualIP: gd.self.VirtualIP,
		Name:      gd.self.Name,
		Version:   gd.self.Version,
	}

	go gd.gossipLoop(ctx, results, selfInfo)
	return nil
}

func (gd *gossipDiscovery) gossipLoop(ctx context.Context, results chan<- DiscoveryResult, selfInfo *NodeInfo) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			msg := PeerExchangeMessage{
				NodeInfo:   selfInfo,
				KnownPeers: gd.getKnown(),
			}
			data, _ := json.Marshal(msg)
			gd.transport.Broadcast("peer_exchange", data)
		}
	}
}

func (gd *gossipDiscovery) getKnown() []string {
	gd.mu.RLock()
	defer gd.mu.RUnlock()
	ids := make([]string, 0, len(gd.known))
	for id := range gd.known {
		ids = append(ids, id)
	}
	return ids
}

func (gd *gossipDiscovery) Stop() error { return nil }

type PeerExchangeMessage struct {
	NodeInfo   *NodeInfo `json:"node_info"`
	KnownPeers []string  `json:"known_peers"`
}

// ---------------------------------------------------------------------------
// DHT discovery (wrapper for existing DHT)
// ---------------------------------------------------------------------------

type dhtDiscovery struct {
	logger    *slog.Logger
	bootstrap []string
	started   bool
	mu        sync.Mutex
}

func (dd *dhtDiscovery) Name() string { return "dht" }

func (dd *dhtDiscovery) Start(ctx context.Context, results chan<- DiscoveryResult) error {
	dd.mu.Lock()
	defer dd.mu.Unlock()
	if dd.started {
		return nil
	}
	dd.started = true

	go dd.dhtLookupLoop(ctx, results)
	return nil
}

func (dd *dhtDiscovery) dhtLookupLoop(ctx context.Context, results chan<- DiscoveryResult) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	select {
	case <-ctx.Done():
		return
	case <-ticker.C:
		for _, bp := range dd.bootstrap {
			select {
			case results <- DiscoveryResult{
				Method:    "dht",
				NodeID:    bp,
				Addresses: []string{bp},
			}:
			default:
			}
		}
	}
}

func (dd *dhtDiscovery) Stop() error {
	dd.mu.Lock()
	defer dd.mu.Unlock()
	dd.started = false
	return nil
}

var _ = json.Marshal
