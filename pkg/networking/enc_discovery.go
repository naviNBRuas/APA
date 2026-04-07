package networking

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"golang.org/x/crypto/hkdf"
)

// EncryptedDiscovery encrypts discovery beacons so they resemble opaque data.
type EncryptedDiscovery struct {
	key []byte
}

// DiscoveryBeacon carries peer identity hints.
type DiscoveryBeacon struct {
	PeerID string   `json:"peer_id"`
	Addrs  []string `json:"addrs"`
	Ts     int64    `json:"ts"`
	Nonce  []byte   `json:"nonce"`
}

// NewEncryptedDiscovery derives a symmetric key from seed material.
func NewEncryptedDiscovery(seed string) (*EncryptedDiscovery, error) {
	if seed == "" {
		seed = "apa-discovery-seed"
	}
	hk := hkdf.New(sha256.New, []byte(seed), nil, []byte("apa.discovery"))
	key := make([]byte, 32)
	if _, err := hk.Read(key); err != nil {
		return nil, fmt.Errorf("derive key: %w", err)
	}
	return &EncryptedDiscovery{key: key}, nil
}

// EncodeBeacon serializes and encrypts a beacon.
func (e *EncryptedDiscovery) EncodeBeacon(b DiscoveryBeacon) ([]byte, error) {
	if len(b.Nonce) != 12 {
		return nil, fmt.Errorf("nonce must be 12 bytes for AES-GCM")
	}
	raw, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	// Additional data could be used for versioning; omitted to blend in.
	ct := gcm.Seal(nil, b.Nonce, raw, nil)
	return ct, nil
}

// DecodeBeacon decrypts and validates a beacon payload.
func (e *EncryptedDiscovery) DecodeBeacon(nonce, ct []byte, maxSkew time.Duration) (DiscoveryBeacon, error) {
	var out DiscoveryBeacon
	if len(nonce) != 12 {
		return out, fmt.Errorf("nonce must be 12 bytes")
	}
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return out, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return out, err
	}
	raw, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return out, fmt.Errorf("decrypt beacon: %w", err)
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, err
	}
	// freshness check
	if maxSkew <= 0 {
		maxSkew = 10 * time.Minute
	}
	ts := time.Unix(out.Ts, 0)
	if time.Since(ts) > maxSkew {
		return out, errors.New("beacon too old")
	}
	if out.PeerID == "" {
		return out, errors.New("missing peer id")
	}
	if _, err := peer.Decode(out.PeerID); err != nil {
		return out, fmt.Errorf("invalid peer id: %w", err)
	}
	return out, nil
}

// BootstrapSources encapsulates possible discovery sources.
type BootstrapSources struct {
	PublicServices []string // e.g., https endpoints returning peer lists
	EmbeddedPeers  []string // shipped with binary/config
	OutOfBand      []string // operator-provided channel
}

// Combined returns the merged bootstrap list.
func (b BootstrapSources) Combined() []string {
	out := append([]string{}, b.PublicServices...)
	out = append(out, b.EmbeddedPeers...)
	out = append(out, b.OutOfBand...)
	return out
}
