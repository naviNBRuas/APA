package agent

import (
	"context"
	"crypto/ed25519"
	"testing"
	"time"

	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/require"
	"log/slog"
)

func TestEphemeralIdentityRotates(t *testing.T) {
	logger := slog.Default()
	priv, _, err := libp2pcrypto.GenerateEd25519Key(nil)
	require.NoError(t, err, "failed to generate base key")

	mgr, err := NewEphemeralIdentityManager(logger, priv, 50*time.Millisecond)
	require.NoError(t, err, "failed to create manager")
	mgr.Start(testCtx(t))
	first := mgr.Current().SessionID
	time.Sleep(80 * time.Millisecond)
	second := mgr.Current().SessionID
	require.NotEqual(t, first, second, "expected session ID to rotate, got same %s", first)
}

func TestEphemeralIdentitySignatureBindsKey(t *testing.T) {
	logger := slog.Default()
	priv, _, err := libp2pcrypto.GenerateEd25519Key(nil)
	require.NoError(t, err, "failed to generate base key")

	mgr, err := NewEphemeralIdentityManager(logger, priv, time.Second)
	require.NoError(t, err, "failed to create manager")
	mgr.Start(testCtx(t))
	sess := mgr.Current()
	raw, err := priv.Raw()
	require.NoError(t, err, "failed to export raw priv key")
	base := ed25519.PrivateKey(raw)
	require.True(t, ed25519.Verify(base.Public().(ed25519.PublicKey), sess.PublicKey, sess.Signature), "signature did not verify with base public key")
}

// helper returns a cancellable context tied to the test.
func testCtx(t *testing.T) (ctx context.Context) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return ctx
}
