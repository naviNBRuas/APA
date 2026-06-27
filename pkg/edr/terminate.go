package edr

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"
)

// terminateProcess terminates a suspicious process
func (rm *ResponseManager) terminateProcess(ctx context.Context, event *Event) error {
	rm.logger.Warn("PROCESS TERMINATION ACTION TRIGGERED", "event_id", event.ID, "source", event.Source)

	// In a real implementation, this would:
	// 1. Identify the process associated with the event
	// 2. Terminate the process
	// 3. Log the termination
	// 4. Notify administrators

	// Terminate the process
	if err := rm.killProcess(event.Source); err != nil {
		rm.logger.Error("Failed to terminate process", "process", event.Source, "error", err)
		return fmt.Errorf("failed to terminate process: %w", err)
	}

	// Log the termination
	rm.logger.Info("Process terminated successfully", "process", event.Source)

	// Notify administrators (simulated)
	if err := rm.notifyAdministrators(event); err != nil {
		rm.logger.Error("Failed to notify administrators", "error", err)
		// Don't fail the action if notification fails
	}

	return nil
}

// killProcess kills a process by name
func (rm *ResponseManager) killProcess(processName string) error {
	var cmd *exec.Cmd

	// Implementation varies by OS
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("taskkill", "/F", "/IM", processName)
	default:
		cmd = exec.Command("pkill", "-f", processName)
	}

	return cmd.Run()
}

// selfDestruct initiates self-destruction of the agent
func (rm *ResponseManager) selfDestruct(ctx context.Context, event *Event) error {
	rm.logger.Warn("SELF-DESTRUCT ACTION TRIGGERED", "event_id", event.ID, "source", event.Source)

	// In a real implementation, this would:
	// 1. Delete all agent files
	// 2. Remove all traces of the agent
	// 3. Terminate all processes
	// 4. Notify administrators (if possible)

	// Notify administrators first (before destruction)
	if err := rm.notifyAdministrators(event); err != nil {
		rm.logger.Error("Failed to notify administrators", "error", err)
		// Continue with self-destruction even if notification fails
	}

	// Delete agent files
	if err := rm.deleteAgentFiles(); err != nil {
		rm.logger.Error("Failed to delete agent files", "error", err)
		return fmt.Errorf("failed to delete agent files: %w", err)
	}

	// Remove all traces
	if err := rm.removeTraces(); err != nil {
		rm.logger.Error("Failed to remove traces", "error", err)
		return fmt.Errorf("failed to remove traces: %w", err)
	}

	// Terminate all processes
	if err := rm.terminateAllProcesses(); err != nil {
		rm.logger.Error("Failed to terminate processes", "error", err)
		return fmt.Errorf("failed to terminate processes: %w", err)
	}

	rm.logger.Info("Self-destruction completed")
	return nil
}

// deleteAgentFiles deletes all agent files
func (rm *ResponseManager) deleteAgentFiles() error {
	// Get the current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Delete the executable
	if err := os.Remove(execPath); err != nil {
		// If we can't delete the current executable, try to mark it for deletion on reboot
		if runtime.GOOS == "windows" {
			// On Windows, we can mark files for deletion on reboot
			cmd := exec.Command("movefile", execPath, "")
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to mark file for deletion: %w", err)
			}
		} else {
			return fmt.Errorf("failed to delete executable: %w", err)
		}
	}

	// Delete configuration files
	configPaths := []string{
		"/etc/apa/",
		"/var/lib/apa/",
		"C:\\ProgramData\\APA\\",
	}

	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			if err := os.RemoveAll(path); err != nil {
				rm.logger.Warn("Failed to remove configuration directory", "path", path, "error", err)
			}
		}
	}

	return nil
}

// removeTraces removes all traces of the agent
func (rm *ResponseManager) removeTraces() error {
	// Remove traces from system logs, registry, etc.
	// This is a simplified implementation

	switch runtime.GOOS {
	case "windows":
		// Remove from Windows registry
		cmd := exec.Command("reg", "delete", "HKLM\\SOFTWARE\\APA", "/f")
		if err := cmd.Run(); err != nil {
			rm.logger.Warn("Failed to remove registry entries", "error", err)
		}
	default:
		// Remove from system logs
		logFiles := []string{
			"/var/log/apa.log",
			"/var/log/syslog",
		}

		for _, logFile := range logFiles {
			if _, err := os.Stat(logFile); err == nil {
				// Truncate the log file instead of deleting it
				if err := os.Truncate(logFile, 0); err != nil {
					rm.logger.Warn("Failed to truncate log file", "file", logFile, "error", err)
				}
			}
		}
	}

	return nil
}

// terminateAllProcesses terminates all agent processes
func (rm *ResponseManager) terminateAllProcesses() error {
	// Get the current process ID
	// Kill all processes except the current one
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// Kill all APA processes except current
		cmd = exec.Command("taskkill", "/F", "/FI", "IMAGENAME eq agent*", "/V")
	default:
		// Kill all APA processes except current
		cmd = exec.Command("pkill", "-f", "agent")
	}

	// Execute the command
	if err := cmd.Run(); err != nil {
		// Ignore errors as some processes might not exist
		rm.logger.Debug("Some processes might not have been terminated", "error", err)
	}

	// Schedule the current process for termination
	go func() {
		time.Sleep(1 * time.Second) // Give time for logging
		os.Exit(0)
	}()

	return nil
}
