package patch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newValidPatch(id, name, severity, target string) *Patch {
	content := []byte(name + " content")
	hasher := sha256.New()
	hasher.Write(content)
	hash := hex.EncodeToString(hasher.Sum(nil))
	return &Patch{
		ID:        id,
		Name:      name,
		Version:   "1.0.0",
		Severity:  severity,
		Target:    target,
		Content:   content,
		Hash:      hash,
		CreatedAt: time.Now(),
	}
}

func TestNewPatchManager(t *testing.T) {
	pm := NewPatchManager(slog.Default())
	require.NotNil(t, pm)
	assert.Empty(t, pm.patches)
	assert.Empty(t, pm.appliedPatches)
	assert.Empty(t, pm.patchBackups)
}

func TestAddPatch_Success(t *testing.T) {
	pm := NewPatchManager(slog.Default())
	patch := newValidPatch("p1", "Test Patch", "medium", "module")

	err := pm.AddPatch(patch)
	assert.NoError(t, err)
	assert.Len(t, pm.patches, 1)
}

func TestAddPatch_HashMismatch(t *testing.T) {
	pm := NewPatchManager(slog.Default())
	patch := newValidPatch("p1", "Test", "medium", "module")
	patch.Hash = "bad-hash"

	err := pm.AddPatch(patch)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "hash mismatch")
	assert.Empty(t, pm.patches)
}

func TestAddPatch_EmptyContent(t *testing.T) {
	pm := NewPatchManager(slog.Default())
	hasher := sha256.New()
	hash := hex.EncodeToString(hasher.Sum(nil))
	patch := &Patch{
		ID:      "p-empty",
		Name:    "Empty",
		Content: []byte{},
		Hash:    hash,
	}

	err := pm.AddPatch(patch)
	assert.NoError(t, err)
}

func TestApplyPatch_Success(t *testing.T) {
	pm := NewPatchManager(slog.Default())
	patch := newValidPatch("p1", "Test", "medium", "module")
	require.NoError(t, pm.AddPatch(patch))

	err := pm.ApplyPatch(context.Background(), "p1")
	assert.NoError(t, err)
	assert.Len(t, pm.appliedPatches, 1)
	assert.Len(t, pm.patchBackups, 1)
}

func TestApplyPatch_NotFound(t *testing.T) {
	pm := NewPatchManager(slog.Default())

	err := pm.ApplyPatch(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestApplyPatch_AgentTarget(t *testing.T) {
	pm := NewPatchManager(slog.Default())
	patch := newValidPatch("p1", "Agent Fix", "critical", "agent")
	require.NoError(t, pm.AddPatch(patch))

	err := pm.ApplyPatch(context.Background(), "p1")
	assert.NoError(t, err)
}

func TestApplyPatch_DriverTarget(t *testing.T) {
	pm := NewPatchManager(slog.Default())
	patch := newValidPatch("p1", "Driver Fix", "high", "driver")
	require.NoError(t, pm.AddPatch(patch))

	err := pm.ApplyPatch(context.Background(), "p1")
	assert.NoError(t, err)
}

func TestApplyPatch_UnknownTarget(t *testing.T) {
	pm := NewPatchManager(slog.Default())
	patch := newValidPatch("p1", "Unknown", "medium", "firmware")
	require.NoError(t, pm.AddPatch(patch))

	err := pm.ApplyPatch(context.Background(), "p1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown patch target")
	assert.Empty(t, pm.appliedPatches)
}

func TestRollbackPatch_Success(t *testing.T) {
	pm := NewPatchManager(slog.Default())
	patch := newValidPatch("p1", "Test", "medium", "module")
	require.NoError(t, pm.AddPatch(patch))
	require.NoError(t, pm.ApplyPatch(context.Background(), "p1"))

	err := pm.RollbackPatch(context.Background(), "p1")
	assert.NoError(t, err)
	assert.Empty(t, pm.appliedPatches)
	assert.Empty(t, pm.patchBackups)
}

func TestRollbackPatch_NotApplied(t *testing.T) {
	pm := NewPatchManager(slog.Default())

	err := pm.RollbackPatch(context.Background(), "p1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not applied")
}

func TestRollbackPatch_MissingBackup(t *testing.T) {
	pm := NewPatchManager(slog.Default())
	patch := newValidPatch("p1", "Test", "medium", "module")
	pm.appliedPatches["p1"] = patch

	err := pm.RollbackPatch(context.Background(), "p1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "backup not found")
}

func TestGetPatchPriority(t *testing.T) {
	pm := NewPatchManager(slog.Default())
	tests := []struct {
		severity string
		expected int
	}{
		{"critical", 1},
		{"high", 2},
		{"medium", 3},
		{"low", 4},
		{"unknown", 5},
		{"", 5},
	}

	for _, tt := range tests {
		patch := newValidPatch("p", "Test", tt.severity, "module")
		assert.Equal(t, tt.expected, pm.GetPatchPriority(patch), "severity=%s", tt.severity)
	}
}

func TestGetPatchesByPriority_Sorted(t *testing.T) {
	pm := NewPatchManager(slog.Default())
	require.NoError(t, pm.AddPatch(newValidPatch("p-low", "Low", "low", "module")))
	require.NoError(t, pm.AddPatch(newValidPatch("p-critical", "Critical", "critical", "module")))
	require.NoError(t, pm.AddPatch(newValidPatch("p-medium", "Medium", "medium", "module")))

	patches := pm.GetPatchesByPriority()
	require.Len(t, patches, 3)
	assert.Equal(t, "p-critical", patches[0].ID)
	assert.Equal(t, "p-medium", patches[1].ID)
	assert.Equal(t, "p-low", patches[2].ID)
}

func TestGetPatchesByPriority_Empty(t *testing.T) {
	pm := NewPatchManager(slog.Default())
	patches := pm.GetPatchesByPriority()
	assert.Empty(t, patches)
}

func TestGetPatchesBySeverity(t *testing.T) {
	pm := NewPatchManager(slog.Default())
	require.NoError(t, pm.AddPatch(newValidPatch("p1", "Critical 1", "critical", "module")))
	require.NoError(t, pm.AddPatch(newValidPatch("p2", "Critical 2", "critical", "agent")))
	require.NoError(t, pm.AddPatch(newValidPatch("p3", "Medium", "medium", "module")))

	critical := pm.GetPatchesBySeverity("critical")
	assert.Len(t, critical, 2)

	medium := pm.GetPatchesBySeverity("medium")
	assert.Len(t, medium, 1)

	none := pm.GetPatchesBySeverity("high")
	assert.Empty(t, none)
}

func TestDistributePatch_Success(t *testing.T) {
	pm := NewPatchManager(slog.Default())
	patch := newValidPatch("p1", "Test", "medium", "module")
	require.NoError(t, pm.AddPatch(patch))

	err := pm.DistributePatch(context.Background(), "p1", []string{"peer1", "peer2"})
	assert.NoError(t, err)
}

func TestDistributePatch_NotFound(t *testing.T) {
	pm := NewPatchManager(slog.Default())

	err := pm.DistributePatch(context.Background(), "p1", []string{"peer1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDistributePatch_EmptyPeers(t *testing.T) {
	pm := NewPatchManager(slog.Default())
	patch := newValidPatch("p1", "Test", "medium", "module")
	require.NoError(t, pm.AddPatch(patch))

	err := pm.DistributePatch(context.Background(), "p1", nil)
	assert.NoError(t, err)
}

func TestDistributePatch_CancelledContext(t *testing.T) {
	pm := NewPatchManager(slog.Default())
	patch := newValidPatch("p1", "Test", "medium", "module")
	require.NoError(t, pm.AddPatch(patch))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := pm.DistributePatch(ctx, "p1", []string{"peer1"})
	assert.NoError(t, err)
}

func TestVerifyPatch_Applied(t *testing.T) {
	pm := NewPatchManager(slog.Default())
	patch := newValidPatch("p1", "Test", "medium", "module")
	require.NoError(t, pm.AddPatch(patch))
	require.NoError(t, pm.ApplyPatch(context.Background(), "p1"))

	err := pm.VerifyPatch("p1")
	assert.NoError(t, err)
}

func TestVerifyPatch_NotApplied(t *testing.T) {
	pm := NewPatchManager(slog.Default())
	patch := newValidPatch("p1", "Test", "medium", "module")
	require.NoError(t, pm.AddPatch(patch))

	err := pm.VerifyPatch("p1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not applied")
}
