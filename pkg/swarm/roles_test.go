package swarm

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
)

func roleLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestRoleRebalanceAssignsTargets(t *testing.T) {
	rm := NewRoleManager(roleLogger(), RolePolicy{TargetScanners: 1, TargetRelays: 1, TargetAggregators: 1, TargetExecutors: 1, MinRoleDuration: time.Second})
	peers := []peer.ID{"p1", "p2", "p3", "p4"}

	scoreFn := func(id peer.ID, role NodeRole) float64 {
		switch role {
		case RoleScanner:
			if id == "p1" {
				return 10
			}
		case RoleRelay:
			if id == "p2" {
				return 9
			}
		case RoleAggregator:
			if id == "p3" {
				return 8
			}
		case RoleExecutor:
			if id == "p4" {
				return 7
			}
		}
		return 1
	}

	rm.Rebalance(peers, scoreFn)

	assert.Equal(t, RoleScanner, rm.GetRole("p1").Role)
	assert.Equal(t, RoleRelay, rm.GetRole("p2").Role)
	assert.Equal(t, RoleAggregator, rm.GetRole("p3").Role)
	assert.Equal(t, RoleExecutor, rm.GetRole("p4").Role)
}

func TestRoleStickinessHonored(t *testing.T) {
	rm := NewRoleManager(roleLogger(), RolePolicy{TargetScanners: 1, MinRoleDuration: 5 * time.Minute})
	peers := []peer.ID{"p1", "p2"}

	scoreFn := func(id peer.ID, role NodeRole) float64 { return 1 }

	rm.SetRole("p1", RoleScanner, 1)
	rm.Rebalance(peers, scoreFn)

	assert.Equal(t, RoleScanner, rm.GetRole("p1").Role, "expected p1 to remain scanner due to stickiness")
}

func TestMetricsSnapshot(t *testing.T) {
	rm := NewRoleManager(roleLogger(), RolePolicy{})
	rm.SetRole("p1", RoleScanner, 1)
	rm.SetRole("p2", RoleRelay, 1)

	snap := rm.Snapshot()
	assert.Equal(t, 1, snap.Scanners)
	assert.Equal(t, 1, snap.Relays)
}

func TestRoleGetRoleForUnknownPeer(t *testing.T) {
	rm := NewRoleManager(roleLogger(), RolePolicy{})
	roleInfo := rm.GetRole("unknown")
	assert.Empty(t, string(roleInfo.Role), "unknown peer should have zero-value role")
}

func TestRoleRebalanceEmptyPeers(t *testing.T) {
	rm := NewRoleManager(roleLogger(), RolePolicy{TargetScanners: 1})
	rm.Rebalance(nil, nil)
	snap := rm.Snapshot()
	assert.Equal(t, 0, snap.Scanners)
}
