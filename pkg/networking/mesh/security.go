package mesh

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"log/slog"
	"sync"
)

// SecurityManager handles authentication and encryption for the mesh.
type SecurityManager struct {
	logger     *slog.Logger
	mu         sync.RWMutex
	privKey    ed25519.PrivateKey
	pubKey     ed25519.PublicKey
	admitted   map[string]bool
	banned     map[string]bool
}

// NewSecurityManager creates a new SecurityManager.
func NewSecurityManager(logger *slog.Logger, privKey ed25519.PrivateKey, pubKey ed25519.PublicKey) *SecurityManager {
	return &SecurityManager{
		logger:   logger,
		privKey:  privKey,
		pubKey:   pubKey,
		admitted: make(map[string]bool),
		banned:   make(map[string]bool),
	}
}

// Sign signs data with the node's private key.
func (sm *SecurityManager) Sign(data []byte) []byte {
	hash := sha256.Sum256(data)
	return ed25519.Sign(sm.privKey, hash[:])
}

// Verify checks a signature against a public key.
func (sm *SecurityManager) Verify(data []byte, sig []byte, pubKey ed25519.PublicKey) bool {
	if len(pubKey) != ed25519.PublicKeySize {
		return false
	}
	hash := sha256.Sum256(data)
	return ed25519.Verify(pubKey, hash[:], sig)
}

// Authenticate verifies a NodeInfo's signature and checks admission policy.
func (sm *SecurityManager) Authenticate(info *NodeInfo) bool {
	if info == nil || info.PublicKey == nil {
		return false
	}

	sm.mu.RLock()
	if sm.banned[info.ID] {
		sm.mu.RUnlock()
		return false
	}
	sm.mu.RUnlock()

	sig := info.Signature
	info.Signature = nil
	data := sm.encodeForSigning(info)
	info.Signature = sig

	if len(info.PublicKey) != ed25519.PublicKeySize {
		return false
	}
	pubKey := ed25519.PublicKey(info.PublicKey)
	if !sm.Verify(data, sig, pubKey) {
		return false
	}

	return true
}

// SignNodeInfo creates a signed NodeInfo for this node.
func (sm *SecurityManager) SignNodeInfo(info *NodeInfo) *NodeInfo {
	info.Signature = nil
	data := sm.encodeForSigning(info)
	sig := sm.Sign(data)
	info.Signature = sig
	return info
}

func (sm *SecurityManager) encodeForSigning(v interface{}) []byte {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		return nil
	}
	return buf.Bytes()
}

// AdmitPeer adds a peer to the admit list.
func (sm *SecurityManager) AdmitPeer(peerID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.admitted[peerID] = true
	sm.logger.Debug("Peer admitted", "peer_id", truncateID(peerID))
}

// BanPeer bans a peer from the network.
func (sm *SecurityManager) BanPeer(peerID string, reason string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.banned[peerID] = true
	delete(sm.admitted, peerID)
	sm.logger.Warn("Peer banned", "peer_id", truncateID(peerID), "reason", reason)
}

// IsAdmitted checks if a peer is admitted.
func (sm *SecurityManager) IsAdmitted(peerID string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.admitted[peerID]
}

// IsBanned checks if a peer is banned.
func (sm *SecurityManager) IsBanned(peerID string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.banned[peerID]
}

// GenerateSessionKey creates a shared session key using the peer's public key.
// In production this would use X25519 ECDH; here we use a keyed hash.
func (sm *SecurityManager) GenerateSessionKey(peerPubKey ed25519.PublicKey) []byte {
	eph := make([]byte, 32)
	rand.Read(eph)
	key := sha256.Sum256(append(eph, peerPubKey...))
	return key[:]
}

// EncryptPayload encrypts data with a session key using AES-256-GCM.
// The returned ciphertext is nonce||sealed.
func (sm *SecurityManager) EncryptPayload(sessionKey []byte, plaintext []byte) ([]byte, error) {
	if len(sessionKey) != 32 {
		return nil, fmt.Errorf("session key must be 32 bytes, got %d", len(sessionKey))
	}
	block, err := aes.NewCipher(sessionKey)
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// DecryptPayload decrypts AES-256-GCM data produced by EncryptPayload.
func (sm *SecurityManager) DecryptPayload(sessionKey []byte, ciphertext []byte) ([]byte, error) {
	if len(sessionKey) != 32 {
		return nil, fmt.Errorf("session key must be 32 bytes, got %d", len(sessionKey))
	}
	block, err := aes.NewCipher(sessionKey)
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, sealed := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plain, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	return plain, nil
}

func init() {
	gob.Register(&NodeInfo{})
}

// NewNodeInfo creates a signed NodeInfo from a MeshNode.
func NewNodeInfo(node *MeshNode, sm *SecurityManager) *NodeInfo {
	pubKey := make([]byte, len(node.PublicKey))
	copy(pubKey, node.PublicKey)

	info := &NodeInfo{
		ID:           node.ID,
		PublicKey:    pubKey,
		VirtualIP:    node.VirtualIP,
		Name:         node.Name,
		Version:      node.Version,
		Capabilities: node.Capabilities,
		Services:     make(map[string]string),
	}
	for k, v := range node.Services {
		info.Services[k] = v
	}
	return sm.SignNodeInfo(info)
}

// NewSession creates a new secure session ID.
func NewSession() string {
	b := make([]byte, 16)
	rand.Read(b)
	return sha256Hex(b[:8])
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return string(encodeHex(h[:8]))
}

func truncateID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

const hexDigits = "0123456789abcdef"

func encodeHex(src []byte) []byte {
	dst := make([]byte, len(src)*2)
	for i, v := range src {
		dst[i*2] = hexDigits[v>>4]
		dst[i*2+1] = hexDigits[v&0x0f]
	}
	return dst
}
