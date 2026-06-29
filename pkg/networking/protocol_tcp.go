package networking

import (
	"context"
	"log/slog"
	"net"
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"
)

type TCPProtocol struct {
	logger        *slog.Logger
	listener      net.Listener
	connections   map[peer.ID]net.Conn
	config        TCPConfig
	messageChan   chan *NetworkMessage
	healthMetrics *ProtocolHealthMetrics

	mu          sync.RWMutex
	isListening bool
}

type TCPConfig struct{}

func NewTCPProtocol(logger *slog.Logger) (*TCPProtocol, error) {
	return &TCPProtocol{
		logger:        logger,
		messageChan:   make(chan *NetworkMessage, 100),
		connections:   make(map[peer.ID]net.Conn),
		healthMetrics: &ProtocolHealthMetrics{ProtocolType: ProtocolTCP},
	}, nil
}

func (tp *TCPProtocol) Initialize(ctx context.Context) error                  { return nil }
func (tp *TCPProtocol) SendMessage(to peer.ID, message *NetworkMessage) error { return nil }
func (tp *TCPProtocol) ReceiveMessages() <-chan *NetworkMessage               { return tp.messageChan }
func (tp *TCPProtocol) GetConnectionInfo() *ConnectionInfo                    { return &ConnectionInfo{} }
func (tp *TCPProtocol) GetHealthMetrics() *ProtocolHealthMetrics              { return tp.healthMetrics }
func (tp *TCPProtocol) Close() error                                          { return nil }
