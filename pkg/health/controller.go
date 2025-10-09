package health

import (
	"context"
	"log/slog"
	"time"
)

// HealthCheck defines an interface for any health check.
type HealthCheck interface {
	Name() string
	Check(ctx context.Context) error
}

// HealthController manages and orchestrates health checks.
type HealthController struct {
	logger *slog.Logger
	checks []HealthCheck
}

// NewHealthController creates a new HealthController.
func NewHealthController(logger *slog.Logger) *HealthController {
	return &HealthController{
		logger: logger,
		checks: []HealthCheck{},
	}
}

// RegisterCheck adds a new health check to the controller.
func (hc *HealthController) RegisterCheck(check HealthCheck) {
	hc.checks = append(hc.checks, check)
}

// StartHealthChecks begins periodic execution of all registered health checks.
func (hc *HealthController) StartHealthChecks(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	hc.logger.Info("Health checks started", "interval", interval)

	for {
		select {
		case <-ctx.Done():
			hc.logger.Info("Health checks stopped.")
			return
		case <-ticker.C:
			hc.runChecks(ctx)
		}
	}
}

func (hc *HealthController) runChecks(ctx context.Context) {
	for _, check := range hc.checks {
		if err := check.Check(ctx); err != nil {
			hc.logger.Error("Health check failed", "check", check.Name(), "error", err)
		} else {
			hc.logger.Debug("Health check passed", "check", check.Name())
		}
	}
}

// ProcessLivenessCheck is a basic health check for process liveness.
type ProcessLivenessCheck struct{
	name string
}

func NewProcessLivenessCheck() *ProcessLivenessCheck {
	return &ProcessLivenessCheck{name: "process-liveness"}
}

func (plc *ProcessLivenessCheck) Name() string {
	return plc.name
}

func (plc *ProcessLivenessCheck) Check(ctx context.Context) error {
	// In a real scenario, this might check if critical goroutines are running,
	// or if the main event loop is responsive.
	// For now, we just return nil to indicate the process is alive.
	return nil
}
