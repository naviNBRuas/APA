package networking

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEncryptedDiscoveryRoundTrip(t *testing.T) {
	ed, err := NewEncryptedDiscovery("test-seed")
	require.NoError(t, err, "init: %v", err)
	nonce := make([]byte, 12)
	_, err = rand.Read(nonce)
	require.NoError(t, err, "nonce: %v", err)
	beacon := DiscoveryBeacon{PeerID: "12D3KooWQAbH8ZqqpwYBFvToNi4zmiEJeZVrDCTnhgypKekJy5oM", Addrs: []string{"/ip4/127.0.0.1/tcp/4001"}, Ts: time.Now().Unix(), Nonce: nonce}
	ct, err := ed.EncodeBeacon(beacon)
	require.NoError(t, err, "encode: %v", err)

	out, err := ed.DecodeBeacon(nonce, ct, time.Minute)
	require.NoError(t, err, "decode: %v", err)
	require.Equal(t, beacon.PeerID, out.PeerID, "mismatch after decode")
	require.Equal(t, beacon.Addrs[0], out.Addrs[0], "mismatch after decode")
}

func TestEncryptedDiscoveryRejectsOld(t *testing.T) {
	ed, _ := NewEncryptedDiscovery("seed")
	nonce := make([]byte, 12)
	_, err := rand.Read(nonce)
	require.NoError(t, err, "Failed to generate nonce: %v", err)
	beacon := DiscoveryBeacon{PeerID: "12D3KooWQAbH8ZqqpwYBFvToNi4zmiEJeZVrDCTnhgypKekJy5oM", Addrs: nil, Ts: time.Now().Add(-2 * time.Hour).Unix(), Nonce: nonce}
	ct, _ := ed.EncodeBeacon(beacon)
	_, err = ed.DecodeBeacon(nonce, ct, 10*time.Minute)
	require.Error(t, err, "expected old beacon rejection")
}
