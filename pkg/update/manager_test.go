package update

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockP2PNetwork is a mock implementation of the P2PNetworkInterface
type MockP2PNetwork struct {
	mock.Mock
}

func (m *MockP2PNetwork) FetchUpdateFromPeer(ctx context.Context, peerID peer.ID, version string) (*ReleaseInfo, []byte, error) {
	args := m.Called(ctx, peerID, version)
	return args.Get(0).(*ReleaseInfo), args.Get(1).([]byte), args.Error(2)
}

func (m *MockP2PNetwork) GetConnectedPeers() []peer.ID {
	args := m.Called()
	return args.Get(0).([]peer.ID)
}

func TestNewManager(t *testing.T) {
	logger := slog.Default()
	cfg := Config{
		ServerURL:     "http://example.com/update",
		CheckInterval: 1 * time.Hour,
		PublicKey:     "0000000000000000000000000000000000000000000000000000000000000000",
		EnableP2P:     true,
	}
	version := "1.0.0"

	manager, err := NewManager(logger, cfg, version)
	assert.NoError(t, err)
	assert.NotNil(t, manager)
	assert.Equal(t, version, manager.CurrentVersion())
}

func TestSetP2PNetwork(t *testing.T) {
	logger := slog.Default()
	cfg := Config{
		ServerURL:     "http://example.com/update",
		CheckInterval: 1 * time.Hour,
		PublicKey:     "0000000000000000000000000000000000000000000000000000000000000000",
		EnableP2P:     true,
	}
	version := "1.0.0"

	manager, err := NewManager(logger, cfg, version)
	assert.NoError(t, err)

	mockP2P := new(MockP2PNetwork)
	manager.SetP2PNetwork(mockP2P)

	// The field is not exported, so we can't directly check it
	// But we can test that the methods work correctly
}

func TestGetCurrentRelease(t *testing.T) {
	logger := slog.Default()
	cfg := Config{
		ServerURL:     "http://example.com/update",
		CheckInterval: 1 * time.Hour,
		PublicKey:     "0000000000000000000000000000000000000000000000000000000000000000",
		EnableP2P:     true,
	}
	version := "1.0.0"

	manager, err := NewManager(logger, cfg, version)
	assert.NoError(t, err)

	release, data, err := manager.GetCurrentRelease()
	assert.NoError(t, err)
	assert.NotNil(t, release)
	assert.Equal(t, version, release.Version)
	assert.Empty(t, data)
}

func TestVerifyArtifact(t *testing.T) {
	// Generate a key pair for testing
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	assert.NoError(t, err)

	// Encode the public key as hex
	pubKeyHex := fmt.Sprintf("%x", pubKey)

	logger := slog.Default()
	cfg := Config{
		ServerURL:     "http://example.com/update",
		CheckInterval: 1 * time.Hour,
		PublicKey:     pubKeyHex,
		EnableP2P:     true,
	}
	version := "1.0.0"

	manager, err := NewManager(logger, cfg, version)
	assert.NoError(t, err)

	// Create test data
	testData := []byte("test binary data")
	hash := sha256.Sum256(testData)
	signature := ed25519.Sign(privKey, hash[:])

	// Encode the signature as hex
	signatureHex := fmt.Sprintf("%x", signature)

	artifact := ArtifactInfo{
		URL:       "http://example.com/binary",
		Signature: signatureHex,
	}

	// Test valid signature
	err = manager.verifyArtifact(artifact, testData)
	assert.NoError(t, err)

	// Test invalid signature
	invalidArtifact := ArtifactInfo{
		URL:       "http://example.com/binary",
		Signature: "invalidsignature",
	}
	err = manager.verifyArtifact(invalidArtifact, testData)
	assert.Error(t, err)
}

func TestCheckForP2PUpdate(t *testing.T) {
	logger := slog.Default()
	cfg := Config{
		ServerURL:     "http://example.com/update",
		CheckInterval: 1 * time.Hour,
		PublicKey:     "0000000000000000000000000000000000000000000000000000000000000000",
		EnableP2P:     true,
	}
	version := "1.0.0"

	manager, err := NewManager(logger, cfg, version)
	assert.NoError(t, err)

	mockP2P := new(MockP2PNetwork)
	manager.SetP2PNetwork(mockP2P)

	// Test when no peers are connected
	mockP2P.On("GetConnectedPeers").Return([]peer.ID{})

	ctx := context.Background()
	release, data, err := manager.checkForP2PUpdate(ctx)
	assert.Error(t, err)
	assert.Nil(t, release)
	assert.Nil(t, data)
	mockP2P.AssertExpectations(t)
}