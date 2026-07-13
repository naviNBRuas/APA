package networking

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

type mockPolicyEnforcerLocal struct{}

func (m mockPolicyEnforcerLocal) Authorize(ctx context.Context, subject, action, resource string) (bool, string, error) {
	return true, "ok", nil
}

func newTestP2P(t *testing.T) (*P2P, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	priv, _, err := crypto.GenerateEd25519Key(nil)
	if err != nil {
		cancel()
		require.NoError(t, err, "failed to generate key")
	}
	pid, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		cancel()
		require.NoError(t, err, "failed to derive peer id")
	}
	cfg := Config{
		ListenAddresses: []string{
			"/ip4/127.0.0.1/tcp/0",
			"/ip4/127.0.0.1/udp/0/quic-v1",
		},
		BootstrapPeers:    []string{"invalid-bootstrap-address"},
		HeartbeatInterval: 200 * time.Millisecond,
		ServiceTag:        "test-svc",
	}
	p, err := NewP2P(ctx, testLogger(t), cfg, pid, priv, mockPolicyEnforcerLocal{})
	if err != nil {
		cancel()
		require.NoError(t, err, "failed to create p2p")
	}
	return p, cancel
}

func testLogger(t *testing.T) *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

func requireP2PIntegration(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping local P2P integration test in short mode")
	}
	if os.Getenv("APA_RUN_P2P_INTEGRATION") != "1" {
		t.Skip("set APA_RUN_P2P_INTEGRATION=1 to run local P2P integration tests")
	}
}

func connectPeers(t *testing.T, a, b *P2P) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	info := peer.AddrInfo{ID: b.host.ID(), Addrs: b.host.Addrs()}
	require.NoError(t, a.host.Connect(ctx, info), "connect failed")
	info2 := peer.AddrInfo{ID: a.host.ID(), Addrs: a.host.Addrs()}
	require.NoError(t, b.host.Connect(ctx, info2), "connect back failed")
}

func TestP2PLocalHeartbeatPropagation(t *testing.T) {
	requireP2PIntegration(t)

	p1, cancel1 := newTestP2P(t)
	defer cancel1()
	defer p1.host.Close()
	p2, cancel2 := newTestP2P(t)
	defer cancel2()
	defer p2.host.Close()

	connectPeers(t, p1, p2)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	require.NoError(t, p1.JoinHeartbeatTopic(ctx), "p1 join heartbeat")
	require.NoError(t, p2.JoinHeartbeatTopic(ctx), "p2 join heartbeat")
	sub, err := p2.pubsub.Subscribe(HeartbeatTopic)
	require.NoError(t, err, "subscribe")
	go p1.StartHeartbeat(ctx, 100*time.Millisecond)

	received := make(chan struct{}, 1)
	go func() {
		msg, err := sub.Next(ctx)
		if err == nil && msg != nil {
			received <- struct{}{}
		}
	}()

	select {
	case <-received:
	case <-ctx.Done():
		require.Fail(t, "timeout waiting for heartbeat")
	}
}

func TestP2PControllerMessageRoundTrip(t *testing.T) {
	requireP2PIntegration(t)

	p1, cancel1 := newTestP2P(t)
	defer cancel1()
	defer p1.host.Close()
	p2, cancel2 := newTestP2P(t)
	defer cancel2()
	defer p2.host.Close()

	connectPeers(t, p1, p2)

	joinCtx, joinCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer joinCancel()
	require.NoError(t, p1.JoinControllerCommTopic(joinCtx), "p1 join controller")
	require.NoError(t, p2.JoinControllerCommTopic(joinCtx), "p2 join controller")
	ch, err := p2.SubscribeControllerMessages(joinCtx)
	require.NoError(t, err, "subscribe controller")

	payload := []byte(`{"type":"test","data":"hello"}`)
	recvCtx, recvCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer recvCancel()

	for {
		require.NoError(t, p1.PublishControllerMessage(joinCtx, payload), "publish controller")

		select {
		case msg := <-ch:
			require.NotNil(t, msg)
			require.Equal(t, "test", msg.Type)
			var data string
			require.NoError(t, json.Unmarshal(msg.Data, &data))
			require.Equal(t, "hello", data)
			return
		case <-time.After(250 * time.Millisecond):
			continue
		case <-recvCtx.Done():
			require.Fail(t, "timeout waiting for controller message")
		}
	}
}
