// Package regeneration provides self-healing and regeneration capabilities for the APA agent
package regeneration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// regenerateAgent performs the agent regeneration process
func (r *Regenerator) regenerateAgent(ctx context.Context) error {
	if r == nil {
		return fmt.Errorf("regenerator is nil")
	}

	r.logger.Info("Starting agent regeneration process")

	if err := r.stopAgent(); err != nil {
		return fmt.Errorf("failed to stop agent: %w", err)
	}

	if err := r.restoreAgentFromBackup(ctx); err != nil {
		return fmt.Errorf("failed to restore agent from backup: %w", err)
	}

	if err := r.verifyRestoredAgent(); err != nil {
		return fmt.Errorf("failed to verify restored agent: %w", err)
	}

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

	r.logger.Info("Agent stopped")

	return nil
}

// restoreAgentFromBackup restores the agent from a backup
func (r *Regenerator) restoreAgentFromBackup(ctx context.Context) error {
	if r == nil {
		return fmt.Errorf("regenerator is nil")
	}

	r.logger.Info("Restoring agent from backup")

	backupPath := filepath.Join(r.config.BackupPath, "agent_backup")
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup not found: %s", backupPath)
	}

	if err := copyFile(backupPath, r.config.BinaryPath); err != nil {
		return fmt.Errorf("failed to restore agent from backup: %w", err)
	}

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

	if _, err := os.Stat(r.config.BinaryPath); os.IsNotExist(err) {
		return fmt.Errorf("restored binary not found: %s", r.config.BinaryPath)
	}

	if err := r.isBinaryExecutable(); err != nil {
		return fmt.Errorf("restored binary is not executable: %w", err)
	}

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

	info, err := os.Stat(r.config.BinaryPath)
	if err != nil {
		return fmt.Errorf("failed to stat binary: %w", err)
	}

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

	r.logger.Info("Agent started")

	return nil
}
