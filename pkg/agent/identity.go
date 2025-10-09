package agent

import (
	"fmt"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Identity represents the unique cryptographic identity of the agent.
type Identity struct {
	PeerID  peer.ID
	PrivKey crypto.PrivKey
}

// NewIdentity creates a new agent identity.
// This implementation generates a new Ed25519 key pair for each run.
// A real-world agent would load/save this key from a secure, persistent store.
func NewIdentity() (*Identity, error) {
	// Generate a new Ed25519 private key.
	privKey, _, err := crypto.GenerateEd25519Key(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Derive the PeerID from the public key.
	peerID, err := peer.IDFromPrivateKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive peer ID from private key: %w", err)
	}

	return &Identity{
		PeerID:  peerID,
		PrivKey: privKey,
	}, nil
}
