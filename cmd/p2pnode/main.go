package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/naviNBRuas/APA/pkg/networking"
)

// allowAllPolicy permits every action; suitable for harness-only binaries.
type allowAllPolicy struct{}

func (allowAllPolicy) Authorize(ctx context.Context, subject, action, resource string) (bool, string, error) {
	return true, "allowed", nil
}

type hostInfo struct {
	PeerID     string   `json:"peer_id"`
	Addrs      []string `json:"addrs"`
	RelayAddrs []string `json:"relay_addrs,omitempty"`
}

type connectRequest struct {
	PeerID    string `json:"peer_id"`
	RelayAddr string `json:"relay_addr"`
	Addr      string `json:"addr"`
}

type publishRequest struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type dhtRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type serverState struct {
	p2p      *networking.P2P
	logger   *slog.Logger
	mu       sync.Mutex
	messages []networking.ControllerMessage
}

func main() {
	var (
		role               = flag.String("role", "node", "role to run: relay or node")
		listenStr          = flag.String("listen-addrs", "/ip4/0.0.0.0/tcp/4001,/ip4/0.0.0.0/udp/4001/quic-v1", "comma separated listen multiaddrs")
		bootstrapStr       = flag.String("bootstrap", "", "comma separated bootstrap peers")
		bootstrapFile      = flag.String("bootstrap-file", "", "path to JSON host info file with addrs[] for bootstrap")
		announceFile       = flag.String("announce-file", "", "path to write this node's host info JSON")
		httpAddr           = flag.String("http", ":8080", "HTTP listen address for control API (blank to disable)")
		heartbeatInterval  = flag.Duration("heartbeat-interval", 2*time.Second, "heartbeat interval")
		serviceTag         = flag.String("service-tag", "apa-harness", "service tag for discovery")
		enableRelayService = flag.Bool("enable-relay-service", false, "expose circuit relay v2 service")
	)
	flag.Parse()

	if strings.EqualFold(*role, "relay") && !*enableRelayService {
		*enableRelayService = true
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	listenAddrs := splitList(*listenStr)
	bootstrapPeers := splitList(*bootstrapStr)
	if *bootstrapFile != "" {
		if fromFile := loadBootstrapFromFile(logger, *bootstrapFile); len(fromFile) > 0 {
			bootstrapPeers = append(bootstrapPeers, fromFile...)
		}
	}

	priv, _, err := crypto.GenerateEd25519Key(nil)
	if err != nil {
		logger.Error("failed to generate key", "error", err)
		os.Exit(1)
	}
	pid, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		logger.Error("failed to derive peer id", "error", err)
		os.Exit(1)
	}

	cfg := networking.Config{
		ListenAddresses:    listenAddrs,
		BootstrapPeers:     bootstrapPeers,
		HeartbeatInterval:  *heartbeatInterval,
		ServiceTag:         *serviceTag,
		EnableRelayService: *enableRelayService,
	}

	p2p, err := networking.NewP2P(ctx, logger, cfg, pid, priv, allowAllPolicy{})
	if err != nil {
		logger.Error("failed to create p2p", "error", err)
		os.Exit(1)
	}

	p2p.StartDiscovery(ctx)

	if err := p2p.JoinHeartbeatTopic(ctx); err != nil {
		logger.Warn("join heartbeat topic", "error", err)
	} else {
		go p2p.StartHeartbeat(ctx, cfg.HeartbeatInterval)
	}

	if err := p2p.JoinControllerCommTopic(ctx); err != nil {
		logger.Warn("join controller topic", "error", err)
	}

	var msgCh <-chan *networking.ControllerMessage
	if ch, err := p2p.SubscribeControllerMessages(ctx); err == nil {
		msgCh = ch
	} else {
		logger.Warn("subscribe controller messages", "error", err)
	}

	state := &serverState{p2p: p2p, logger: logger}
	go state.consumeMessages(msgCh)

	// Always write announce file for automation if requested.
	if *announceFile != "" {
		info := hostInfoFromP2P(p2p, *enableRelayService)
		if err := writeHostInfo(*announceFile, info); err != nil {
			logger.Warn("failed to write announce file", "error", err)
		}
	}

	if *httpAddr != "" {
		go state.serveHTTP(ctx, *httpAddr)
	}

	logger.Info("p2p node started", "role", *role, "peer", p2p.HostID())
	<-ctx.Done()
	logger.Info("shutting down p2p node")
	_ = p2p.Shutdown()
}

func (s *serverState) consumeMessages(ch <-chan *networking.ControllerMessage) {
	if ch == nil {
		return
	}
	for msg := range ch {
		if msg == nil {
			continue
		}
		s.mu.Lock()
		s.messages = append(s.messages, *msg)
		s.mu.Unlock()
	}
}

func (s *serverState) serveHTTP(ctx context.Context, addr string) {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/info", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, hostInfoFromP2P(s.p2p, s.p2p != nil && s.p2p.HostID() != ""))
	})

	mux.HandleFunc("/messages", func(w http.ResponseWriter, _ *http.Request) {
		s.mu.Lock()
		msgs := append([]networking.ControllerMessage(nil), s.messages...)
		s.mu.Unlock()
		writeJSON(w, msgs)
	})

	mux.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		info := hostInfoFromP2P(s.p2p, false)
		metrics := map[string]interface{}{
			"peer_id":    info.PeerID,
			"addrs":      info.Addrs,
			"peer_count": s.p2p.PeerCount(),
			"topics": map[string]bool{
				"heartbeat":  s.p2p.IsHeartbeatJoined(),
				"controller": s.p2p.IsControllerJoined(),
				"leader":     s.p2p.IsLeaderElectionJoined(),
			},
		}
		writeJSON(w, metrics)
	})

	mux.HandleFunc("/publish", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req publishRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ctxPub, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		msg := networking.ControllerMessage{Type: req.Type, Data: req.Data}
		if err := s.p2p.PublishControllerMessage(ctxPub, mustJSON(msg)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	})

	mux.HandleFunc("/connect", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req connectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.PeerID == "" {
			http.Error(w, "peer_id required", http.StatusBadRequest)
			return
		}
		ctxDial, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		if err := s.connect(ctxDial, req); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("/put-dht", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req dhtRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ctxPut, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		if err := s.p2p.PutDHTValue(ctxPut, req.Key, []byte(req.Value)); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("/get-dht", func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		if key == "" {
			http.Error(w, "key required", http.StatusBadRequest)
			return
		}
		ctxGet, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		val, err := s.p2p.GetDHTValue(ctxGet, key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, map[string]string{"value": string(val)})
	})

	server := &http.Server{Addr: addr, Handler: mux}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.logger.Error("http server error", "error", err)
	}
}

func (s *serverState) connect(ctx context.Context, req connectRequest) error {
	targetID, err := peer.Decode(req.PeerID)
	if err != nil {
		return fmt.Errorf("invalid peer id: %w", err)
	}

	if req.RelayAddr != "" {
		if err := s.p2p.ConnectViaRelay(ctx, req.RelayAddr, targetID); err != nil {
			return fmt.Errorf("relay connect failed: %w", err)
		}
		return nil
	}

	if req.Addr != "" {
		addr, err := ma.NewMultiaddr(req.Addr)
		if err != nil {
			return fmt.Errorf("invalid addr: %w", err)
		}
		info, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			return fmt.Errorf("addr info: %w", err)
		}
		if err := s.p2p.ConnectToAddrInfo(ctx, *info); err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		return nil
	}

	return fmt.Errorf("either relay_addr or addr required")
}

func hostInfoFromP2P(p *networking.P2P, includeRelay bool) hostInfo {
	var addrs []string
	pid := p.HostID()
	for _, a := range p.HostAddrs() {
		addrs = append(addrs, fmt.Sprintf("%s/p2p/%s", a.String(), pid))
	}
	info := hostInfo{PeerID: pid, Addrs: addrs}
	if includeRelay {
		for _, a := range addrs {
			info.RelayAddrs = append(info.RelayAddrs, fmt.Sprintf("%s/p2p-circuit", a))
		}
	}
	return info
}

func writeHostInfo(path string, info hostInfo) error {
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func loadBootstrapFromFile(logger *slog.Logger, path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		logger.Warn("read bootstrap file", "error", err)
		return nil
	}
	var info hostInfo
	if err := json.Unmarshal(data, &info); err != nil {
		logger.Warn("parse bootstrap file", "error", err)
		return nil
	}
	return append([]string{}, info.Addrs...)
}

func splitList(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func mustJSON(msg networking.ControllerMessage) []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		return []byte(`{"type":"error","data":"marshal"}`)
	}
	return b
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
