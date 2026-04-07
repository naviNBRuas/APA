package networking

import (
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// ReputationProvider supplies peer reputation scores (0-100).
type ReputationProvider interface {
	GetScore(peerID string) float64
}

// NetworkStatsProvider supplies network stats per peer.
type NetworkStatsProvider interface {
	GetNetworkStats(peer.ID) *NetworkStats
}

// ForwardPolicy defines gating thresholds.
type ForwardPolicy struct {
	MinReputation float64
	MaxLatency    time.Duration
	MaxPacketLoss float64 // 0.0-1.0
	MinBandwidth  float64 // Mbps

	BucketBytes       int           // token bucket capacity
	RefillBytesPerSec int           // refill rate
	Cooldown          time.Duration // optional cooldown when refused
}

func (p ForwardPolicy) withDefaults() ForwardPolicy {
	if p.MinReputation == 0 {
		p.MinReputation = 40
	}
	if p.MaxLatency == 0 {
		p.MaxLatency = 500 * time.Millisecond
	}
	if p.MaxPacketLoss == 0 {
		p.MaxPacketLoss = 0.2
	}
	if p.MinBandwidth == 0 {
		p.MinBandwidth = 5 // Mbps
	}
	if p.BucketBytes == 0 {
		p.BucketBytes = 256 * 1024
	}
	if p.RefillBytesPerSec == 0 {
		p.RefillBytesPerSec = 256 * 1024
	}
	return p
}

// SelectiveForwarder enforces self-throttling propagation decisions.
type SelectiveForwarder struct {
	policy ForwardPolicy
	rep    ReputationProvider
	net    NetworkStatsProvider

	mu       sync.Mutex
	tokens   int
	lastFill time.Time
	lastDeny time.Time
}

// NewSelectiveForwarder creates a forward decider with the given providers.
func NewSelectiveForwarder(policy ForwardPolicy, rep ReputationProvider, net NetworkStatsProvider) *SelectiveForwarder {
	policy = policy.withDefaults()
	return &SelectiveForwarder{
		policy:   policy,
		rep:      rep,
		net:      net,
		tokens:   policy.BucketBytes,
		lastFill: time.Now(),
	}
}

// AllowForward implements ForwardDecider.
func (sf *SelectiveForwarder) AllowForward(target peer.ID, payloadBytes int) bool {
	now := time.Now()

	if sf.policy.Cooldown > 0 {
		sf.mu.Lock()
		if now.Sub(sf.lastDeny) < sf.policy.Cooldown {
			sf.mu.Unlock()
			return false
		}
		sf.mu.Unlock()
	}

	if sf.rep != nil {
		if score := sf.rep.GetScore(string(target)); score < sf.policy.MinReputation {
			sf.markDeny(now)
			return false
		}
	}

	if sf.net != nil {
		stats := sf.net.GetNetworkStats(target)
		if stats != nil {
			if stats.Latency > sf.policy.MaxLatency {
				sf.markDeny(now)
				return false
			}
			if stats.PacketLoss > sf.policy.MaxPacketLoss {
				sf.markDeny(now)
				return false
			}
			if stats.Bandwidth < sf.policy.MinBandwidth {
				sf.markDeny(now)
				return false
			}
		}
	}

	if payloadBytes <= 0 {
		return true
	}

	sf.mu.Lock()
	defer sf.mu.Unlock()

	sf.refillLocked(now)

	if sf.tokens < payloadBytes {
		sf.lastDeny = now
		return false
	}
	sf.tokens -= payloadBytes
	return true
}

func (sf *SelectiveForwarder) refillLocked(now time.Time) {
	elapsed := now.Sub(sf.lastFill)
	if elapsed <= 0 {
		return
	}
	add := int(elapsed.Seconds() * float64(sf.policy.RefillBytesPerSec))
	if add > 0 {
		sf.tokens += add
		if sf.tokens > sf.policy.BucketBytes {
			sf.tokens = sf.policy.BucketBytes
		}
		sf.lastFill = now
	}
}

func (sf *SelectiveForwarder) markDeny(now time.Time) {
	if sf.policy.Cooldown == 0 {
		return
	}
	sf.mu.Lock()
	if now.After(sf.lastDeny) {
		sf.lastDeny = now
	}
	sf.mu.Unlock()
}
