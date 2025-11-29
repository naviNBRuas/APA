package swarm

import (
	"context"
	"log/slog"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// RoutingManager manages adaptive routing based on peer reputation and network conditions
type RoutingManager struct {
	logger          *slog.Logger
	reputation      *ReputationSystem
	networkMonitor  *NetworkMonitor
	routingTable    map[string][]Route // resource -> routes
	mu              sync.RWMutex
}

// Route represents a route to a resource through a peer
type Route struct {
	PeerID     peer.ID `json:"peer_id"`
	Resource   string  `json:"resource"`
	Cost       float64 `json:"cost"`
	Latency    time.Duration `json:"latency"`
	Bandwidth  float64 `json:"bandwidth"` // Mbps
	LastUsed   time.Time `json:"last_used"`
	SuccessRate float64 `json:"success_rate"`
}

// NetworkStats represents network statistics for a peer
type NetworkStats struct {
	Latency    time.Duration
	Bandwidth  float64 // Mbps
	PacketLoss float64 // Percentage
}

// NetworkMonitor monitors network conditions
type NetworkMonitor struct {
	logger *slog.Logger
	stats  map[peer.ID]*NetworkStats
	mu     sync.RWMutex
}

// NewRoutingManager creates a new routing manager
func NewRoutingManager(logger *slog.Logger, reputation *ReputationSystem) *RoutingManager {
	nm := &NetworkMonitor{
		logger: logger,
		stats:  make(map[peer.ID]*NetworkStats),
	}

	return &RoutingManager{
		logger:         logger,
		reputation:     reputation,
		networkMonitor: nm,
		routingTable:   make(map[string][]Route),
	}
}

// NewNetworkMonitor creates a new network monitor
func NewNetworkMonitor(logger *slog.Logger) *NetworkMonitor {
	return &NetworkMonitor{
		logger: logger,
		stats:  make(map[peer.ID]*NetworkStats),
	}
}

// UpdateNetworkStats updates network statistics for a peer
func (nm *NetworkMonitor) UpdateNetworkStats(peerID peer.ID, latency time.Duration, bandwidth float64, packetLoss float64) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	stats, exists := nm.stats[peerID]
	if !exists {
		stats = &NetworkStats{}
		nm.stats[peerID] = stats
	}

	stats.Latency = latency
	stats.Bandwidth = bandwidth
	stats.PacketLoss = packetLoss

	nm.logger.Debug("Updated network stats for peer", 
		"peer_id", peerID, 
		"latency", latency, 
		"bandwidth", bandwidth, 
		"packet_loss", packetLoss)
}

// GetNetworkStats returns network statistics for a peer
func (nm *NetworkMonitor) GetNetworkStats(peerID peer.ID) *NetworkStats {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	stats, exists := nm.stats[peerID]
	if !exists {
		return &NetworkStats{
			Latency:    100 * time.Millisecond, // Default latency
			Bandwidth:  10.0,                   // Default bandwidth (Mbps)
			PacketLoss: 0.0,                    // Default packet loss
		}
	}

	return &NetworkStats{
		Latency:    stats.Latency,
		Bandwidth:  stats.Bandwidth,
		PacketLoss: stats.PacketLoss,
	}
}

// AddRoute adds a route to the routing table
func (rm *RoutingManager) AddRoute(resource string, route Route) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	routes, exists := rm.routingTable[resource]
	if !exists {
		routes = make([]Route, 0)
	}

	// Check if route already exists
	for i, r := range routes {
		if r.PeerID == route.PeerID {
			// Update existing route
			routes[i] = route
			rm.routingTable[resource] = routes
			return
		}
	}

	// Add new route
	routes = append(routes, route)
	rm.routingTable[resource] = routes

	rm.logger.Debug("Added route to routing table", 
		"resource", resource, 
		"peer_id", route.PeerID, 
		"cost", route.Cost)
}

// RemoveRoute removes a route from the routing table
func (rm *RoutingManager) RemoveRoute(resource string, peerID peer.ID) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	routes, exists := rm.routingTable[resource]
	if !exists {
		return
	}

	// Find and remove the route
	for i, route := range routes {
		if route.PeerID == peerID {
			routes = append(routes[:i], routes[i+1:]...)
			rm.routingTable[resource] = routes
			rm.logger.Debug("Removed route from routing table", 
				"resource", resource, 
				"peer_id", peerID)
			return
		}
	}
}

// GetBestRoute returns the best route to a resource based on cost, reputation, and network conditions
func (rm *RoutingManager) GetBestRoute(ctx context.Context, resource string) *Route {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	routes, exists := rm.routingTable[resource]
	if !exists || len(routes) == 0 {
		return nil
	}

	// Sort routes by composite score
	sortedRoutes := make([]Route, len(routes))
	copy(sortedRoutes, routes)

	sort.Slice(sortedRoutes, func(i, j int) bool {
		scoreI := rm.calculateRouteScore(&sortedRoutes[i])
		scoreJ := rm.calculateRouteScore(&sortedRoutes[j])
		return scoreI > scoreJ // Higher score is better
	})

	// Return the best route
	bestRoute := sortedRoutes[0]
	
	// Update last used time
	bestRoute.LastUsed = time.Now()
	
	rm.logger.Debug("Selected best route", 
		"resource", resource, 
		"peer_id", bestRoute.PeerID, 
		"score", rm.calculateRouteScore(&bestRoute))

	return &bestRoute
}

// GetMultipleRoutes returns multiple routes to a resource, sorted by preference
func (rm *RoutingManager) GetMultipleRoutes(ctx context.Context, resource string, count int) []Route {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	routes, exists := rm.routingTable[resource]
	if !exists || len(routes) == 0 {
		return nil
	}

	// Sort routes by composite score
	sortedRoutes := make([]Route, len(routes))
	copy(sortedRoutes, routes)

	sort.Slice(sortedRoutes, func(i, j int) bool {
		scoreI := rm.calculateRouteScore(&sortedRoutes[i])
		scoreJ := rm.calculateRouteScore(&sortedRoutes[j])
		return scoreI > scoreJ // Higher score is better
	})

	// Return up to count routes
	if len(sortedRoutes) > count {
		sortedRoutes = sortedRoutes[:count]
	}

	// Update last used time for returned routes
	now := time.Now()
	for i := range sortedRoutes {
		sortedRoutes[i].LastUsed = now
	}

	rm.logger.Debug("Selected multiple routes", 
		"resource", resource, 
		"count", len(sortedRoutes))

	return sortedRoutes
}

// calculateRouteScore calculates a composite score for a route based on multiple factors
func (rm *RoutingManager) calculateRouteScore(route *Route) float64 {
	// Get reputation score (0-100)
	reputationScore := rm.reputation.GetScore(string(route.PeerID))

	// Get network stats
	networkStats := rm.networkMonitor.GetNetworkStats(route.PeerID)

	// Normalize factors to 0-1 scale
	normalizedReputation := reputationScore / 100.0
	
	// Lower latency is better (invert and normalize)
	maxLatency := 1000.0 // ms
	normalizedLatency := 1.0 - (float64(networkStats.Latency.Milliseconds()) / maxLatency)
	if normalizedLatency < 0 {
		normalizedLatency = 0
	}
	
	// Higher bandwidth is better (normalize to 0-1, assuming max 1000 Mbps)
	maxBandwidth := 1000.0
	normalizedBandwidth := networkStats.Bandwidth / maxBandwidth
	if normalizedBandwidth > 1.0 {
		normalizedBandwidth = 1.0
	}
	
	// Lower packet loss is better (invert and normalize)
	normalizedPacketLoss := 1.0 - (networkStats.PacketLoss / 100.0)
	if normalizedPacketLoss < 0 {
		normalizedPacketLoss = 0
	}
	
	// Lower cost is better (invert and normalize)
	maxCost := 100.0
	normalizedCost := 1.0 - (route.Cost / maxCost)
	if normalizedCost < 0 {
		normalizedCost = 0
	}
	
	// Higher success rate is better
	normalizedSuccessRate := route.SuccessRate
	
	// Calculate weighted composite score
	// Weights can be adjusted based on priorities
	reputationWeight := 0.3
	latencyWeight := 0.2
	bandwidthWeight := 0.2
	packetLossWeight := 0.1
	costWeight := 0.1
	successRateWeight := 0.1
	
	compositeScore := (normalizedReputation * reputationWeight) +
		(normalizedLatency * latencyWeight) +
		(normalizedBandwidth * bandwidthWeight) +
		(normalizedPacketLoss * packetLossWeight) +
		(normalizedCost * costWeight) +
		(normalizedSuccessRate * successRateWeight)

	return compositeScore
}

// GetRandomRoute returns a random route to a resource (useful for load balancing)
func (rm *RoutingManager) GetRandomRoute(ctx context.Context, resource string) *Route {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	routes, exists := rm.routingTable[resource]
	if !exists || len(routes) == 0 {
		return nil
	}

	// Select a random route
	randomIndex := rand.Intn(len(routes))
	route := routes[randomIndex]
	
	// Update last used time
	route.LastUsed = time.Now()
	
	rm.logger.Debug("Selected random route", 
		"resource", resource, 
		"peer_id", route.PeerID)

	return &route
}

// GetTrustedRoutes returns routes only through trusted peers
func (rm *RoutingManager) GetTrustedRoutes(ctx context.Context, resource string, trustThreshold float64) []Route {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	routes, exists := rm.routingTable[resource]
	if !exists || len(routes) == 0 {
		return nil
	}

	// Filter routes to only include trusted peers
	var trustedRoutes []Route
	for _, route := range routes {
		if rm.reputation.IsTrustedPeer(string(route.PeerID), trustThreshold) {
			trustedRoutes = append(trustedRoutes, route)
		}
	}

	rm.logger.Debug("Selected trusted routes", 
		"resource", resource, 
		"trusted_count", len(trustedRoutes), 
		"total_count", len(routes))

	return trustedRoutes
}