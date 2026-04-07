package agent

import (
	"context"
	"crypto/ed25519"
	"testing"
	"time"

	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"log/slog"
)

func TestEphemeralIdentityRotates(t *testing.T) {
	logger := slog.Default()
	priv, _, err := libp2pcrypto.GenerateEd25519Key(nil)
	if err != nil {
		t.Fatalf("failed to generate base key: %v", err)
	}

	mgr, err := NewEphemeralIdentityManager(logger, priv, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	mgr.Start(testCtx(t))
	first := mgr.Current().SessionID
	time.Sleep(80 * time.Millisecond)
	second := mgr.Current().SessionID
	if first == second {
		t.Fatalf("expected session ID to rotate, got same %s", first)
	}
}

func TestEphemeralIdentitySignatureBindsKey(t *testing.T) {
	logger := slog.Default()
	priv, _, err := libp2pcrypto.GenerateEd25519Key(nil)
	if err != nil {
		t.Fatalf("failed to generate base key: %v", err)
	}

	mgr, err := NewEphemeralIdentityManager(logger, priv, time.Second)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	mgr.Start(testCtx(t))
	sess := mgr.Current()
	raw, err := priv.Raw()
	if err != nil {
		t.Fatalf("failed to export raw priv key: %v", err)
	}
	base := ed25519.PrivateKey(raw)
	if !ed25519.Verify(base.Public().(ed25519.PublicKey), sess.PublicKey, sess.Signature) {
		t.Fatalf("signature did not verify with base public key")
	}
}

// helper returns a cancellable context tied to the test.
func testCtx(t *testing.T) (ctx context.Context) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return ctx
}
