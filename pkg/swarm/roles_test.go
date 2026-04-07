package swarm

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
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

	if rm.GetRole("p1").Role != RoleScanner || rm.GetRole("p2").Role != RoleRelay || rm.GetRole("p3").Role != RoleAggregator || rm.GetRole("p4").Role != RoleExecutor {
		t.Fatalf("roles not assigned as expected")
	}
}

func TestRoleStickinessHonored(t *testing.T) {
	rm := NewRoleManager(roleLogger(), RolePolicy{TargetScanners: 1, MinRoleDuration: 5 * time.Minute})
	peers := []peer.ID{"p1", "p2"}

	scoreFn := func(id peer.ID, role NodeRole) float64 { return 1 }

	rm.SetRole("p1", RoleScanner, 1)
	rm.Rebalance(peers, scoreFn)

	if rm.GetRole("p1").Role != RoleScanner {
		t.Fatalf("expected p1 to remain scanner due to stickiness")
	}
}

func TestMetricsSnapshot(t *testing.T) {
	rm := NewRoleManager(roleLogger(), RolePolicy{})
	rm.SetRole("p1", RoleScanner, 1)
	rm.SetRole("p2", RoleRelay, 1)

	snap := rm.Snapshot()
	if snap.Scanners != 1 || snap.Relays != 1 {
		t.Fatalf("unexpected metrics snapshot: %+v", snap)
	}
}
