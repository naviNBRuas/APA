package platform

import (
	"log/slog"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

type AdaptationEngine struct {
	logger     *slog.Logger
	thresholds AdaptationThresholds
	triggers   []AdaptationTrigger
	history    []*AdaptationEvent
	strategy   AdaptationStrategy
}

func NewAdaptationEngine(logger *slog.Logger, thresholds AdaptationThresholds) *AdaptationEngine {
	return &AdaptationEngine{
		logger:     logger,
		thresholds: thresholds,
		triggers:   make([]AdaptationTrigger, 0),
		history:    make([]*AdaptationEvent, 0),
		strategy:   AdaptationStrategy{AdaptationMode: "reactive"},
	}
}

func (ae *AdaptationEngine) EvaluateAdaptation(metrics *ResourceMetrics, profile *PlatformProfile) (bool, []string) {
	var reasons []string
	vm, err := mem.VirtualMemory()
	if err == nil && ae.thresholds.MemoryPressureThreshold > 0 {
		if vm.UsedPercent > ae.thresholds.MemoryPressureThreshold {
			reasons = append(reasons, "memory_pressure")
		}
	}
	cpuPercent, err := cpu.Percent(0, false)
	if err == nil && len(cpuPercent) > 0 && ae.thresholds.CPULoadThreshold > 0 && cpuPercent[0] > ae.thresholds.CPULoadThreshold {
		reasons = append(reasons, "cpu_pressure")
	}
	if runtime.NumGoroutine() > 10000 {
		reasons = append(reasons, "high_goroutine_count")
	}
	if len(reasons) > 0 {
		ae.history = append(ae.history, &AdaptationEvent{
			Timestamp: time.Now(),
			Trigger:   "resource_pressure",
		})
	}
	return len(reasons) > 0, reasons
}

func (ae *AdaptationEngine) Shutdown() {
	ae.logger.Info("Adaptation engine shutting down")
}
