package selfhealing

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"
)

// NewRestartProcessStrategy creates a new restart process strategy
func NewRestartProcessStrategy() *RestartProcessStrategy {
	return &RestartProcessStrategy{
		name:        "restart-process",
		description: "Restarts failed processes to restore functionality",
		priority:    80,
		config:      make(map[string]interface{}),
	}
}

// Name returns the name of the strategy
func (r *RestartProcessStrategy) Name() string {
	return r.name
}

// Description returns the description of the strategy
func (r *RestartProcessStrategy) Description() string {
	return r.description
}

// CanHandle determines if this strategy can handle the given health issue
func (r *RestartProcessStrategy) CanHandle(issue *HealthIssue) bool {
	return issue.Type == "process" || issue.Component == "process"
}

// Apply applies the restart process strategy
func (r *RestartProcessStrategy) Apply(ctx context.Context, issue *HealthIssue) (*HealingResult, error) {
	startTime := time.Now()

	processName := issue.Component
	if name, ok := issue.Context["process_name"].(string); ok {
		processName = name
	}

	if err := r.terminateProcess(processName); err != nil {
		return nil, fmt.Errorf("failed to terminate process: %w", err)
	}

	time.Sleep(100 * time.Millisecond)

	if err := r.startProcess(processName); err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	if err := r.verifyProcess(processName); err != nil {
		return &HealingResult{
			Success:     false,
			ActionTaken: fmt.Sprintf("Attempted to restart process '%s'", processName),
			Message:     fmt.Sprintf("Process restart failed: %v", err),
			Metrics: map[string]interface{}{
				"restart_time_ms": time.Since(startTime).Milliseconds(),
			},
			RetryNeeded: true,
		}, nil
	}

	result := &HealingResult{
		Success:     true,
		ActionTaken: fmt.Sprintf("Restarted process '%s'", processName),
		Message:     "Process restarted successfully",
		Metrics: map[string]interface{}{
			"restart_time_ms": time.Since(startTime).Milliseconds(),
		},
		RetryNeeded: false,
	}

	return result, nil
}

// terminateProcess terminates a process by name
func (r *RestartProcessStrategy) terminateProcess(processName string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("taskkill", "/F", "/IM", processName)
	default:
		cmd = exec.Command("pkill", "-f", processName)
	}

	return cmd.Run()
}

// startProcess starts a process by name
func (r *RestartProcessStrategy) startProcess(processName string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", processName)
	default:
		cmd = exec.Command(processName)
	}

	return cmd.Start()
}

// verifyProcess verifies that a process is running
func (r *RestartProcessStrategy) verifyProcess(processName string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s", processName))
	default:
		cmd = exec.Command("pgrep", "-f", processName)
	}

	return cmd.Run()
}

// Priority returns the priority of this strategy
func (r *RestartProcessStrategy) Priority() int {
	return r.priority
}

// Configure configures the strategy
func (r *RestartProcessStrategy) Configure(config map[string]interface{}) error {
	r.config = config
	return nil
}
