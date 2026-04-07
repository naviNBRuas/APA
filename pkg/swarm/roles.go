package swarm

import (
	"log/slog"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// NodeRole enumerates dynamic specialization roles.
type NodeRole string

const (
	RoleUnknown    NodeRole = "unknown"
	RoleScanner    NodeRole = "scanner"
	RoleRelay      NodeRole = "relay"
	RoleAggregator NodeRole = "aggregator"
	RoleExecutor   NodeRole = "executor"
)

// RoleProfile captures a peer's role plus metadata.
type RoleProfile struct {
	Role       NodeRole
	AssignedAt time.Time
	Score      float64 // suitability score used during assignment
}

// RolePolicy configures desired role ratios and reassignment cadence.
type RolePolicy struct {
	TargetScanners    int
	TargetRelays      int
	TargetAggregators int
	TargetExecutors   int
	MinRoleDuration   time.Duration
}

func (p RolePolicy) withDefaults() RolePolicy {
	if p.TargetScanners == 0 {
		p.TargetScanners = 3
	}
	if p.TargetRelays == 0 {
		p.TargetRelays = 5
	}
	if p.TargetAggregators == 0 {
		p.TargetAggregators = 3
	}
	if p.TargetExecutors == 0 {
		p.TargetExecutors = 6
	}
	if p.MinRoleDuration == 0 {
		p.MinRoleDuration = 10 * time.Minute
	}
	return p
}

// RoleManager handles dynamic role assignment across peers.
type RoleManager struct {
	logger  *slog.Logger
	policy  RolePolicy
	mu      sync.RWMutex
	roles   map[peer.ID]RoleProfile
	rng     *rand.Rand
	metrics RoleMetrics
}

// RoleMetrics exposes counts per role.
type RoleMetrics struct {
	Scanners    int
	Relays      int
	Aggregators int
	Executors   int
}

// NewRoleManager returns a role manager with defaults.
func NewRoleManager(logger *slog.Logger, policy RolePolicy) *RoleManager {
	policy = policy.withDefaults()
	return &RoleManager{
		logger: logger,
		policy: policy,
		roles:  make(map[peer.ID]RoleProfile),
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Snapshot returns a copy of current role metrics.
func (rm *RoleManager) Snapshot() RoleMetrics {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.metrics
}

// GetRole returns the current role for a peer.
func (rm *RoleManager) GetRole(id peer.ID) RoleProfile {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.roles[id]
}

// SetRole forcibly sets a peer's role.
func (rm *RoleManager) SetRole(id peer.ID, role NodeRole, score float64) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.setRoleLocked(id, role, score)
	rm.recomputeMetricsLocked()
}

func (rm *RoleManager) setRoleLocked(id peer.ID, role NodeRole, score float64) {
	prev, ok := rm.roles[id]
	if ok && time.Since(prev.AssignedAt) < rm.policy.MinRoleDuration {
		// Stickiness: do not change roles until minimum duration elapsed
		return
	}
	rm.roles[id] = RoleProfile{Role: role, AssignedAt: time.Now(), Score: score}
	rm.recomputeMetricsLocked()
}

func (rm *RoleManager) recomputeMetricsLocked() {
	counts := RoleMetrics{}
	for _, p := range rm.roles {
		switch p.Role {
		case RoleScanner:
			counts.Scanners++
		case RoleRelay:
			counts.Relays++
		case RoleAggregator:
			counts.Aggregators++
		case RoleExecutor:
			counts.Executors++
		}
	}
	rm.metrics = counts
}

// Rebalance assigns roles dynamically based on policy targets and a suitability score per peer.
// The scoreFn should return a higher value for better suitability for the requested role.
func (rm *RoleManager) Rebalance(peers []peer.ID, scoreFn func(peer.ID, NodeRole) float64) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.policy = rm.policy.withDefaults()

	// Reset counts but keep existing assignments when within policy and sticky
	rm.recomputeMetricsLocked()

	assign := func(target int, role NodeRole) {
		// collect candidates with scores
		type cand struct {
			id    peer.ID
			score float64
		}
		var cs []cand
		for _, id := range peers {
			cs = append(cs, cand{id: id, score: scoreFn(id, role)})
		}
		// sort by score descending
		sort.Slice(cs, func(i, j int) bool { return cs[i].score > cs[j].score })

		current := rm.countRoleLocked(role)
		needed := target - current
		if needed <= 0 {
			return
		}
		for i := 0; i < len(cs) && needed > 0; i++ {
			c := cs[i]
			prev := rm.roles[c.id]
			if prev.Role == role && time.Since(prev.AssignedAt) < rm.policy.MinRoleDuration {
				continue
			}
			rm.setRoleLocked(c.id, role, c.score)
			needed--
		}
	}

	assign(rm.policy.TargetScanners, RoleScanner)
	assign(rm.policy.TargetRelays, RoleRelay)
	assign(rm.policy.TargetAggregators, RoleAggregator)
	assign(rm.policy.TargetExecutors, RoleExecutor)
}

func (rm *RoleManager) countRoleLocked(role NodeRole) int {
	switch role {
	case RoleScanner:
		return rm.metrics.Scanners
	case RoleRelay:
		return rm.metrics.Relays
	case RoleAggregator:
		return rm.metrics.Aggregators
	case RoleExecutor:
		return rm.metrics.Executors
	default:
		return 0
	}
}
