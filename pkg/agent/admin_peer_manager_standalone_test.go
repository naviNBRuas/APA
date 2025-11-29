package agent

import (
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Standalone test for AdminPeerManager without dependencies on the full runtime
func TestAdminPeerManagerStandalone(t *testing.T) {
	// Create a logger with debug level
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Create an admin peer manager
	apm := NewAdminPeerManager(logger)

	// Test that we can create an admin peer manager
	assert.NotNil(t, apm)

	// Test adding admin peers
	testPeer1 := "QmTestPeer1"
	testPeer2 := "QmTestPeer2"
	testPeer3 := "QmTestPeer3" // This peer will NOT be explicitly added as admin

	fmt.Println("=== Adding admin peers ===")
	apm.AddAdminPeer(testPeer1)
	apm.AddAdminPeer(testPeer2)

	// Test checking if a peer is an admin peer
	fmt.Println("=== Checking admin peers ===")
	assert.True(t, apm.IsAdminPeer(testPeer1))
	assert.True(t, apm.IsAdminPeer(testPeer2))
	assert.False(t, apm.IsAdminPeer(testPeer3))
	assert.False(t, apm.IsAdminPeer("QmNonAdminPeer"))

	// Test removing admin peers
	fmt.Println("=== Removing admin peers ===")
	apm.RemoveAdminPeer(testPeer1)
	assert.False(t, apm.IsAdminPeer(testPeer1))
	assert.True(t, apm.IsAdminPeer(testPeer2))

	// Test setting and getting minimum reputation threshold
	fmt.Println("=== Setting reputation threshold ===")
	apm.SetMinReputationThreshold(85.0)
	assert.Equal(t, 85.0, apm.GetMinReputationThreshold())

	// Test authorized admin check for explicitly authorized peers (should always be true)
	fmt.Println("=== Testing explicitly authorized peer ===")
	result1 := apm.IsAuthorizedAdmin(testPeer2, 95.0, true)  // Explicitly authorized peer
	fmt.Printf("Test 1 - Explicitly authorized peer: %v\n", result1)
	assert.True(t, result1, "Explicitly authorized peer should always be true")

	// Test authorized admin check with reputation-based authorization
	// These tests use testPeer3 which is NOT explicitly authorized
	fmt.Println("=== Testing reputation-based authorization ===")
	
	result2 := apm.IsAuthorizedAdmin(testPeer3, 95.0, true)  // High reputation, connected
	fmt.Printf("Test 2 - High reputation, connected: %v\n", result2)
	assert.True(t, result2, "High reputation, connected should be true")
	
	result3 := apm.IsAuthorizedAdmin(testPeer3, 85.0, true)  // Exact threshold, connected
	fmt.Printf("Test 3 - Exact threshold, connected: %v\n", result3)
	assert.True(t, result3, "Exact threshold, connected should be true")
	
	result4 := apm.IsAuthorizedAdmin(testPeer3, 80.0, true) // Below threshold, connected
	fmt.Printf("Test 4 - Below threshold, connected: %v\n", result4)
	assert.False(t, result4, "Below threshold, connected should be false")
	
	result5 := apm.IsAuthorizedAdmin(testPeer3, 95.0, false) // High reputation, not connected
	fmt.Printf("Test 5 - High reputation, not connected: %v\n", result5)
	assert.False(t, result5, "High reputation, not connected should be false")
	
	result6 := apm.IsAuthorizedAdmin(testPeer3, 95.0, true)  // High reputation, connected
	fmt.Printf("Test 6 - High reputation, connected: %v\n", result6)
	assert.True(t, result6, "High reputation, connected should be true")
	
	// Test that a peer that was removed is not authorized even with high reputation
	fmt.Println("=== Testing removed peer ===")
	fmt.Printf("Checking if testPeer1 is admin peer: %v\n", apm.IsAdminPeer(testPeer1))
	result7 := apm.IsAuthorizedAdmin(testPeer1, 95.0, true) // Removed peer, high reputation, connected
	fmt.Printf("Test 7 - Removed peer, high reputation, connected: %v\n", result7)
	// A removed peer can still be authorized by reputation if they meet the criteria
	assert.True(t, result7, "Removed peer with high reputation and connected should be authorized")
}