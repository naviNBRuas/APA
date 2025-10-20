package recovery

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/naviNBRuas/APA/pkg/module"
	"github.com/naviNBRuas/APA/pkg/networking"
	manager "github.com/naviNBRuas/APA/pkg/controller/manager"

	"github.com/libp2p/go-libp2p/core/peer"
	"gopkg.in/yaml.v3"
)

// RecoveryController manages the agent's recovery mechanisms.
type RecoveryController struct {
	logger          *slog.Logger
	config          any
	applyConfigFunc func(configData []byte) error
	p2p             *networking.P2P
	moduleManager   *module.Manager
	controllerManager *manager.Manager
}

// NewRecoveryController creates a new RecoveryController.
func NewRecoveryController(logger *slog.Logger, config any, applyConfigFunc func(configData []byte) error, p2p *networking.P2P, moduleManager *module.Manager, controllerManager *manager.Manager) *RecoveryController {
	return &RecoveryController{
		logger:          logger,
		config:          config,
		applyConfigFunc: applyConfigFunc,
		p2p:             p2p,
		moduleManager:   moduleManager,
		controllerManager: controllerManager,
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

	peerID, err := peer.Decode(nodeID)
	if err != nil {
		return fmt.Errorf("failed to decode peer ID: %w", err)
	}

	// 1. Isolate the node from the network (e.g., disconnect from peers, block incoming connections)
	if rc.p2p != nil {
		rc.logger.Info("Attempting to isolate node from P2P network", "node", nodeID)
		// Disconnect from the peer
		if err := rc.p2p.Host.Network().ClosePeer(peerID); err != nil {
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

	// 4. Trigger further recovery actions (e.g., reporting, re-imaging)

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

	return rc.applyConfigFunc(data)
}
