package agent

import (
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
)

// EphemeralConfig controls session identity rotation.
type EphemeralConfig struct {
	RotationInterval time.Duration `yaml:"rotation_interval"`
}

// EphemeralSession captures the current session-bound identity.
type EphemeralSession struct {
	SessionID string
	PublicKey ed25519.PublicKey
	Signature []byte // signed by base private key over PublicKey
	ExpiresAt time.Time
}

// EphemeralIdentityManager rotates session identities derived from a base key.
type EphemeralIdentityManager struct {
	logger   *slog.Logger
	baseKey  ed25519.PrivateKey
	interval time.Duration

	mu      sync.RWMutex
	current EphemeralSession

	cancel context.CancelFunc
}

func NewEphemeralIdentityManager(logger *slog.Logger, baseKey libp2pcrypto.PrivKey, interval time.Duration) (*EphemeralIdentityManager, error) {
	edKey, err := toEd25519(baseKey)
	if err != nil {
		return nil, err
	}
	if interval <= 0 {
		interval = 10 * time.Minute
	}
	return &EphemeralIdentityManager{logger: logger, baseKey: edKey, interval: interval}, nil
}

// Start begins rotation on the provided context.
func (m *EphemeralIdentityManager) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.rotate()
	go m.loop(ctx)
}

// Stop halts rotation.
func (m *EphemeralIdentityManager) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
}

// ForceRotate triggers an immediate identity rotation.
func (m *EphemeralIdentityManager) ForceRotate() {
	m.rotate()
}

// Current returns the current session identity snapshot.
func (m *EphemeralIdentityManager) Current() EphemeralSession {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

func (m *EphemeralIdentityManager) loop(ctx context.Context) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.rotate()
		}
	}
}

func (m *EphemeralIdentityManager) rotate() {
	// Derive deterministic-but-randomized seed using base key and random salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		m.logger.Warn("ephemeral identity: failed to read random salt", "error", err)
	}
	h := hmac.New(sha256.New, m.baseKey)
	h.Write(salt)
	seed := h.Sum(nil)
	if len(seed) < ed25519.SeedSize {
		// pad if ever needed (unlikely)
		padded := make([]byte, ed25519.SeedSize)
		copy(padded, seed)
		seed = padded
	}
	seed = seed[:ed25519.SeedSize]

	priv := ed25519.NewKeyFromSeed(seed)
	pub := priv.Public().(ed25519.PublicKey)

	// SessionID derived from public key hash
	pubHash := sha256.Sum256(pub)
	sessionID := hex.EncodeToString(pubHash[:])

	// Sign ephemeral public key with base key for attestable binding
	sig := ed25519.Sign(m.baseKey, pub)

	session := EphemeralSession{
		SessionID: sessionID,
		PublicKey: pub,
		Signature: sig,
		ExpiresAt: time.Now().Add(m.interval),
	}

	m.mu.Lock()
	m.current = session
	m.mu.Unlock()
	m.logger.Debug("ephemeral identity rotated", "session_id", sessionID, "expires_at", session.ExpiresAt)
}

// toEd25519 converts a libp2p private key into a crypto/ed25519 private key.
func toEd25519(key libp2pcrypto.PrivKey) (ed25519.PrivateKey, error) {
	if key.Type() != libp2pcrypto.Ed25519 {
		return nil, fmt.Errorf("ephemeral identity requires ed25519 key, got %v", key.Type())
	}
	raw, err := key.Raw()
	if err != nil {
		return nil, fmt.Errorf("failed to extract raw key: %w", err)
	}
	if len(raw) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("unexpected ed25519 private key size %d", len(raw))
	}
	return ed25519.PrivateKey(raw), nil
}
