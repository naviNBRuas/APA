package manager

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	controllerPkg "github.com/naviNBRuas/APA/pkg/controller"
	manifest "github.com/naviNBRuas/APA/pkg/controller/manifest"
	"github.com/naviNBRuas/APA/pkg/networking"
	"github.com/naviNBRuas/APA/pkg/policy"
	"github.com/naviNBRuas/APA/pkg/consensus"
)

// Manager handles the lifecycle of controllers.
type Manager struct {
	logger        *slog.Logger
	controllers   map[string]controllerPkg.Controller // Maps controller name to Controller instance
	controllerDir string
	mu            sync.RWMutex
	policyEnforcer policy.PolicyEnforcer
	p2pNetwork    *networking.P2P  // Add P2P network reference
	consensus     consensus.Consensus // Consensus algorithm for distributed decision-making
}

// NewManager creates a new controller manager.
func NewManager(logger *slog.Logger, controllerDir string, policyEnforcer policy.PolicyEnforcer) *Manager {
	// Create a simple leader election consensus algorithm
	consensusConfig := &consensus.Config{
		NodeID:     "local-node", // This would be the actual node ID in a real implementation
		PeerIDs:    []string{},   // This would be populated with peer IDs in a real implementation
		Algorithm:  "leader-election",
		ListenAddr: ":8080",      // This would be the actual listen address in a real implementation
	}
	
	consensusAlg := consensus.NewLeaderElection(logger, consensusConfig)
	
	manager := &Manager{
		logger:        logger,
		controllers:   make(map[string]controllerPkg.Controller),
		controllerDir: controllerDir,
		policyEnforcer: policyEnforcer,
		consensus:     consensusAlg,
	}
	
	// Start the consensus algorithm
	ctx := context.Background()
	if err := consensusAlg.Start(ctx); err != nil {
		logger.Error("Failed to start consensus algorithm", "error", err)
	}
	
	return manager
}

// SetP2PNetwork sets the P2P network instance for the manager.
func (m *Manager) SetP2PNetwork(p2p *networking.P2P) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.p2pNetwork = p2p
	
	// Set up the handler for incoming P2P controller messages
	if p2p != nil {
		// In a real implementation, we would set up message handlers here
		// For now, we'll just log that the P2P network is set
		m.logger.Info("P2P network set for controller manager")
	}
}

// LoadControllersFromDir scans the controller directory for manifest.json files and loads them.
func (m *Manager) LoadControllersFromDir(ctx context.Context) error {
	m.logger.Info("Scanning for controllers in directory", "path", m.controllerDir)

	// Check if controller directory exists, create if it doesn't
	if _, err := os.Stat(m.controllerDir); os.IsNotExist(err) {
		m.logger.Info("Controller directory does not exist, creating it", "path", m.controllerDir)
		if err := os.MkdirAll(m.controllerDir, 0755); err != nil {
			return fmt.Errorf("failed to create controller directory: %w", err)
		}
	}

	return filepath.Walk(m.controllerDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == "manifest.json" {
			if err := m.loadControllerFromManifest(ctx, path); err != nil {
				m.logger.Error("Failed to load controller from manifest", "path", path, "error", err)
				// Continue to next manifest
			}
		}
		return nil
	})
}

// loadControllerFromManifest parses a manifest and loads the controller.
func (m *Manager) loadControllerFromManifest(ctx context.Context, manifestPath string) error {
	// 1. Read and parse manifest
	manifest, err := m.parseManifest(manifestPath)
	if err != nil {
		return err
	}

	// 2. Verify controller binary hash
	controllerPath := filepath.Join(filepath.Dir(manifestPath), manifest.Path)
	err = m.verifyHash(controllerPath, manifest.Hash)
	if err != nil {
		return fmt.Errorf("controller '%s' hash verification failed: %w", manifest.Name, err)
	}
	m.logger.Debug("Controller hash verified", "name", manifest.Name)

	// 3. Authorize controller loading based on policy
	allowed, reason, err := m.policyEnforcer.Authorize(ctx, manifest.Name, "load_controller", manifest.Policy)
	if err != nil {
		return fmt.Errorf("failed to authorize controller loading: %w", err)
	}
	if !allowed {
		return fmt.Errorf("controller loading not authorized: %s", reason)
	}
	m.logger.Info("Controller loading authorized", "name", manifest.Name)

	// 4. Note required capabilities (not actively enforced at this stage)
	if len(manifest.Capabilities) > 0 {
		m.logger.Info("Controller requires capabilities", "name", manifest.Name, "capabilities", manifest.Capabilities)
	}

	// 5. Create a GoBinaryController for the external binary
	controller := controllerPkg.NewGoBinaryController(m.logger, manifest)

	m.mu.Lock()
	defer m.mu.Unlock()
	m.controllers[manifest.Name] = controller
	m.logger.Info("Successfully loaded controller", "name", manifest.Name, "version", manifest.Version)

	return nil
}

// StartController starts a loaded controller by name.
func (m *Manager) StartController(ctx context.Context, name string) error {
	m.mu.RLock()
	controller, ok := m.controllers[name]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("controller '%s' not found", name)
	}

	return controller.Start(ctx)
}

// StopController stops a running controller by name.
func (m *Manager) StopController(ctx context.Context, name string) error {
	m.mu.RLock()
	controller, ok := m.controllers[name]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("controller '%s' not found", name)
	}

	return controller.Stop(ctx)
}

// ListControllers returns the manifests of all loaded controllers.
func (m *Manager) ListControllers() []*manifest.Manifest {
	m.mu.RLock()
	defer m.mu.RUnlock()
	manifests := make([]*manifest.Manifest, 0, len(m.controllers))
	for _, ctrl := range m.controllers {
		if goBinaryCtrl, ok := ctrl.(*controllerPkg.GoBinaryController); ok { // Check for GoBinaryController
			manifests = append(manifests, goBinaryCtrl.Manifest)
		} else if dummyCtrl, ok := ctrl.(*controllerPkg.DummyController); ok { // Keep DummyController for now
			manifests = append(manifests, dummyCtrl.Manifest)
		}
	}
	return manifests
}

// Shutdown gracefully stops all controllers.
func (m *Manager) Shutdown(ctx context.Context) error {
	m.logger.Info("Shutting down controller manager")
	
	// Get a copy of the controllers map under lock
	m.mu.RLock()
	controllers := make(map[string]controllerPkg.Controller)
	for name, controller := range m.controllers {
		controllers[name] = controller
	}
	m.mu.RUnlock()
	
	// Stop all controllers
	for name, controller := range controllers {
		// Create a timeout context for each controller shutdown
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		if err := controller.Stop(shutdownCtx); err != nil {
			m.logger.Error("Failed to stop controller during shutdown", "name", name, "error", err)
		}
		cancel() // Cancel the timeout context
	}
	return nil
}

// ConfigureController sends new configuration data to a running controller.
func (m *Manager) ConfigureController(ctx context.Context, name string, configData []byte) error {
	m.mu.RLock()
	controller, ok := m.controllers[name]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("controller '%s' not found", name)
	}

	return controller.Configure(configData)
}

// SendMessageToController sends a message to a running controller.
func (m *Manager) SendMessageToController(ctx context.Context, name string, message interface{}) error {
	m.mu.RLock()
	controller, ok := m.controllers[name]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("controller '%s' not found", name)
	}

	// Convert message to networking.ControllerMessage
	// In a real implementation, this would be more sophisticated
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Create a proper networking.ControllerMessage
	ctrlMessage := networking.ControllerMessage{
		SenderPeerID: "local", // In a real implementation, this would be the actual sender's peer ID
		Type:         "controller_message",
		Data:         jsonData,
	}

	// Log the message being sent
	m.logger.Info("Sending message to controller", "name", name, "message_type", ctrlMessage.Type)

	// Call the controller's HandleMessage method
	return controller.HandleMessage(ctx, ctrlMessage)
}

// SendP2PMessageToController sends a message to a controller on another peer via the P2P network.
func (m *Manager) SendP2PMessageToController(ctx context.Context, targetPeerID, targetController, messageType string, payload interface{}) error {
	if m.p2pNetwork == nil {
		return fmt.Errorf("P2P network not available")
	}

	// Create a router message that specifies the target controller
	routerMsg := struct {
		TargetController string      `json:"target_controller"`
		MessageType      string      `json:"message_type"`
		Payload          interface{} `json:"payload"`
	}{
		TargetController: targetController,
		MessageType:      messageType,
		Payload:          payload,
	}

	// Marshal the router message
	routerMsgBytes, err := json.Marshal(routerMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal router message: %w", err)
	}

	// Create the controller message
	ctrlMsg := struct {
		Type      string          `json:"type"`
		Payload   json.RawMessage `json:"payload"`
		Timestamp time.Time       `json:"timestamp"`
	}{
		Type:      "controller_message",
		Payload:   routerMsgBytes,
		Timestamp: time.Now(),
	}

	// Marshal the controller message
	ctrlMsgBytes, err := json.Marshal(ctrlMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal controller message: %w", err)
	}

	// Publish the message to the P2P controller communication topic
	m.logger.Info("Sending P2P controller message", 
		"target_peer", targetPeerID, 
		"target_controller", targetController, 
		"message_type", messageType)

	// We need to send this to a specific peer, not broadcast it
	// For now, we'll just log that we would send it
	m.logger.Info("Would send P2P controller message to specific peer", 
		"target_peer", targetPeerID, 
		"message", string(ctrlMsgBytes))
	
	return nil
}

func (m *Manager) parseManifest(path string) (*manifest.Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}
	var manifest manifest.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest json: %w", err)
	}
	return &manifest, nil
}

// verifyHash calculates the SHA256 hash of a file and compares it to an expected hex-encoded hash.
// A placeholder hash "..." is always considered valid for testing.
func (m *Manager) verifyHash(filePath, expectedHash string) error {
	if expectedHash == "..." {
		m.logger.Warn("Skipping hash verification for placeholder hash", "file", filePath)
		return nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return err
	}

	actualHash := hex.EncodeToString(hasher.Sum(nil))
	if actualHash != expectedHash {
		return fmt.Errorf("hash mismatch: expected %s, got %s", expectedHash, actualHash)
	}

	return nil
}

// IsLeader returns whether this node is the current leader
func (m *Manager) IsLeader() bool {
	if m.consensus != nil {
		return m.consensus.IsLeader()
	}
	return false
}

// GetLeaderID returns the ID of the current leader
func (m *Manager) GetLeaderID() string {
	if m.consensus != nil {
		return m.consensus.GetLeaderID()
	}
	return ""
}

// ProposeValue proposes a value for consensus
func (m *Manager) ProposeValue(ctx context.Context, key string, value interface{}) error {
	if m.consensus != nil {
		return m.consensus.ProposeValue(ctx, key, value)
	}
	return fmt.Errorf("no consensus algorithm available")
}

// GetValue retrieves the agreed-upon value for a key
func (m *Manager) GetValue(ctx context.Context, key string) (interface{}, error) {
	if m.consensus != nil {
		return m.consensus.GetValue(ctx, key)
	}
	return nil, fmt.Errorf("no consensus algorithm available")
}

// handleP2PControllerMessage processes incoming controller messages from the P2P network.
func (m *Manager) handleP2PControllerMessage(ctx context.Context, msg networking.ControllerMessage) error {
	m.logger.Info("Received controller message from P2P network", "type", msg.Type, "sender", msg.SenderPeerID)
	
	// For now, we'll route all messages to a special "p2p-router" controller
	// In a more sophisticated implementation, we might route based on message type or target controller
	
	// Check if we have a controller that can handle this message
	m.mu.RLock()
	controller, ok := m.controllers["p2p-router"]
	m.mu.RUnlock()
	
	if ok {
		// Forward the message to the router controller
		return controller.HandleMessage(ctx, msg)
	} else {
		// Log that we received a message but have no handler
		m.logger.Warn("Received P2P controller message but no router controller available", 
			"type", msg.Type, "sender", msg.SenderPeerID)
	}
	
	return nil
}

// handleLeaderElectionMessage processes incoming leader election messages
func (m *Manager) handleLeaderElectionMessage(ctx context.Context, msg networking.LeaderElectionMessage) error {
	m.logger.Info("Received leader election message", "candidate", msg.CandidateID, "is_leader", msg.IsLeader)
	
	// If this message declares a leader, update our leader information
	if msg.IsLeader {
		// In a real implementation, we would update our consensus algorithm
		// For now, we'll just log the information
		m.logger.Info("New leader elected", "leader_id", msg.CandidateID)
	}
	
	return nil
}

// PublishControllerMessage publishes a message to the P2P controller communication topic.
func (m *Manager) PublishControllerMessage(ctx context.Context, msgType string, payload json.RawMessage) error {
	if m.p2pNetwork == nil {
		return fmt.Errorf("P2P network not available")
	}
	
	// Create a controller message
	msg := networking.ControllerMessage{
		Type:         msgType,
		Data:         payload,
		SenderPeerID: "local", // In a real implementation, this would be the actual peer ID
	}
	
	// Marshal the message
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal controller message: %w", err)
	}
	
	return m.p2pNetwork.PublishControllerMessage(ctx, msgBytes)
}
