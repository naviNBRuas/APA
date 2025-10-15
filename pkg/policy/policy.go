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
func (p *PolicyEnforcerImpl) Authorize(ctx context.Context, author string, action string, resource string) (bool, string, error) {
	// For now, we only check if the author of a module is trusted.
	if action == "load_module" {
		for _, trustedAuthor := range p.config.TrustedAuthors {
			if author == trustedAuthor {
				return true, "authorized", nil
			}
		}
		return false, "unauthorized: author not trusted", nil
	}

	return false, "unauthorized: action not supported by policy", nil
}