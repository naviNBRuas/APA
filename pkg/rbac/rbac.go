package rbac

import (
	"context"
	"fmt"
	"log/slog"
)

// RBAC defines the interface for Role-Based Access Control
type RBAC interface {
	// Authorize checks if a user has permission to perform an action on a resource
	Authorize(ctx context.Context, user string, action string, resource string) (bool, string, error)
	
	// AddRole adds a new role to the system
	AddRole(role string) error
	
	// AssignRole assigns a role to a user
	AssignRole(user string, role string) error
	
	// AddPermission adds a permission to a role
	AddPermission(role string, action string, resource string) error
	
	// RemoveRole removes a role from the system
	RemoveRole(role string) error
	
	// RevokeRole revokes a role from a user
	RevokeRole(user string, role string) error
	
	// RemovePermission removes a permission from a role
	RemovePermission(role string, action string, resource string) error
}

// Config holds configuration for the RBAC system
type Config struct {
	PolicyFile string `yaml:"policy_file"`
}

// RBACImpl implements the RBAC interface using OPA/Rego
type RBACImpl struct {
	logger *slog.Logger
	config *Config
	roles  map[string]map[string]map[string]bool // role -> action -> resource -> allowed
	users  map[string][]string                   // user -> roles
}

// NewRBAC creates a new RBAC implementation
func NewRBAC(logger *slog.Logger, config *Config) *RBACImpl {
	return &RBACImpl{
		logger: logger,
		config: config,
		roles:  make(map[string]map[string]map[string]bool),
		users:  make(map[string][]string),
	}
}

// Authorize checks if a user has permission to perform an action on a resource
func (r *RBACImpl) Authorize(ctx context.Context, user string, action string, resource string) (bool, string, error) {
	// Get the user's roles
	roles, exists := r.users[user]
	if !exists {
		return false, "user not found", nil
	}
	
	// Check each role for the required permission
	for _, role := range roles {
		// Check if the role exists
		actions, roleExists := r.roles[role]
		if !roleExists {
			continue
		}
		
		// Check if the action exists for this role
		resources, actionExists := actions[action]
		if !actionExists {
			continue
		}
		
		// Check if the resource is allowed for this action
		if allowed, resourceExists := resources[resource]; resourceExists && allowed {
			return true, "authorized", nil
		}
		
		// Check for wildcard resource permissions
		if allowed, wildcardExists := resources["*"]; wildcardExists && allowed {
			return true, "authorized", nil
		}
	}
	
	return false, "permission denied", nil
}

// AddRole adds a new role to the system
func (r *RBACImpl) AddRole(role string) error {
	if _, exists := r.roles[role]; exists {
		return fmt.Errorf("role %s already exists", role)
	}
	
	r.roles[role] = make(map[string]map[string]bool)
	r.logger.Info("Added role", "role", role)
	return nil
}

// AssignRole assigns a role to a user
func (r *RBACImpl) AssignRole(user string, role string) error {
	// Check if the role exists
	if _, exists := r.roles[role]; !exists {
		return fmt.Errorf("role %s does not exist", role)
	}
	
	// Check if the user already has this role
	roles, userExists := r.users[user]
	if userExists {
		for _, r := range roles {
			if r == role {
				return fmt.Errorf("user %s already has role %s", user, role)
			}
		}
	}
	
	// Assign the role to the user
	r.users[user] = append(r.users[user], role)
	r.logger.Info("Assigned role to user", "user", user, "role", role)
	return nil
}

// AddPermission adds a permission to a role
func (r *RBACImpl) AddPermission(role string, action string, resource string) error {
	// Check if the role exists
	actions, roleExists := r.roles[role]
	if !roleExists {
		return fmt.Errorf("role %s does not exist", role)
	}
	
	// Create the action map if it doesn't exist
	if _, actionExists := actions[action]; !actionExists {
		actions[action] = make(map[string]bool)
	}
	
	// Add the permission
	actions[action][resource] = true
	r.logger.Info("Added permission to role", "role", role, "action", action, "resource", resource)
	return nil
}

// RemoveRole removes a role from the system
func (r *RBACImpl) RemoveRole(role string) error {
	if _, exists := r.roles[role]; !exists {
		return fmt.Errorf("role %s does not exist", role)
	}
	
	delete(r.roles, role)
	r.logger.Info("Removed role", "role", role)
	return nil
}

// RevokeRole revokes a role from a user
func (r *RBACImpl) RevokeRole(user string, role string) error {
	roles, userExists := r.users[user]
	if !userExists {
		return fmt.Errorf("user %s does not exist", user)
	}
	
	for i, ro := range roles {
		if ro == role {
			// Remove the role from the user's roles
			r.users[user] = append(roles[:i], roles[i+1:]...)
			r.logger.Info("Revoked role from user", "user", user, "role", role)
			return nil
		}
	}
	
	return fmt.Errorf("user %s does not have role %s", user, role)
}

// RemovePermission removes a permission from a role
func (r *RBACImpl) RemovePermission(role string, action string, resource string) error {
	// Check if the role exists
	actions, roleExists := r.roles[role]
	if !roleExists {
		return fmt.Errorf("role %s does not exist", role)
	}
	
	// Check if the action exists for this role
	resources, actionExists := actions[action]
	if !actionExists {
		return fmt.Errorf("action %s does not exist for role %s", action, role)
	}
	
	// Check if the resource exists for this action
	if _, resourceExists := resources[resource]; !resourceExists {
		return fmt.Errorf("resource %s does not exist for action %s in role %s", resource, action, role)
	}
	
	// Remove the permission
	delete(resources, resource)
	r.logger.Info("Removed permission from role", "role", role, "action", action, "resource", resource)
	return nil
}