package agent

import (
	"log/slog"
	"sync"
)

// AdminPeerManager manages admin peer identification and authentication
type AdminPeerManager struct {
	mu sync.RWMutex

	// List of peer IDs that are authorized as admin peers
	authorizedAdminPeers map[string]bool

	// Minimum reputation score required for admin access
	minReputationThreshold float64

	// Logger for the admin peer manager
	logger *slog.Logger
}

// NewAdminPeerManager creates a new admin peer manager
func NewAdminPeerManager(logger *slog.Logger) *AdminPeerManager {
	return &AdminPeerManager{
		authorizedAdminPeers:   make(map[string]bool),
		minReputationThreshold: 90.0,
		logger:                 logger,
	}
}

// AddAdminPeer adds a peer ID to the list of authorized admin peers
func (apm *AdminPeerManager) AddAdminPeer(peerID string) {
	apm.mu.Lock()
	apm.authorizedAdminPeers[peerID] = true
	apm.mu.Unlock()
	apm.logger.Info("Added admin peer", "peer_id", peerID)
}

// RemoveAdminPeer removes a peer ID from the list of authorized admin peers
func (apm *AdminPeerManager) RemoveAdminPeer(peerID string) {
	apm.mu.Lock()
	delete(apm.authorizedAdminPeers, peerID)
	apm.mu.Unlock()
	apm.logger.Info("Removed admin peer", "peer_id", peerID)
}

// IsAdminPeer checks if a peer ID is authorized as an admin peer
func (apm *AdminPeerManager) IsAdminPeer(peerID string) bool {
	apm.mu.RLock()
	result := apm.authorizedAdminPeers[peerID]
	apm.mu.RUnlock()
	apm.logger.Debug("IsAdminPeer check", "peer_id", peerID, "result", result)
	return result
}

// SetMinReputationThreshold sets the minimum reputation score required for admin access
func (apm *AdminPeerManager) SetMinReputationThreshold(threshold float64) {
	apm.mu.Lock()
	apm.minReputationThreshold = threshold
	apm.mu.Unlock()
	apm.logger.Info("Set minimum reputation threshold", "threshold", threshold)
}

// GetMinReputationThreshold returns the minimum reputation score required for admin access
func (apm *AdminPeerManager) GetMinReputationThreshold() float64 {
	apm.mu.RLock()
	defer apm.mu.RUnlock()
	return apm.minReputationThreshold
}

// IsAuthorizedAdmin checks if a peer is authorized for admin access based on multiple criteria
func (apm *AdminPeerManager) IsAuthorizedAdmin(peerID string, reputationScore float64, isConnected bool) bool {
	// Check if peer is explicitly authorized as an admin peer
	if apm.IsAdminPeer(peerID) {
		apm.logger.Debug("Peer is explicitly authorized as admin", "peer_id", peerID)
		return true
	}

	// Check if peer has sufficient reputation and is connected
	apm.mu.RLock()
	threshold := apm.minReputationThreshold
	apm.mu.RUnlock()
	if isConnected && reputationScore >= threshold {
		apm.logger.Debug("Peer authorized by reputation", "peer_id", peerID, "reputation", reputationScore, "threshold", threshold)
		return true
	}

	// Peer is not authorized for admin access
	apm.logger.Debug("Peer not authorized for admin access", "peer_id", peerID)
	return false
}
