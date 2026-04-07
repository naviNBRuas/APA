package agent

import (
	"sync"
	"time"
)

// Credential holds a token with expiry for access retention.
type Credential struct {
	Token     string
	ExpiresAt time.Time
}

// CredentialVault stores and reuses long-lived credentials.
type CredentialVault struct {
	mu    sync.Mutex
	store map[string]Credential
}

// NewCredentialVault creates an empty vault.
func NewCredentialVault() *CredentialVault {
	return &CredentialVault{store: make(map[string]Credential)}
}

// Put stores a credential under a key.
func (v *CredentialVault) Put(key, token string, ttl time.Duration) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.store[key] = Credential{Token: token, ExpiresAt: time.Now().Add(ttl)}
}

// Get retrieves a credential if valid.
func (v *CredentialVault) Get(key string) (Credential, bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	cred, ok := v.store[key]
	if !ok {
		return Credential{}, false
	}
	if time.Now().After(cred.ExpiresAt) {
		delete(v.store, key)
		return Credential{}, false
	}
	return cred, true
}
