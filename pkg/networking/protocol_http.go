package networking

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
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
	listenAddr  string
}

type HTTPConfig struct {
	ListenAddr string `json:"listen_addr"`
}

func NewHTTPProtocol(logger *slog.Logger) (*HTTPProtocol, error) {
	return &HTTPProtocol{
		logger:        logger,
		client:        &http.Client{Timeout: 30 * time.Second},
		messageChan:   make(chan *NetworkMessage, 100),
		endpoints:     make(map[peer.ID]string),
		healthMetrics: &ProtocolHealthMetrics{ProtocolType: ProtocolHTTP},
	}, nil
}

func (hp *HTTPProtocol) Initialize(ctx context.Context) error {
	addr := hp.config.ListenAddr
	if addr == "" {
		addr = ":0"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/message", hp.handleIncoming)
	hp.server = &http.Server{Addr: addr, Handler: mux}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("http init: %w", err)
	}
	hp.listenAddr = listener.Addr().String()
	go func() {
		if err := hp.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			hp.logger.Error("HTTP server error", "error", err)
		}
	}()
	hp.logger.Info("HTTP protocol initialized", "listen_addr", hp.listenAddr)
	return nil
}

func (hp *HTTPProtocol) handleIncoming(w http.ResponseWriter, r *http.Request) {
	var msg NetworkMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "invalid message", http.StatusBadRequest)
		return
	}
	select {
	case hp.messageChan <- &msg:
	default:
		hp.logger.Warn("HTTP message channel full, dropping")
	}
	w.WriteHeader(http.StatusOK)
}

func (hp *HTTPProtocol) RegisterPeerEndpoint(id peer.ID, addr string) {
	hp.mu.Lock()
	hp.endpoints[id] = addr
	hp.mu.Unlock()
}

func (hp *HTTPProtocol) SendMessage(to peer.ID, message *NetworkMessage) error {
	hp.mu.RLock()
	addr, ok := hp.endpoints[to]
	hp.mu.RUnlock()
	if !ok {
		return fmt.Errorf("http: no endpoint registered for %s", to)
	}
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("http marshal: %w", err)
	}
	resp, err := hp.client.Post(addr+"/message", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("http send: %w", err)
	}
	_ = resp.Body.Close()
	hp.healthMetrics.TotalMessagesSent++
	return nil
}

func (hp *HTTPProtocol) ReceiveMessages() <-chan *NetworkMessage { return hp.messageChan }

func (hp *HTTPProtocol) GetConnectionInfo() *ConnectionInfo {
	hp.mu.RLock()
	addr := hp.listenAddr
	hp.mu.RUnlock()
	return &ConnectionInfo{
		LocalAddress:  addr,
		Protocol:      ProtocolHTTP,
		Status:        ConnectionStatusConnected,
		Established:   time.Now(),
		LastActivity:  time.Now(),
	}
}

func (hp *HTTPProtocol) GetHealthMetrics() *ProtocolHealthMetrics { return hp.healthMetrics }

func (hp *HTTPProtocol) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if hp.server != nil {
		return hp.server.Shutdown(ctx)
	}
	return nil
}
