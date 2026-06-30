package networking

import (
	"context"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/miekg/dns"
)

type DNSProtocol struct {
	logger        *slog.Logger
	client        *dns.Client
	server        *dns.Server
	config        DNSConfig
	messageChan   chan *NetworkMessage
	healthMetrics *ProtocolHealthMetrics

	mu    sync.RWMutex
	peers map[peer.ID]string
}

type DNSConfig struct {
	ListenAddr string `json:"listen_addr"`
	Domain     string `json:"domain"`
}

type DNSCacheEntry struct {
	Data      []byte    `json:"data"`
	Timestamp time.Time `json:"timestamp"`
	Expires   time.Time `json:"expires"`
	Hits      int       `json:"hits"`
}

var dnsEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

func NewDNSProtocol(logger *slog.Logger, config DNSConfig) (*DNSProtocol, error) {
	domain := config.Domain
	if domain == "" {
		domain = "apa.dns"
	}
	return &DNSProtocol{
		logger: logger,
		client: &dns.Client{Net: "udp", Timeout: 5 * time.Second},
		config: DNSConfig{
			ListenAddr: config.ListenAddr,
			Domain:     domain,
		},
		messageChan:   make(chan *NetworkMessage, 100),
		healthMetrics: &ProtocolHealthMetrics{ProtocolType: ProtocolDNS},
		peers:         make(map[peer.ID]string),
	}, nil
}

func (dp *DNSProtocol) Initialize(ctx context.Context) error {
	addr := dp.config.ListenAddr
	if addr == "" {
		addr = ":0"
	}

	mux := dns.NewServeMux()
	mux.HandleFunc(dp.config.Domain, dp.handleDNSQuery)

	dp.server = &dns.Server{
		Addr:    addr,
		Net:     "udp",
		Handler: mux,
	}

	go func() {
		if err := dp.server.ListenAndServe(); err != nil {
			dp.logger.Error("DNS server error", "error", err)
		}
	}()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = dp.server.ShutdownContext(shutdownCtx)
	}()

	dp.logger.Info("DNS protocol initialized", "listen_addr", addr, "domain", dp.config.Domain)
	return nil
}

func (dp *DNSProtocol) handleDNSQuery(w dns.ResponseWriter, r *dns.Msg) {
	if len(r.Question) == 0 {
		return
	}
	q := r.Question[0]
	if q.Qtype != dns.TypeTXT {
		return
	}

	msg := new(dns.Msg)
	msg.SetReply(r)
	msg.Authoritative = true

	data, err := dp.decodeQueryName(q.Name)
	if err != nil {
		_ = w.WriteMsg(msg)
		return
	}

	var netMsg NetworkMessage
	if err := json.Unmarshal(data, &netMsg); err != nil {
		_ = w.WriteMsg(msg)
		return
	}

	dp.healthMetrics.TotalMessagesRecv++
	select {
	case dp.messageChan <- &netMsg:
	default:
	}

	msg.Answer = append(msg.Answer, &dns.TXT{
		Hdr: dns.RR_Header{
			Name:   q.Name,
			Rrtype: dns.TypeTXT,
			Class:  dns.ClassINET,
			Ttl:    60,
		},
		Txt: []string{"ok"},
	})
	_ = w.WriteMsg(msg)
}

func (dp *DNSProtocol) decodeQueryName(name string) ([]byte, error) {
	labels := dns.SplitDomainName(name)
	if len(labels) < 2 {
		return nil, fmt.Errorf("too few labels")
	}
	encoded := ""
	for i := 0; i < len(labels)-1; i++ {
		encoded += labels[i]
	}
	return dnsEncoding.DecodeString(encoded)
}

func (dp *DNSProtocol) encodeMessage(msg *NetworkMessage) string {
	data, err := json.Marshal(msg)
	if err != nil {
		return ""
	}
	encoded := dnsEncoding.EncodeToString(data)
	return encoded + "." + dp.config.Domain + "."
}

func (dp *DNSProtocol) RegisterPeerEndpoint(id peer.ID, addr string) {
	dp.mu.Lock()
	dp.peers[id] = addr
	dp.mu.Unlock()
}

func (dp *DNSProtocol) SendMessage(to peer.ID, message *NetworkMessage) error {
	dp.mu.RLock()
	addr, ok := dp.peers[to]
	dp.mu.RUnlock()
	if !ok {
		return fmt.Errorf("dns: unknown peer %s", to)
	}

	queryName := dp.encodeMessage(message)
	if queryName == "" {
		return fmt.Errorf("dns: failed to encode message")
	}

	m := new(dns.Msg)
	m.SetQuestion(queryName, dns.TypeTXT)

	_, _, err := dp.client.Exchange(m, addr)
	if err != nil {
		return fmt.Errorf("dns exchange: %w", err)
	}
	dp.healthMetrics.TotalMessagesSent++
	return nil
}

func (dp *DNSProtocol) ReceiveMessages() <-chan *NetworkMessage {
	return dp.messageChan
}

func (dp *DNSProtocol) GetConnectionInfo() *ConnectionInfo {
	return &ConnectionInfo{
		LocalAddress: dp.config.ListenAddr,
		Protocol:     ProtocolDNS,
		Status:       ConnectionConnected,
		Established:  time.Now(),
		LastActivity: time.Now(),
	}
}

func (dp *DNSProtocol) GetHealthMetrics() *ProtocolHealthMetrics {
	return dp.healthMetrics
}

func (dp *DNSProtocol) Close() error {
	if dp.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return dp.server.ShutdownContext(ctx)
	}
	return nil
}
