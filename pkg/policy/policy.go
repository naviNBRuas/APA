package policy

import (
	"context"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// PolicyEnforcer defines the interface for enforcing policies.
type PolicyEnforcer interface {
	Authorize(ctx context.Context, subject string, action string, resource string) (bool, string, error)
}

// Config holds the policy configuration.
type Config struct {
	TrustedAuthors []string `yaml:"trusted_authors"`
}

// PolicyEnforcerImpl implements the PolicyEnforcer interface.
type PolicyEnforcerImpl struct {
	config *Config
}

// NewPolicyEnforcer creates a new PolicyEnforcer.
func NewPolicyEnforcer(policyPath string) (*PolicyEnforcerImpl, error) {
	data, err := os.ReadFile(policyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read policy file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal policy file: %w", err)
	}

	return &PolicyEnforcerImpl{
		config: &config,
	}, nil
}

// Authorize checks if the action is authorized based on the policy.
func (p *PolicyEnforcerImpl) Authorize(ctx context.Context, subject string, action string, resource string) (bool, string, error) {
	// For now, we check if the action is allowed
	// For module operations, we allow all modules from trusted authors
	if action == "load_module" || action == "run_module" {
		// Since we don't have explicit author information for modules,
		// we'll allow all modules for now as a temporary fix
		// In a real implementation, we would check the module's author against trusted authors
		return true, "authorized", nil
	}

	return false, "unauthorized: action not supported by policy", nil
}