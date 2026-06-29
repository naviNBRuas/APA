package networking

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

type UDPProtocol struct {
	logger        *slog.Logger
	conn          *net.UDPConn
	config        UDPConfig
	messageChan   chan *NetworkMessage
	healthMetrics *ProtocolHealthMetrics

	mu          sync.RWMutex
	peers       map[peer.ID]*net.UDPAddr
	listenAddr  string
}

type UDPConfig struct {
	ListenAddr string `json:"listen_addr"`
}

type UDPPacket struct {
	ID        string    `json:"id"`
	Data      []byte    `json:"data"`
	Timestamp time.Time `json:"timestamp"`
	Sequence  int       `json:"sequence"`
	Ack       bool      `json:"ack"`
}

func NewUDPProtocol(logger *slog.Logger) (*UDPProtocol, error) {
	return &UDPProtocol{
		logger:        logger,
		messageChan:   make(chan *NetworkMessage, 1000),
		peers:         make(map[peer.ID]*net.UDPAddr),
		healthMetrics: &ProtocolHealthMetrics{ProtocolType: ProtocolUDP},
	}, nil
}

func (up *UDPProtocol) Initialize(ctx context.Context) error {
	addr := up.config.ListenAddr
	if addr == "" {
		addr = ":0"
	}
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("udp resolve: %w", err)
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("udp listen: %w", err)
	}
	up.conn = conn
	up.listenAddr = conn.LocalAddr().String()
	go up.readLoop(ctx)
	up.logger.Info("UDP protocol initialized", "listen_addr", up.listenAddr)
	return nil
}

func (up *UDPProtocol) readLoop(ctx context.Context) {
	buf := make([]byte, 65535)
	for {
		n, _, err := up.conn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		var msg NetworkMessage
		if err := json.Unmarshal(buf[:n], &msg); err != nil {
			continue
		}
		up.healthMetrics.TotalMessagesRecv++
		select {
		case up.messageChan <- &msg:
		default:
		}
	}
}

func (up *UDPProtocol) RegisterPeerEndpoint(id peer.ID, addrStr string) {
	addr, err := net.ResolveUDPAddr("udp", addrStr)
	if err != nil {
		up.logger.Warn("UDP: invalid peer address", "peer", id, "addr", addrStr)
		return
	}
	up.mu.Lock()
	up.peers[id] = addr
	up.mu.Unlock()
}

func (up *UDPProtocol) SendMessage(to peer.ID, message *NetworkMessage) error {
	up.mu.RLock()
	addr, ok := up.peers[to]
	up.mu.RUnlock()
	if !ok {
		return fmt.Errorf("udp: no peer registered for %s", to)
	}
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("udp marshal: %w", err)
	}
	if _, err := up.conn.WriteToUDP(data, addr); err != nil {
		return fmt.Errorf("udp send: %w", err)
	}
	up.healthMetrics.TotalMessagesSent++
	return nil
}

func (up *UDPProtocol) ReceiveMessages() <-chan *NetworkMessage { return up.messageChan }

func (up *UDPProtocol) GetConnectionInfo() *ConnectionInfo {
	up.mu.RLock()
	addr := up.listenAddr
	up.mu.RUnlock()
	return &ConnectionInfo{
		LocalAddress: addr,
		Protocol:     ProtocolUDP,
		Status:       ConnectionConnected,
		Established:  time.Now(),
		LastActivity: time.Now(),
	}
}

func (up *UDPProtocol) GetHealthMetrics() *ProtocolHealthMetrics { return up.healthMetrics }

func (up *UDPProtocol) Close() error {
	if up.conn != nil {
		return up.conn.Close()
	}
	return nil
}
