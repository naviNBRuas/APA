package patch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"testing"
	"time"
)

func TestPatchManager(t *testing.T) {
	logger := slog.Default()
	patchManager := NewPatchManager(logger)

	// Create a test patch
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

	// Test adding a patch
	err := patchManager.AddPatch(patch)
	if err != nil {
		t.Fatalf("Failed to add patch: %v", err)
	}

	// Test applying a patch
	err = patchManager.ApplyPatch(context.Background(), patch.ID)
	if err != nil {
		t.Errorf("Failed to apply patch: %v", err)
	}

	// Test rolling back a patch
	err = patchManager.RollbackPatch(context.Background(), patch.ID)
	if err != nil {
		t.Errorf("Failed to rollback patch: %v", err)
	}

	// Test patch priority
	priority := patchManager.GetPatchPriority(patch)
	if priority != 3 { // medium severity should be priority 3
		t.Errorf("Expected priority 3 for medium severity patch, got %d", priority)
	}

	// Test getting patches by priority
	patches := patchManager.GetPatchesByPriority()
	if len(patches) != 1 {
		t.Errorf("Expected 1 patch, got %d", len(patches))
	}

	// Test getting patches by severity
	mediumPatches := patchManager.GetPatchesBySeverity("medium")
	if len(mediumPatches) != 1 {
		t.Errorf("Expected 1 medium severity patch, got %d", len(mediumPatches))
	}

	// Test distributing a patch
	peerAddresses := []string{"192.168.1.100:4001", "192.168.1.101:4001"}
	err = patchManager.DistributePatch(context.Background(), patch.ID, peerAddresses)
	if err != nil {
		t.Errorf("Failed to distribute patch: %v", err)
	}

	// Test verifying a patch
	err = patchManager.VerifyPatch(patch.ID)
	if err == nil {
		t.Error("Expected verification to fail for non-applied patch")
	}
}

func TestPatchIntegrityVerification(t *testing.T) {
	logger := slog.Default()
	patchManager := NewPatchManager(logger)

	// Create a test patch with incorrect hash
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

	// Test adding a patch with incorrect hash
	err := patchManager.AddPatch(patch)
	if err == nil {
		t.Error("Expected patch addition to fail with incorrect hash")
	}
}

func TestPatchPrioritization(t *testing.T) {
	logger := slog.Default()
	patchManager := NewPatchManager(logger)

	// Create patches with different severities
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

	// Add patches
	patchManager.AddPatch(criticalPatch)
	patchManager.AddPatch(highPatch)
	patchManager.AddPatch(mediumPatch)

	// Get patches by priority
	patches := patchManager.GetPatchesByPriority()
	if len(patches) != 3 {
		t.Errorf("Expected 3 patches, got %d", len(patches))
	}

	// Check that critical patch is first (highest priority)
	if patches[0].ID != "critical-patch" {
		t.Errorf("Expected critical patch first, got %s", patches[0].ID)
	}

	// Check that high patch is second
	if patches[1].ID != "high-patch" {
		t.Errorf("Expected high patch second, got %s", patches[1].ID)
	}

	// Check that medium patch is third
	if patches[2].ID != "medium-patch" {
		t.Errorf("Expected medium patch third, got %s", patches[2].ID)
	}
}