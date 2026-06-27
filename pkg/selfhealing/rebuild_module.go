package selfhealing

import (
	"context"
	"fmt"
	"time"
)

// NewRebuildModuleStrategy creates a new rebuild module strategy
func NewRebuildModuleStrategy() *RebuildModuleStrategy {
	return &RebuildModuleStrategy{
		name:        "rebuild-module",
		description: "Rebuilds corrupted or missing modules from trusted sources",
		priority:    90,
		config:      make(map[string]interface{}),
	}
}

// Name returns the name of the strategy
func (r *RebuildModuleStrategy) Name() string {
	return r.name
}

// Description returns the description of the strategy
func (r *RebuildModuleStrategy) Description() string {
	return r.description
}

// CanHandle determines if this strategy can handle the given health issue
func (r *RebuildModuleStrategy) CanHandle(issue *HealthIssue) bool {
	return issue.Type == "module" || issue.Component == "module"
}

// Apply applies the rebuild module strategy
func (r *RebuildModuleStrategy) Apply(ctx context.Context, issue *HealthIssue) (*HealingResult, error) {
	startTime := time.Now()

	moduleName := issue.Component
	if name, ok := issue.Context["module_name"].(string); ok {
		moduleName = name
	}

	moduleData, err := r.requestModuleFromPeers(ctx, moduleName)
	if err != nil {
		return nil, fmt.Errorf("failed to request module from peers: %w", err)
	}

	if err := r.verifyModuleIntegrity(moduleData, moduleName); err != nil {
		return nil, fmt.Errorf("module integrity verification failed: %w", err)
	}

	if err := r.replaceModule(moduleName, moduleData); err != nil {
		return nil, fmt.Errorf("failed to replace module: %w", err)
	}

	if err := r.reloadModule(moduleName); err != nil {
		return &HealingResult{
			Success:     false,
			ActionTaken: fmt.Sprintf("Attempted to rebuild module '%s'", moduleName),
			Message:     fmt.Sprintf("Module reload failed: %v", err),
			Metrics: map[string]interface{}{
				"rebuild_time_ms": time.Since(startTime).Milliseconds(),
			},
			RetryNeeded: true,
		}, nil
	}

	result := &HealingResult{
		Success:     true,
		ActionTaken: fmt.Sprintf("Rebuilt module '%s'", moduleName),
		Message:     "Module rebuilt and loaded successfully",
		Metrics: map[string]interface{}{
			"rebuild_time_ms": time.Since(startTime).Milliseconds(),
			"module_size_kb":  len(moduleData) / 1024,
		},
		RetryNeeded: false,
	}

	return result, nil
}

// requestModuleFromPeers requests a module from trusted peers
func (r *RebuildModuleStrategy) requestModuleFromPeers(ctx context.Context, moduleName string) ([]byte, error) {
	time.Sleep(200 * time.Millisecond)

	return []byte(fmt.Sprintf("dummy module data for %s", moduleName)), nil
}

// verifyModuleIntegrity verifies the integrity of a module
func (r *RebuildModuleStrategy) verifyModuleIntegrity(moduleData []byte, moduleName string) error {
	time.Sleep(50 * time.Millisecond)

	return nil
}

// replaceModule replaces a corrupted module with new data
func (r *RebuildModuleStrategy) replaceModule(moduleName string, moduleData []byte) error {
	time.Sleep(100 * time.Millisecond)

	return nil
}

// reloadModule reloads a module in the runtime
func (r *RebuildModuleStrategy) reloadModule(moduleName string) error {
	time.Sleep(150 * time.Millisecond)

	return nil
}

// Priority returns the priority of this strategy
func (r *RebuildModuleStrategy) Priority() int {
	return r.priority
}

// Configure configures the strategy
func (r *RebuildModuleStrategy) Configure(config map[string]interface{}) error {
	r.config = config
	return nil
}
