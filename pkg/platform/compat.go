package platform

import (
	"log/slog"
	"runtime"
	"strings"
)

type CompatibilityLayer struct {
	logger    *slog.Logger
	overrides CompatibilityOverrides
	patches   map[string]CompatibilityPatch
	adapters  map[string]PlatformAdapter
}

func NewCompatibilityLayer(logger *slog.Logger, overrides CompatibilityOverrides) *CompatibilityLayer {
	return &CompatibilityLayer{
		logger:    logger,
		overrides: overrides,
		patches:   make(map[string]CompatibilityPatch),
		adapters:  make(map[string]PlatformAdapter),
	}
}

func (cl *CompatibilityLayer) ScanForIssues() []string {
	var issues []string
	switch runtime.GOOS {
	case "windows":
		issues = append(issues, "case_insensitive_fs")
	case "linux":
		issues = append(issues, "case_sensitive_fs")
	}
	return issues
}

func (cl *CompatibilityLayer) GetPatchForIssue(issue string) (*CompatibilityPatch, bool) {
	patch, ok := cl.patches[issue]
	if ok {
		return &patch, true
	}
	return nil, false
}

func (cl *CompatibilityLayer) ApplyPatch(patch *CompatibilityPatch) error {
	cl.patches[patch.Name] = *patch
	cl.logger.Info("Applied compatibility patch", "name", patch.Name, "targets", strings.Join(patch.TargetPlatforms, ","))
	return nil
}

func (cl *CompatibilityLayer) EnableCompatibilityMode() error {
	cl.logger.Info("Compatibility mode enabled", "os", runtime.GOOS, "arch", runtime.GOARCH)
	return nil
}
