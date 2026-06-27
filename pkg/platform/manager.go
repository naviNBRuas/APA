// Package platform provides advanced cross-platform compatibility and platform-specific optimizations.
package platform

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

// PlatformManager handles platform-specific optimizations and compatibility.
type PlatformManager struct {
	logger             *slog.Logger
	config             PlatformConfig
	detector           *PlatformDetector
	optimizer          *PlatformOptimizer
	compatibilityLayer *CompatibilityLayer
	resourceManager    *ResourceManager
	adaptationEngine   *AdaptationEngine

	mu             sync.RWMutex
	currentProfile *PlatformProfile
	isRunning      bool
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

// NewPlatformManager creates a new platform manager with advanced capabilities.
func NewPlatformManager(logger *slog.Logger, config PlatformConfig) (*PlatformManager, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	ctx, cancel := context.WithCancel(context.Background())

	pm := &PlatformManager{
		logger: logger,
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize components
	if err := pm.initializeComponents(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize platform manager components: %w", err)
	}

	logger.Info("Platform manager initialized successfully",
		"auto_detection", config.EnableAutoDetection,
		"optimizations", config.EnableOptimizations,
		"compatibility", config.EnableCompatibility)

	return pm, nil
}

// initializeComponents sets up all platform management components.
func (pm *PlatformManager) initializeComponents() error {
	var errs []error

	// Initialize platform detector
	pm.detector = NewPlatformDetector(pm.logger, 5*time.Minute)

	// Initialize optimizer with platform-specific strategies
	pm.optimizer = NewPlatformOptimizer(pm.logger, pm.config.OptimizationStrategies)

	// Initialize compatibility layer
	pm.compatibilityLayer = NewCompatibilityLayer(pm.logger, pm.config.CompatibilityOverrides)

	// Initialize resource manager
	pm.resourceManager = NewResourceManager(pm.logger, pm.config.ResourceLimits)

	// Initialize adaptation engine
	pm.adaptationEngine = NewAdaptationEngine(pm.logger, pm.config.AdaptationThresholds)

	// Detect initial platform profile
	if pm.config.EnableAutoDetection {
		profile, err := pm.detector.DetectPlatform()
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to detect platform: %w", err))
		} else {
			pm.currentProfile = profile
			pm.logger.Info("Platform profile detected",
				"os", profile.OS.Name,
				"architecture", profile.Architecture.Type,
				"confidence", profile.ConfidenceScore)
		}
	}

	// Apply initial optimizations
	if pm.config.EnableOptimizations && pm.currentProfile != nil {
		if err := pm.optimizer.ApplyOptimizations(pm.currentProfile); err != nil {
			errs = append(errs, fmt.Errorf("failed to apply optimizations: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("initialization errors: %v", errs)
	}

	return nil
}

// Start begins platform management operations.
func (pm *PlatformManager) Start() error {
	pm.mu.Lock()
	if pm.isRunning {
		pm.mu.Unlock()
		return fmt.Errorf("platform manager is already running")
	}
	pm.isRunning = true
	pm.mu.Unlock()

	pm.logger.Info("Starting platform management")

	// Start resource monitoring
	pm.wg.Add(1)
	go pm.resourceMonitoringLoop()

	// Start adaptation engine
	pm.wg.Add(1)
	go pm.adaptationLoop()

	// Start compatibility monitoring
	pm.wg.Add(1)
	go pm.compatibilityMonitoringLoop()

	return nil
}

// Stop gracefully shuts down platform management.
func (pm *PlatformManager) Stop() {
	pm.mu.Lock()
	if !pm.isRunning {
		pm.mu.Unlock()
		return
	}
	pm.isRunning = false
	pm.mu.Unlock()

	pm.logger.Info("Stopping platform management")

	// Cancel context to stop all goroutines
	pm.cancel()

	// Wait for all components to finish
	pm.wg.Wait()

	// Cleanup resources
	pm.cleanup()

	pm.logger.Info("Platform management stopped")
}

// cleanup releases all resources.
func (pm *PlatformManager) cleanup() {
	if pm.resourceManager != nil {
		pm.resourceManager.Shutdown()
	}

	if pm.adaptationEngine != nil {
		pm.adaptationEngine.Shutdown()
	}
}

// resourceMonitoringLoop continuously monitors system resources.
func (pm *PlatformManager) resourceMonitoringLoop() {
	defer pm.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			pm.monitorResources()
		}
	}
}

// monitorResources collects and analyzes resource metrics.
func (pm *PlatformManager) monitorResources() {
	if pm.resourceManager == nil {
		return
	}

	metrics, err := pm.collectResourceMetrics()
	if err != nil {
		pm.logger.Error("Failed to collect resource metrics", "error", err)
		return
	}

	pm.resourceManager.UpdateMetrics(metrics)

	// Check for resource alerts
	alerts := pm.resourceManager.CheckAlerts()
	for _, alert := range alerts {
		pm.handleResourceAlert(alert)
	}
}

// adaptationLoop handles dynamic platform adaptation.
func (pm *PlatformManager) adaptationLoop() {
	defer pm.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			pm.evaluatePlatformAdaptation()
		}
	}
}

// compatibilityMonitoringLoop monitors compatibility issues.
func (pm *PlatformManager) compatibilityMonitoringLoop() {
	defer pm.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			pm.checkCompatibilityIssues()
		}
	}
}

// GetPlatformProfile returns the current platform profile.
func (pm *PlatformManager) GetPlatformProfile() *PlatformProfile {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pm.currentProfile == nil {
		return nil
	}

	// Return a copy to prevent external modification
	profile := *pm.currentProfile
	return &profile
}

// ForcePlatformDetection forces immediate platform detection.
func (pm *PlatformManager) ForcePlatformDetection() (*PlatformProfile, error) {
	if pm.detector == nil {
		return nil, fmt.Errorf("platform detector not initialized")
	}

	profile, err := pm.detector.DetectPlatform()
	if err != nil {
		return nil, fmt.Errorf("platform detection failed: %w", err)
	}

	pm.mu.Lock()
	pm.currentProfile = profile
	pm.mu.Unlock()

	pm.logger.Info("Platform detection forced",
		"os", profile.OS.Name,
		"architecture", profile.Architecture.Type)

	return profile, nil
}

// ApplyPlatformOptimizations applies optimizations for the current platform.
func (pm *PlatformManager) ApplyPlatformOptimizations() error {
	pm.mu.RLock()
	profile := pm.currentProfile
	pm.mu.RUnlock()

	if profile == nil {
		return fmt.Errorf("no platform profile available")
	}

	if pm.optimizer == nil {
		return fmt.Errorf("platform optimizer not initialized")
	}

	return pm.optimizer.ApplyOptimizations(profile)
}

// HandleResourceAlert processes resource alerts.
func (pm *PlatformManager) handleResourceAlert(alert *ResourceAlert) {
	pm.logger.Warn("Resource alert triggered",
		"resource", alert.Resource,
		"level", alert.Level,
		"value", alert.Value,
		"threshold", alert.Threshold)

	// Trigger adaptation if critical
	if alert.Level == AlertCritical || alert.Level == AlertEmergency {
		go pm.evaluatePlatformAdaptation()
	}
}

// evaluatePlatformAdaptation determines if platform adaptation is needed.
func (pm *PlatformManager) evaluatePlatformAdaptation() {
	if pm.adaptationEngine == nil || pm.currentProfile == nil {
		return
	}

	metrics := pm.resourceManager.GetCurrentMetrics()
	if metrics == nil {
		return
	}

	adaptationNeeded, changes := pm.adaptationEngine.EvaluateAdaptation(metrics, pm.currentProfile)
	if adaptationNeeded {
		pm.logger.Info("Platform adaptation triggered", "changes", len(changes))

		// Apply adaptations
		for _, change := range changes {
			if err := pm.applyAdaptation(change); err != nil {
				pm.logger.Error("Failed to apply adaptation", "change", change, "error", err)
			}
		}

		// Update profile after adaptation
		if newProfile, err := pm.detector.DetectPlatform(); err == nil {
			pm.mu.Lock()
			pm.currentProfile = newProfile
			pm.mu.Unlock()
		}
	}
}

// applyAdaptation implements a specific adaptation change.
func (pm *PlatformManager) applyAdaptation(change string) error {
	pm.logger.Info("Applying platform adaptation", "change", change)

	switch change {
	case "increase_resources":
		return pm.resourceManager.ScaleUp()
	case "decrease_resources":
		return pm.resourceManager.ScaleDown()
	case "optimize_power":
		return pm.applyPowerOptimization()
	case "adjust_scheduling":
		return pm.applySchedulingOptimization()
	case "enable_compatibility_mode":
		return pm.compatibilityLayer.EnableCompatibilityMode()
	default:
		return fmt.Errorf("unknown adaptation: %s", change)
	}
}

// checkCompatibilityIssues scans for platform compatibility problems.
func (pm *PlatformManager) checkCompatibilityIssues() {
	if pm.compatibilityLayer == nil {
		return
	}

	issues := pm.compatibilityLayer.ScanForIssues()
	if len(issues) > 0 {
		pm.logger.Warn("Compatibility issues detected", "count", len(issues))
		for _, issue := range issues {
			pm.handleCompatibilityIssue(issue)
		}
	}
}

// handleCompatibilityIssue addresses a specific compatibility problem.
func (pm *PlatformManager) handleCompatibilityIssue(issue string) {
	pm.logger.Info("Handling compatibility issue", "issue", issue)

	// Apply appropriate patch or workaround
	if patch, exists := pm.compatibilityLayer.GetPatchForIssue(issue); exists {
		if err := pm.compatibilityLayer.ApplyPatch(patch); err != nil {
			pm.logger.Error("Failed to apply compatibility patch", "patch", patch.Name, "error", err)
		}
	}
}

// Helper methods for resource collection
func (pm *PlatformManager) collectResourceMetrics() (*ResourceMetrics, error) {
	// Collect CPU metrics
	cpuPercent, err := cpu.Percent(0, false)
	if err != nil {
		return nil, fmt.Errorf("failed to collect CPU metrics: %w", err)
	}

	// Collect memory metrics
	memStats, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to collect memory metrics: %w", err)
	}

	// Collect disk I/O metrics
	ioCounters, err := disk.IOCounters()
	if err != nil {
		return nil, fmt.Errorf("failed to collect disk I/O metrics: %w", err)
	}

	// Collect network metrics
	netIO, err := net.IOCounters(false)
	if err != nil {
		return nil, fmt.Errorf("failed to collect network metrics: %w", err)
	}

	// Collect system load
	loadAvg, err := load.Avg()
	if err != nil {
		// Not available on all platforms, use default values
		loadAvg = &load.AvgStat{Load1: 0, Load5: 0, Load15: 0}
	}

	var diskStats disk.IOCountersStat
	for _, v := range ioCounters {
		diskStats = v
		break
	}

	var netStats net.IOCountersStat
	if len(netIO) > 0 {
		netStats = netIO[0]
	}

	cpuUsage := 0.0
	if len(cpuPercent) > 0 {
		cpuUsage = cpuPercent[0]
	}

	metrics := &ResourceMetrics{
		CPUUsage:    cpuUsage,
		MemoryUsage: memStats.UsedPercent,
		DiskIO: DiskIOMetrics{
			ReadBytes:  diskStats.ReadBytes,
			WriteBytes: diskStats.WriteBytes,
			ReadOps:    diskStats.ReadCount,
			WriteOps:   diskStats.WriteCount,
		},
		NetworkIO: NetworkMetrics{
			BytesSent:   netStats.BytesSent,
			BytesRecv:   netStats.BytesRecv,
			PacketsSent: netStats.PacketsSent,
			PacketsRecv: netStats.PacketsRecv,
		},
		LoadAverage: LoadAverage{
			OneMinute:      loadAvg.Load1,
			FiveMinutes:    loadAvg.Load5,
			FifteenMinutes: loadAvg.Load15,
		},
	}

	return metrics, nil
}

// Utility methods for platform-specific optimizations
func (pm *PlatformManager) applyPowerOptimization() error {
	// Implementation would adjust CPU governor, frequency scaling, etc.
	pm.logger.Info("Applying power optimization strategy")
	return nil
}

func (pm *PlatformManager) applySchedulingOptimization() error {
	// Implementation would adjust thread affinity, process priorities, etc.
	pm.logger.Info("Applying scheduling optimization strategy")
	return nil
}
