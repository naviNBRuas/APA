package networking

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/libp2p/go-libp2p/core/peer"
)

type WebSocketProtocol struct {
	logger        *slog.Logger
	upgrader      websocket.Upgrader
	server        *http.Server
	connections   map[peer.ID]*websocket.Conn
	config        WebSocketConfig
	messageChan   chan *NetworkMessage
	healthMetrics *ProtocolHealthMetrics

	mu         sync.RWMutex
	listenAddr string
	peers      map[peer.ID]string
}

type WebSocketConfig struct {
	ListenAddr string `json:"listen_addr"`
}

func NewWebSocketProtocol(logger *slog.Logger, config WebSocketConfig) (*WebSocketProtocol, error) {
	return &WebSocketProtocol{
		logger: logger,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		config:        config,
		connections:   make(map[peer.ID]*websocket.Conn),
		messageChan:   make(chan *NetworkMessage, 100),
		healthMetrics: &ProtocolHealthMetrics{ProtocolType: ProtocolWebSocket},
		peers:         make(map[peer.ID]string),
	}, nil
}

func (wp *WebSocketProtocol) Initialize(ctx context.Context) error {
	addr := wp.config.ListenAddr
	if addr == "" {
		addr = ":0"
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("websocket listen: %w", err)
	}
	wp.listenAddr = listener.Addr().String()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", wp.handleWebSocket)

	wp.server = &http.Server{
		Handler: mux,
	}

	go func() {
		if err := wp.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			wp.logger.Error("WebSocket server error", "error", err)
		}
	}()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = wp.server.Shutdown(shutdownCtx)
	}()

	wp.logger.Info("WebSocket protocol initialized", "listen_addr", wp.listenAddr)
	return nil
}

func (wp *WebSocketProtocol) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := wp.upgrader.Upgrade(w, r, nil)
	if err != nil {
		wp.logger.Warn("WebSocket upgrade failed", "error", err)
		return
	}

	wp.healthMetrics.ConnectionStatus = ConnectionConnected
	defer func() {
		_ = conn.Close()
		wp.healthMetrics.ConnectionStatus = ConnectionDisconnected
	}()

	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var msg NetworkMessage
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			continue
		}
		wp.healthMetrics.TotalMessagesRecv++
		select {
		case wp.messageChan <- &msg:
		default:
		}
	}
}

func (wp *WebSocketProtocol) RegisterPeerEndpoint(id peer.ID, addr string) {
	wp.mu.Lock()
	wp.peers[id] = addr
	wp.mu.Unlock()
}

func (wp *WebSocketProtocol) SendMessage(to peer.ID, message *NetworkMessage) error {
	wp.mu.RLock()
	addr, ok := wp.peers[to]
	wp.mu.RUnlock()
	if !ok {
		return fmt.Errorf("websocket: unknown peer %s", to)
	}

	conn, _, err := websocket.DefaultDialer.Dial(addr+"/ws", nil)
	if err != nil {
		return fmt.Errorf("websocket dial: %w", err)
	}
	defer func() { _ = conn.Close() }()

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("websocket marshal: %w", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("websocket write: %w", err)
	}
	wp.healthMetrics.TotalMessagesSent++
	return nil
}

func (wp *WebSocketProtocol) ReceiveMessages() <-chan *NetworkMessage {
	return wp.messageChan
}

func (wp *WebSocketProtocol) GetConnectionInfo() *ConnectionInfo {
	return &ConnectionInfo{
		LocalAddress:  wp.listenAddr,
		Protocol:      ProtocolWebSocket,
		Status:        wp.healthMetrics.ConnectionStatus,
		Established:   wp.healthMetrics.CreatedAt,
		LastActivity:  time.Now(),
	}
}

func (wp *WebSocketProtocol) GetHealthMetrics() *ProtocolHealthMetrics {
	return wp.healthMetrics
}

func (wp *WebSocketProtocol) Close() error {
	if wp.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return wp.server.Shutdown(ctx)
	}
	return nil
}
