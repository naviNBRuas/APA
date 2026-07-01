package platform

import (
	"log/slog"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

type ResourceManager struct {
	logger       *slog.Logger
	limits       ResourceLimits
	monitor      *ResourceMonitor
	allocation   *ResourceAllocator
	optimization *ResourceOptimizer
}

func NewResourceManager(logger *slog.Logger, limits ResourceLimits) *ResourceManager {
	return &ResourceManager{
		logger: logger,
		limits: limits,
		monitor: &ResourceMonitor{
			logger:       logger,
			samplingRate: 1 * time.Second,
			metrics:      &ResourceMetrics{},
			alerts:       make(chan *ResourceAlert, 100),
		},
		allocation: &ResourceAllocator{
			logger:             logger,
			policies:           make(map[string]AllocationPolicy),
			currentAllocations: make(map[string]*ResourceAllocation),
		},
		optimization: &ResourceOptimizer{
			logger:     logger,
			models:     make(map[string]*OptimizationModel),
			strategies: make(map[string]OptimizationStrategy),
		},
	}
}

func (rm *ResourceManager) UpdateMetrics(metrics *ResourceMetrics) {
	rm.monitor.metrics = metrics
}

func (rm *ResourceManager) CheckAlerts() []*ResourceAlert {
	var alerts []*ResourceAlert
	vm, err := mem.VirtualMemory()
	if err == nil && rm.limits.MaxMemoryMB > 0 {
		usedMB := vm.Used / 1024 / 1024
		if usedMB > rm.limits.MaxMemoryMB {
			alerts = append(alerts, &ResourceAlert{
				Resource:  "memory",
				Message:   "Memory usage exceeded limit",
				Timestamp: time.Now(),
			})
		}
	}
	cpuPercent, err := cpu.Percent(0, false)
	if err == nil && len(cpuPercent) > 0 && rm.limits.MaxCPUPercent > 0 && cpuPercent[0] > rm.limits.MaxCPUPercent {
		alerts = append(alerts, &ResourceAlert{
			Resource:  "cpu",
			Message:   "CPU usage exceeded limit",
			Timestamp: time.Now(),
		})
	}
	return alerts
}

func (rm *ResourceManager) GetCurrentMetrics() *ResourceMetrics {
	vm, _ := mem.VirtualMemory()
	cpuPercent, _ := cpu.Percent(0, false)
	metrics := &ResourceMetrics{
		MemoryUsage: vm.UsedPercent,
	}
	if len(cpuPercent) > 0 {
		metrics.CPUUsage = cpuPercent[0]
	}
	return metrics
}

func (rm *ResourceManager) ScaleUp() error {
	rm.logger.Info("Scaling up resource allocation")
	return nil
}

func (rm *ResourceManager) ScaleDown() error {
	rm.logger.Info("Scaling down resource allocation")
	return nil
}

func (rm *ResourceManager) Shutdown() {
	rm.logger.Info("Resource manager shutting down")
}
