package networking

import (
	"context"
	"log/slog"
	"testing"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
)

func TestNewRelayProxyManager(t *testing.T) {
	logger := slog.Default()
	h, err := libp2p.New()
	assert.NoError(t, err)

	rpm := NewRelayProxyManager(logger, h)

	assert.NotNil(t, rpm)
	assert.Equal(t, h, rpm.host)
}

func TestEstablishRelayConnection(t *testing.T) {
	logger := slog.Default()
	h, err := libp2p.New()
	assert.NoError(t, err)

	rpm := NewRelayProxyManager(logger, h)

	targetPeer := peer.ID("target-peer")
	relayPeer := peer.ID("relay-peer")

	err = rpm.EstablishRelayConnection(context.Background(), targetPeer, relayPeer)

	assert.Error(t, err)
}

func TestEstablishProxyConnection(t *testing.T) {
	logger := slog.Default()
	h, err := libp2p.New()
	assert.NoError(t, err)

	rpm := NewRelayProxyManager(logger, h)

	err = rpm.EstablishProxyConnection(context.Background(), "127.0.0.1:8080", "127.0.0.1:1080")

	assert.Error(t, err)
}

func TestEstablishHTTPProxyConnection(t *testing.T) {
	logger := slog.Default()
	h, err := libp2p.New()
	assert.NoError(t, err)

	rpm := NewRelayProxyManager(logger, h)

	err = rpm.EstablishHTTPProxyConnection(context.Background(), "127.0.0.1:8080", "127.0.0.1:1080")

	assert.Error(t, err)
}

func TestFindRelayPeers(t *testing.T) {
	logger := slog.Default()
	h, err := libp2p.New()
	assert.NoError(t, err)

	rpm := NewRelayProxyManager(logger, h)

	peers, err := rpm.FindRelayPeers(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, peers)
	assert.Len(t, peers, 0)
}

func TestFindProxyServers(t *testing.T) {
	logger := slog.Default()
	h, err := libp2p.New()
	assert.NoError(t, err)

	rpm := NewRelayProxyManager(logger, h)

	servers, err := rpm.FindProxyServers(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, servers)
	assert.Len(t, servers, 0)
}
