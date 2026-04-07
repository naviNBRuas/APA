package networking

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	mathrand "math/rand"
	"time"
)

// OverlayMode enumerates supported disguise transports.
//
// These overlays encapsulate agent payloads to resemble common client traffic,
// leveraging transport-layer ambiguity to reduce fingerprintability.
type OverlayMode string

const (
	OverlayHTTPS     OverlayMode = "https"
	OverlayWebSocket OverlayMode = "websocket"
	OverlayQUIC      OverlayMode = "quic"
	OverlayDNS       OverlayMode = "dns"
)

// OverlayConfig controls cover-traffic synthesis.
type OverlayConfig struct {
	Hosts      []string
	Paths      []string
	UserAgents []string
	DNSNames   []string
	MinPadding int
	MaxPadding int
}

func (c OverlayConfig) withDefaults() OverlayConfig {
	if len(c.Hosts) == 0 {
		c.Hosts = []string{"cdn.jsdelivr.net", "fonts.gstatic.com", "api.segment.io", "i.scdn.co"}
	}
	if len(c.Paths) == 0 {
		c.Paths = []string{"/v1/metrics", "/collect", "/analytics", "/resources", "/graphql"}
	}
	if len(c.UserAgents) == 0 {
		c.UserAgents = []string{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3 Safari/605.1.15",
			"Mozilla/5.0 (X11; Linux x86_64; rv:121.0) Gecko/20100101 Firefox/121.0",
		}
	}
	if len(c.DNSNames) == 0 {
		c.DNSNames = []string{"www.google.com", "www.cloudflare.com", "api.github.com", "cdn.cloudflare.net"}
	}
	if c.MinPadding <= 0 {
		c.MinPadding = 32
	}
	if c.MaxPadding <= 0 {
		c.MaxPadding = 96
	}
	if c.MaxPadding < c.MinPadding {
		c.MaxPadding = c.MinPadding
	}
	return c
}

// OverlayEnvelope carries synthesized metadata and padding that should resemble benign traffic.
type OverlayEnvelope struct {
	Mode          OverlayMode
	Host          string
	Path          string
	Headers       map[string]string
	Payload       []byte
	Padding       []byte
	ALPN          string
	DNSName       string
	TransportHint string
}

// OverlayMux constructs envelopes for multiple disguise transports.
type OverlayMux struct {
	logger *slog.Logger
	cfg    OverlayConfig
	rng    *mathrand.Rand
}

// NewOverlayMux returns a mux with randomized seed and sensible defaults.
func NewOverlayMux(logger *slog.Logger, cfg OverlayConfig) *OverlayMux {
	cfg = cfg.withDefaults()
	src := mathrand.NewSource(time.Now().UnixNano())
	return &OverlayMux{logger: logger, cfg: cfg, rng: mathrand.New(src)}
}

// BuildEnvelope generates a cover envelope for the requested overlay mode.
func (m *OverlayMux) BuildEnvelope(mode OverlayMode, payload []byte) (OverlayEnvelope, error) {
	switch mode {
	case OverlayHTTPS:
		return m.buildHTTPS(payload), nil
	case OverlayWebSocket:
		return m.buildWebSocket(payload), nil
	case OverlayQUIC:
		return m.buildQUIC(payload), nil
	case OverlayDNS:
		return m.buildDNS(payload), nil
	default:
		return OverlayEnvelope{}, fmt.Errorf("unsupported overlay mode: %s", mode)
	}
}

func (m *OverlayMux) buildHTTPS(payload []byte) OverlayEnvelope {
	host := m.pick(m.cfg.Hosts)
	path := m.pick(m.cfg.Paths)
	ua := m.pick(m.cfg.UserAgents)
	headers := map[string]string{
		"Host":            host,
		"User-Agent":      ua,
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
		"Accept-Language": m.pick([]string{"en-US,en;q=0.9", "en-GB,en;q=0.8", "en;q=0.7"}),
		"Cache-Control":   "no-cache",
	}
	return OverlayEnvelope{
		Mode:    OverlayHTTPS,
		Host:    host,
		Path:    path,
		Headers: headers,
		Payload: append([]byte(nil), payload...),
		Padding: m.padding(),
	}
}

func (m *OverlayMux) buildWebSocket(payload []byte) OverlayEnvelope {
	host := m.pick(m.cfg.Hosts)
	path := m.pick(m.cfg.Paths)
	ua := m.pick(m.cfg.UserAgents)
	key := m.wsKey()
	headers := map[string]string{
		"Host":                   host,
		"User-Agent":             ua,
		"Upgrade":                "websocket",
		"Connection":             "Upgrade",
		"Sec-WebSocket-Version":  "13",
		"Sec-WebSocket-Key":      key,
		"Sec-WebSocket-Protocol": m.pick([]string{"chat", "json", "grpc-web"}),
	}
	return OverlayEnvelope{
		Mode:          OverlayWebSocket,
		Host:          host,
		Path:          path,
		Headers:       headers,
		Payload:       append([]byte(nil), payload...),
		Padding:       m.padding(),
		TransportHint: "http/1.1",
	}
}

func (m *OverlayMux) buildQUIC(payload []byte) OverlayEnvelope {
	host := m.pick(m.cfg.Hosts)
	path := m.pick(m.cfg.Paths)
	alpn := m.pick([]string{"h3", "h3-29", "hq-interop"})
	headers := map[string]string{
		"Authority": host,
		"Path":      path,
		"Scheme":    "https",
	}
	return OverlayEnvelope{
		Mode:          OverlayQUIC,
		Host:          host,
		Path:          path,
		Headers:       headers,
		Payload:       append([]byte(nil), payload...),
		Padding:       m.padding(),
		ALPN:          alpn,
		TransportHint: "quic",
	}
}

func (m *OverlayMux) buildDNS(payload []byte) OverlayEnvelope {
	name := m.pick(m.cfg.DNSNames)
	headers := map[string]string{
		"Accept":          "application/dns-message",
		"Cache-Control":   "no-cache",
		"Accept-Language": m.pick([]string{"en-US,en;q=0.9", "en-GB,en;q=0.8"}),
	}
	hint := m.pick([]string{"doh-get", "doh-post", "udp"})
	return OverlayEnvelope{
		Mode:          OverlayDNS,
		DNSName:       name,
		Headers:       headers,
		Payload:       append([]byte(nil), payload...),
		Padding:       m.padding(),
		TransportHint: hint,
	}
}

func (m *OverlayMux) padding() []byte {
	if m.cfg.MaxPadding <= 0 {
		return nil
	}
	span := m.cfg.MaxPadding - m.cfg.MinPadding
	if span < 0 {
		span = 0
	}
	length := m.cfg.MinPadding
	if span > 0 {
		length += m.rng.Intn(span + 1)
	}
	if length == 0 {
		return nil
	}

	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		// best-effort; fall back to pseudo-random
		for i := range buf {
			buf[i] = byte(m.rng.Intn(256))
		}
	}
	return buf
}

func (m *OverlayMux) pick(values []string) string {
	if len(values) == 0 {
		return ""
	}
	if len(values) == 1 {
		return values[0]
	}
	idx := m.rng.Intn(len(values))
	return values[idx]
}

func (m *OverlayMux) wsKey() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		for i := range b {
			b[i] = byte(m.rng.Intn(256))
		}
	}
	return base64.StdEncoding.EncodeToString(b)
}
