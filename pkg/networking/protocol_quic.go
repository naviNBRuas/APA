package networking

import (
	"context"
	"crypto/tls"
	"log/slog"
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/quic-go/quic-go"
)

type QUICProtocol struct {
	logger        *slog.Logger
	listener      quic.Listener
	connections   map[peer.ID]interface{}
	config        QUICConfig
	messageChan   chan *NetworkMessage
	healthMetrics *ProtocolHealthMetrics

	mu        sync.RWMutex
	tlsConfig *tls.Config
}

type QUICConfig struct{}

func NewQUICProtocol(logger *slog.Logger) (*QUICProtocol, error) {
	return &QUICProtocol{
		logger:        logger,
		messageChan:   make(chan *NetworkMessage, 100),
		connections:   make(map[peer.ID]interface{}),
		healthMetrics: &ProtocolHealthMetrics{ProtocolType: ProtocolQUIC},
	}, nil
}

func (qp *QUICProtocol) Initialize(ctx context.Context) error                  { return nil }
func (qp *QUICProtocol) SendMessage(to peer.ID, message *NetworkMessage) error { return nil }
func (qp *QUICProtocol) ReceiveMessages() <-chan *NetworkMessage               { return qp.messageChan }
func (qp *QUICProtocol) GetConnectionInfo() *ConnectionInfo                    { return &ConnectionInfo{} }
func (qp *QUICProtocol) GetHealthMetrics() *ProtocolHealthMetrics              { return qp.healthMetrics }
func (qp *QUICProtocol) Close() error                                          { return nil }
