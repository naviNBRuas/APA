package networking

import (
	"context"
	"log/slog"
	"testing"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAdvancedDiscovery(t *testing.T) {
	// Create a libp2p host for testing
	ctx := context.Background()
	host, err := libp2p.New()
	require.NoError(t, err)
	defer host.Close()

	// Create a DHT for testing
	dhtInstance, err := dht.New(ctx, host)
	require.NoError(t, err)

	// Create AdvancedDiscovery instance
	logger := slog.Default()
	ad := NewAdvancedDiscovery(logger, host, dhtInstance, "test-service")

	// Verify the instance was created correctly
	assert.NotNil(t, ad)
	assert.Equal(t, host, ad.host)
	assert.Equal(t, dhtInstance, ad.dht)
	assert.Equal(t, "test-service", ad.serviceTag)
}

func TestMarkPeerConnected(t *testing.T) {
	// Create a libp2p host for testing
	ctx := context.Background()
	host, err := libp2p.New()
	require.NoError(t, err)
	defer host.Close()

	// Create a DHT for testing
	dhtInstance, err := dht.New(ctx, host)
	require.NoError(t, err)

	// Create AdvancedDiscovery instance
	logger := slog.Default()
	ad := NewAdvancedDiscovery(logger, host, dhtInstance, "test-service")

	// Generate a test peer ID
	peerID := peer.ID("test-peer-id")

	// Mark peer as connected
	ad.MarkPeerConnected(peerID)

	// Get connected peers and verify
	connectedPeers := ad.GetConnectedPeers()
	assert.Len(t, connectedPeers, 1)
	assert.Equal(t, peerID, connectedPeers[0])
}

func TestMarkPeerDisconnected(t *testing.T) {
	// Create a libp2p host for testing
	ctx := context.Background()
	host, err := libp2p.New()
	require.NoError(t, err)
	defer host.Close()

	// Create a DHT for testing
	dhtInstance, err := dht.New(ctx, host)
	require.NoError(t, err)

	// Create AdvancedDiscovery instance
	logger := slog.Default()
	ad := NewAdvancedDiscovery(logger, host, dhtInstance, "test-service")

	// Generate a test peer ID
	peerID := peer.ID("test-peer-id")

	// Mark peer as connected first
	ad.MarkPeerConnected(peerID)

	// Mark peer as disconnected
	ad.MarkPeerDisconnected(peerID)

	// Get connected peers and verify
	connectedPeers := ad.GetConnectedPeers()
	assert.Len(t, connectedPeers, 0)
}

func TestReputationRouting(t *testing.T) {
	// Create a libp2p host for testing
	ctx := context.Background()
	host, err := libp2p.New()
	require.NoError(t, err)
	defer host.Close()

	// Create a DHT for testing
	dhtInstance, err := dht.New(ctx, host)
	require.NoError(t, err)

	// Create AdvancedDiscovery instance
	logger := slog.Default()
	ad := NewAdvancedDiscovery(logger, host, dhtInstance, "test-service")

	// Test recording interactions
	peerID := peer.ID("test-peer-id")
	ad.reputationRouting.reputation.RecordInteraction(peerID, ModuleTransfer, Success)

	// Test getting reputation score
	score := ad.reputationRouting.reputation.GetReputationScore(peerID)
	assert.Greater(t, score, 50.0) // Should be higher than initial 50.0

	// Test selecting optimal peer with no excluded peers
	// Since we have one peer with interactions, it should be selected
	optimalPeer := ad.reputationRouting.SelectOptimalPeer(ModuleTransfer, []peer.ID{})
	// Should return the peer with the highest reputation
	assert.Equal(t, peerID, optimalPeer)
}

func TestRelayProxyManager(t *testing.T) {
	// Create a libp2p host for testing
	ctx := context.Background()
	host, err := libp2p.New()
	require.NoError(t, err)
	defer host.Close()

	// Create a DHT for testing
	dhtInstance, err := dht.New(ctx, host)
	require.NoError(t, err)

	// Create AdvancedDiscovery instance
	logger := slog.Default()
	ad := NewAdvancedDiscovery(logger, host, dhtInstance, "test-service")

	// Test establishing relay connection
	peerID := peer.ID("test-peer-id")
	relayPeer := peer.ID("relay-peer-id")
	err = ad.relayProxyMgr.EstablishRelayConnection(ctx, peerID, relayPeer)
	assert.NoError(t, err)

	// Test establishing proxy connection
	err = ad.relayProxyMgr.EstablishProxyConnection(ctx, "/ip4/127.0.0.1/tcp/8080", "/ip4/127.0.0.1/tcp/9090")
	assert.NoError(t, err)

	// Test finding relay peers
	relayPeers, err := ad.relayProxyMgr.FindRelayPeers(ctx)
	assert.NoError(t, err)
	assert.Len(t, relayPeers, 3) // Should return mock relay peers
}

func TestBluetoothDiscovery(t *testing.T) {
	// Create a libp2p host for testing
	ctx := context.Background()
	host, err := libp2p.New()
	require.NoError(t, err)
	defer host.Close()

	// Create a DHT for testing
	dhtInstance, err := dht.New(ctx, host)
	require.NoError(t, err)

	// Create AdvancedDiscovery instance
	logger := slog.Default()
	ad := NewAdvancedDiscovery(logger, host, dhtInstance, "test-service")

	// Test discovering nearby peers
	peers, err := ad.bluetoothDisc.DiscoverNearbyPeers(ctx)
	assert.NoError(t, err)
	assert.Len(t, peers, 2) // Should return mock Bluetooth peers
}