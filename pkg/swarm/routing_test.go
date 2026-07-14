package swarm

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func routingTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestNewNetworkMonitor(t *testing.T) {
	nm := NewNetworkMonitor(routingTestLogger())
	require.NotNil(t, nm)
}

func TestNetworkMonitor_UpdateGetStats(t *testing.T) {
	nm := NewNetworkMonitor(routingTestLogger())
	pid := peer.ID("test-peer")

	nm.UpdateNetworkStats(pid, 50*time.Millisecond, 100.0, 1.5)
	stats := nm.GetNetworkStats(pid)
	require.NotNil(t, stats)
	assert.Equal(t, 50*time.Millisecond, stats.Latency)
	assert.Equal(t, 100.0, stats.Bandwidth)
	assert.Equal(t, 1.5, stats.PacketLoss)
}

func TestNetworkMonitor_GetStats_UnknownPeer(t *testing.T) {
	nm := NewNetworkMonitor(routingTestLogger())
	pid := peer.ID("unknown")

	stats := nm.GetNetworkStats(pid)
	require.NotNil(t, stats)
	assert.Equal(t, 100*time.Millisecond, stats.Latency)
	assert.Equal(t, 10.0, stats.Bandwidth)
	assert.Equal(t, 0.0, stats.PacketLoss)
}

func TestRoutingManager_New(t *testing.T) {
	logger := routingTestLogger()
	rep := NewReputationSystem(logger)
	rm := NewRoutingManager(logger, rep)
	require.NotNil(t, rm)
	require.NotNil(t, rm.NetworkMonitor())
}

func TestRoutingManager_AddRoute_New(t *testing.T) {
	logger := routingTestLogger()
	rep := NewReputationSystem(logger)
	rm := NewRoutingManager(logger, rep)
	pid := peer.ID("peer-1")

	rm.AddRoute("res-1", Route{
		PeerID:      pid,
		Resource:    "res-1",
		Cost:        10.0,
		Latency:     50 * time.Millisecond,
		Bandwidth:   100.0,
		SuccessRate: 0.95,
	})

	best := rm.GetBestRoute(nil, "res-1")
	require.NotNil(t, best)
	assert.Equal(t, pid, best.PeerID)
}

func TestRoutingManager_AddRoute_UpdateExisting(t *testing.T) {
	logger := routingTestLogger()
	rep := NewReputationSystem(logger)
	rm := NewRoutingManager(logger, rep)
	pid := peer.ID("peer-1")

	rm.AddRoute("res-1", Route{PeerID: pid, Resource: "res-1", Cost: 10.0, SuccessRate: 0.9})
	rm.AddRoute("res-1", Route{PeerID: pid, Resource: "res-1", Cost: 5.0, SuccessRate: 0.99})

	best := rm.GetBestRoute(nil, "res-1")
	require.NotNil(t, best)
	assert.Equal(t, 5.0, best.Cost)
}

func TestRoutingManager_RemoveRoute(t *testing.T) {
	logger := routingTestLogger()
	rep := NewReputationSystem(logger)
	rm := NewRoutingManager(logger, rep)
	pid := peer.ID("peer-1")

	rm.AddRoute("res-1", Route{PeerID: pid, Resource: "res-1", Cost: 10.0})
	rm.RemoveRoute("res-1", pid)

	best := rm.GetBestRoute(nil, "res-1")
	assert.Nil(t, best)
}

func TestRoutingManager_RemoveRoute_NotFound(t *testing.T) {
	logger := routingTestLogger()
	rep := NewReputationSystem(logger)
	rm := NewRoutingManager(logger, rep)

	rm.RemoveRoute("nonexistent", peer.ID("nobody"))
}

func TestRoutingManager_GetBestRoute_NoRoutes(t *testing.T) {
	logger := routingTestLogger()
	rep := NewReputationSystem(logger)
	rm := NewRoutingManager(logger, rep)

	best := rm.GetBestRoute(nil, "unknown")
	assert.Nil(t, best)
}

func TestRoutingManager_GetBestRoute_PicksHighestScore(t *testing.T) {
	logger := routingTestLogger()
	rep := NewReputationSystem(logger)
	rm := NewRoutingManager(logger, rep)

	pa := peer.ID("peer-a")
	pb := peer.ID("peer-b")

	rm.AddRoute("res-1", Route{PeerID: pa, Resource: "res-1", Cost: 50.0, SuccessRate: 0.5})
	rm.AddRoute("res-1", Route{PeerID: pb, Resource: "res-1", Cost: 10.0, SuccessRate: 0.99})

	best := rm.GetBestRoute(nil, "res-1")
	require.NotNil(t, best)
	assert.Equal(t, pb, best.PeerID)
}

func TestRoutingManager_GetMultipleRoutes(t *testing.T) {
	logger := routingTestLogger()
	rep := NewReputationSystem(logger)
	rm := NewRoutingManager(logger, rep)

	rm.AddRoute("res-1", Route{PeerID: peer.ID("p1"), Resource: "res-1", Cost: 30, SuccessRate: 0.7, Latency: 100 * time.Millisecond, Bandwidth: 50})
	rm.AddRoute("res-1", Route{PeerID: peer.ID("p2"), Resource: "res-1", Cost: 10, SuccessRate: 0.99, Latency: 20 * time.Millisecond, Bandwidth: 200})
	rm.AddRoute("res-1", Route{PeerID: peer.ID("p3"), Resource: "res-1", Cost: 20, SuccessRate: 0.85, Latency: 50 * time.Millisecond, Bandwidth: 100})

	routes := rm.GetMultipleRoutes(nil, "res-1", 2)
	require.Len(t, routes, 2)
	assert.Equal(t, peer.ID("p2"), routes[0].PeerID)
}

func TestRoutingManager_GetMultipleRoutes_NoRoutes(t *testing.T) {
	logger := routingTestLogger()
	rep := NewReputationSystem(logger)
	rm := NewRoutingManager(logger, rep)

	routes := rm.GetMultipleRoutes(nil, "unknown", 5)
	assert.Nil(t, routes)
}

func TestRoutingManager_GetMultipleRoutes_LessThanCount(t *testing.T) {
	logger := routingTestLogger()
	rep := NewReputationSystem(logger)
	rm := NewRoutingManager(logger, rep)

	rm.AddRoute("res-1", Route{PeerID: peer.ID("p1"), Resource: "res-1", Cost: 10, SuccessRate: 0.9})

	routes := rm.GetMultipleRoutes(nil, "res-1", 10)
	require.Len(t, routes, 1)
}

func TestRoutingManager_GetRandomRoute(t *testing.T) {
	logger := routingTestLogger()
	rep := NewReputationSystem(logger)
	rm := NewRoutingManager(logger, rep)

	rm.AddRoute("res-1", Route{PeerID: peer.ID("p1"), Resource: "res-1", Cost: 10, SuccessRate: 0.9})
	rm.AddRoute("res-1", Route{PeerID: peer.ID("p2"), Resource: "res-1", Cost: 20, SuccessRate: 0.8})

	route := rm.GetRandomRoute(nil, "res-1")
	require.NotNil(t, route)
	assert.Equal(t, "res-1", route.Resource)
}

func TestRoutingManager_GetRandomRoute_NoRoutes(t *testing.T) {
	logger := routingTestLogger()
	rep := NewReputationSystem(logger)
	rm := NewRoutingManager(logger, rep)

	route := rm.GetRandomRoute(nil, "unknown")
	assert.Nil(t, route)
}

func TestRoutingManager_GetTrustedRoutes(t *testing.T) {
	logger := routingTestLogger()
	rep := NewReputationSystem(logger)
	rm := NewRoutingManager(logger, rep)

	trusted := peer.ID("trusted-peer")
	untrusted := peer.ID("untrusted-peer")

	rep.RecordInteraction(string(trusted), ModuleTransfer, Success)
	rep.RecordInteraction(string(trusted), ModuleTransfer, Success)

	rm.AddRoute("res-1", Route{PeerID: trusted, Resource: "res-1", Cost: 10, SuccessRate: 0.9})
	rm.AddRoute("res-1", Route{PeerID: untrusted, Resource: "res-1", Cost: 10, SuccessRate: 0.9})

	routes := rm.GetTrustedRoutes(nil, "res-1", 55.0)
	require.Len(t, routes, 1)
	assert.Equal(t, trusted, routes[0].PeerID)
}

func TestRoutingManager_GetTrustedRoutes_NoRoutes(t *testing.T) {
	logger := routingTestLogger()
	rep := NewReputationSystem(logger)
	rm := NewRoutingManager(logger, rep)

	routes := rm.GetTrustedRoutes(nil, "unknown", 60.0)
	assert.Nil(t, routes)
}

func TestRoutingManager_GetTrustedRoutes_EmptyRoutingTable(t *testing.T) {
	logger := routingTestLogger()
	rep := NewReputationSystem(logger)
	rm := NewRoutingManager(logger, rep)

	rm.AddRoute("res-1", Route{PeerID: peer.ID("p1"), Resource: "res-1", Cost: 10, SuccessRate: 0.9})

	routes := rm.GetTrustedRoutes(nil, "unknown", 60.0)
	assert.Nil(t, routes)
}

func TestNetworkMonitor_GetStats_ReturnsCopy(t *testing.T) {
	nm := NewNetworkMonitor(routingTestLogger())
	pid := peer.ID("test-peer")

	nm.UpdateNetworkStats(pid, 50*time.Millisecond, 100.0, 1.5)
	stats := nm.GetNetworkStats(pid)
	stats.Latency = 999 * time.Hour

	stats2 := nm.GetNetworkStats(pid)
	assert.Equal(t, 50*time.Millisecond, stats2.Latency)
}
