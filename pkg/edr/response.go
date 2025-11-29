package edr

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// ResponseAction defines an automated response action
type ResponseAction struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ActionType  string    `json:"action_type"` // quarantine, terminate, isolate, self-destruct
	Severity    string    `json:"severity"`    // low, medium, high, critical
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
}

// ResponseManager handles automated response actions
type ResponseManager struct {
	logger        *slog.Logger
	actions       map[string]*ResponseAction
	responseRules map[string][]string // Map of severity to action IDs
}

// NewResponseManager creates a new response manager
func NewResponseManager(logger *slog.Logger) *ResponseManager {
	return &ResponseManager{
		logger:        logger,
		actions:       make(map[string]*ResponseAction),
		responseRules: make(map[string][]string),
	}
}

// AddAction adds a response action to the manager
func (rm *ResponseManager) AddAction(action *ResponseAction) error {
	rm.actions[action.ID] = action
	rm.logger.Info("Added response action", "id", action.ID, "name", action.Name, "type", action.ActionType)
	return nil
}

// RemoveAction removes a response action from the manager
func (rm *ResponseManager) RemoveAction(actionID string) error {
	if _, exists := rm.actions[actionID]; !exists {
		return nil // Already removed or doesn't exist
	}
	
	delete(rm.actions, actionID)
	rm.logger.Info("Removed response action", "id", actionID)
	return nil
}

// EnableAction enables a response action
func (rm *ResponseManager) EnableAction(actionID string) error {
	action, exists := rm.actions[actionID]
	if !exists {
		return nil // Action doesn't exist
	}
	
	action.Enabled = true
	rm.logger.Info("Enabled response action", "id", actionID)
	return nil
}

// DisableAction disables a response action
func (rm *ResponseManager) DisableAction(actionID string) error {
	action, exists := rm.actions[actionID]
	if !exists {
		return nil // Action doesn't exist
	}
	
	action.Enabled = false
	rm.logger.Info("Disabled response action", "id", actionID)
	return nil
}

// AddResponseRule adds a rule for automatic response based on event severity
func (rm *ResponseManager) AddResponseRule(severity string, actionIDs []string) error {
	rm.responseRules[severity] = actionIDs
	rm.logger.Info("Added response rule", "severity", severity, "action_count", len(actionIDs))
	return nil
}

// ExecuteResponse executes response actions for a given event
func (rm *ResponseManager) ExecuteResponse(ctx context.Context, event *Event) error {
	// Get actions for this event's severity
	actionIDs, exists := rm.responseRules[event.Severity]
	if !exists {
		rm.logger.Debug("No response rules for severity", "severity", event.Severity)
		return nil
	}
	
	rm.logger.Info("Executing response actions for event", 
		"event_id", event.ID, 
		"severity", event.Severity, 
		"action_count", len(actionIDs))
	
	// Execute each action
	for _, actionID := range actionIDs {
		if err := rm.executeAction(ctx, actionID, event); err != nil {
			rm.logger.Error("Failed to execute response action", "action_id", actionID, "error", err)
		}
	}
	
	return nil
}

// executeAction executes a single response action
func (rm *ResponseManager) executeAction(ctx context.Context, actionID string, event *Event) error {
	action, exists := rm.actions[actionID]
	if !exists {
		return nil // Action doesn't exist
	}
	
	if !action.Enabled {
		rm.logger.Debug("Skipping disabled action", "action_id", actionID)
		return nil
	}
	
	rm.logger.Info("Executing response action", 
		"action_id", actionID, 
		"action_type", action.ActionType,
		"event_id", event.ID)
	
	// Execute the appropriate action based on type
	switch action.ActionType {
	case "quarantine":
		return rm.quarantineNode(ctx, event)
	case "terminate":
		return rm.terminateProcess(ctx, event)
	case "isolate":
		return rm.isolateNetwork(ctx, event)
	case "self-destruct":
		return rm.selfDestruct(ctx, event)
	default:
		rm.logger.Warn("Unknown action type", "action_type", action.ActionType)
		return nil
	}
}

// quarantineNode quarantines the current node
func (rm *ResponseManager) quarantineNode(ctx context.Context, event *Event) error {
	rm.logger.Warn("QUARANTINE ACTION TRIGGERED", "event_id", event.ID, "source", event.Source)
	
	// In a real implementation, this would:
	// 1. Disconnect from the network
	// 2. Block all outgoing connections
	// 3. Notify administrators
	// 4. Enter a safe mode
	
	// Disconnect from network
	if err := rm.disconnectNetwork(); err != nil {
		rm.logger.Error("Failed to disconnect network", "error", err)
		return fmt.Errorf("failed to disconnect network: %w", err)
	}
	
	// Block all outgoing connections
	if err := rm.blockOutgoingConnections(); err != nil {
		rm.logger.Error("Failed to block outgoing connections", "error", err)
		return fmt.Errorf("failed to block outgoing connections: %w", err)
	}
	
	// Notify administrators (simulated)
	if err := rm.notifyAdministrators(event); err != nil {
		rm.logger.Error("Failed to notify administrators", "error", err)
		// Don't fail the action if notification fails
	}
	
	// Enter safe mode
	if err := rm.enterSafeMode(); err != nil {
		rm.logger.Error("Failed to enter safe mode", "error", err)
		return fmt.Errorf("failed to enter safe mode: %w", err)
	}
	
	rm.logger.Info("Node quarantined successfully")
	return nil
}

// disconnectNetwork disconnects the node from the network
func (rm *ResponseManager) disconnectNetwork() error {
	// Implementation varies by OS
	switch runtime.GOOS {
	case "windows":
		// Disable network adapters on Windows
		cmd := exec.Command("netsh", "interface", "set", "interface", "name=\"Ethernet\"", "admin=disable")
		if err := cmd.Run(); err != nil {
			// Try alternative approach
			cmd = exec.Command("ipconfig", "/release")
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to disconnect network: %w", err)
			}
		}
	case "darwin":
		// Disable network on macOS
		cmd := exec.Command("ifconfig", "en0", "down")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to disconnect network: %w", err)
		}
	default:
		// Disable network on Linux
		cmd := exec.Command("ifconfig", "eth0", "down")
		if err := cmd.Run(); err != nil {
			// Try alternative approach
			cmd = exec.Command("ip", "link", "set", "eth0", "down")
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to disconnect network: %w", err)
			}
		}
	}
	
	return nil
}

// blockOutgoingConnections blocks all outgoing connections
func (rm *ResponseManager) blockOutgoingConnections() error {
	// Implementation varies by OS
	switch runtime.GOOS {
	case "windows":
		// Block outgoing connections using Windows Firewall
		cmd := exec.Command("netsh", "advfirewall", "set", "allprofiles", "firewallpolicy", "blockinbound,blockoutbound")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to block outgoing connections: %w", err)
		}
	case "darwin":
		// Block outgoing connections using pfctl on macOS
		cmd := exec.Command("pfctl", "-f", "/etc/pf.conf")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to block outgoing connections: %w", err)
		}
	default:
		// Block outgoing connections using iptables on Linux
		cmd := exec.Command("iptables", "-P", "OUTPUT", "DROP")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to block outgoing connections: %w", err)
		}
	}
	
	return nil
}

// notifyAdministrators notifies administrators about the quarantine
func (rm *ResponseManager) notifyAdministrators(event *Event) error {
	// In a real implementation, this would send notifications to administrators
	// For now, we'll just log the notification
	rm.logger.Info("NOTIFICATION SENT TO ADMINISTRATORS", 
		"event_id", event.ID, 
		"severity", event.Severity,
		"source", event.Source,
		"details", event.Details)
	
	return nil
}

// enterSafeMode enters a safe mode for the agent
func (rm *ResponseManager) enterSafeMode() error {
	// In a real implementation, this would put the agent in a safe mode
	// For now, we'll just log the action
	rm.logger.Info("Entering safe mode")
	
	return nil
}

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

// isolateNetwork isolates the node from the network
func (rm *ResponseManager) isolateNetwork(ctx context.Context, event *Event) error {
	rm.logger.Warn("NETWORK ISOLATION ACTION TRIGGERED", "event_id", event.ID, "source", event.Source)
	
	// In a real implementation, this would:
	// 1. Block all network traffic
	// 2. Close all network connections
	// 3. Configure firewall rules
	// 4. Notify administrators
	
	// Block all network traffic
	if err := rm.blockAllTraffic(); err != nil {
		rm.logger.Error("Failed to block network traffic", "error", err)
		return fmt.Errorf("failed to block network traffic: %w", err)
	}
	
	// Close all network connections
	if err := rm.closeAllConnections(); err != nil {
		rm.logger.Error("Failed to close network connections", "error", err)
		return fmt.Errorf("failed to close network connections: %w", err)
	}
	
	// Configure firewall rules
	if err := rm.configureFirewall(); err != nil {
		rm.logger.Error("Failed to configure firewall", "error", err)
		return fmt.Errorf("failed to configure firewall: %w", err)
	}
	
	// Notify administrators (simulated)
	if err := rm.notifyAdministrators(event); err != nil {
		rm.logger.Error("Failed to notify administrators", "error", err)
		// Don't fail the action if notification fails
	}
	
	rm.logger.Info("Network isolated successfully")
	return nil
}

// blockAllTraffic blocks all network traffic
func (rm *ResponseManager) blockAllTraffic() error {
	// Implementation varies by OS
	switch runtime.GOOS {
	case "windows":
		// Block all traffic using Windows Firewall
		cmd := exec.Command("netsh", "advfirewall", "set", "allprofiles", "firewallpolicy", "blockinbound,blockoutbound")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to block traffic: %w", err)
		}
	case "darwin":
		// Block all traffic using pfctl on macOS
		blockRules := `
block drop all
pass quick on lo0
`
		cmd := exec.Command("echo", blockRules, "|", "pfctl", "-f", "-")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to block traffic: %w", err)
		}
	default:
		// Block all traffic using iptables on Linux
		cmd := exec.Command("iptables", "-P", "INPUT", "DROP")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to block traffic: %w", err)
		}
		
		cmd = exec.Command("iptables", "-P", "OUTPUT", "DROP")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to block traffic: %w", err)
		}
		
		cmd = exec.Command("iptables", "-P", "FORWARD", "DROP")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to block traffic: %w", err)
		}
	}
	
	return nil
}

// closeAllConnections closes all network connections
func (rm *ResponseManager) closeAllConnections() error {
	// Implementation varies by OS
	switch runtime.GOOS {
	case "windows":
		// Close connections using netsh on Windows
		cmd := exec.Command("netsh", "interface", "ipv4", "reset")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to close connections: %w", err)
		}
	default:
		// Close connections using ss on Linux/macOS
		cmd := exec.Command("ss", "-K")
		if err := cmd.Run(); err != nil {
			// Alternative approach
			cmd = exec.Command("pkill", "-f", "sshd")
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to close connections: %w", err)
			}
		}
	}
	
	return nil
}

// configureFirewall configures firewall rules
func (rm *ResponseManager) configureFirewall() error {
	// Implementation varies by OS
	switch runtime.GOOS {
	case "windows":
		// Configure Windows Firewall
		cmd := exec.Command("netsh", "advfirewall", "set", "allprofiles", "state", "on")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to configure firewall: %w", err)
		}
	case "darwin":
		// Configure pf on macOS
		cmd := exec.Command("pfctl", "-e")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to configure firewall: %w", err)
		}
	default:
		// Configure iptables on Linux
		cmd := exec.Command("iptables", "-F")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to configure firewall: %w", err)
		}
	}
	
	return nil
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
	currentPID := os.Getpid()
	
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

// GetAvailableActions returns all available response actions
func (rm *ResponseManager) GetAvailableActions() []*ResponseAction {
	var actions []*ResponseAction
	
	for _, action := range rm.actions {
		actions = append(actions, action)
	}
	
	return actions
}

// GetResponseRules returns all response rules
func (rm *ResponseManager) GetResponseRules() map[string][]string {
	return rm.responseRules
}