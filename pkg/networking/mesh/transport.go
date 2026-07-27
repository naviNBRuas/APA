package mesh

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type AuthCallback func(addr string, info *NodeInfo) (*MeshNode, error)

type TransportMethod interface {
	Name() TransportType
	Priority() int
	Listen(ctx context.Context) error
	Connect(ctx context.Context, addr string) error
	Send(peerID string, data []byte) error
	Broadcast(data []byte) error
	Receive() <-chan IncomingMessage
	Close() error
	IsAvailable() bool
}

type IncomingMessage struct {
	From      string
	Data      []byte
	Type      string
	Transport TransportType
}

type TransportChain struct {
	logger       *slog.Logger
	cfg          MeshConfig
	mu           sync.RWMutex
	methods      []TransportMethod
	authCallback AuthCallback
	msgCh        chan IncomingMessage
	peerRoutes   map[string]TransportType
	peerConns    map[string]bool
}

func NewTransportChain(logger *slog.Logger, cfg MeshConfig) *TransportChain {
	tc := &TransportChain{
		logger:     logger,
		cfg:        cfg,
		msgCh:      make(chan IncomingMessage, 512),
		peerRoutes: make(map[string]TransportType),
		peerConns:  make(map[string]bool),
	}
	tc.buildChain()
	return tc
}

func (tc *TransportChain) buildChain() {
	msgCh := tc.msgCh
	idx := 0

	methods := []TransportMethod{}

	methods = append(methods, &webSocketTransport{
		logger: tc.logger,
		port:   tc.cfg.ListenPort,
		msgCh:  msgCh,
		wspri:  idx,
	})
	idx++

	methods = append(methods, &httpTransport{
		logger: tc.logger,
		port:   tc.cfg.ListenPort,
		msgCh:  msgCh,
		httpri: idx,
	})
	idx++

	methods = append(methods, &tcpTransport{
		logger: tc.logger,
		port:   tc.cfg.ListenPort,
		msgCh:  msgCh,
		tcppri: idx,
	})
	idx++

	if tc.cfg.EnableLAN {
		methods = append(methods, &lanTransport{
			logger: tc.logger,
			msgCh:  msgCh,
			lanpri: idx,
		})
		idx++
	}

	if tc.cfg.EnableRelay {
		methods = append(methods, &relayTransport{
			logger: tc.logger,
			msgCh:  msgCh,
			relpri: idx,
			relays: tc.cfg.RelayAddresses,
		})
		idx++
	}

	tc.methods = methods
}

func (tc *TransportChain) SetNodeID(nodeID string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	for _, m := range tc.methods {
		if r, ok := m.(*relayTransport); ok {
			r.nodeID = nodeID
		}
	}
}

func (tc *TransportChain) SetAuthCallback(cb AuthCallback) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.authCallback = cb
}

func (tc *TransportChain) Start(ctx context.Context) error {
	for _, m := range tc.methods {
		if err := m.Listen(ctx); err != nil {
			tc.logger.Warn("Transport listen failed", "transport", m.Name(), "error", err)
			continue
		}
		tc.logger.Info("Transport listening", "transport", m.Name())
	}
	return nil
}

func (tc *TransportChain) Stop() {
	for _, m := range tc.methods {
		_ = m.Close()
	}
}

func (tc *TransportChain) Available() []string {
	var avail []string
	for _, m := range tc.methods {
		if m.IsAvailable() {
			avail = append(avail, string(m.Name()))
		}
	}
	return avail
}

func (tc *TransportChain) Connect(ctx context.Context, peerID string, addr string) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	for _, m := range tc.methods {
		if !m.IsAvailable() {
			continue
		}
		tc.logger.Debug("Attempting connect", "peer", truncateID(peerID), "addr", addr, "transport", m.Name())
		if err := m.Connect(ctx, addr); err == nil {
			tc.peerRoutes[peerID] = m.Name()
			tc.peerConns[peerID] = true
			tc.logger.Info("Connected to peer", "peer_id", truncateID(peerID), "transport", m.Name())
			return nil
		}
	}
	return fmt.Errorf("all transports failed for peer %s", truncateID(peerID))
}

func (tc *TransportChain) SendToPeer(peerID string, msgType string, payload []byte) error {
	env := Envelope{
		Type:    msgType,
		To:      peerID,
		Payload: payload,
		SentAt:  time.Now().UTC().UnixNano(),
	}
	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}

	tc.mu.RLock()
	transportType, hasRoute := tc.peerRoutes[peerID]
	tc.mu.RUnlock()

	if hasRoute {
		for _, m := range tc.methods {
			if m.Name() == transportType {
				return m.Send(peerID, data)
			}
		}
	}

	for _, m := range tc.methods {
		if m.IsAvailable() {
			if err := m.Send(peerID, data); err == nil {
				return nil
			}
		}
	}
	return fmt.Errorf("no transport available to send to %s", truncateID(peerID))
}

func (tc *TransportChain) Broadcast(msgType string, payload []byte) error {
	env := Envelope{
		Type:    msgType,
		Payload: payload,
		SentAt:  time.Now().UTC().UnixNano(),
	}
	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}
	var lastErr error
	for _, m := range tc.methods {
		if m.IsAvailable() {
			if err := m.Broadcast(data); err != nil {
				lastErr = err
			}
		}
	}
	return lastErr
}

func (tc *TransportChain) Receive() <-chan IncomingMessage {
	return tc.msgCh
}

type Envelope struct {
	Type    string `json:"type"`
	From    string `json:"from"`
	To      string `json:"to"`
	Payload []byte `json:"payload"`
	SentAt  int64  `json:"sent_at"`
	TTL     int    `json:"ttl,omitempty"`
	Hops    int    `json:"hops,omitempty"`
}

// ---------------------------------------------------------------------------
// WebSocket transport
// ---------------------------------------------------------------------------

type webSocketTransport struct {
	logger   *slog.Logger
	port     int
	msgCh    chan IncomingMessage
	wspri    int
	listener net.Listener
	server   *http.Server
	peers    map[string]net.Conn
	peersMu  sync.RWMutex
	started  bool
	mu       sync.Mutex
}

func (w *webSocketTransport) Name() TransportType     { return TransportWebSocket }
func (w *webSocketTransport) Priority() int            { return w.wspri }
func (w *webSocketTransport) IsAvailable() bool         { return w.started }
func (w *webSocketTransport) Receive() <-chan IncomingMessage { return w.msgCh }

func (w *webSocketTransport) Listen(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	addr := fmt.Sprintf(":%d", w.port)
	if w.port == 0 {
		addr = ":0"
	}

	w.peers = make(map[string]net.Conn)
	mux := http.NewServeMux()
	mux.HandleFunc("/mesh/ws", func(rw http.ResponseWriter, r *http.Request) {
		upgrade := strings.ToLower(r.Header.Get("Upgrade"))
		if upgrade != "websocket" {
			http.Error(rw, "upgrade required", http.StatusUpgradeRequired)
			return
		}
		hj, ok := rw.(http.Hijacker)
		if !ok {
			http.Error(rw, "hijack not supported", http.StatusInternalServerError)
			return
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			return
		}
		peerID := fmt.Sprintf("ws-%d", rand.Int63())
		w.peersMu.Lock()
		w.peers[peerID] = conn
		w.peersMu.Unlock()
		go w.readConn(peerID, conn)
	})

	server := &http.Server{Handler: mux, ReadTimeout: 30 * time.Second}
	w.server = server

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("websocket listen: %w", err)
	}
	w.listener = listener

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			w.logger.Error("WebSocket server error", "error", err)
		}
	}()

	w.port = listener.Addr().(*net.TCPAddr).Port
	w.started = true
	w.logger.Info("WebSocket transport listening", "port", w.port)
	return nil
}

func (w *webSocketTransport) readConn(peerID string, conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 65536)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			w.peersMu.Lock()
			delete(w.peers, peerID)
			w.peersMu.Unlock()
			return
		}
		msg := IncomingMessage{
			From:      peerID,
			Data:      append([]byte{}, buf[:n]...),
			Transport: TransportWebSocket,
		}
		select {
		case w.msgCh <- msg:
		default:
		}
	}
}

func (w *webSocketTransport) Connect(ctx context.Context, addr string) error {
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("ws connect: %w", err)
	}
	req := fmt.Sprintf("GET /mesh/ws HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\nSec-WebSocket-Version: 13\r\n\r\n", addr)
	if _, err := conn.Write([]byte(req)); err != nil {
		conn.Close()
		return fmt.Errorf("ws connect handshake: %w", err)
	}
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		conn.Close()
		return fmt.Errorf("ws connect response: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSwitchingProtocols {
		conn.Close()
		return fmt.Errorf("ws connect: unexpected status %d", resp.StatusCode)
	}
	peerID := fmt.Sprintf("ws-out-%d", rand.Int63())
	w.peersMu.Lock()
	w.peers[peerID] = conn
	w.peersMu.Unlock()
	go w.readConn(peerID, conn)
	return nil
}

func (w *webSocketTransport) Send(peerID string, data []byte) error {
	w.peersMu.RLock()
	conn, ok := w.peers[peerID]
	w.peersMu.RUnlock()
	if !ok {
		return fmt.Errorf("peer %s not connected via websocket", truncateID(peerID))
	}
	_, err := conn.Write(data)
	return err
}

func (w *webSocketTransport) Broadcast(data []byte) error {
	w.peersMu.RLock()
	defer w.peersMu.RUnlock()
	for _, conn := range w.peers {
		conn.Write(data)
	}
	return nil
}

func (w *webSocketTransport) Close() error {
	if w.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		w.server.Shutdown(ctx)
	}
	if w.listener != nil {
		w.listener.Close()
	}
	return nil
}

// ---------------------------------------------------------------------------
// HTTP transport
// ---------------------------------------------------------------------------

type httpTransport struct {
	logger   *slog.Logger
	port     int
	msgCh    chan IncomingMessage
	httpri   int
	listener net.Listener
	server   *http.Server
	client   *http.Client
	peers    map[string]time.Time
	peersMu  sync.RWMutex
	started  bool
	mu       sync.Mutex
}

func (h *httpTransport) Name() TransportType     { return TransportHTTPS }
func (h *httpTransport) Priority() int            { return h.httpri }
func (h *httpTransport) IsAvailable() bool         { return h.started }
func (h *httpTransport) Receive() <-chan IncomingMessage { return h.msgCh }

func (h *httpTransport) Listen(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	addr := fmt.Sprintf(":%d", h.port)
	if h.port == 0 {
		addr = ":0"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/mesh/peers", func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.peersMu.RLock()
		peers := make([]string, 0, len(h.peers))
		for p := range h.peers {
			peers = append(peers, p)
		}
		h.peersMu.RUnlock()
		json.NewEncoder(rw).Encode(peers)
	})
	mux.HandleFunc("/mesh/msg", func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		buf := make([]byte, r.ContentLength)
		r.Body.Read(buf)

		host := r.Host
		if host == "" {
			host = r.RemoteAddr
		}
		h.peersMu.Lock()
		h.peers[host] = time.Now().UTC()
		h.peersMu.Unlock()

		msg := IncomingMessage{
			From:      r.RemoteAddr,
			Data:      buf,
			Transport: TransportHTTPS,
		}
		select {
		case h.msgCh <- msg:
		default:
		}
		rw.WriteHeader(http.StatusOK)
	})

	server := &http.Server{Handler: mux, ReadTimeout: 30 * time.Second}
	h.server = server
	h.peers = make(map[string]time.Time)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("http listen: %w", err)
	}
	h.listener = listener

	go server.Serve(listener)
	h.port = listener.Addr().(*net.TCPAddr).Port
	h.client = &http.Client{Timeout: 10 * time.Second}
	h.started = true
	h.logger.Info("HTTP transport listening", "port", h.port)
	return nil
}

func (h *httpTransport) Connect(_ context.Context, _ string) error { return nil }

func (h *httpTransport) Send(peerID string, data []byte) error {
	req, err := http.NewRequest(http.MethodPost,
		fmt.Sprintf("http://%s/mesh/msg", peerID),
		bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (h *httpTransport) Broadcast(data []byte) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.client == nil {
		return nil
	}
	knownPeers := []string{}
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/mesh/peers", h.port))
	if err == nil {
		json.NewDecoder(resp.Body).Decode(&knownPeers)
		resp.Body.Close()
	}
	for _, peer := range knownPeers {
		req, err := http.NewRequest(http.MethodPost,
			fmt.Sprintf("http://%s/mesh/msg", peer),
			bytes.NewReader(data))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/octet-stream")
		h.client.Do(req)
	}
	return nil
}

func (h *httpTransport) Close() error {
	if h.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		h.server.Shutdown(ctx)
	}
	if h.listener != nil {
		h.listener.Close()
	}
	return nil
}

// ---------------------------------------------------------------------------
// TCP transport
// ---------------------------------------------------------------------------

type tcpTransport struct {
	logger   *slog.Logger
	port     int
	msgCh    chan IncomingMessage
	tcppri   int
	listener net.Listener
	peers    map[string]net.Conn
	peersMu  sync.RWMutex
	started  bool
	mu       sync.Mutex
}

func (t *tcpTransport) Name() TransportType     { return TransportTCP }
func (t *tcpTransport) Priority() int            { return t.tcppri }
func (t *tcpTransport) IsAvailable() bool         { return t.started }
func (t *tcpTransport) Receive() <-chan IncomingMessage { return t.msgCh }

func (t *tcpTransport) Listen(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	addr := fmt.Sprintf(":%d", t.port)
	if t.port == 0 {
		addr = ":0"
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("tcp listen: %w", err)
	}
	t.listener = listener
	t.peers = make(map[string]net.Conn)

	go t.acceptLoop()
	t.port = listener.Addr().(*net.TCPAddr).Port
	t.started = true
	t.logger.Info("TCP transport listening", "port", t.port)
	return nil
}

func (t *tcpTransport) acceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			return
		}
		peerID := fmt.Sprintf("tcp-%d", rand.Int63())
		t.peersMu.Lock()
		t.peers[peerID] = conn
		t.peersMu.Unlock()
		go t.readConn(peerID, conn)
	}
}

func (t *tcpTransport) readConn(peerID string, conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 65536)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			t.peersMu.Lock()
			delete(t.peers, peerID)
			t.peersMu.Unlock()
			return
		}
		msg := IncomingMessage{
			From:      peerID,
			Data:      append([]byte{}, buf[:n]...),
			Transport: TransportTCP,
		}
		select {
		case t.msgCh <- msg:
		default:
		}
	}
}

func (t *tcpTransport) Connect(ctx context.Context, addr string) error {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("tcp connect: %w", err)
	}
	peerID := fmt.Sprintf("tcp-out-%d", rand.Int63())
	t.peersMu.Lock()
	t.peers[peerID] = conn
	t.peersMu.Unlock()
	go t.readConn(peerID, conn)
	return nil
}

func (t *tcpTransport) Send(peerID string, data []byte) error {
	t.peersMu.RLock()
	conn, ok := t.peers[peerID]
	t.peersMu.RUnlock()
	if !ok {
		return fmt.Errorf("peer %s not connected via tcp", truncateID(peerID))
	}
	_, err := conn.Write(data)
	return err
}

func (t *tcpTransport) Broadcast(data []byte) error {
	t.peersMu.RLock()
	defer t.peersMu.RUnlock()
	for _, conn := range t.peers {
		conn.Write(data)
	}
	return nil
}

func (t *tcpTransport) Close() error {
	if t.listener != nil {
		t.listener.Close()
	}
	return nil
}

// ---------------------------------------------------------------------------
// LAN transport
// ---------------------------------------------------------------------------

type lanTransport struct {
	logger   *slog.Logger
	port     int
	msgCh    chan IncomingMessage
	lanpri   int
	listener net.Listener
	peers    map[string]net.Conn
	peersMu  sync.RWMutex
	started  bool
	mu       sync.Mutex
}

func (l *lanTransport) Name() TransportType     { return TransportLAN }
func (l *lanTransport) Priority() int            { return l.lanpri }
func (l *lanTransport) IsAvailable() bool         { return l.started }
func (l *lanTransport) Receive() <-chan IncomingMessage { return l.msgCh }

func (l *lanTransport) Listen(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return fmt.Errorf("lan listen: %w", err)
	}
	l.listener = listener
	l.peers = make(map[string]net.Conn)

	go l.acceptLoop()
	l.port = listener.Addr().(*net.TCPAddr).Port
	l.started = true
	l.logger.Info("LAN transport listening", "port", l.port)
	return nil
}

func (l *lanTransport) acceptLoop() {
	for {
		conn, err := l.listener.Accept()
		if err != nil {
			return
		}
		peerID := fmt.Sprintf("lan-%d", rand.Int63())
		l.peersMu.Lock()
		l.peers[peerID] = conn
		l.peersMu.Unlock()
		go l.readConn(peerID, conn)
	}
}

func (l *lanTransport) readConn(peerID string, conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 65536)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			l.peersMu.Lock()
			delete(l.peers, peerID)
			l.peersMu.Unlock()
			return
		}
		msg := IncomingMessage{
			From:      peerID,
			Data:      append([]byte{}, buf[:n]...),
			Transport: TransportLAN,
		}
		select {
		case l.msgCh <- msg:
		default:
		}
	}
}

func (l *lanTransport) Connect(ctx context.Context, addr string) error {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	peerID := fmt.Sprintf("lan-out-%d", rand.Int63())
	l.peersMu.Lock()
	l.peers[peerID] = conn
	l.peersMu.Unlock()
	go l.readConn(peerID, conn)
	return nil
}

func (l *lanTransport) Send(peerID string, data []byte) error {
	l.peersMu.RLock()
	conn, ok := l.peers[peerID]
	l.peersMu.RUnlock()
	if !ok {
		return fmt.Errorf("peer %s not connected via lan", truncateID(peerID))
	}
	_, err := conn.Write(data)
	return err
}

func (l *lanTransport) Broadcast(data []byte) error {
	l.peersMu.RLock()
	defer l.peersMu.RUnlock()
	for _, conn := range l.peers {
		conn.Write(data)
	}
	return nil
}

func (l *lanTransport) Close() error {
	if l.listener != nil {
		l.listener.Close()
	}
	return nil
}

// ---------------------------------------------------------------------------
// Relay transport
// ---------------------------------------------------------------------------

type relayTransport struct {
	logger   *slog.Logger
	msgCh    chan IncomingMessage
	relpri   int
	relays   []string
	conn     net.Conn
	enc      *json.Encoder
	dec      *json.Decoder
	nodeID   string
	peers    map[string]bool
	peersMu  sync.RWMutex
	started  bool
	mu       sync.Mutex
	ctx      context.Context
	cancel   context.CancelFunc
}

func (r *relayTransport) Name() TransportType       { return TransportRelay }
func (r *relayTransport) Priority() int              { return r.relpri }
func (r *relayTransport) IsAvailable() bool           { return r.started }
func (r *relayTransport) Receive() <-chan IncomingMessage { return r.msgCh }

func (r *relayTransport) Listen(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.peers = make(map[string]bool)
	r.ctx, r.cancel = context.WithCancel(ctx)
	r.started = true
	r.logger.Info("Relay transport ready")
	return nil
}

func (r *relayTransport) Connect(ctx context.Context, addr string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.conn != nil {
		return nil
	}

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("relay connect: %w", err)
	}
	r.conn = conn
	r.enc = json.NewEncoder(conn)
	r.dec = json.NewDecoder(conn)

	regMsg := RelayMessage{
		Type:    "relay_register",
		Payload: mustMarshal(map[string]string{"id": r.nodeID}),
	}
	if err := r.enc.Encode(regMsg); err != nil {
		conn.Close()
		r.conn = nil
		return fmt.Errorf("relay register: %w", err)
	}

	r.logger.Info("Connected to relay", "addr", addr)
	go r.relayReadLoop()
	return nil
}

func (r *relayTransport) relayReadLoop() {
	for {
		var msg RelayMessage
		if err := r.dec.Decode(&msg); err != nil {
			r.mu.Lock()
			r.started = false
			if r.conn != nil {
				r.conn.Close()
				r.conn = nil
			}
			r.mu.Unlock()
			return
		}
		switch msg.Type {
		case "relay_deliver", "relay_forward":
			incoming := IncomingMessage{
				From:      msg.From,
				Data:      msg.Payload,
				Transport: TransportRelay,
			}
			select {
			case r.msgCh <- incoming:
			default:
			}
		case "relay_list_response":
			var ids []string
			if json.Unmarshal(msg.Payload, &ids) == nil {
				r.peersMu.Lock()
				for _, id := range ids {
					r.peers[id] = true
				}
				r.peersMu.Unlock()
			}
		case "relay_pong":
		}
	}
}

func (r *relayTransport) Send(peerID string, data []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.enc == nil {
		return fmt.Errorf("relay not connected")
	}
	return r.enc.Encode(RelayMessage{
		Type:    "relay_forward",
		To:      peerID,
		Payload: data,
	})
}

func (r *relayTransport) Broadcast(data []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.enc == nil {
		return fmt.Errorf("relay not connected")
	}
	return r.enc.Encode(RelayMessage{
		Type:    "relay_broadcast",
		Payload: data,
	})
}

func (r *relayTransport) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cancel != nil {
		r.cancel()
	}
	if r.conn != nil {
		r.conn.Close()
		r.conn = nil
	}
	r.started = false
	return nil
}
