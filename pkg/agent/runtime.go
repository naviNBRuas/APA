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

	"github.com/naviNBRuas/APA/pkg/controller"
	manager "github.com/naviNBRuas/APA/pkg/controller/manager"
	"github.com/naviNBRuas/APA/pkg/controller/task-orchestrator"
	"github.com/naviNBRuas/APA/pkg/health"
	"github.com/naviNBRuas/APA/pkg/module"
	"github.com/naviNBRuas/APA/pkg/networking"
	"github.com/naviNBRuas/APA/pkg/policy"
	"github.com/naviNBRuas/APA/pkg/recovery"
	"github.com/naviNBRuas/APA/pkg/update"
	"gopkg.in/yaml.v3"
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

	// Initialize Recovery Controller
	recoveryController := recovery.NewRecoveryController(logger, config, rt.ApplyConfig, p2p, moduleManager, controllerManager)
	rt.recoveryController = recoveryController

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
	mux.HandleFunc("/admin/health", rt.healthHandler)
	mux.HandleFunc("/admin/status", rt.statusHandler)
	mux.HandleFunc("/admin/modules/list", rt.listModulesHandler)
	mux.HandleFunc("/admin/update/check", rt.updateCheckHandler)

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
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}

// statusHandler is the handler for the /admin/status endpoint.
func (rt *Runtime) statusHandler(w http.ResponseWriter, r *http.Request) {
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

// listModulesHandler is the handler for the /admin/modules/list endpoint.
func (rt *Runtime) listModulesHandler(w http.ResponseWriter, r *http.Request) {
	modules := rt.moduleManager.ListModules()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(modules); err != nil {
		rt.logger.Error("Failed to encode modules list response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// updateCheckHandler is the handler for the /admin/update/check endpoint.
func (rt *Runtime) updateCheckHandler(w http.ResponseWriter, r *http.Request) {
	go rt.updateManager.CheckForUpdate()
	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintln(w, "Update check initiated.")
}
