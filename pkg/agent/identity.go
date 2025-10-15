package agent

import (
	"encoding/hex"
	"encoding/json"
	"os"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Identity holds the agent's cryptographic identity.
type Identity struct {
	PeerID  peer.ID
	PrivKey crypto.PrivKey
}

// identityState is used to persist the private key to disk.
type identityState struct {
	PrivateKey string `json:"private_key"`
}

// NewIdentity creates a new identity for the agent, loading it from disk if it
// exists or creating a new one otherwise.
func NewIdentity(identityPath string) (*Identity, error) {


	if _, err := os.Stat(identityPath); os.IsNotExist(err) {
		return createAndSaveIdentity(identityPath)
	}

	return loadIdentity(identityPath)
}

// createAndSaveIdentity generates a new Ed25519 key pair and saves it to the specified path.
func createAndSaveIdentity(path string) (*Identity, error) {
	privKey, _, err := crypto.GenerateEd25519Key(nil)
	if err != nil {
		return nil, err
	}

	peerID, err := peer.IDFromPrivateKey(privKey)
	if err != nil {
		return nil, err
	}

	privBytes, err := crypto.MarshalPrivateKey(privKey)
	if err != nil {
		return nil, err
	}

	state := &identityState{
		PrivateKey: hex.EncodeToString(privBytes),
	}

	data, err := json.Marshal(state)
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return nil, err
	}

	return &Identity{
		PeerID:  peerID,
		PrivKey: privKey,
	}, nil
}

// loadIdentity loads an identity from the specified path.
func loadIdentity(path string) (*Identity, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var state identityState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	privBytes, err := hex.DecodeString(state.PrivateKey)
	if err != nil {
		return nil, err
	}

	privKey, err := crypto.UnmarshalPrivateKey(privBytes)
	if err != nil {
		return nil, err
	}

	peerID, err := peer.IDFromPrivateKey(privKey)
	if err != nil {
		return nil, err
	}

	return &Identity{
		PeerID:  peerID,
		PrivKey: privKey,
	}, nil
}