package agent

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/naviNBRuas/APA/pkg/opa"
	"github.com/stretchr/testify/assert"
)

func TestCreateAuthzInput(t *testing.T) {
	// Create a logger
	logger := slog.Default()

	// Create a mock runtime with minimal setup for testing
	rt := &Runtime{
		logger: logger,
		adminPeerManager: NewAdminPeerManager(logger),
	}

	// Add a test admin peer
	testPeerID := "QmTestPeer"
	rt.adminPeerManager.AddAdminPeer(testPeerID)

	// Create a test HTTP request with peer ID header
	req := httptest.NewRequest(http.MethodGet, "/admin/health", nil)
	req.Header.Set("X-Peer-ID", testPeerID)

	// Test creating authz input
	input := rt.createAuthzInput(req)

	// Verify the input contains expected fields
	assert.Equal(t, http.MethodGet, input["method"])
	assert.Equal(t, "/admin/health", input["path"])
	assert.Equal(t, testPeerID, input["peer_id"])
	assert.Equal(t, true, input["peer_is_admin"])
}

func TestAdminAuthorization(t *testing.T) {
	// Create a logger
	logger := slog.Default()

	// Create a mock runtime with minimal setup for testing
	rt := &Runtime{
		logger: logger,
		adminPeerManager: NewAdminPeerManager(logger),
		adminPolicyEngine: opa.NewOPAPolicyEngine(),
	}

	// Add a test admin peer
	testPeerID := "QmTestPeer"
	rt.adminPeerManager.AddAdminPeer(testPeerID)

	// Create a test HTTP request with peer ID header
	req := httptest.NewRequest(http.MethodGet, "/admin/health", nil)
	req.Header.Set("X-Peer-ID", testPeerID)

	// Test creating authz input
	input := rt.createAuthzInput(req)

	// Verify the input contains expected fields
	assert.Equal(t, true, input["peer_is_admin"])

	// Test with a non-admin peer
	nonAdminPeerID := "QmNonAdminPeer"
	req.Header.Set("X-Peer-ID", nonAdminPeerID)
	input = rt.createAuthzInput(req)
	assert.Equal(t, false, input["peer_is_admin"])
}