package networking

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdvancedDiscoveryIntegration(t *testing.T) {
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

	// Test that all components are properly initialized
	assert.NotNil(t, ad)
	assert.NotNil(t, ad.relayProxyMgr)
	assert.NotNil(t, ad.bluetoothDisc)
	assert.NotNil(t, ad.reputationRouting)

	// Test marking peers as connected/disconnected
	peerID := peer.ID("test-peer")

	// Mark peer as connected
	ad.MarkPeerConnected(peerID)

	// Get connected peers and verify
	connectedPeers := ad.GetConnectedPeers()
	assert.Len(t, connectedPeers, 1)
	assert.Equal(t, peerID, connectedPeers[0])

	// Mark peer as disconnected
	ad.MarkPeerDisconnected(peerID)

	// Get connected peers and verify
	connectedPeers = ad.GetConnectedPeers()
	assert.Len(t, connectedPeers, 0)

	// Test getting connected peers
	ad.MarkPeerConnected(peer.ID("peer-1"))
	ad.MarkPeerConnected(peer.ID("peer-2"))
	connectedPeers = ad.GetConnectedPeers()
	assert.Len(t, connectedPeers, 2)

	// Test reputation routing
	peer1 := peer.ID("peer-1")
	peer2 := peer.ID("peer-2")

	// Record interactions for both peers
	for i := 0; i < 5; i++ {
		ad.reputationRouting.reputation.RecordInteraction(peer1, ModuleTransfer, Success)
		ad.reputationRouting.reputation.RecordInteraction(peer2, ModuleTransfer, Success)
	}

	// Get best peers (should include both peers)
	bestPeers := ad.reputationRouting.GetBestPeers(2)
	assert.Len(t, bestPeers, 2)

	// Get reputation score for peer2
	score := ad.reputationRouting.reputation.GetReputationScore(peer2)
	assert.Greater(t, score, 50.0) // Should be higher than initial 50.0

	// Test network stats
	ad.reputationRouting.networkStats.UpdateNetworkStats(peer1, 50*time.Millisecond, 50.0)

	// Test secure connection through port
	// err = ad.EstablishSecureConnectionThroughPort(ctx, "/ip4/127.0.0.1/tcp/8080", 8080)
	// assert.NoError(t, err)

	// Test relay connection
	relayPeers, err := ad.relayProxyMgr.FindRelayPeers(ctx)
	assert.NoError(t, err)
	assert.Len(t, relayPeers, 0)

	// Test Bluetooth discovery
	btPeers, err := ad.bluetoothDisc.DiscoverNearbyPeers(ctx)
	assert.NoError(t, err)
	assert.Len(t, btPeers, 2) // Should return mock Bluetooth peers
}


