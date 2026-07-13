package agent

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/naviNBRuas/APA/pkg/networking"
	"github.com/naviNBRuas/APA/pkg/swarm"
)

func (rt *Runtime) runTopologyMutator(ctx context.Context, mut *swarm.TopologyMutator) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			res := mut.Mutate(ctx)
			if len(res.Prune) > 0 || len(res.Attach) > 0 {
				rt.logger.Info("topology mutation applied", "prune", res.Prune, "attach", res.Attach)
			}
		}
	}
}

func (rt *Runtime) runElasticityLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if rt.p2p == nil || rt.elasticityManager == nil {
				continue
			}
			demand := rt.p2p.PeerCount()
			actions := rt.elasticityManager.ObserveDemand(demand)
			if len(actions) > 0 {
				rt.elasticityManager.Apply(actions)
				rt.logger.Info("elasticity adjustment", "demand", demand, "actions", actions, "capacity", rt.elasticityManager.Snapshot())
			}
		}
	}
}

func (rt *Runtime) ReportSuspiciousPeer(sig swarm.SuspicionSignal) {
	if rt.sinkResistance == nil {
		return
	}
	if triggered := rt.sinkResistance.Observe(sig); triggered {
		rt.logger.Warn("sink resistance triggered re-key", "peer", sig.Peer, "reason", sig.Reason, "source", sig.Source)
	}
}

type networkStatsAdapter struct {
	nm *swarm.NetworkMonitor
}

func (a *networkStatsAdapter) GetNetworkStats(pid peer.ID) *networking.NetworkStats {
	if a == nil || a.nm == nil {
		return nil
	}
	s := a.nm.GetNetworkStats(pid)
	if s == nil {
		return nil
	}
	return &networking.NetworkStats{Latency: s.Latency, Bandwidth: s.Bandwidth, PacketLoss: s.PacketLoss}
}
