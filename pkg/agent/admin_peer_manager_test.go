package agent

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdminPeerManager(t *testing.T) {
	// Create a logger
	logger := slog.Default()

	// Create an admin peer manager
	apm := NewAdminPeerManager(logger)

	// Test that we can create an admin peer manager
	assert.NotNil(t, apm)

	// Test adding admin peers
	testPeer1 := "QmTestPeer1"
	testPeer2 := "QmTestPeer2"

	apm.AddAdminPeer(testPeer1)
	apm.AddAdminPeer(testPeer2)

	// Test checking if a peer is an admin peer
	assert.True(t, apm.IsAdminPeer(testPeer1))
	assert.True(t, apm.IsAdminPeer(testPeer2))
	assert.False(t, apm.IsAdminPeer("QmNonAdminPeer"))

	// Test removing admin peers
	apm.RemoveAdminPeer(testPeer1)
	assert.False(t, apm.IsAdminPeer(testPeer1))
	assert.True(t, apm.IsAdminPeer(testPeer2))

	// Test setting and getting minimum reputation threshold
	apm.SetMinReputationThreshold(85.0)
	assert.Equal(t, 85.0, apm.GetMinReputationThreshold())

	// Test authorized admin check with reputation score
	assert.True(t, apm.IsAuthorizedAdmin(testPeer2, 95.0, true))  // High reputation, connected (explicitly authorized)
	assert.True(t, apm.IsAuthorizedAdmin(testPeer2, 85.0, true))  // Exact threshold, connected (explicitly authorized)
	assert.True(t, apm.IsAuthorizedAdmin(testPeer2, 80.0, true)) // Below threshold, connected (explicitly authorized)
	assert.True(t, apm.IsAuthorizedAdmin(testPeer2, 95.0, false)) // High reputation, not connected (explicitly authorized)
	assert.True(t, apm.IsAuthorizedAdmin(testPeer2, 95.0, true))  // High reputation, connected (explicitly authorized)
}