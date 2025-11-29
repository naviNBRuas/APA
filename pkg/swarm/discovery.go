package swarm

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// ResourceManager manages resource discovery and availability
type ResourceManager struct {
	logger          *slog.Logger
	reputation      *ReputationSystem
	routing         *RoutingManager
	topology        *TopologyManager
	resources       map[string]*ResourceInfo // resourceID -> resource info
	resourceOwners  map[string][]peer.ID     // resourceID -> owners
	mu              sync.RWMutex
}

// ResourceInfo represents information about a resource
type ResourceInfo struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Version     string    `json:"version"`
	Size        int64     `json:"size"`
	Hash        string    `json:"hash"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ResourceAnnouncement represents a resource announcement from a peer
type ResourceAnnouncement struct {
	PeerID      peer.ID       `json:"peer_id"`
	Resource    *ResourceInfo `json:"resource"`
	Timestamp   time.Time     `json:"timestamp"`
	Expiration  time.Time     `json:"expiration"`
}

// ResourceQuery represents a query for resources
type ResourceQuery struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Version  string   `json:"version"`
	Tags     []string `json:"tags"`
	MinScore float64  `json:"min_score"`
}

// NewResourceManager creates a new resource manager
func NewResourceManager(logger *slog.Logger, reputation *ReputationSystem, routing *RoutingManager, topology *TopologyManager) *ResourceManager {
	return &ResourceManager{
		logger:         logger,
		reputation:     reputation,
		routing:        routing,
		topology:       topology,
		resources:      make(map[string]*ResourceInfo),
		resourceOwners: make(map[string][]peer.ID),
	}
}

// AnnounceResource announces a resource available from this peer
func (rm *ResourceManager) AnnounceResource(ctx context.Context, resource *ResourceInfo) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Store resource information
	rm.resources[resource.ID] = resource

	// Add this peer as an owner (assuming local peer)
	// In a real implementation, this would come from the network announcement
	// owners, exists := rm.resourceOwners[resource.ID]
	// if !exists {
	// 	owners = make([]peer.ID, 0)
	// }
	// owners = append(owners, localPeerID)
	// rm.resourceOwners[resource.ID] = owners

	rm.logger.Debug("Announced resource", 
		"resource_id", resource.ID, 
		"name", resource.Name, 
		"type", resource.Type)
}

// HandleResourceAnnouncement handles a resource announcement from another peer
func (rm *ResourceManager) HandleResourceAnnouncement(ctx context.Context, announcement *ResourceAnnouncement) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Store resource information
	rm.resources[announcement.Resource.ID] = announcement.Resource

	// Update ownership mapping
	owners, exists := rm.resourceOwners[announcement.Resource.ID]
	if !exists {
		owners = make([]peer.ID, 0)
	}

	// Check if peer is already in owners list
	alreadyOwner := false
	for _, owner := range owners {
		if owner == announcement.PeerID {
			alreadyOwner = true
			break
		}
	}

	if !alreadyOwner {
		owners = append(owners, announcement.PeerID)
		rm.resourceOwners[announcement.Resource.ID] = owners
	}

	// Add route for this resource
	route := Route{
		PeerID:     announcement.PeerID,
		Resource:   announcement.Resource.ID,
		Cost:       10.0, // Base cost, would be calculated in real implementation
		Latency:    50 * time.Millisecond, // Placeholder
		Bandwidth:  50.0, // Mbps, placeholder
		LastUsed:   time.Now(),
		SuccessRate: 1.0, // Placeholder
	}
	rm.routing.AddRoute(announcement.Resource.ID, route)

	rm.logger.Debug("Handled resource announcement", 
		"resource_id", announcement.Resource.ID, 
		"peer_id", announcement.PeerID, 
		"expiration", announcement.Expiration)
}

// RemoveExpiredResources removes expired resource announcements
func (rm *ResourceManager) RemoveExpiredResources(ctx context.Context) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	now := time.Now()
	var expiredResources []string

	for resourceID, resource := range rm.resources {
		// In a real implementation, we would check expiration times
		// For now, we'll just remove resources that haven't been updated in 24 hours
		if now.Sub(resource.UpdatedAt) > 24*time.Hour {
			expiredResources = append(expiredResources, resourceID)
		}
	}

	for _, resourceID := range expiredResources {
		delete(rm.resources, resourceID)
		delete(rm.resourceOwners, resourceID)
		
		// Remove routes for this resource
		// In a real implementation, we would need to be more specific about which routes to remove
		rm.logger.Debug("Removed expired resource", "resource_id", resourceID)
	}

	if len(expiredResources) > 0 {
		rm.logger.Info("Cleaned up expired resources", "count", len(expiredResources))
	}
}

// FindResources searches for resources matching the query
func (rm *ResourceManager) FindResources(ctx context.Context, query *ResourceQuery) []*ResourceInfo {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	var matchingResources []*ResourceInfo

	for _, resource := range rm.resources {
		// Check name match
		if query.Name != "" && resource.Name != query.Name {
			continue
		}

		// Check type match
		if query.Type != "" && resource.Type != query.Type {
			continue
		}

		// Check version match
		if query.Version != "" && resource.Version != query.Version {
			continue
		}

		// Check tags match (all tags must be present)
		if len(query.Tags) > 0 {
			tagMatch := true
			for _, requiredTag := range query.Tags {
				found := false
				for _, resourceTag := range resource.Tags {
					if resourceTag == requiredTag {
						found = true
						break
					}
				}
				if !found {
					tagMatch = false
					break
				}
			}
			if !tagMatch {
				continue
			}
		}

		matchingResources = append(matchingResources, resource)
	}

	rm.logger.Debug("Found matching resources", 
		"query", query, 
		"count", len(matchingResources))

	return matchingResources
}

// GetResource returns information about a specific resource
func (rm *ResourceManager) GetResource(ctx context.Context, resourceID string) *ResourceInfo {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	resource, exists := rm.resources[resourceID]
	if !exists {
		return nil
	}

	// Return a copy to prevent external modification
	return &ResourceInfo{
		ID:          resource.ID,
		Name:        resource.Name,
		Type:        resource.Type,
		Version:     resource.Version,
		Size:        resource.Size,
		Hash:        resource.Hash,
		Description: resource.Description,
		Tags:        append([]string(nil), resource.Tags...),
		CreatedAt:   resource.CreatedAt,
		UpdatedAt:   resource.UpdatedAt,
	}
}

// GetResourceOwners returns the peers that own a specific resource
func (rm *ResourceManager) GetResourceOwners(ctx context.Context, resourceID string) []peer.ID {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	owners, exists := rm.resourceOwners[resourceID]
	if !exists {
		return nil
	}

	// Return a copy to prevent external modification
	ownerCopy := make([]peer.ID, len(owners))
	copy(ownerCopy, owners)
	return ownerCopy
}

// GetResourcesByPeer returns all resources owned by a specific peer
func (rm *ResourceManager) GetResourcesByPeer(ctx context.Context, peerID peer.ID) []*ResourceInfo {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	var peerResources []*ResourceInfo

	for resourceID, owners := range rm.resourceOwners {
		for _, owner := range owners {
			if owner == peerID {
				resource, exists := rm.resources[resourceID]
				if exists {
					peerResources = append(peerResources, resource)
				}
				break
			}
		}
	}

	rm.logger.Debug("Found resources for peer", 
		"peer_id", peerID, 
		"count", len(peerResources))

	return peerResources
}

// GetAllResources returns all known resources
func (rm *ResourceManager) GetAllResources(ctx context.Context) []*ResourceInfo {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	var allResources []*ResourceInfo
	for _, resource := range rm.resources {
		allResources = append(allResources, resource)
	}

	rm.logger.Debug("Retrieved all resources", "count", len(allResources))

	return allResources
}

// GetResourceCount returns the total number of known resources
func (rm *ResourceManager) GetResourceCount(ctx context.Context) int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return len(rm.resources)
}

// GetTrustedResourceOwners returns trusted peers that own a specific resource
func (rm *ResourceManager) GetTrustedResourceOwners(ctx context.Context, resourceID string, trustThreshold float64) []peer.ID {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	owners, exists := rm.resourceOwners[resourceID]
	if !exists {
		return nil
	}

	var trustedOwners []peer.ID
	for _, owner := range owners {
		if rm.reputation.IsTrustedPeer(string(owner), trustThreshold) {
			trustedOwners = append(trustedOwners, owner)
		}
	}

	rm.logger.Debug("Found trusted resource owners", 
		"resource_id", resourceID, 
		"trusted_count", len(trustedOwners), 
		"total_count", len(owners))

	return trustedOwners
}