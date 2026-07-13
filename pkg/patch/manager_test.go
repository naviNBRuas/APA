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

func TestPatchManager(t *testing.T) {
	logger := slog.Default()
	patchManager := NewPatchManager(logger)

	content := []byte("This is a test patch content")
	hasher := sha256.New()
	hasher.Write(content)
	hash := hex.EncodeToString(hasher.Sum(nil))

	patch := &Patch{
		ID:          "test-patch-001",
		Name:        "Test Patch",
		Version:     "1.0.0",
		Description: "A test patch for testing purposes",
		Severity:    "medium",
		Target:      "module",
		Content:     content,
		Hash:        hash,
		CreatedAt:   time.Now(),
	}

	err := patchManager.AddPatch(patch)
	require.NoError(t, err, "Failed to add patch")

	err = patchManager.ApplyPatch(context.Background(), patch.ID)
	assert.NoError(t, err, "Failed to apply patch")

	err = patchManager.RollbackPatch(context.Background(), patch.ID)
	assert.NoError(t, err, "Failed to rollback patch")

	priority := patchManager.GetPatchPriority(patch)
	assert.Equal(t, 3, priority)

	patches := patchManager.GetPatchesByPriority()
	assert.Equal(t, 1, len(patches))

	mediumPatches := patchManager.GetPatchesBySeverity("medium")
	assert.Equal(t, 1, len(mediumPatches))

	peerAddresses := []string{"192.168.1.100:4001", "192.168.1.101:4001"}
	err = patchManager.DistributePatch(context.Background(), patch.ID, peerAddresses)
	assert.NoError(t, err, "Failed to distribute patch")

	err = patchManager.VerifyPatch(patch.ID)
	assert.Error(t, err, "Expected verification to fail for non-applied patch")
}

func TestPatchIntegrityVerification(t *testing.T) {
	logger := slog.Default()
	patchManager := NewPatchManager(logger)

	content := []byte("This is a test patch content")
	wrongHash := "incorrect-hash"

	patch := &Patch{
		ID:          "test-patch-002",
		Name:        "Test Patch with Wrong Hash",
		Version:     "1.0.0",
		Description: "A test patch with incorrect hash",
		Severity:    "high",
		Target:      "agent",
		Content:     content,
		Hash:        wrongHash,
		CreatedAt:   time.Now(),
	}

	err := patchManager.AddPatch(patch)
	assert.Error(t, err, "Expected patch addition to fail with incorrect hash")
}

func TestPatchPrioritization(t *testing.T) {
	logger := slog.Default()
	patchManager := NewPatchManager(logger)

	criticalContent := []byte("critical patch content")
	criticalHasher := sha256.New()
	criticalHasher.Write(criticalContent)
	criticalHash := hex.EncodeToString(criticalHasher.Sum(nil))

	criticalPatch := &Patch{
		ID:       "critical-patch",
		Name:     "Critical Patch",
		Severity: "critical",
		Target:   "agent",
		Content:  criticalContent,
		Hash:     criticalHash,
	}

	highContent := []byte("high patch content")
	highHasher := sha256.New()
	highHasher.Write(highContent)
	highHash := hex.EncodeToString(highHasher.Sum(nil))

	highPatch := &Patch{
		ID:       "high-patch",
		Name:     "High Patch",
		Severity: "high",
		Target:   "module",
		Content:  highContent,
		Hash:     highHash,
	}

	mediumContent := []byte("medium patch content")
	mediumHasher := sha256.New()
	mediumHasher.Write(mediumContent)
	mediumHash := hex.EncodeToString(mediumHasher.Sum(nil))

	mediumPatch := &Patch{
		ID:       "medium-patch",
		Name:     "Medium Patch",
		Severity: "medium",
		Target:   "driver",
		Content:  mediumContent,
		Hash:     mediumHash,
	}

	if err := patchManager.AddPatch(criticalPatch); err != nil {
		assert.NoError(t, err, "Failed to add critical patch")
	}
	if err := patchManager.AddPatch(highPatch); err != nil {
		assert.NoError(t, err, "Failed to add high patch")
	}
	if err := patchManager.AddPatch(mediumPatch); err != nil {
		assert.NoError(t, err, "Failed to add medium patch")
	}

	patches := patchManager.GetPatchesByPriority()
	assert.Equal(t, 3, len(patches))
	assert.Equal(t, "critical-patch", patches[0].ID)
	assert.Equal(t, "high-patch", patches[1].ID)
	assert.Equal(t, "medium-patch", patches[2].ID)
}
