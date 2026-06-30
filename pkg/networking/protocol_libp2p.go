package networking

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

type LibP2PProtocol struct {
	logger        *slog.Logger
	host          host.Host
	config        LibP2PConfig
	messageChan   chan *NetworkMessage
	healthMetrics *ProtocolHealthMetrics

	mu        sync.RWMutex
	peers     map[peer.ID]*PeerConnection
	listenAddr string
}

type LibP2PConfig struct {
	ListenAddrs  []string `json:"listen_addrs"`
	ProtocolID   string   `json:"protocol_id"`
}

const defaultLibP2PProtocolID = "/apa/msg/1.0.0"

func NewLibP2PProtocol(logger *slog.Logger, config LibP2PConfig) (*LibP2PProtocol, error) {
	pid := config.ProtocolID
	if pid == "" {
		pid = defaultLibP2PProtocolID
	}
	return &LibP2PProtocol{
		logger:        logger,
		config:        LibP2PConfig{ProtocolID: pid},
		messageChan:   make(chan *NetworkMessage, 100),
		healthMetrics: &ProtocolHealthMetrics{ProtocolType: ProtocolLibP2P},
		peers:         make(map[peer.ID]*PeerConnection),
	}, nil
}

func (lp *LibP2PProtocol) Initialize(ctx context.Context) error {
	var opts []libp2p.Option
	if len(lp.config.ListenAddrs) > 0 {
		opts = append(opts, libp2p.ListenAddrStrings(lp.config.ListenAddrs...))
	} else {
		opts = append(opts, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	}

	h, err := libp2p.New(opts...)
	if err != nil {
		return fmt.Errorf("libp2p host: %w", err)
	}
	lp.host = h

	for _, a := range h.Addrs() {
		if lp.listenAddr != "" {
			lp.listenAddr += ","
		}
		lp.listenAddr = a.String() + "/p2p/" + h.ID().String()
	}

	h.SetStreamHandler(protocol.ID(lp.config.ProtocolID), lp.handleStream)
	lp.logger.Info("libp2p protocol initialized", "peer_id", h.ID(), "addrs", h.Addrs())

	go func() {
		<-ctx.Done()
		_ = h.Close()
	}()

	return nil
}

func (lp *LibP2PProtocol) handleStream(s network.Stream) {
	defer func() { _ = s.Close() }()
	peerID := s.Conn().RemotePeer()
	lp.mu.Lock()
	lp.peers[peerID] = &PeerConnection{
		PeerID:    peerID,
		Connected: time.Now(),
		LastSeen:  time.Now(),
	}
	lp.mu.Unlock()

	scanner := bufio.NewScanner(s)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		var msg NetworkMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}
		lp.healthMetrics.TotalMessagesRecv++
		select {
		case lp.messageChan <- &msg:
		default:
		}
	}
}

func (lp *LibP2PProtocol) RegisterPeerEndpoint(id peer.ID, addr string) {
	_ = addr
	lp.mu.RLock()
	_, exists := lp.peers[id]
	lp.mu.RUnlock()
	if !exists {
		lp.mu.Lock()
		lp.peers[id] = &PeerConnection{
			PeerID:    id,
			Connected: time.Now(),
			LastSeen:  time.Now(),
		}
		lp.mu.Unlock()
	}
}

func (lp *LibP2PProtocol) SendMessage(to peer.ID, message *NetworkMessage) error {
	s, err := lp.host.NewStream(context.Background(), to, protocol.ID(lp.config.ProtocolID))
	if err != nil {
		return fmt.Errorf("libp2p new stream: %w", err)
	}
	defer func() { _ = s.Close() }()

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("libp2p marshal: %w", err)
	}
	if _, err := s.Write(data); err != nil {
		return fmt.Errorf("libp2p write: %w", err)
	}
	lp.healthMetrics.TotalMessagesSent++
	return nil
}

func (lp *LibP2PProtocol) ReceiveMessages() <-chan *NetworkMessage {
	return lp.messageChan
}

func (lp *LibP2PProtocol) GetConnectionInfo() *ConnectionInfo {
	status := ConnectionConnected
	if lp.host == nil {
		status = ConnectionDisconnected
	}
	return &ConnectionInfo{
		LocalAddress: lp.listenAddr,
		Protocol:     ProtocolLibP2P,
		Status:       status,
		Established:  time.Now(),
		LastActivity: time.Now(),
	}
}

func (lp *LibP2PProtocol) GetHealthMetrics() *ProtocolHealthMetrics {
	return lp.healthMetrics
}

func (lp *LibP2PProtocol) Close() error {
	if lp.host != nil {
		return lp.host.Close()
	}
	return nil
}
