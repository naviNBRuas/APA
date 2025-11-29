package recovery

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"
	"crypto/sha256"
	"encoding/hex"

	"github.com/naviNBRuas/APA/pkg/module"
	controllerManifest "github.com/naviNBRuas/APA/pkg/controller/manifest"

	"github.com/libp2p/go-libp2p/core/peer"
	"gopkg.in/yaml.v3"
)

// P2PService defines the interface for P2P operations used by RecoveryController.
type P2PService interface {
	FetchModule(ctx context.Context, peerID peer.ID, name, version string) (*module.Manifest, []byte, error)
	ClosePeer(peerID peer.ID) error
}

// ModuleManagerService defines the interface for ModuleManager operations used by RecoveryController.
type ModuleManagerService interface {
	SaveAndLoadModule(manifest *module.Manifest, wasmBytes []byte) error
	ListModules() []*module.Manifest
	StopModule(name string) error
}

// ControllerManagerService defines the interface for ControllerManager operations used by RecoveryController.
type ControllerManagerService interface {
	ListControllers() []*controllerManifest.Manifest
	StopController(ctx context.Context, name string) error
}

// RecoveryController manages the agent's recovery mechanisms.
type RecoveryController struct {
	logger          *slog.Logger
	config          any
	applyConfigFunc func(configData []byte) error
	p2p             P2PService
	moduleManager   ModuleManagerService
	controllerManager ControllerManagerService
	snapshotPath    string
	quarantineList  map[string]time.Time
}

// NewRecoveryController creates a new RecoveryController.
func NewRecoveryController(logger *slog.Logger, config any, applyConfigFunc func(configData []byte) error, p2p P2PService, moduleManager ModuleManagerService, controllerManager ControllerManagerService) *RecoveryController {
	return &RecoveryController{
		logger:          logger,
		config:          config,
		applyConfigFunc: applyConfigFunc,
		p2p:             p2p,
		moduleManager:   moduleManager,
		controllerManager: controllerManager,
		snapshotPath:    "agent-snapshot.yaml",
		quarantineList:  make(map[string]time.Time),
	}
}

// RequestPeerCopy requests a module artifact from a trusted peer.
func (rc *RecoveryController) RequestPeerCopy(ctx context.Context, moduleName string, peerIDStr string) error {
	rc.logger.Info("Requesting peer copy", "module", moduleName, "peer", peerIDStr)

	peerID, err := peer.Decode(peerIDStr)
	if err != nil {
		return fmt.Errorf("failed to decode peer ID: %w", err)
	}

	// For now, we assume version is latest
	version := "latest"
	manifest, wasmBytes, err := rc.p2p.FetchModule(ctx, peerID, moduleName, version)
	if err != nil {
		return fmt.Errorf("failed to fetch module from peer: %w", err)
	}

	// Verify the manifest and WASM bytes
	if err := rc.verifyModule(manifest, wasmBytes); err != nil {
		return fmt.Errorf("module verification failed: %w", err)
	}

	if err := rc.moduleManager.SaveAndLoadModule(manifest, wasmBytes); err != nil {
		return fmt.Errorf("failed to save and load fetched module: %w", err)
	}

	rc.logger.Info("Successfully fetched and loaded module from peer", "module", moduleName, "peer", peerIDStr)
	return nil
}

// verifyModule performs basic verification of a module's manifest and WASM bytes
func (rc *RecoveryController) verifyModule(manifest *module.Manifest, wasmBytes []byte) error {
	// Check that required fields are present
	if manifest.Name == "" {
		return fmt.Errorf("module manifest missing name")
	}
	if manifest.Version == "" {
		return fmt.Errorf("module manifest missing version")
	}
	if manifest.Hash == "" {
		return fmt.Errorf("module manifest missing hash")
	}

	// Calculate the actual hash of the WASM bytes
	hasher := sha256.New()
	hasher.Write(wasmBytes)
	actualHash := hex.EncodeToString(hasher.Sum(nil))

	// Compare with the expected hash from the manifest
	if actualHash != manifest.Hash {
		return fmt.Errorf("module hash mismatch: expected %s, got %s", manifest.Hash, actualHash)
	}

	rc.logger.Info("Module verification successful", "module", manifest.Name, "hash", actualHash)
	return nil
}

// QuarantineNode marks a node as quarantined.
func (rc *RecoveryController) QuarantineNode(ctx context.Context, nodeID string) error {
	rc.logger.Warn("Quarantining node", "node", nodeID)

	peerID, err := peer.Decode(nodeID)
	if err != nil {
		return fmt.Errorf("failed to decode peer ID: %w", err)
	}

	// 1. Isolate the node from the network (e.g., disconnect from peers, block incoming connections)
	if rc.p2p != nil {
		rc.logger.Info("Attempting to isolate node from P2P network", "node", nodeID)
		// Disconnect from the peer
		if err := rc.p2p.ClosePeer(peerID); err != nil {
			rc.logger.Error("Failed to close peer connection during quarantine", "peer", peerID, "error", err)
		} else {
			rc.logger.Info("Successfully closed peer connection", "peer", peerID)
		}
	}

	// 2. Stop all running modules on the node
	if rc.moduleManager != nil {
		rc.logger.Info("Attempting to stop all running modules on node", "node", nodeID)
		for _, manifest := range rc.moduleManager.ListModules() {
			if err := rc.moduleManager.StopModule(manifest.Name); err != nil {
				rc.logger.Error("Failed to stop module during quarantine", "module", manifest.Name, "error", err)
			}
		}
	}

	// Stop all running controllers on the node
	if rc.controllerManager != nil {
		rc.logger.Info("Attempting to stop all running controllers on node", "node", nodeID)
		for _, manifest := range rc.controllerManager.ListControllers() {
			if err := rc.controllerManager.StopController(ctx, manifest.Name); err != nil {
				rc.logger.Error("Failed to stop controller during quarantine", "controller", manifest.Name, "error", err)
			}
		}
	}

	// 3. Prevent it from running new modules or participating in P2P (handled by policy enforcement)
	rc.logger.Warn("Dynamic policy update for quarantined node is not yet fully implemented.", "node", nodeID)

	// 4. Add node to quarantine list with timestamp
	rc.quarantineList[nodeID] = time.Now()

	// 5. Trigger further recovery actions (e.g., reporting, re-imaging)
	rc.logger.Info("Initiating recovery actions for quarantined node", "node", nodeID)

	rc.logger.Info("Node quarantined successfully", "node", nodeID)
	return nil
}

// CreateSnapshot saves the agent's state.
func (rc *RecoveryController) CreateSnapshot(ctx context.Context) error {
	rc.logger.Info("Creating agent snapshot")
	data, err := yaml.Marshal(rc.config)
	if err != nil {
		return err
	}
	return os.WriteFile("agent-snapshot.yaml", data, 0644)
}

// RestoreSnapshot restores the agent's state from a snapshot.
func (rc *RecoveryController) RestoreSnapshot(ctx context.Context) error {
	rc.logger.Info("Restoring agent from snapshot")
	data, err := os.ReadFile("agent-snapshot.yaml")
	if err != nil {
		return err
	}

	if rc.applyConfigFunc == nil {
		return fmt.Errorf("applyConfigFunc is not set in RecoveryController")
	}

	// Apply the configuration to reconfigure all components
	if err := rc.applyConfigFunc(data); err != nil {
		return fmt.Errorf("failed to apply configuration during restore: %w", err)
	}

	// After applying the config, we need to ensure modules are properly loaded
	// This would typically be handled by the ApplyConfigFunc which restarts the agent
	// But we can add additional verification steps here if needed

	rc.logger.Info("Agent state successfully restored from snapshot")
	return nil
}
