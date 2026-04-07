package networking

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
)

// TransportOption represents an application-layer transport candidate.
type TransportOption struct {
	Name      string
	Available bool
}

// EncryptedMessenger encapsulates payloads inside common protocols with end-to-end encryption.
type EncryptedMessenger struct {
	key []byte
}

// NewEncryptedMessenger builds a messenger with a 32-byte key.
func NewEncryptedMessenger(key []byte) (*EncryptedMessenger, error) {
	if len(key) != 32 {
		return nil, errors.New("key must be 32 bytes")
	}
	return &EncryptedMessenger{key: append([]byte(nil), key...)}, nil
}

// Seal encrypts payload and returns nonce+ciphertext.
func (m *EncryptedMessenger) Seal(plain []byte) (nonce, ct []byte, err error) {
	block, err := aes.NewCipher(m.key)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce = make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, err
	}
	ct = gcm.Seal(nil, nonce, plain, nil)
	return nonce, ct, nil
}

// Open decrypts payload using the provided nonce.
func (m *EncryptedMessenger) Open(nonce, ct []byte) ([]byte, error) {
	block, err := aes.NewCipher(m.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce, ct, nil)
}

// SelectTransport chooses the first available transport from the preference list.
func SelectTransport(candidates []TransportOption) (TransportOption, error) {
	for _, c := range candidates {
		if c.Available {
			return c, nil
		}
	}
	return TransportOption{}, fmt.Errorf("no transport available")
}
