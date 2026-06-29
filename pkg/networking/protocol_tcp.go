package networking

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

type TCPProtocol struct {
	logger        *slog.Logger
	config        TCPConfig
	messageChan   chan *NetworkMessage
	healthMetrics *ProtocolHealthMetrics

	mu        sync.RWMutex
	listener  net.Listener
	peers     map[peer.ID]string
	listenAddr string
}

type TCPConfig struct {
	ListenAddr string `json:"listen_addr"`
}

func NewTCPProtocol(logger *slog.Logger) (*TCPProtocol, error) {
	return &TCPProtocol{
		logger:        logger,
		messageChan:   make(chan *NetworkMessage, 100),
		peers:         make(map[peer.ID]string),
		healthMetrics: &ProtocolHealthMetrics{ProtocolType: ProtocolTCP},
	}, nil
}

func (tp *TCPProtocol) Initialize(ctx context.Context) error {
	addr := tp.config.ListenAddr
	if addr == "" {
		addr = ":0"
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("tcp init: %w", err)
	}
	tp.listener = listener
	tp.listenAddr = listener.Addr().String()
	go tp.acceptLoop(ctx)
	tp.logger.Info("TCP protocol initialized", "listen_addr", tp.listenAddr)
	return nil
}

func (tp *TCPProtocol) acceptLoop(ctx context.Context) {
	for {
		conn, err := tp.listener.Accept()
		if err != nil {
			return
		}
		go tp.handleConn(conn)
	}
}

func (tp *TCPProtocol) handleConn(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		var msg NetworkMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}
		tp.healthMetrics.TotalMessagesRecv++
		select {
		case tp.messageChan <- &msg:
		default:
		}
	}
}

func (tp *TCPProtocol) RegisterPeerEndpoint(id peer.ID, addr string) {
	tp.mu.Lock()
	tp.peers[id] = addr
	tp.mu.Unlock()
}

func (tp *TCPProtocol) SendMessage(to peer.ID, message *NetworkMessage) error {
	tp.mu.RLock()
	addr, ok := tp.peers[to]
	tp.mu.RUnlock()
	if !ok {
		return fmt.Errorf("tcp: no peer registered for %s", to)
	}
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("tcp dial: %w", err)
	}
	defer func() { _ = conn.Close() }()
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("tcp marshal: %w", err)
	}
	data = append(data, '\n')
	if _, err := conn.Write(data); err != nil {
		return fmt.Errorf("tcp write: %w", err)
	}
	tp.healthMetrics.TotalMessagesSent++
	return nil
}

func (tp *TCPProtocol) ReceiveMessages() <-chan *NetworkMessage { return tp.messageChan }

func (tp *TCPProtocol) GetConnectionInfo() *ConnectionInfo {
	tp.mu.RLock()
	addr := tp.listenAddr
	tp.mu.RUnlock()
	return &ConnectionInfo{
		LocalAddress: addr,
		Protocol:     ProtocolTCP,
		Status:       ConnectionStatusConnected,
		Established:  time.Now(),
		LastActivity: time.Now(),
	}
}

func (tp *TCPProtocol) GetHealthMetrics() *ProtocolHealthMetrics { return tp.healthMetrics }

func (tp *TCPProtocol) Close() error {
	if tp.listener != nil {
		return tp.listener.Close()
	}
	return nil
}
