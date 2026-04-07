package networking

import (
	"encoding/base64"
	"io"
	"log/slog"
	"testing"
)

func testOverlayLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestHTTPSOverlayLooksPlausible(t *testing.T) {
	mux := NewOverlayMux(testOverlayLogger(), OverlayConfig{})
	env, err := mux.BuildEnvelope(OverlayHTTPS, []byte("hello"))
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if env.Host == "" || env.Path == "" {
		t.Fatalf("expected host and path to be populated")
	}
	if env.Headers["User-Agent"] == "" || env.Headers["Accept"] == "" {
		t.Fatalf("expected plausible headers present")
	}
	if len(env.Padding) == 0 {
		t.Fatalf("expected padding to be applied")
	}
}

func TestWebSocketOverlayHasHandshakeKey(t *testing.T) {
	mux := NewOverlayMux(testOverlayLogger(), OverlayConfig{})
	env, err := mux.BuildEnvelope(OverlayWebSocket, []byte("ping"))
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	key := env.Headers["Sec-WebSocket-Key"]
	if key == "" {
		t.Fatalf("expected websocket key header")
	}
	raw, err := base64.StdEncoding.DecodeString(key)
	if err != nil || len(raw) != 16 {
		t.Fatalf("expected base64-encoded 16-byte key, got len=%d err=%v", len(raw), err)
	}
	if len(env.Padding) == 0 {
		t.Fatalf("expected padding")
	}
}

func TestQUICOverlayContainsALPNAndPadding(t *testing.T) {
	mux := NewOverlayMux(testOverlayLogger(), OverlayConfig{MinPadding: 8, MaxPadding: 16})
	env, err := mux.BuildEnvelope(OverlayQUIC, []byte("data"))
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if env.ALPN == "" {
		t.Fatalf("expected ALPN set")
	}
	if len(env.Padding) < 8 || len(env.Padding) > 16 {
		t.Fatalf("padding outside expected range: %d", len(env.Padding))
	}
}

func TestDNSOverlayUsesConfiguredNames(t *testing.T) {
	mux := NewOverlayMux(testOverlayLogger(), OverlayConfig{DNSNames: []string{"example.com"}})
	env, err := mux.BuildEnvelope(OverlayDNS, []byte("q"))
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if env.DNSName != "example.com" {
		t.Fatalf("expected DNS name to come from config, got %s", env.DNSName)
	}
}

func TestPaddingUsesRange(t *testing.T) {
	mux := NewOverlayMux(testOverlayLogger(), OverlayConfig{MinPadding: 1, MaxPadding: 2})
	env, err := mux.BuildEnvelope(OverlayHTTPS, []byte("x"))
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if l := len(env.Padding); l < 1 || l > 2 {
		t.Fatalf("padding length out of range: %d", l)
	}
}
