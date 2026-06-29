package networking

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

type HTTPProtocol struct {
	logger        *slog.Logger
	client        *http.Client
	server        *http.Server
	config        HTTPConfig
	messageChan   chan *NetworkMessage
	healthMetrics *ProtocolHealthMetrics

	mu          sync.RWMutex
	endpoints   map[peer.ID]string
	activeConns map[string]*http.Response
}

type HTTPConfig struct{}

func NewHTTPProtocol(logger *slog.Logger) (*HTTPProtocol, error) {
	return &HTTPProtocol{
		logger:        logger,
		client:        &http.Client{Timeout: 30 * time.Second},
		messageChan:   make(chan *NetworkMessage, 100),
		endpoints:     make(map[peer.ID]string),
		activeConns:   make(map[string]*http.Response),
		healthMetrics: &ProtocolHealthMetrics{ProtocolType: ProtocolHTTP},
	}, nil
}

func (hp *HTTPProtocol) Initialize(ctx context.Context) error                  { return nil }
func (hp *HTTPProtocol) SendMessage(to peer.ID, message *NetworkMessage) error { return nil }
func (hp *HTTPProtocol) ReceiveMessages() <-chan *NetworkMessage               { return hp.messageChan }
func (hp *HTTPProtocol) GetConnectionInfo() *ConnectionInfo                    { return &ConnectionInfo{} }
func (hp *HTTPProtocol) GetHealthMetrics() *ProtocolHealthMetrics              { return hp.healthMetrics }
func (hp *HTTPProtocol) Close() error                                          { return nil }
