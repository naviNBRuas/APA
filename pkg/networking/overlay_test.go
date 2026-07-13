package networking

import (
	"encoding/base64"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

func testOverlayLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestHTTPSOverlayLooksPlausible(t *testing.T) {
	mux := NewOverlayMux(testOverlayLogger(), OverlayConfig{})
	env, err := mux.BuildEnvelope(OverlayHTTPS, []byte("hello"))
	require.NoError(t, err, "build failed")

	require.NotEmpty(t, env.Host, "expected host to be populated")
	require.NotEmpty(t, env.Path, "expected path to be populated")
	require.NotEmpty(t, env.Headers["User-Agent"], "expected User-Agent header")
	require.NotEmpty(t, env.Headers["Accept"], "expected Accept header")
	require.NotEmpty(t, env.Padding, "expected padding to be applied")
}

func TestWebSocketOverlayHasHandshakeKey(t *testing.T) {
	mux := NewOverlayMux(testOverlayLogger(), OverlayConfig{})
	env, err := mux.BuildEnvelope(OverlayWebSocket, []byte("ping"))
	require.NoError(t, err, "build failed")

	key := env.Headers["Sec-WebSocket-Key"]
	require.NotEmpty(t, key, "expected websocket key header")

	raw, err := base64.StdEncoding.DecodeString(key)
	require.NoError(t, err)
	require.Equal(t, 16, len(raw), "expected base64-encoded 16-byte key")
	require.NotEmpty(t, env.Padding, "expected padding")
}

func TestQUICOverlayContainsALPNAndPadding(t *testing.T) {
	mux := NewOverlayMux(testOverlayLogger(), OverlayConfig{MinPadding: 8, MaxPadding: 16})
	env, err := mux.BuildEnvelope(OverlayQUIC, []byte("data"))
	require.NoError(t, err, "build failed")

	require.NotEmpty(t, env.ALPN, "expected ALPN set")
	require.GreaterOrEqual(t, len(env.Padding), 8)
	require.LessOrEqual(t, len(env.Padding), 16)
}

func TestDNSOverlayUsesConfiguredNames(t *testing.T) {
	mux := NewOverlayMux(testOverlayLogger(), OverlayConfig{DNSNames: []string{"example.com"}})
	env, err := mux.BuildEnvelope(OverlayDNS, []byte("q"))
	require.NoError(t, err, "build failed")

	require.Equal(t, "example.com", env.DNSName)
}

func TestPaddingUsesRange(t *testing.T) {
	mux := NewOverlayMux(testOverlayLogger(), OverlayConfig{MinPadding: 1, MaxPadding: 2})
	env, err := mux.BuildEnvelope(OverlayHTTPS, []byte("x"))
	require.NoError(t, err, "build failed")

	l := len(env.Padding)
	require.GreaterOrEqual(t, l, 1, "padding length out of range")
	require.LessOrEqual(t, l, 2, "padding length out of range")
}
