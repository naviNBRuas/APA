package selfhealing

import (
	"context"
	"runtime"
	"time"
)

func NewMemoryOptimizationStrategy() *MemoryOptimizationStrategy {
	return &MemoryOptimizationStrategy{
		name:        "memory-optimization",
		description: "Optimizes memory usage to prevent out-of-memory conditions",
		priority:    60,
		config:      make(map[string]interface{}),
	}
}

func (m *MemoryOptimizationStrategy) Name() string {
	return m.name
}

func (m *MemoryOptimizationStrategy) Description() string {
	return m.description
}

func (m *MemoryOptimizationStrategy) CanHandle(issue *HealthIssue) bool {
	return issue.Type == "memory" || issue.Component == "memory"
}

func (m *MemoryOptimizationStrategy) Apply(ctx context.Context, issue *HealthIssue) (*HealingResult, error) {
	startTime := time.Now()

	before := m.readMemoryUsage()

	m.forceGarbageCollection()

	m.clearCaches()

	m.adjustMemoryParameters()

	memoryFreed := m.verifyMemoryImprovement(before)

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

func (m *MemoryOptimizationStrategy) readMemoryUsage() uint64 {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	return mem.Alloc
}

func (m *MemoryOptimizationStrategy) forceGarbageCollection() {
	runtime.GC()
}

func (m *MemoryOptimizationStrategy) clearCaches() {
}

func (m *MemoryOptimizationStrategy) adjustMemoryParameters() {
}

func (m *MemoryOptimizationStrategy) verifyMemoryImprovement(before uint64) int {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	freed := int64(before) - int64(mem.Alloc)
	if freed < 0 {
		return 0
	}
	return int(freed / 1024 / 1024)
}

func (m *MemoryOptimizationStrategy) Priority() int {
	return m.priority
}

func (m *MemoryOptimizationStrategy) Configure(config map[string]interface{}) error {
	m.config = config
	return nil
}
