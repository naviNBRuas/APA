package task_orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/naviNBRuas/APA/pkg/networking"
)

// TaskOrchestrator is an example decentralized controller module.
type TaskOrchestrator struct {
	logger *slog.Logger
	name   string
}

// NewTaskOrchestrator creates a new TaskOrchestrator controller.
func NewTaskOrchestrator(logger *slog.Logger) *TaskOrchestrator {
	return &TaskOrchestrator{
		logger: logger,
		name:   "task-orchestrator",
	}
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
	return nil
}
