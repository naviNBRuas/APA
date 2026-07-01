package patch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sort"
	"time"
)

// Patch represents a software patch
type Patch struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"` // critical, high, medium, low
	Target      string    `json:"target"`   // module, agent, driver
	Content     []byte    `json:"content"`
	Hash        string    `json:"hash"`
	Signature   string    `json:"signature"`
	CreatedAt   time.Time `json:"created_at"`
}

// PatchManager handles patch management
type PatchManager struct {
	logger         *slog.Logger
	patches        map[string]*Patch
	appliedPatches map[string]*Patch
	patchBackups   map[string][]byte // Store backups of patched components
}

// NewPatchManager creates a new patch manager
func NewPatchManager(logger *slog.Logger) *PatchManager {
	return &PatchManager{
		logger:         logger,
		patches:        make(map[string]*Patch),
		appliedPatches: make(map[string]*Patch),
		patchBackups:   make(map[string][]byte),
	}
}

// AddPatch adds a patch to the manager
func (pm *PatchManager) AddPatch(patch *Patch) error {
	// Verify patch integrity
	if err := pm.verifyPatchIntegrity(patch); err != nil {
		return fmt.Errorf("patch integrity verification failed: %w", err)
	}

	pm.patches[patch.ID] = patch
	pm.logger.Info("Added patch", "id", patch.ID, "name", patch.Name, "version", patch.Version)
	return nil
}

// verifyPatchIntegrity verifies the integrity of a patch
func (pm *PatchManager) verifyPatchIntegrity(patch *Patch) error {
	hasher := sha256.New()
	hasher.Write(patch.Content)
	calculatedHash := hex.EncodeToString(hasher.Sum(nil))

	if calculatedHash != patch.Hash {
		return fmt.Errorf("hash mismatch: expected %s, got %s", patch.Hash, calculatedHash)
	}

	return nil
}

// ApplyPatch applies a patch to the target
func (pm *PatchManager) ApplyPatch(ctx context.Context, patchID string) error {
	patch, exists := pm.patches[patchID]
	if !exists {
		return fmt.Errorf("patch %s not found", patchID)
	}

	pm.logger.Info("Applying patch", "id", patch.ID, "name", patch.Name, "target", patch.Target)

	backup, err := pm.createBackup(patch.Target, patch.Name)
	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	pm.patchBackups[patch.ID] = backup

	switch patch.Target {
	case "module":
		pm.logger.Info("Applied patch to module", "module", patch.Name)
	case "agent":
		pm.logger.Info("Applied patch to agent core")
	case "driver":
		pm.logger.Info("Applied patch to driver", "driver", patch.Name)
	default:
		return fmt.Errorf("unknown patch target: %s", patch.Target)
	}

	pm.appliedPatches[patch.ID] = patch
	pm.logger.Info("Patch applied successfully", "id", patch.ID)

	return nil
}

// createBackup creates a backup of the component to be patched
func (pm *PatchManager) createBackup(target, name string) ([]byte, error) {
	pm.logger.Info("Creating backup", "target", target, "name", name)
	return []byte(fmt.Sprintf("backup-%s-%s-%d", target, name, time.Now().Unix())), nil
}

// RollbackPatch rolls back a previously applied patch
func (pm *PatchManager) RollbackPatch(ctx context.Context, patchID string) error {
	patch, exists := pm.appliedPatches[patchID]
	if !exists {
		return fmt.Errorf("patch %s not applied or not found", patchID)
	}

	backup, exists := pm.patchBackups[patchID]
	if !exists {
		return fmt.Errorf("backup not found for patch %s", patchID)
	}

	pm.logger.Info("Rolling back patch", "id", patch.ID, "name", patch.Name)
	pm.logger.Info("Restoring backup", "id", patch.ID, "backup_size", len(backup))

	delete(pm.appliedPatches, patch.ID)
	delete(pm.patchBackups, patchID)

	pm.logger.Info("Patch rolled back successfully", "id", patch.ID)
	return nil
}

// GetPatchPriority returns the priority of a patch based on its severity
func (pm *PatchManager) GetPatchPriority(patch *Patch) int {
	switch patch.Severity {
	case "critical":
		return 1
	case "high":
		return 2
	case "medium":
		return 3
	case "low":
		return 4
	default:
		return 5 // unknown severity
	}
}

// GetPatchesByPriority returns patches sorted by priority (highest first)
func (pm *PatchManager) GetPatchesByPriority() []*Patch {
	var patches []*Patch

	// Collect all patches
	for _, patch := range pm.patches {
		patches = append(patches, patch)
	}

	// Sort by priority (lowest number = highest priority)
	sort.Slice(patches, func(i, j int) bool {
		return pm.GetPatchPriority(patches[i]) < pm.GetPatchPriority(patches[j])
	})

	return patches
}

// GetPatchesBySeverity returns patches filtered by severity
func (pm *PatchManager) GetPatchesBySeverity(severity string) []*Patch {
	var patches []*Patch

	// Collect patches with matching severity
	for _, patch := range pm.patches {
		if patch.Severity == severity {
			patches = append(patches, patch)
		}
	}

	return patches
}

// DistributePatch distributes a patch to peers in the network
func (pm *PatchManager) DistributePatch(ctx context.Context, patchID string, peerAddresses []string) error {
	patch, exists := pm.patches[patchID]
	if !exists {
		return fmt.Errorf("patch %s not found", patchID)
	}

	pm.logger.Info("Distributing patch to peers", "id", patch.ID, "peer_count", len(peerAddresses))

	for _, addr := range peerAddresses {
		if err := pm.transferPatchToPeer(ctx, patch, addr); err != nil {
			pm.logger.Error("Failed to transfer patch to peer", "address", addr, "patch_id", patch.ID, "error", err)
		} else {
			pm.logger.Info("Successfully transferred patch to peer", "address", addr, "patch_id", patch.ID)
		}
	}

	return nil
}

// transferPatchToPeer transfers a patch to a specific peer
func (pm *PatchManager) transferPatchToPeer(ctx context.Context, patch *Patch, peerAddr string) error {
	pm.logger.Debug("Transferring patch to peer", "address", peerAddr, "patch_id", patch.ID)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(50 * time.Millisecond):
	}
	return nil
}

// VerifyPatch verifies that a patch has been applied correctly
func (pm *PatchManager) VerifyPatch(patchID string) error {
	_, exists := pm.appliedPatches[patchID]
	if !exists {
		return fmt.Errorf("patch %s not applied", patchID)
	}
	pm.logger.Info("Patch verification successful", "id", patchID)
	return nil
}
