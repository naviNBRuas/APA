package mesh

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"
)

type RelayServer struct {
	logger    *slog.Logger
	port      int
	listener  net.Listener
	clients   map[string]relayClient
	mu        sync.RWMutex
	started   bool
	ctx       context.Context
	cancel    context.CancelFunc
}

type relayClient struct {
	ID       string
	Conn     net.Conn
	Joined   time.Time
	LastSeen time.Time
	Recv     chan RelayMessage
}

type RelayMessage struct {
	Type    string `json:"type"`
	From    string `json:"from"`
	To      string `json:"to"`
	Payload []byte `json:"payload,omitempty"`
}

func NewRelayServer(logger *slog.Logger, port int) *RelayServer {
	return &RelayServer{
		logger:  logger,
		port:    port,
		clients: make(map[string]relayClient),
	}
}

func (rs *RelayServer) Start() error {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if rs.started {
		return nil
	}
	addr := fmt.Sprintf(":%d", rs.port)
	if rs.port == 0 {
		addr = ":0"
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("relay listen: %w", err)
	}
	rs.listener = listener
	rs.ctx, rs.cancel = context.WithCancel(context.Background())
	rs.started = true
	rs.port = listener.Addr().(*net.TCPAddr).Port
	rs.logger.Info("Relay server listening", "port", rs.port)
	go rs.acceptLoop()
	return nil
}

func (rs *RelayServer) Stop() {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if !rs.started {
		return
	}
	rs.cancel()
	if rs.listener != nil {
		rs.listener.Close()
	}
	for id, c := range rs.clients {
		c.Conn.Close()
		delete(rs.clients, id)
	}
	rs.started = false
	rs.logger.Info("Relay server stopped")
}

func (rs *RelayServer) Addr() string {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	if rs.listener != nil {
		return rs.listener.Addr().String()
	}
	return ""
}

func (rs *RelayServer) ClientCount() int {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return len(rs.clients)
}

func (rs *RelayServer) acceptLoop() {
	for {
		conn, err := rs.listener.Accept()
		if err != nil {
			select {
			case <-rs.ctx.Done():
				return
			default:
				rs.logger.Error("Relay accept error", "error", err)
				return
			}
		}
		go rs.handleClient(conn)
	}
}

func (rs *RelayServer) handleClient(conn net.Conn) {
	defer conn.Close()

	clientID := fmt.Sprintf("relay-%d", time.Now().UnixNano())

	client := relayClient{
		ID:       clientID,
		Conn:     conn,
		Joined:   time.Now().UTC(),
		LastSeen: time.Now().UTC(),
		Recv:     make(chan RelayMessage, 64),
	}
	rs.mu.Lock()
	rs.clients[clientID] = client
	rs.mu.Unlock()

	rs.logger.Debug("Relay client connected", "client_id", truncateID(clientID), "addr", conn.RemoteAddr())

	defer func() {
		rs.mu.Lock()
		delete(rs.clients, clientID)
		rs.mu.Unlock()
		rs.logger.Debug("Relay client disconnected", "client_id", truncateID(clientID))
	}()

	go rs.clientSendLoop(clientID, conn, client.Recv)

	dec := json.NewDecoder(conn)
	enc := json.NewEncoder(conn)
	for {
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		var msg RelayMessage
		if err := dec.Decode(&msg); err != nil {
			return
		}
		msg.From = clientID

		rs.mu.RLock()
		_, exists := rs.clients[clientID]
		rs.mu.RUnlock()
		if !exists {
			return
		}

		rs.mu.Lock()
		if c, ok := rs.clients[clientID]; ok {
			c.LastSeen = time.Now().UTC()
			rs.clients[clientID] = c
		}
		rs.mu.Unlock()

		switch msg.Type {
		case "relay_register":
			if msg.Payload != nil {
				var reg struct {
					ID string `json:"id"`
				}
				if json.Unmarshal(msg.Payload, &reg) == nil && reg.ID != "" {
					rs.mu.Lock()
					delete(rs.clients, clientID)
					rs.clients[reg.ID] = relayClient{
						ID:       reg.ID,
						Conn:     conn,
						Joined:   time.Now().UTC(),
						LastSeen: time.Now().UTC(),
						Recv:     client.Recv,
					}
					rs.mu.Unlock()
					clientID = reg.ID
					rs.logger.Info("Relay client registered", "node_id", truncateID(clientID))
				}
			}

		case "relay_forward":
			rs.mu.RLock()
			target, ok := rs.clients[msg.To]
			rs.mu.RUnlock()
			if ok && target.Recv != nil {
				select {
				case target.Recv <- RelayMessage{
					Type:    "relay_deliver",
					From:    msg.From,
					Payload: msg.Payload,
				}:
				default:
				}
			}

		case "relay_broadcast":
			rs.mu.RLock()
			for id, c := range rs.clients {
				if id == clientID || c.Recv == nil {
					continue
				}
				select {
				case c.Recv <- RelayMessage{
					Type:    "relay_deliver",
					From:    msg.From,
					Payload: msg.Payload,
				}:
				default:
				}
			}
			rs.mu.RUnlock()

		case "relay_list":
			rs.mu.RLock()
			ids := make([]string, 0, len(rs.clients))
			for id := range rs.clients {
				ids = append(ids, id)
			}
			rs.mu.RUnlock()
			enc.Encode(RelayMessage{
				Type:    "relay_list_response",
				Payload: mustMarshal(ids),
			})

		case "relay_ping":
			enc.Encode(RelayMessage{Type: "relay_pong"})
		}
	}
}

func (rs *RelayServer) clientSendLoop(id string, conn net.Conn, recv <-chan RelayMessage) {
	enc := json.NewEncoder(conn)
	for {
		select {
		case <-rs.ctx.Done():
			return
		case msg, ok := <-recv:
			if !ok {
				return
			}
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			enc.Encode(msg)
		}
	}
}

func (tc *TransportChain) SetRelayAddresses(addrs []string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	for _, m := range tc.methods {
		if r, ok := m.(*relayTransport); ok {
			r.relays = addrs
		}
	}
}

var _ json.Marshaler
