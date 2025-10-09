import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/naviNBRuas/APA/pkg/module"
	"github.com/naviNBRuas/APA/pkg/networking"
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
	P2P                networking.Config `yaml:"p2p"`
	Update             update.Config     `yaml:"update"`
}

// Runtime is the core agent runtime. It manages all agent components.
type Runtime struct {
	config        *Config
	logger        *slog.Logger
	identity      *Identity
	server        *http.Server
	moduleManager *module.Manager
	p2p           *networking.P2P
	updateManager *update.Manager
}

// NewRuntime creates a new agent runtime.
func NewRuntime(configPath string, version string) (*Runtime, error) {
	ctx := context.Background()

	// Load configuration
	config, err := loadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

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
	identity, err := NewIdentity()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize identity: %w", err)
	}
	logger.Info("Identity initialized", "agent_peer_id", identity.PeerID)

	// Initialize Module Manager
	moduleManager, err := module.NewManager(ctx, logger, config.ModulePath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize module manager: %w", err)
	}

	// Initialize P2P Networking
	p2p, err := networking.NewP2P(ctx, logger, config.P2P, identity.PeerID, identity.PrivKey)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize P2P networking: %w", err)
	}

	// Initialize Update Manager
	updateManager, err := update.NewManager(logger, config.Update, version)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize update manager: %w", err)
	}

	rt := &Runtime{
		config:        config,
		logger:        logger,
		identity:      identity,
		moduleManager: moduleManager,
		p2p:           p2p,
		updateManager: updateManager,
	}

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

	return rt, nil
}

// Start starts the agent runtime and blocks until shutdown.
func (rt *Runtime) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rt.logger.Info("Starting APA agent runtime", "version", rt.updateManager.currentVersion)

	// Start the update checker
	go rt.updateManager.StartPeriodicCheck(ctx, rt.config.Update.CheckInterval)

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
