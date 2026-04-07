package swarm

import (
	"context"
	"sort"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// TopologyMutator periodically rewires the mesh to improve latency and churn shape.
type TopologyMutator struct {
	tm      *TopologyManager
	policy  MutationPolicy
	localID peer.ID
}

// MutationPolicy controls pruning and attachment.
type MutationPolicy struct {
	TargetDegree int
	MaxLatency   time.Duration
	Interval     time.Duration
}

func (p MutationPolicy) withDefaults() MutationPolicy {
	if p.TargetDegree == 0 {
		p.TargetDegree = 12
	}
	if p.MaxLatency == 0 {
		p.MaxLatency = 600 * time.Millisecond
	}
	if p.Interval == 0 {
		p.Interval = 5 * time.Minute
	}
	return p
}

// MutationResult captures suggested rewiring actions.
type MutationResult struct {
	Prune  []peer.ID
	Attach []peer.ID
	RanAt  time.Time
}

type score struct {
	id    peer.ID
	value float64
}

func NewTopologyMutator(tm *TopologyManager, policy MutationPolicy, localID peer.ID) *TopologyMutator {
	return &TopologyMutator{tm: tm, policy: policy.withDefaults(), localID: localID}
}

// Mutate proposes pruning high-latency edges and attaching to better peers.
func (m *TopologyMutator) Mutate(ctx context.Context) MutationResult {
	res := MutationResult{RanAt: time.Now()}
	policy := m.policy.withDefaults()

	connected := m.tm.GetConnectedPeers()
	if len(connected) == 0 {
		return res
	}

	// Prune worst latency peers beyond target degree.
	var peers []score
	for _, pid := range connected {
		edge := m.tm.EdgeInfoBetween(m.localID, pid)
		latency := time.Duration(0)
		quality := 0.0
		if edge != nil {
			latency = edge.Latency
			quality = edge.Quality
		}
		val := float64(latency/time.Millisecond) - quality*100
		peers = append(peers, score{id: pid, value: val})
	}

	sort.Slice(peers, func(i, j int) bool { return peers[i].value > peers[j].value })
	if len(peers) > policy.TargetDegree {
		excess := len(peers) - policy.TargetDegree
		res.Prune = append(res.Prune, toIDs(peers[:excess])...)
	}

	// Attach: use topology suggestions for new connections.
	suggestions := m.tm.FindOptimalPeers(ctx, policy.TargetDegree, nil)
	res.Attach = append(res.Attach, suggestions...)
	return res
}

func toIDs(scores []score) []peer.ID {
	out := make([]peer.ID, 0, len(scores))
	for _, s := range scores {
		out = append(out, s.id)
	}
	return out
}
