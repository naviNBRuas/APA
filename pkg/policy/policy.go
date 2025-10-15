package policy

import (
	"context"
)

// PolicyEnforcer defines the interface for enforcing policies.
type PolicyEnforcer interface {
	Authorize(ctx context.Context, token string, action string, resource string) (bool, string, error)
}

// AuthService defines the interface for authenticating and authorizing peers.
type AuthService interface {
	Authenticate(ctx context.Context, peerID string, credentials []byte) (string, error)
	AuthorizeConnection(ctx context.Context, peerID string, role string) (bool, error)
}

// DummyPolicyEnforcer is a placeholder implementation of PolicyEnforcer.
// It currently only checks for a static token.
type DummyPolicyEnforcer struct{}

// Authorize checks if the token is "super-secret-token".
func (d *DummyPolicyEnforcer) Authorize(ctx context.Context, token string, action string, resource string) (bool, string, error) {
	if token == "super-secret-token" {
		return true, "authorized", nil
	}
	return false, "unauthorized", nil
}