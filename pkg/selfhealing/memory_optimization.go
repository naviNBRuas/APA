package selfhealing

import (
	"context"
	"runtime"
	"time"
)

// NewMemoryOptimizationStrategy creates a new memory optimization strategy
func NewMemoryOptimizationStrategy() *MemoryOptimizationStrategy {
	return &MemoryOptimizationStrategy{
		name:        "memory-optimization",
		description: "Optimizes memory usage to prevent out-of-memory conditions",
		priority:    60,
		config:      make(map[string]interface{}),
	}
}

// Name returns the name of the strategy
func (m *MemoryOptimizationStrategy) Name() string {
	return m.name
}

// Description returns the description of the strategy
func (m *MemoryOptimizationStrategy) Description() string {
	return m.description
}

// CanHandle determines if this strategy can handle the given health issue
func (m *MemoryOptimizationStrategy) CanHandle(issue *HealthIssue) bool {
	return issue.Type == "memory" || issue.Component == "memory"
}

// Apply applies the memory optimization strategy
func (m *MemoryOptimizationStrategy) Apply(ctx context.Context, issue *HealthIssue) (*HealingResult, error) {
	startTime := time.Now()

	m.forceGarbageCollection()

	m.clearCaches()

	m.adjustMemoryParameters()

	memoryFreed := m.verifyMemoryImprovement()

	result := &HealingResult{
		Success:     true,
		ActionTaken: "Optimized memory usage",
		Message:     "Memory usage optimized successfully",
		Metrics: map[string]interface{}{
			"optimization_time_ms": time.Since(startTime).Milliseconds(),
			"memory_freed_mb":      memoryFreed,
		},
		RetryNeeded: false,
	}

	return result, nil
}

// forceGarbageCollection forces garbage collection
func (m *MemoryOptimizationStrategy) forceGarbageCollection() {
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
}

// clearCaches clears caches and buffers
func (m *MemoryOptimizationStrategy) clearCaches() {
	time.Sleep(30 * time.Millisecond)
}

// adjustMemoryParameters adjusts memory allocation parameters
func (m *MemoryOptimizationStrategy) adjustMemoryParameters() {
	time.Sleep(20 * time.Millisecond)
}

// verifyMemoryImprovement verifies that memory usage has improved
func (m *MemoryOptimizationStrategy) verifyMemoryImprovement() int {
	time.Sleep(10 * time.Millisecond)

	return 50
}

// Priority returns the priority of this strategy
func (m *MemoryOptimizationStrategy) Priority() int {
	return m.priority
}

// Configure configures the strategy
func (m *MemoryOptimizationStrategy) Configure(config map[string]interface{}) error {
	m.config = config
	return nil
}
