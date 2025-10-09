package agent

import (
	"fmt"
	"os"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Identity represents the unique cryptographic identity of the agent.
type Identity struct {
	PeerID  peer.ID
	PrivKey crypto.PrivKey
}

// NewIdentity creates a new agent identity, loading from a file if it exists,
// or generating a new one and saving it for future use.
func NewIdentity() (*Identity, error) {
	const keyFilePath = "agent_identity.key"

	// Attempt to load the private key from the file.
	privKey, err := loadIdentity(keyFilePath)
	if err == nil {
		// Key loaded successfully.
		peerID, err := peer.IDFromPrivateKey(privKey)
		if err != nil {
			return nil, fmt.Errorf("failed to derive peer ID from loaded private key: %w", err)
		}
		return &Identity{PeerID: peerID, PrivKey: privKey}, nil
	}

	// If the key file doesn't exist, generate a new key.
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load identity: %w", err)
	}

	// Generate a new Ed25519 private key.
	newPrivKey, _, err := crypto.GenerateEd25519Key(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Save the new private key to the file.
	if err := saveIdentity(keyFilePath, newPrivKey); err != nil {
		return nil, fmt.Errorf("failed to save new identity: %w", err)
	}

	// Derive the PeerID from the public key.
	peerID, err := peer.IDFromPrivateKey(newPrivKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive peer ID from new private key: %w", err)
	}

	return &Identity{
		PeerID:  peerID,
		PrivKey: newPrivKey,
	}, nil
}

// loadIdentity reads a private key from a file.
func loadIdentity(path string) (crypto.PrivKey, error) {
	keyBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return crypto.UnmarshalEd25519PrivateKey(keyBytes)
}

// saveIdentity writes a private key to a file.
func saveIdentity(path string, key crypto.PrivKey) error {
	keyBytes, err := crypto.MarshalPrivateKey(key)
	if err != nil {
		return err
	}
	return os.WriteFile(path, keyBytes, 0600)
}
