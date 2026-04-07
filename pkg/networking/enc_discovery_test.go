package networking

import (
	"crypto/rand"
	"testing"
	"time"
)

func TestEncryptedDiscoveryRoundTrip(t *testing.T) {
	ed, err := NewEncryptedDiscovery("test-seed")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		t.Fatalf("nonce: %v", err)
	}
	// Use a valid multibase peer ID string.
	beacon := DiscoveryBeacon{PeerID: "12D3KooWQAbH8ZqqpwYBFvToNi4zmiEJeZVrDCTnhgypKekJy5oM", Addrs: []string{"/ip4/127.0.0.1/tcp/4001"}, Ts: time.Now().Unix(), Nonce: nonce}
	ct, err := ed.EncodeBeacon(beacon)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	out, err := ed.DecodeBeacon(nonce, ct, time.Minute)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.PeerID != beacon.PeerID || out.Addrs[0] != beacon.Addrs[0] {
		t.Fatalf("mismatch after decode")
	}
}

func TestEncryptedDiscoveryRejectsOld(t *testing.T) {
	ed, _ := NewEncryptedDiscovery("seed")
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		t.Fatalf("Failed to generate nonce: %v", err)
	}
	beacon := DiscoveryBeacon{PeerID: "12D3KooWQAbH8ZqqpwYBFvToNi4zmiEJeZVrDCTnhgypKekJy5oM", Addrs: nil, Ts: time.Now().Add(-2 * time.Hour).Unix(), Nonce: nonce}
	ct, _ := ed.EncodeBeacon(beacon)
	if _, err := ed.DecodeBeacon(nonce, ct, 10*time.Minute); err == nil {
		t.Fatalf("expected old beacon rejection")
	}
}
