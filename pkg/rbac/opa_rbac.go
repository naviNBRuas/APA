package rbac

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"gopkg.in/yaml.v3"
)

// OPARBAC implements the RBAC interface using OPA/Rego policies
type OPARBAC struct {
	logger *slog.Logger
	config *Config
	roles  map[string]map[string]map[string]bool // role -> action -> resource -> allowed
	users  map[string][]string                   // user -> roles
}

// Role represents a role in the RBAC system
type Role struct {
	Name        string              `yaml:"name"`
	Permissions []Permission        `yaml:"permissions"`
}

// Permission represents a permission in the RBAC system
type Permission struct {
	Action   string   `yaml:"action"`
	Resource string   `yaml:"resource"`
}

// Policy represents the RBAC policy
type Policy struct {
	Roles []Role `yaml:"roles"`
	Users []User `yaml:"users"`
}

// User represents a user in the RBAC system
type User struct {
	Name  string   `yaml:"name"`
	Roles []string `yaml:"roles"`
}

// NewOPARBAC creates a new OPA-based RBAC implementation
func NewOPARBAC(logger *slog.Logger, config *Config) (*OPARBAC, error) {
	rb := &OPARBAC{
		logger: logger,
		config: config,
		roles:  make(map[string]map[string]map[string]bool),
		users:  make(map[string][]string),
	}
	
	// Load the policy from file if specified
	if config.PolicyFile != "" {
		if err := rb.loadPolicyFromFile(config.PolicyFile); err != nil {
			return nil, fmt.Errorf("failed to load policy from file: %w", err)
		}
	}
	
	return rb, nil
}

// loadPolicyFromFile loads the RBAC policy from a YAML file
func (rb *OPARBAC) loadPolicyFromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read policy file: %w", err)
	}
	
	var policy Policy
	if err := yaml.Unmarshal(data, &policy); err != nil {
		return fmt.Errorf("failed to unmarshal policy file: %w", err)
	}
	
	// Load roles
	for _, role := range policy.Roles {
		if _, exists := rb.roles[role.Name]; !exists {
			rb.roles[role.Name] = make(map[string]map[string]bool)
		}
		
		for _, perm := range role.Permissions {
			if _, actionExists := rb.roles[role.Name][perm.Action]; !actionExists {
				rb.roles[role.Name][perm.Action] = make(map[string]bool)
			}
			rb.roles[role.Name][perm.Action][perm.Resource] = true
		}
	}
	
	// Load users
	for _, user := range policy.Users {
		rb.users[user.Name] = user.Roles
	}
	
	rb.logger.Info("Loaded RBAC policy from file", "filename", filename)
	return nil
}

// Authorize checks if a user has permission to perform an action on a resource
func (rb *OPARBAC) Authorize(ctx context.Context, user string, action string, resource string) (bool, string, error) {
	// Get the user's roles
	roles, exists := rb.users[user]
	if !exists {
		return false, "user not found", nil
	}
	
	// Check each role for the required permission
	for _, role := range roles {
		// Check if the role exists
		actions, roleExists := rb.roles[role]
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
func (rb *OPARBAC) AddRole(role string) error {
	if _, exists := rb.roles[role]; exists {
		return fmt.Errorf("role %s already exists", role)
	}
	
	rb.roles[role] = make(map[string]map[string]bool)
	rb.logger.Info("Added role", "role", role)
	return nil
}

// AssignRole assigns a role to a user
func (rb *OPARBAC) AssignRole(user string, role string) error {
	// Check if the role exists
	if _, exists := rb.roles[role]; !exists {
		return fmt.Errorf("role %s does not exist", role)
	}
	
	// Check if the user already has this role
	roles, userExists := rb.users[user]
	if userExists {
		for _, r := range roles {
			if r == role {
				return fmt.Errorf("user %s already has role %s", user, role)
			}
		}
	}
	
	// Assign the role to the user
	rb.users[user] = append(rb.users[user], role)
	rb.logger.Info("Assigned role to user", "user", user, "role", role)
	return nil
}

// AddPermission adds a permission to a role
func (rb *OPARBAC) AddPermission(role string, action string, resource string) error {
	// Check if the role exists
	actions, roleExists := rb.roles[role]
	if !roleExists {
		return fmt.Errorf("role %s does not exist", role)
	}
	
	// Create the action map if it doesn't exist
	if _, actionExists := actions[action]; !actionExists {
		actions[action] = make(map[string]bool)
	}
	
	// Add the permission
	actions[action][resource] = true
	rb.logger.Info("Added permission to role", "role", role, "action", action, "resource", resource)
	return nil
}

// RemoveRole removes a role from the system
func (rb *OPARBAC) RemoveRole(role string) error {
	if _, exists := rb.roles[role]; !exists {
		return fmt.Errorf("role %s does not exist", role)
	}
	
	delete(rb.roles, role)
	rb.logger.Info("Removed role", "role", role)
	return nil
}

// RevokeRole revokes a role from a user
func (rb *OPARBAC) RevokeRole(user string, role string) error {
	roles, userExists := rb.users[user]
	if !userExists {
		return fmt.Errorf("user %s does not exist", user)
	}
	
	for i, r := range roles {
		if r == role {
			// Remove the role from the user's roles
			rb.users[user] = append(roles[:i], roles[i+1:]...)
			rb.logger.Info("Revoked role from user", "user", user, "role", role)
			return nil
		}
	}
	
	return fmt.Errorf("user %s does not have role %s", user, role)
}

// RemovePermission removes a permission from a role
func (rb *OPARBAC) RemovePermission(role string, action string, resource string) error {
	// Check if the role exists
	actions, roleExists := rb.roles[role]
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
	rb.logger.Info("Removed permission from role", "role", role, "action", action, "resource", resource)
	return nil
}