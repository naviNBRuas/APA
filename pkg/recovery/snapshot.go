package recovery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/naviNBRuas/APA/pkg/module"
	controllerManifest "github.com/naviNBRuas/APA/pkg/controller/manifest"
	"gopkg.in/yaml.v3"
)

// SnapshotData represents the complete state of the agent at a point in time
type SnapshotData struct {
	Timestamp       time.Time                     `json:"timestamp"`
	Version         string                        `json:"version"`
	AgentID         string                        `json:"agent_id"`
	Configuration   interface{}                   `json:"configuration"`
	Modules         []*module.Manifest            `json:"modules"`
	Controllers     []*controllerManifest.Manifest `json:"controllers"`
	OperationalState map[string]interface{}       `json:"operational_state"`
	Checksum        string                        `json:"checksum"`
}

// ExtendedRecoveryController extends the RecoveryController with enhanced snapshot capabilities
type ExtendedRecoveryController struct {
	*RecoveryController
	snapshotStoragePath string
}

// NewExtendedRecoveryController creates a new ExtendedRecoveryController
func NewExtendedRecoveryController(
	logger *slog.Logger,
	config interface{},
	applyConfigFunc func(configData []byte) error,
	p2p P2PService,
	moduleManager ModuleManagerService,
	controllerManager ControllerManagerService,
	snapshotStoragePath string,
) *ExtendedRecoveryController {
	baseController := NewRecoveryController(logger, config, applyConfigFunc, p2p, moduleManager, controllerManager)
	return &ExtendedRecoveryController{
		RecoveryController:  baseController,
		snapshotStoragePath: snapshotStoragePath,
	}
}

// CreateComprehensiveSnapshot creates a complete snapshot of the agent's state
func (erc *ExtendedRecoveryController) CreateComprehensiveSnapshot(ctx context.Context, agentID string) (string, error) {
	erc.logger.Info("Creating comprehensive agent snapshot", "agent_id", agentID)

	// Gather all components for the snapshot
	snapshot := &SnapshotData{
		Timestamp:       time.Now(),
		Version:         "1.0",
		AgentID:         agentID,
		Configuration:   erc.config,
		Modules:         make([]*module.Manifest, 0),
		Controllers:     make([]*controllerManifest.Manifest, 0),
		OperationalState: make(map[string]interface{}),
	}

	// Collect module information
	if erc.moduleManager != nil {
		modules := erc.moduleManager.ListModules()
		snapshot.Modules = modules
		erc.logger.Debug("Collected module information", "module_count", len(modules))
	}

	// Collect controller information
	if erc.controllerManager != nil {
		controllers := erc.controllerManager.ListControllers()
		snapshot.Controllers = controllers
		erc.logger.Debug("Collected controller information", "controller_count", len(controllers))
	}

	// Collect operational state (this would be extended in a real implementation)
	snapshot.OperationalState["quarantine_list"] = erc.quarantineList
	snapshot.OperationalState["snapshot_timestamp"] = time.Now()

	// Calculate checksum for integrity verification
	checksum, err := erc.calculateSnapshotChecksum(snapshot)
	if err != nil {
		return "", fmt.Errorf("failed to calculate snapshot checksum: %w", err)
	}
	snapshot.Checksum = checksum

	// Serialize snapshot data
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to serialize snapshot data: %w", err)
	}

	// Create snapshot filename with timestamp
	filename := fmt.Sprintf("snapshot_%s_%s.json", agentID, time.Now().Format("20060102_150405"))
	filepath := filepath.Join(erc.snapshotStoragePath, filename)

	// Ensure the snapshot storage directory exists
	if err := os.MkdirAll(erc.snapshotStoragePath, 0755); err != nil {
		return "", fmt.Errorf("failed to create snapshot storage directory: %w", err)
	}

	// Write snapshot to file
	err = os.WriteFile(filepath, data, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write snapshot file: %w", err)
	}

	erc.logger.Info("Comprehensive snapshot created successfully", "file", filepath)
	return filepath, nil
}

// RestoreFromComprehensiveSnapshot restores the agent's state from a comprehensive snapshot
func (erc *ExtendedRecoveryController) RestoreFromComprehensiveSnapshot(ctx context.Context, snapshotFile string) error {
	erc.logger.Info("Restoring agent from comprehensive snapshot", "file", snapshotFile)

	// Read snapshot data from file
	data, err := os.ReadFile(snapshotFile)
	if err != nil {
		return fmt.Errorf("failed to read snapshot file: %w", err)
	}

	// Deserialize snapshot data
	var snapshot SnapshotData
	err = json.Unmarshal(data, &snapshot)
	if err != nil {
		return fmt.Errorf("failed to deserialize snapshot data: %w", err)
	}

	// Verify checksum
	calculatedChecksum, err := erc.calculateSnapshotChecksum(&snapshot)
	if err != nil {
		return fmt.Errorf("failed to calculate snapshot checksum for verification: %w", err)
	}

	if calculatedChecksum != snapshot.Checksum {
		return fmt.Errorf("snapshot checksum verification failed: expected %s, got %s", snapshot.Checksum, calculatedChecksum)
	}

	erc.logger.Info("Snapshot checksum verified successfully")

	// Restore configuration
	configData, err := yaml.Marshal(snapshot.Configuration)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration data: %w", err)
	}

	if erc.applyConfigFunc == nil {
		return fmt.Errorf("applyConfigFunc is not set in RecoveryController")
	}

	// Apply the configuration to reconfigure all components
	if err := erc.applyConfigFunc(configData); err != nil {
		return fmt.Errorf("failed to apply configuration during restore: %w", err)
	}

	// Restore modules (in a real implementation, this would involve fetching and loading module binaries)
	erc.logger.Info("Would restore modules in a real implementation", "module_count", len(snapshot.Modules))

	// Restore controllers (in a real implementation, this would involve fetching and loading controller binaries)
	erc.logger.Info("Would restore controllers in a real implementation", "controller_count", len(snapshot.Controllers))

	// Restore operational state
	erc.quarantineList = make(map[string]time.Time)
	if ql, ok := snapshot.OperationalState["quarantine_list"].(map[string]interface{}); ok {
		for k, v := range ql {
			if t, ok := v.(string); ok {
				if parsedTime, err := time.Parse(time.RFC3339, t); err == nil {
					erc.quarantineList[k] = parsedTime
				}
			}
		}
	}

	erc.logger.Info("Agent state successfully restored from comprehensive snapshot")
	return nil
}

// ListSnapshots lists all available snapshots
func (erc *ExtendedRecoveryController) ListSnapshots() ([]string, error) {
	erc.logger.Info("Listing available snapshots")

	// Read directory
	entries, err := os.ReadDir(erc.snapshotStoragePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Directory doesn't exist, return empty list
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read snapshot directory: %w", err)
	}

	// Filter for snapshot files
	var snapshots []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" && len(entry.Name()) > 8 {
			// Check if it's likely a snapshot file (starts with "snapshot_")
			if entry.Name()[:9] == "snapshot_" {
				snapshots = append(snapshots, filepath.Join(erc.snapshotStoragePath, entry.Name()))
			}
		}
	}

	erc.logger.Info("Found snapshots", "count", len(snapshots))
	return snapshots, nil
}

// DeleteSnapshot deletes a snapshot file
func (erc *ExtendedRecoveryController) DeleteSnapshot(snapshotFile string) error {
	erc.logger.Info("Deleting snapshot", "file", snapshotFile)

	// Verify the file is in the snapshot storage path
	absSnapshotFile, err := filepath.Abs(snapshotFile)
	if err != nil {
		return fmt.Errorf("failed to get absolute path of snapshot file: %w", err)
	}

	absStoragePath, err := filepath.Abs(erc.snapshotStoragePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path of storage directory: %w", err)
	}

	// Ensure the file is within the snapshot storage path for security
	if len(absSnapshotFile) <= len(absStoragePath) || absSnapshotFile[:len(absStoragePath)] != absStoragePath {
		return fmt.Errorf("snapshot file is not within the snapshot storage path")
	}

	err = os.Remove(snapshotFile)
	if err != nil {
		return fmt.Errorf("failed to delete snapshot file: %w", err)
	}

	erc.logger.Info("Snapshot deleted successfully", "file", snapshotFile)
	return nil
}

// calculateSnapshotChecksum calculates SHA-256 checksum of snapshot data (excluding the checksum field)
func (erc *ExtendedRecoveryController) calculateSnapshotChecksum(snapshot *SnapshotData) (string, error) {
	// Create a copy of the snapshot without the checksum field
	snapshotCopy := *snapshot
	snapshotCopy.Checksum = ""

	// Serialize the snapshot data
	data, err := json.Marshal(snapshotCopy)
	if err != nil {
		return "", fmt.Errorf("failed to serialize snapshot for checksum calculation: %w", err)
	}

	// Calculate SHA-256 hash
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// SchedulePeriodicSnapshots schedules automatic snapshots at regular intervals
func (erc *ExtendedRecoveryController) SchedulePeriodicSnapshots(ctx context.Context, agentID string, interval time.Duration) {
	erc.logger.Info("Scheduling periodic snapshots", "interval", interval)

	// In a real implementation, this would:
	// 1. Create a ticker for the specified interval
	// 2. Create snapshots periodically
	// 3. Handle cleanup of old snapshots
	
	// For now, we'll just log the action
	erc.logger.Info("Would schedule periodic snapshots every", "interval", interval)
}