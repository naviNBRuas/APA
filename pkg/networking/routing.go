package networking

import (
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

type IntelligentRoutingEngine struct {
	logger           *slog.Logger
	routingTable     map[peer.ID][]RouteOption
	performanceCache map[string]*RoutePerformance
	strategy         RoutingStrategy

	mu sync.RWMutex
}

type RouteOption struct {
	Protocol     ProtocolType
	Address      string
	Cost         float64
	Reliability  float64
	Latency      time.Duration
	Bandwidth    float64
	Security     SecurityLevel
	LastUsed     time.Time
	SuccessCount int64
	FailureCount int64
}

type RoutePerformance struct{}

func NewIntelligentRoutingEngine(logger *slog.Logger) *IntelligentRoutingEngine {
	return &IntelligentRoutingEngine{
		logger:           logger,
		routingTable:     make(map[peer.ID][]RouteOption),
		performanceCache: make(map[string]*RoutePerformance),
	}
}

func (ire *IntelligentRoutingEngine) SelectBestRoute(to peer.ID, message *NetworkMessage) *RouteOption {
	ire.mu.RLock()
	routes, ok := ire.routingTable[to]
	ire.mu.RUnlock()

	if !ok || len(routes) == 0 {
		return nil
	}

	var best *RouteOption
	bestScore := math.Inf(-1)

	for i := range routes {
		r := &routes[i]
		latencyWeight := 1.0
		if r.Latency > 0 {
			latencyWeight = 1.0 / (1.0 + r.Latency.Seconds())
		}
		score := r.Reliability*0.4 + r.Bandwidth*0.3 + latencyWeight*0.2 - r.Cost*0.1
		if score > bestScore {
			bestScore = score
			best = r
		}
	}

	return best
}
