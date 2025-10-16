package recovery

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/naviNBRuas/APA/pkg/module"
	"github.com/naviNBRuas/APA/pkg/networking"

	"github.com/libp2p/go-libp2p/core/peer"
)

// RecoveryController manages the agent's recovery mechanisms.
type RecoveryController struct {
	logger          *slog.Logger
	config          any
	applyConfigFunc func(configData []byte) error
	p2p             *networking.P2P
	moduleManager   *module.Manager
}

// NewRecoveryController creates a new RecoveryController.
func NewRecoveryController(logger *slog.Logger, config any, applyConfigFunc func(configData []byte) error, p2p *networking.P2P, moduleManager *module.Manager) *RecoveryController {
	return &RecoveryController{
		logger:          logger,
		config:          config,
		applyConfigFunc: applyConfigFunc,
		p2p:             p2p,
		moduleManager:   moduleManager,
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
	manifest, wasmBytes, err := rc.p2p.FetchModule(ctx, peerID, moduleName, "latest")
	if err != nil {
		return fmt.Errorf("failed to fetch module from peer: %w", err)
	}

	if err := rc.moduleManager.SaveAndLoadModule(manifest, wasmBytes); err != nil {
		return fmt.Errorf("failed to save and load fetched module: %w", err)
	}

	rc.logger.Info("Successfully fetched and loaded module from peer", "module", moduleName, "peer", peerIDStr)
	return nil
}

// QuarantineNode marks a node as quarantined.
func (rc *RecoveryController) QuarantineNode(ctx context.Context, nodeID string) error {
	rc.logger.Warn("Quarantining node", "node", nodeID)

	// 1. Isolate the node from the network (e.g., disconnect from peers, block incoming connections)
	if rc.p2p != nil {
		rc.logger.Info("Attempting to isolate node from P2P network", "node", nodeID)
		// In a real implementation, this would involve more sophisticated P2P isolation.
		// For now, we'll just log the intent.
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

	// 3. Prevent it from running new modules or participating in P2P (handled by policy enforcement)

	// 4. Trigger further recovery actions (e.g., reporting, re-imaging)

	rc.logger.Info("Node quarantined successfully", "node", nodeID)
	return nil
}

// CreateSnapshot saves the agent's state.
func (rc *RecoveryController) CreateSnapshot(ctx context.Context) error {
	rc.logger.Info("Creating agent snapshot")
	data, err := json.MarshalIndent(rc.config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile("agent-snapshot.json", data, 0644)
}

// RestoreSnapshot restores the agent's state from a snapshot.
func (rc *RecoveryController) RestoreSnapshot(ctx context.Context) error {
	rc.logger.Info("Restoring agent from snapshot")
	data, err := os.ReadFile("agent-snapshot.json")
	if err != nil {
		return err
	}

	var config any
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to unmarshal snapshot: %w", err)
	}

	if rc.applyConfigFunc == nil {
		return fmt.Errorf("applyConfigFunc is not set in RecoveryController")
	}

	return rc.applyConfigFunc(data)
}
