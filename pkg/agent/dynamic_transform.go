package agent

import (
	"crypto/sha256"
	"fmt"

	"github.com/naviNBRuas/APA/pkg/polymorphic"
)

// TransformationManager builds per-run variants of code to avoid stable fingerprints.
type TransformationManager struct {
	engine *polymorphic.Engine
	logger Logger
}

// Logger is the subset of slog.Logger used here (for testability).
type Logger interface {
	Info(msg string, args ...any)
}

// NewTransformationManager constructs a manager with the provided engine and logger.
func NewTransformationManager(engine *polymorphic.Engine, logger Logger) *TransformationManager {
	return &TransformationManager{engine: engine, logger: logger}
}

// NextVariant produces a transformed payload and fingerprint; the original can be recovered via ReverseVariant.
func (tm *TransformationManager) NextVariant(payload []byte) ([]byte, string, error) {
	if tm == nil || tm.engine == nil {
		return nil, "", fmt.Errorf("transformer not initialized")
	}
	variant, err := tm.engine.TransformCode(payload)
	if err != nil {
		return nil, "", err
	}
	fp := sha256.Sum256(variant)
	if tm.logger != nil {
		tm.logger.Info("generated runtime variant", "fingerprint", fmt.Sprintf("%x", fp[:8]))
	}
	return variant, fmt.Sprintf("%x", fp[:]), nil
}

// ReverseVariant restores the original payload from a variant produced by NextVariant.
func (tm *TransformationManager) ReverseVariant(variant []byte) ([]byte, error) {
	if tm == nil || tm.engine == nil {
		return nil, fmt.Errorf("transformer not initialized")
	}
	return tm.engine.ReverseTransformation(variant)
}

// Cleanup releases transformation resources.
func (tm *TransformationManager) Cleanup() {}
