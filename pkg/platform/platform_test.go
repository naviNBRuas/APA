package platform

import (
	"io"
	"log/slog"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestNewPlatformDetector(t *testing.T) {
	pd := NewPlatformDetector(testLogger(), time.Minute)
	require.NotNil(t, pd)
	assert.Equal(t, time.Minute, pd.cacheExpiry)
}

func TestDetectCurrentPlatform(t *testing.T) {
	pt := detectCurrentPlatform()

	// Map the current GOOS/GOARCH to expected PlatformType
	var expected PlatformType
	switch {
	case runtime.GOOS == "linux" && runtime.GOARCH == "amd64":
		expected = PlatformLinuxAMD64
	case runtime.GOOS == "linux" && runtime.GOARCH == "arm64":
		expected = PlatformLinuxARM64
	case runtime.GOOS == "linux" && runtime.GOARCH == "arm":
		expected = PlatformLinuxARM
	case runtime.GOOS == "linux" && runtime.GOARCH == "386":
		expected = PlatformLinux386
	case runtime.GOOS == "linux" && runtime.GOARCH == "riscv64":
		expected = PlatformLinuxRISCV64
	case runtime.GOOS == "windows" && runtime.GOARCH == "amd64":
		expected = PlatformWindowsAMD64
	case runtime.GOOS == "windows" && runtime.GOARCH == "arm64":
		expected = PlatformWindowsARM64
	case runtime.GOOS == "windows" && runtime.GOARCH == "386":
		expected = PlatformWindows386
	case runtime.GOOS == "darwin" && runtime.GOARCH == "amd64":
		expected = PlatformDarwinAMD64
	case runtime.GOOS == "darwin" && runtime.GOARCH == "arm64":
		expected = PlatformDarwinARM64
	case runtime.GOOS == "freebsd" && runtime.GOARCH == "amd64":
		expected = PlatformFreeBSDAMD64
	default:
		expected = PlatformUnknown
	}
	assert.Equal(t, expected, pt)
}

func TestDetectPlatform(t *testing.T) {
	pd := NewPlatformDetector(testLogger(), time.Minute)
	profile, err := pd.DetectPlatform()
	require.NoError(t, err)
	require.NotNil(t, profile)

	assert.Equal(t, runtime.GOOS, profile.OS.Name)
	assert.Equal(t, runtime.GOARCH, profile.Architecture.Type)
	assert.Equal(t, runtime.Version(), profile.Runtime.GoVersion)
	assert.Equal(t, runtime.Compiler, profile.Runtime.Compiler)
	assert.True(t, profile.ConfidenceScore > 0)
	assert.False(t, profile.ProfileTimestamp.IsZero())
}

func TestNewPlatformOptimizer(t *testing.T) {
	po := NewPlatformOptimizer(testLogger(), nil)
	require.NotNil(t, po)
	assert.NotNil(t, po.profiles)
	assert.Nil(t, po.strategies)
	assert.Equal(t, PlatformType(runtime.GOOS), po.currentOS)
}

func TestApplyOptimizations(t *testing.T) {
	po := NewPlatformOptimizer(testLogger(), nil)
	profile, err := NewPlatformDetector(testLogger(), time.Minute).DetectPlatform()
	require.NoError(t, err)

	err = po.ApplyOptimizations(profile)
	require.NoError(t, err)
}

func TestOptimizerWithStrategies(t *testing.T) {
	strategies := map[PlatformType]OptimizationStrategy{
		PlatformLinuxAMD64: {},
	}
	po := NewPlatformOptimizer(testLogger(), strategies)
	profile, _ := NewPlatformDetector(testLogger(), time.Minute).DetectPlatform()
	err := po.ApplyOptimizations(profile)
	require.NoError(t, err)
}

func TestNewCompatibilityLayer(t *testing.T) {
	cl := NewCompatibilityLayer(testLogger(), CompatibilityOverrides{})
	require.NotNil(t, cl)
	assert.NotNil(t, cl.patches)
	assert.NotNil(t, cl.adapters)
}

func TestScanForIssues(t *testing.T) {
	cl := NewCompatibilityLayer(testLogger(), CompatibilityOverrides{})
	issues := cl.ScanForIssues()
	require.NotNil(t, issues)

	if runtime.GOOS == "linux" {
		assert.Contains(t, issues, "case_sensitive_fs")
	} else if runtime.GOOS == "windows" {
		assert.Contains(t, issues, "case_insensitive_fs")
	}
}

func TestGetPatchForIssue_Missing(t *testing.T) {
	cl := NewCompatibilityLayer(testLogger(), CompatibilityOverrides{})
	patch, ok := cl.GetPatchForIssue("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, patch)
}

func TestGetPatchForIssue_Found(t *testing.T) {
	cl := NewCompatibilityLayer(testLogger(), CompatibilityOverrides{})
	err := cl.ApplyPatch(&CompatibilityPatch{Name: "test_patch", TargetPlatforms: []string{"linux"}})
	require.NoError(t, err)

	patch, ok := cl.GetPatchForIssue("test_patch")
	assert.True(t, ok)
	require.NotNil(t, patch)
	assert.Equal(t, "test_patch", patch.Name)
}

func TestApplyPatch(t *testing.T) {
	cl := NewCompatibilityLayer(testLogger(), CompatibilityOverrides{})
	patch := &CompatibilityPatch{Name: "fs_fix", TargetPlatforms: []string{"linux", "darwin"}}
	err := cl.ApplyPatch(patch)
	require.NoError(t, err)

	stored, ok := cl.patches["fs_fix"]
	assert.True(t, ok)
	assert.Equal(t, "fs_fix", stored.Name)
}

func TestEnableCompatibilityMode(t *testing.T) {
	cl := NewCompatibilityLayer(testLogger(), CompatibilityOverrides{})
	err := cl.EnableCompatibilityMode()
	require.NoError(t, err)
}

func TestNewAdaptationEngine(t *testing.T) {
	ae := NewAdaptationEngine(testLogger(), AdaptationThresholds{})
	require.NotNil(t, ae)
	assert.Equal(t, "reactive", ae.strategy.AdaptationMode)
	assert.Empty(t, ae.triggers)
	assert.Empty(t, ae.history)
}

func TestEvaluateAdaptation_NoThresholds(t *testing.T) {
	ae := NewAdaptationEngine(testLogger(), AdaptationThresholds{})
	needed, reasons := ae.EvaluateAdaptation(&ResourceMetrics{}, &PlatformProfile{})
	assert.False(t, needed)
	assert.Empty(t, reasons)
}

func TestAdaptationEngine_Shutdown(t *testing.T) {
	ae := NewAdaptationEngine(testLogger(), AdaptationThresholds{})
	ae.Shutdown()
}

func TestNewResourceManager(t *testing.T) {
	rm := NewResourceManager(testLogger(), ResourceLimits{})
	require.NotNil(t, rm)
	require.NotNil(t, rm.monitor)
	require.NotNil(t, rm.allocation)
	require.NotNil(t, rm.optimization)
}

func TestResourceManager_UpdateMetrics(t *testing.T) {
	rm := NewResourceManager(testLogger(), ResourceLimits{})
	metrics := &ResourceMetrics{CPUUsage: 50.0, MemoryUsage: 60.0}
	rm.UpdateMetrics(metrics)
	assert.Equal(t, 50.0, rm.monitor.metrics.CPUUsage)
	assert.Equal(t, 60.0, rm.monitor.metrics.MemoryUsage)
}

func TestResourceManager_GetCurrentMetrics(t *testing.T) {
	rm := NewResourceManager(testLogger(), ResourceLimits{})
	metrics := rm.GetCurrentMetrics()
	require.NotNil(t, metrics)
	assert.True(t, metrics.MemoryUsage >= 0)
}

func TestResourceManager_ScaleUp(t *testing.T) {
	rm := NewResourceManager(testLogger(), ResourceLimits{})
	err := rm.ScaleUp()
	require.NoError(t, err)
}

func TestResourceManager_ScaleDown(t *testing.T) {
	rm := NewResourceManager(testLogger(), ResourceLimits{})
	err := rm.ScaleDown()
	require.NoError(t, err)
}

func TestResourceManager_Shutdown(t *testing.T) {
	rm := NewResourceManager(testLogger(), ResourceLimits{})
	rm.Shutdown()
}

func TestNewPlatformManager_NilLogger(t *testing.T) {
	pm, err := NewPlatformManager(nil, PlatformConfig{})
	assert.Error(t, err)
	assert.Nil(t, pm)
	assert.Contains(t, err.Error(), "logger is required")
}

func TestNewPlatformManager_NoAutoDetection(t *testing.T) {
	pm, err := NewPlatformManager(testLogger(), PlatformConfig{
		EnableAutoDetection: false,
		EnableOptimizations: false,
		EnableCompatibility: false,
	})
	require.NoError(t, err)
	require.NotNil(t, pm)
	assert.False(t, pm.isRunning)
	assert.Nil(t, pm.currentProfile)
	pm.Stop()
}

func TestNewPlatformManager_Full(t *testing.T) {
	pm, err := NewPlatformManager(testLogger(), PlatformConfig{
		EnableAutoDetection: true,
		EnableOptimizations: true,
		EnableCompatibility: true,
	})
	require.NoError(t, err)
	require.NotNil(t, pm)
	assert.NotNil(t, pm.currentProfile)
	assert.NotNil(t, pm.detector)
	assert.NotNil(t, pm.optimizer)
	assert.NotNil(t, pm.compatibilityLayer)
	assert.NotNil(t, pm.resourceManager)
	assert.NotNil(t, pm.adaptationEngine)
	pm.Stop()
}

func TestGetPlatformProfile_Nil(t *testing.T) {
	pm, err := NewPlatformManager(testLogger(), PlatformConfig{
		EnableAutoDetection: false,
	})
	require.NoError(t, err)
	defer pm.Stop()

	profile := pm.GetPlatformProfile()
	assert.Nil(t, profile)
}

func TestGetPlatformProfile_ReturnsCopy(t *testing.T) {
	pm, err := NewPlatformManager(testLogger(), PlatformConfig{
		EnableAutoDetection: false,
	})
	require.NoError(t, err)
	defer pm.Stop()

	_ = pm.GetPlatformProfile()

	profile, err := pm.detector.DetectPlatform()
	require.NoError(t, err)

	pm.mu.Lock()
	pm.currentProfile = profile
	pm.mu.Unlock()

	retrieved := pm.GetPlatformProfile()
	require.NotNil(t, retrieved)
	assert.Equal(t, runtime.GOOS, retrieved.OS.Name)

	retrieved.OS.Name = "hacked"
	// Original should not be affected (copy semantics)
	pm.mu.RLock()
	assert.Equal(t, runtime.GOOS, pm.currentProfile.OS.Name)
	pm.mu.RUnlock()
}

func TestStartStop(t *testing.T) {
	pm, err := NewPlatformManager(testLogger(), PlatformConfig{
		EnableAutoDetection: false,
	})
	require.NoError(t, err)

	err = pm.Start()
	require.NoError(t, err)
	assert.True(t, pm.isRunning)

	// Starting again should fail
	err = pm.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	pm.Stop()
	assert.False(t, pm.isRunning)

	// Double stop should not panic
	pm.Stop()
}

func TestForcePlatformDetection(t *testing.T) {
	pm, err := NewPlatformManager(testLogger(), PlatformConfig{
		EnableAutoDetection: false,
	})
	require.NoError(t, err)
	defer pm.Stop()

	profile, err := pm.ForcePlatformDetection()
	require.NoError(t, err)
	require.NotNil(t, profile)
	assert.Equal(t, runtime.GOOS, profile.OS.Name)

	// currentProfile should now be set
	retrieved := pm.GetPlatformProfile()
	require.NotNil(t, retrieved)
	assert.Equal(t, runtime.GOOS, retrieved.OS.Name)
}

func TestApplyPlatformOptimizations_NoProfile(t *testing.T) {
	pm, err := NewPlatformManager(testLogger(), PlatformConfig{
		EnableAutoDetection: false,
	})
	require.NoError(t, err)
	defer pm.Stop()

	err = pm.ApplyPlatformOptimizations()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no platform profile available")
}

func TestApplyPlatformOptimizations(t *testing.T) {
	pm, err := NewPlatformManager(testLogger(), PlatformConfig{
		EnableAutoDetection: false,
	})
	require.NoError(t, err)
	defer pm.Stop()

	_, err = pm.ForcePlatformDetection()
	require.NoError(t, err)

	err = pm.ApplyPlatformOptimizations()
	require.NoError(t, err)
}

func TestCollectResourceMetrics(t *testing.T) {
	pm, err := NewPlatformManager(testLogger(), PlatformConfig{
		EnableAutoDetection: false,
	})
	require.NoError(t, err)
	defer pm.Stop()

	metrics, err := pm.collectResourceMetrics()
	require.NoError(t, err)
	require.NotNil(t, metrics)

	assert.True(t, metrics.CPUUsage >= 0)
	assert.True(t, metrics.MemoryUsage >= 0)
	assert.True(t, metrics.LoadAverage.OneMinute >= 0)
}

func TestHandleResourceAlert_NonCritical(t *testing.T) {
	pm, err := NewPlatformManager(testLogger(), PlatformConfig{
		EnableAutoDetection: false,
	})
	require.NoError(t, err)
	defer pm.Stop()

	// Non-critical alert should not trigger adaptation
	alert := &ResourceAlert{
		Resource: "memory",
		Level:    AlertWarning,
		Value:    85.0,
		Threshold: 80.0,
	}
	pm.handleResourceAlert(alert)
}

func TestCheckCompatibilityIssues(t *testing.T) {
	pm, err := NewPlatformManager(testLogger(), PlatformConfig{
		EnableAutoDetection: false,
	})
	require.NoError(t, err)
	defer pm.Stop()

	pm.checkCompatibilityIssues()
}

func TestApplyAdaptation_Unknown(t *testing.T) {
	pm, err := NewPlatformManager(testLogger(), PlatformConfig{
		EnableAutoDetection: false,
	})
	require.NoError(t, err)
	defer pm.Stop()

	err = pm.applyAdaptation("unknown_adaptation")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown adaptation")
}

func TestApplyAdaptation_IncreaseResources(t *testing.T) {
	pm, err := NewPlatformManager(testLogger(), PlatformConfig{
		EnableAutoDetection: false,
	})
	require.NoError(t, err)
	defer pm.Stop()

	err = pm.applyAdaptation("increase_resources")
	require.NoError(t, err)
}

func TestApplyAdaptation_DecreaseResources(t *testing.T) {
	pm, err := NewPlatformManager(testLogger(), PlatformConfig{
		EnableAutoDetection: false,
	})
	require.NoError(t, err)
	defer pm.Stop()

	err = pm.applyAdaptation("decrease_resources")
	require.NoError(t, err)
}

func TestApplyAdaptation_OptimizePower(t *testing.T) {
	pm, err := NewPlatformManager(testLogger(), PlatformConfig{
		EnableAutoDetection: false,
	})
	require.NoError(t, err)
	defer pm.Stop()

	err = pm.applyAdaptation("optimize_power")
	require.NoError(t, err)
}

func TestApplyAdaptation_AdjustScheduling(t *testing.T) {
	pm, err := NewPlatformManager(testLogger(), PlatformConfig{
		EnableAutoDetection: false,
	})
	require.NoError(t, err)
	defer pm.Stop()

	err = pm.applyAdaptation("adjust_scheduling")
	require.NoError(t, err)
}

func TestApplyAdaptation_CompatibilityMode(t *testing.T) {
	pm, err := NewPlatformManager(testLogger(), PlatformConfig{
		EnableAutoDetection: false,
	})
	require.NoError(t, err)
	defer pm.Stop()

	err = pm.applyAdaptation("enable_compatibility_mode")
	require.NoError(t, err)
}

func TestNewPlatformManager_WithContextCancel(t *testing.T) {
	pm, err := NewPlatformManager(testLogger(), PlatformConfig{
		EnableAutoDetection: false,
	})
	require.NoError(t, err)
	defer pm.Stop()

	require.NotNil(t, pm.cancel)
	pm.cancel()
}

func TestNewPlatformManager_DetectAndOptimize(t *testing.T) {
	pm, err := NewPlatformManager(testLogger(), PlatformConfig{
		EnableAutoDetection: true,
		EnableOptimizations: true,
	})
	require.NoError(t, err)
	require.NotNil(t, pm)

	profile := pm.GetPlatformProfile()
	require.NotNil(t, profile)
	assert.Equal(t, runtime.GOOS, profile.OS.Name)
	pm.Stop()
}

func TestContextCancellation(t *testing.T) {
	pm, err := NewPlatformManager(testLogger(), PlatformConfig{
		EnableAutoDetection: false,
	})
	require.NoError(t, err)

	ctx := pm.ctx
	require.NotNil(t, ctx)

	select {
	case <-ctx.Done():
		t.Fatal("context should not be cancelled initially")
	default:
	}

	err = pm.Start()
	require.NoError(t, err)

	pm.Stop()

	select {
	case <-ctx.Done():
		// expected after Stop
	default:
		t.Fatal("context should be cancelled after Stop")
	}
}

func TestResourceManager_CheckAlerts(t *testing.T) {
	rm := NewResourceManager(testLogger(), ResourceLimits{
		MaxCPUPercent: 0,
		MaxMemoryMB:   0,
	})
	alerts := rm.CheckAlerts()
	assert.Empty(t, alerts)
}

func TestCompatibilityLayer_Overrides(t *testing.T) {
	overrides := CompatibilityOverrides{
		FilePathSeparators: map[string]string{"linux": "/", "windows": "\\"},
	}
	cl := NewCompatibilityLayer(testLogger(), overrides)
	assert.Equal(t, "/", cl.overrides.FilePathSeparators["linux"])
}

func TestAdaptationEngine_EvaluateAdaptation_Thresholds(t *testing.T) {
	ae := NewAdaptationEngine(testLogger(), AdaptationThresholds{
		CPULoadThreshold:        200.0,
		MemoryPressureThreshold: 200.0,
	})
	metrics := &ResourceMetrics{CPUUsage: 1.0, MemoryUsage: 1.0}
	needed, reasons := ae.EvaluateAdaptation(metrics, &PlatformProfile{})
	if needed {
		t.Logf("adaptation triggered by: %v", reasons)
	}
}
