package agent

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
)

// TrustedChannel verifies payloads delivered via signed ecosystems (updates, dependencies).
type TrustedChannel struct {
	sharedKey []byte
}

func NewTrustedChannel(sharedKey []byte) *TrustedChannel {
	return &TrustedChannel{sharedKey: append([]byte(nil), sharedKey...)}
}

// Sign produces an HMAC for the payload metadata.
func (t *TrustedChannel) Sign(payload []byte) []byte {
	mac := hmac.New(sha256.New, t.sharedKey)
	mac.Write(payload)
	return mac.Sum(nil)
}

// Verify checks the provided signature matches.
func (t *TrustedChannel) Verify(payload, sig []byte) error {
	mac := hmac.New(sha256.New, t.sharedKey)
	mac.Write(payload)
	expected := mac.Sum(nil)
	if !hmac.Equal(expected, sig) {
		return fmt.Errorf("signature mismatch")
	}
	return nil
}
