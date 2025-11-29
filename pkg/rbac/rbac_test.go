package rbac

import (
	"context"
	"log/slog"
	"testing"
)

// TestRBAC tests the basic functionality of the RBAC implementation
func TestRBAC(t *testing.T) {
	// Create a logger
	logger := slog.Default()
	
	// Create an RBAC config
	config := &Config{
		PolicyFile: "",
	}
	
	// Create an RBAC implementation
	rb := NewRBAC(logger, config)
	
	// Test that we can create an RBAC implementation
	if rb == nil {
		t.Error("Failed to create RBAC implementation")
	}
	
	// Test adding a role
	role := "admin"
	if err := rb.AddRole(role); err != nil {
		t.Errorf("Failed to add role: %v", err)
	}
	
	// Test adding a duplicate role
	if err := rb.AddRole(role); err == nil {
		t.Error("Should have failed to add duplicate role")
	}
	
	// Test adding a permission to the role
	action := "read"
	resource := "file"
	if err := rb.AddPermission(role, action, resource); err != nil {
		t.Errorf("Failed to add permission: %v", err)
	}
	
	// Test assigning a role to a user
	user := "alice"
	if err := rb.AssignRole(user, role); err != nil {
		t.Errorf("Failed to assign role: %v", err)
	}
	
	// Test assigning a duplicate role to a user
	if err := rb.AssignRole(user, role); err == nil {
		t.Error("Should have failed to assign duplicate role to user")
	}
	
	// Test authorizing a user with permission
	ctx := context.Background()
	allowed, reason, err := rb.Authorize(ctx, user, action, resource)
	if err != nil {
		t.Errorf("Failed to authorize user: %v", err)
	}
	
	if !allowed {
		t.Errorf("User should be authorized, but got reason: %s", reason)
	}
	
	// Test authorizing a user without permission
	action2 := "write"
	allowed, reason, err = rb.Authorize(ctx, user, action2, resource)
	if err != nil {
		t.Errorf("Failed to authorize user: %v", err)
	}
	
	if allowed {
		t.Error("User should not be authorized")
	}
	
	// Test authorizing a non-existent user
	allowed, reason, err = rb.Authorize(ctx, "bob", action, resource)
	if err != nil {
		t.Errorf("Failed to authorize user: %v", err)
	}
	
	if allowed {
		t.Error("Non-existent user should not be authorized")
	}
	
	// Test revoking a role from a user
	if err := rb.RevokeRole(user, role); err != nil {
		t.Errorf("Failed to revoke role: %v", err)
	}
	
	// Test revoking a role from a user who doesn't have it
	if err := rb.RevokeRole(user, role); err == nil {
		t.Error("Should have failed to revoke role from user who doesn't have it")
	}
	
	// Test removing a permission from a role
	if err := rb.RemovePermission(role, action, resource); err != nil {
		t.Errorf("Failed to remove permission: %v", err)
	}
	
	// Test removing a role
	if err := rb.RemoveRole(role); err != nil {
		t.Errorf("Failed to remove role: %v", err)
	}
	
	// Test removing a non-existent role
	if err := rb.RemoveRole(role); err == nil {
		t.Error("Should have failed to remove non-existent role")
	}
}