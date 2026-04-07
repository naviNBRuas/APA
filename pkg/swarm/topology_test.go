package swarm

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
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
	if len(disconnect) != 0 {
		t.Fatalf("expected no disconnects for under-connected node, got %v", disconnect)
	}

	if len(connect) != 2 {
		t.Fatalf("expected 2 suggested connections, got %d", len(connect))
	}

	for _, peerID := range connect {
		tier := tm.topology.Nodes[peerID].Tier
		if tier == NodeTierEdge {
			t.Fatalf("edge node should not prefer another edge; got suggestion %s", peerID)
		}
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
	if len(connect) != 0 {
		t.Fatalf("expected no connect suggestions when over-connected, got %v", connect)
	}
	if len(disconnect) != 1 {
		t.Fatalf("expected 1 disconnect suggestion, got %d", len(disconnect))
	}

	if disconnect[0] != edgeWeak {
		t.Fatalf("expected weakest connection %s to be suggested for removal, got %s", edgeWeak, disconnect[0])
	}
}

func TestRegionBiasInfluencesScoring(t *testing.T) {
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
	if len(connect) != 1 {
		t.Fatalf("expected single suggestion, got %d", len(connect))
	}

	if connect[0] != relaySame {
		t.Fatalf("expected region bias to favor %s, got %s", relaySame, connect[0])
	}
}
