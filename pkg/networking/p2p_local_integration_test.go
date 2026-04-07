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
)

// mockPolicyEnforcerLocal accepts all actions.
type mockPolicyEnforcerLocal struct{}

func (m mockPolicyEnforcerLocal) Authorize(ctx context.Context, subject, action, resource string) (bool, string, error) {
	return true, "ok", nil
}

// newTestP2P spins up a P2P node bound to loopback on random ports.
func newTestP2P(t *testing.T) (*P2P, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	priv, _, err := crypto.GenerateEd25519Key(nil)
	if err != nil {
		cancel()
		t.Fatalf("failed to generate key: %v", err)
	}
	pid, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		cancel()
		t.Fatalf("failed to derive peer id: %v", err)
	}
	cfg := Config{
		ListenAddresses: []string{
			"/ip4/127.0.0.1/tcp/0",
			"/ip4/127.0.0.1/udp/0/quic-v1",
		},
		// Keep this non-empty to avoid NewP2P falling back to public default
		// bootstrap peers in local integration tests.
		BootstrapPeers:    []string{"invalid-bootstrap-address"},
		HeartbeatInterval: 200 * time.Millisecond,
		ServiceTag:        "test-svc",
	}
	p, err := NewP2P(ctx, testLogger(t), cfg, pid, priv, mockPolicyEnforcerLocal{})
	if err != nil {
		cancel()
		t.Fatalf("failed to create p2p: %v", err)
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

// connectPeers connects a to b bidirectionally.
func connectPeers(t *testing.T, a, b *P2P) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	info := peer.AddrInfo{ID: b.host.ID(), Addrs: b.host.Addrs()}
	if err := a.host.Connect(ctx, info); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	info2 := peer.AddrInfo{ID: a.host.ID(), Addrs: a.host.Addrs()}
	if err := b.host.Connect(ctx, info2); err != nil {
		t.Fatalf("connect back failed: %v", err)
	}
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
	if err := p1.JoinHeartbeatTopic(ctx); err != nil {
		t.Fatalf("p1 join heartbeat: %v", err)
	}
	if err := p2.JoinHeartbeatTopic(ctx); err != nil {
		t.Fatalf("p2 join heartbeat: %v", err)
	}
	sub, err := p2.pubsub.Subscribe(HeartbeatTopic)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
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
		t.Fatalf("timeout waiting for heartbeat")
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
	if err := p1.JoinControllerCommTopic(joinCtx); err != nil {
		t.Fatalf("p1 join controller: %v", err)
	}
	if err := p2.JoinControllerCommTopic(joinCtx); err != nil {
		t.Fatalf("p2 join controller: %v", err)
	}
	ch, err := p2.SubscribeControllerMessages(joinCtx)
	if err != nil {
		t.Fatalf("subscribe controller: %v", err)
	}

	payload := []byte(`{"type":"test","data":"hello"}`)
	recvCtx, recvCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer recvCancel()

	for {
		if err := p1.PublishControllerMessage(joinCtx, payload); err != nil {
			t.Fatalf("publish controller: %v", err)
		}

		select {
		case msg := <-ch:
			if msg == nil || msg.Type != "test" {
				t.Fatalf("unexpected message: %+v", msg)
			}
			var data string
			if err := json.Unmarshal(msg.Data, &data); err != nil || data != "hello" {
				t.Fatalf("unexpected data: %s err=%v", string(msg.Data), err)
			}
			return
		case <-time.After(250 * time.Millisecond):
			continue
		case <-recvCtx.Done():
			t.Fatalf("timeout waiting for controller message")
		}
	}
}
