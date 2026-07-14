package swarm

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestEdgePrefersRelayAndBackbone(t *testing.T) {
	logger := newTestLogger()
	rep := NewReputationSystem(logger)
	routing := NewRoutingManager(logger, rep)
	tm := NewTopologyManager(logger, rep, routing)
	tm.SetPolicy(TierPolicy{TargetEdgeDegree: 2, RegionBias: 0.3, QualityWeight: 0.5, ReputationWeight: 0.2})

	local := peer.ID("edge-local")
	tm.UpdatePeerConnection(local, true, []string{"us-east"}, nil)
	tm.SetPeerTier(local, NodeTierEdge)

	relay1 := peer.ID("relay-1")
	tm.UpdatePeerConnection(relay1, false, []string{"us-east"}, nil)
	tm.SetPeerTier(relay1, NodeTierRelay)
	tm.UpdateEdge(local, relay1, 10*time.Millisecond, 200, 0.9)

	relay2 := peer.ID("relay-2")
	tm.UpdatePeerConnection(relay2, false, []string{"us-west"}, nil)
	tm.SetPeerTier(relay2, NodeTierRelay)
	tm.UpdateEdge(local, relay2, 20*time.Millisecond, 150, 0.8)

	backbone := peer.ID("backbone-1")
	tm.UpdatePeerConnection(backbone, false, []string{"us-east"}, nil)
	tm.SetPeerTier(backbone, NodeTierBackbone)
	tm.UpdateEdge(local, backbone, 30*time.Millisecond, 300, 0.85)

	edgeNeighbor := peer.ID("edge-neighbor")
	tm.UpdatePeerConnection(edgeNeighbor, false, []string{"us-east"}, nil)
	tm.SetPeerTier(edgeNeighbor, NodeTierEdge)
	tm.UpdateEdge(local, edgeNeighbor, 5*time.Millisecond, 100, 0.95)

	connect, disconnect := tm.SuggestTierAdjustments(local, NodeTierEdge)
	require.Empty(t, disconnect, "expected no disconnects for under-connected node")
	require.Len(t, connect, 2, "expected 2 suggested connections")

	for _, peerID := range connect {
		tier := tm.topology.Nodes[peerID].Tier
		assert.NotEqualf(t, NodeTierEdge, tier, "edge node should not prefer another edge; got suggestion %s", peerID)
	}
}

func TestOverConnectedRelaySuggestsLowestScoreDisconnect(t *testing.T) {
	logger := newTestLogger()
	rep := NewReputationSystem(logger)
	routing := NewRoutingManager(logger, rep)
	tm := NewTopologyManager(logger, rep, routing)
	tm.SetPolicy(TierPolicy{TargetRelayDegree: 2, RegionBias: 0.2, QualityWeight: 0.3, ReputationWeight: 0.5})

	local := peer.ID("relay-local")
	tm.UpdatePeerConnection(local, true, []string{"eu"}, nil)
	tm.SetPeerTier(local, NodeTierRelay)

	edgeStrong := peer.ID("edge-strong")
	tm.UpdatePeerConnection(edgeStrong, true, []string{"eu"}, nil)
	tm.SetPeerTier(edgeStrong, NodeTierEdge)
	tm.UpdateEdge(local, edgeStrong, 15*time.Millisecond, 150, 0.9)
	rep.RecordInteraction(string(edgeStrong), ModuleTransfer, Success)
	rep.RecordInteraction(string(edgeStrong), ModuleTransfer, Success)

	edgeWeak := peer.ID("edge-weak")
	tm.UpdatePeerConnection(edgeWeak, true, []string{"eu"}, nil)
	tm.SetPeerTier(edgeWeak, NodeTierEdge)
	tm.UpdateEdge(local, edgeWeak, 40*time.Millisecond, 50, 0.4)
	rep.RecordInteraction(string(edgeWeak), ModuleTransfer, Failure)

	backbone := peer.ID("backbone")
	tm.UpdatePeerConnection(backbone, true, []string{"us"}, nil)
	tm.SetPeerTier(backbone, NodeTierBackbone)
	tm.UpdateEdge(local, backbone, 25*time.Millisecond, 200, 0.85)

	connect, disconnect := tm.SuggestTierAdjustments(local, NodeTierRelay)
	require.Empty(t, connect, "expected no connect suggestions when over-connected")
	require.Len(t, disconnect, 1, "expected 1 disconnect suggestion")
	assert.Equal(t, edgeWeak, disconnect[0], "expected weakest connection to be suggested for removal")
}

func TestTopologyManager_RegionFor_Known(t *testing.T) {
	logger := newTestLogger()
	rep := NewReputationSystem(logger)
	routing := NewRoutingManager(logger, rep)
	tm := NewTopologyManager(logger, rep, routing)
	pid := peer.ID("peer-1")

	tm.UpdatePeerConnection(pid, false, []string{"us-east"}, nil)
	assert.Equal(t, "us-east", tm.RegionFor(pid))
}

func TestTopologyManager_RegionFor_Unknown(t *testing.T) {
	logger := newTestLogger()
	rep := NewReputationSystem(logger)
	routing := NewRoutingManager(logger, rep)
	tm := NewTopologyManager(logger, rep, routing)

	assert.Equal(t, "", tm.RegionFor(peer.ID("nobody")))
}

func TestTopologyManager_GetPeerConnection_Known(t *testing.T) {
	logger := newTestLogger()
	rep := NewReputationSystem(logger)
	routing := NewRoutingManager(logger, rep)
	tm := NewTopologyManager(logger, rep, routing)
	pid := peer.ID("peer-1")

	tm.UpdatePeerConnection(pid, true, []string{"us-east"}, []string{"storage"})

	conn := tm.GetPeerConnection(pid)
	require.NotNil(t, conn)
	assert.True(t, conn.Connected)
	assert.Equal(t, []string{"us-east"}, conn.Regions)
	assert.Equal(t, []string{"storage"}, conn.Capabilities)
	assert.Equal(t, 1, conn.Connections)
}

func TestTopologyManager_GetPeerConnection_Unknown(t *testing.T) {
	logger := newTestLogger()
	rep := NewReputationSystem(logger)
	routing := NewRoutingManager(logger, rep)
	tm := NewTopologyManager(logger, rep, routing)

	assert.Nil(t, tm.GetPeerConnection(peer.ID("nobody")))
}

func TestTopologyManager_GetPeerConnection_ReturnsCopy(t *testing.T) {
	logger := newTestLogger()
	rep := NewReputationSystem(logger)
	routing := NewRoutingManager(logger, rep)
	tm := NewTopologyManager(logger, rep, routing)
	pid := peer.ID("peer-1")

	tm.UpdatePeerConnection(pid, true, []string{"us-east"}, []string{"storage"})

	conn := tm.GetPeerConnection(pid)
	conn.Connected = false
	conn.Regions[0] = "hacked"

	conn2 := tm.GetPeerConnection(pid)
	assert.True(t, conn2.Connected)
	assert.Equal(t, "us-east", conn2.Regions[0])
}

func TestTopologyManager_GetTopology_ReturnsCopy(t *testing.T) {
	logger := newTestLogger()
	rep := NewReputationSystem(logger)
	routing := NewRoutingManager(logger, rep)
	tm := NewTopologyManager(logger, rep, routing)
	pid := peer.ID("peer-1")

	tm.UpdatePeerConnection(pid, false, nil, nil)
	tm.UpdatePeerConnection(peer.ID("peer-2"), false, nil, nil)
	tm.UpdateEdge(pid, peer.ID("peer-2"), 10*time.Millisecond, 100, 0.9)

	topo := tm.GetTopology()
	require.NotNil(t, topo)
	assert.Len(t, topo.Nodes, 2)
	assert.Len(t, topo.Edges, 1)

	topo.Nodes[pid] = &NodeInfo{PeerID: peer.ID("hacker")}
	delete(topo.Edges, string(pid)+"->peer-2")

	topo2 := tm.GetTopology()
	assert.Len(t, topo2.Nodes, 2)
	assert.Len(t, topo2.Edges, 1)
}

func TestTopologyManager_FindOptimalPeers_Basic(t *testing.T) {
	logger := newTestLogger()
	rep := NewReputationSystem(logger)
	routing := NewRoutingManager(logger, rep)
	tm := NewTopologyManager(logger, rep, routing)
	ctx := context.Background()

	tm.UpdatePeerConnection(peer.ID("candidate-1"), false, nil, nil)
	tm.UpdatePeerConnection(peer.ID("candidate-2"), false, nil, nil)

	peers := tm.FindOptimalPeers(ctx, 2, nil)
	require.Len(t, peers, 2)
}

func TestTopologyManager_FindOptimalPeers_ExcludesConnected(t *testing.T) {
	logger := newTestLogger()
	rep := NewReputationSystem(logger)
	routing := NewRoutingManager(logger, rep)
	tm := NewTopologyManager(logger, rep, routing)
	ctx := context.Background()

	tm.UpdatePeerConnection(peer.ID("connected-peer"), true, nil, nil)
	tm.UpdatePeerConnection(peer.ID("available-peer"), false, nil, nil)

	peers := tm.FindOptimalPeers(ctx, 5, nil)
	require.Len(t, peers, 1)
	assert.Equal(t, peer.ID("available-peer"), peers[0])
}

func TestTopologyManager_FindOptimalPeers_RequiresCapabilities(t *testing.T) {
	logger := newTestLogger()
	rep := NewReputationSystem(logger)
	routing := NewRoutingManager(logger, rep)
	tm := NewTopologyManager(logger, rep, routing)
	ctx := context.Background()

	tm.UpdatePeerConnection(peer.ID("has-storage"), false, nil, []string{"storage", "compute"})
	tm.UpdatePeerConnection(peer.ID("compute-only"), false, nil, []string{"compute"})

	peers := tm.FindOptimalPeers(ctx, 5, []string{"storage"})
	require.Len(t, peers, 1)
	assert.Equal(t, peer.ID("has-storage"), peers[0])
}

func TestTopologyManager_FindOptimalPeers_NoCandidates(t *testing.T) {
	logger := newTestLogger()
	rep := NewReputationSystem(logger)
	routing := NewRoutingManager(logger, rep)
	tm := NewTopologyManager(logger, rep, routing)
	ctx := context.Background()

	peers := tm.FindOptimalPeers(ctx, 5, nil)
	assert.Empty(t, peers)
}

func TestTopologyManager_FindOptimalPeers_LimitsCount(t *testing.T) {
	logger := newTestLogger()
	rep := NewReputationSystem(logger)
	routing := NewRoutingManager(logger, rep)
	tm := NewTopologyManager(logger, rep, routing)
	ctx := context.Background()

	tm.UpdatePeerConnection(peer.ID("a"), false, nil, nil)
	tm.UpdatePeerConnection(peer.ID("b"), false, nil, nil)
	tm.UpdatePeerConnection(peer.ID("c"), false, nil, nil)

	peers := tm.FindOptimalPeers(ctx, 2, nil)
	require.Len(t, peers, 2)
}

func TestTopologyManager_FindOptimalPeers_PrefersHigherReputation(t *testing.T) {
	logger := newTestLogger()
	rep := NewReputationSystem(logger)
	routing := NewRoutingManager(logger, rep)
	tm := NewTopologyManager(logger, rep, routing)
	ctx := context.Background()

	highRep := peer.ID("high-rep")
	lowRep := peer.ID("low-rep")

	tm.UpdatePeerConnection(highRep, false, nil, nil)
	tm.UpdatePeerConnection(lowRep, false, nil, nil)

	rep.RecordInteraction(string(highRep), ModuleTransfer, Success)
	rep.RecordInteraction(string(highRep), ModuleTransfer, Success)
	rep.RecordInteraction(string(highRep), ModuleTransfer, Success)
	rep.RecordInteraction(string(lowRep), ModuleTransfer, Failure)
	rep.RecordInteraction(string(lowRep), ModuleTransfer, Failure)

	peers := tm.FindOptimalPeers(ctx, 2, nil)
	require.Len(t, peers, 2)
	assert.Equal(t, highRep, peers[0])
}

func TestTopologyManager_OptimizeTopology_UnderConnected(t *testing.T) {
	logger := newTestLogger()
	rep := NewReputationSystem(logger)
	routing := NewRoutingManager(logger, rep)
	tm := NewTopologyManager(logger, rep, routing)
	ctx := context.Background()

	tm.UpdatePeerConnection(peer.ID("candidate-1"), false, nil, nil)
	tm.UpdatePeerConnection(peer.ID("candidate-2"), false, nil, nil)

	tm.OptimizeTopology(ctx)
}

func TestTopologyManager_OptimizeTopology_OverConnected(t *testing.T) {
	logger := newTestLogger()
	rep := NewReputationSystem(logger)
	routing := NewRoutingManager(logger, rep)
	tm := NewTopologyManager(logger, rep, routing)
	ctx := context.Background()

	for i := range 25 {
		pid := peer.ID(peer.ID([]byte{byte(i)}))
		tm.UpdatePeerConnection(pid, true, nil, nil)
	}

	tm.OptimizeTopology(ctx)
}

func TestTopologyManager_OptimizeTopology_NormalRange(t *testing.T) {
	logger := newTestLogger()
	rep := NewReputationSystem(logger)
	routing := NewRoutingManager(logger, rep)
	tm := NewTopologyManager(logger, rep, routing)
	ctx := context.Background()

	for i := range 10 {
		pid := peer.ID(peer.ID([]byte{byte(i)}))
		tm.UpdatePeerConnection(pid, true, nil, nil)
	}

	tm.OptimizeTopology(ctx)
}

func TestTopologyBiasInfluencesScoring(t *testing.T) {
	logger := newTestLogger()
	rep := NewReputationSystem(logger)
	routing := NewRoutingManager(logger, rep)
	tm := NewTopologyManager(logger, rep, routing)
	tm.SetPolicy(TierPolicy{TargetEdgeDegree: 1, RegionBias: 1.0, QualityWeight: 0.1, ReputationWeight: 0.1})

	local := peer.ID("edge-local")
	tm.UpdatePeerConnection(local, true, []string{"asia"}, nil)
	tm.SetPeerTier(local, NodeTierEdge)

	relaySame := peer.ID("relay-same")
	tm.UpdatePeerConnection(relaySame, false, []string{"asia"}, nil)
	tm.SetPeerTier(relaySame, NodeTierRelay)
	tm.UpdateEdge(local, relaySame, 30*time.Millisecond, 100, 0.7)

	relayFar := peer.ID("relay-far")
	tm.UpdatePeerConnection(relayFar, false, []string{"eu"}, nil)
	tm.SetPeerTier(relayFar, NodeTierRelay)
	tm.UpdateEdge(local, relayFar, 10*time.Millisecond, 100, 0.9)

	connect, _ := tm.SuggestTierAdjustments(local, NodeTierEdge)
	require.Len(t, connect, 1, "expected single suggestion")
	assert.Equal(t, relaySame, connect[0], "expected region bias to favor same-region relay")
}
