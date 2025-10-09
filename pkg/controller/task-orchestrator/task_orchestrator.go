package task_orchestrator

import (
	"context"
	"log/slog"
	"time"
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
