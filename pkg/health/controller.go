package health

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
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

	if hc.logger != nil {
		hc.logger.Info("Health checks started", "interval", interval)
	}

	for {
		select {
		case <-ctx.Done():
			if hc.logger != nil {
				hc.logger.Info("Health checks stopped.")
			}
			return
		case <-ticker.C:
			hc.runChecks(ctx)
		}
	}
}

func (hc *HealthController) runChecks(ctx context.Context) {
	for _, check := range hc.checks {
		if err := check.Check(ctx); err != nil {
			if hc.logger != nil {
				hc.logger.Error("Health check failed", "check", check.Name(), "error", err)
			}
		} else {
			if hc.logger != nil {
				hc.logger.Debug("Health check passed", "check", check.Name())
			}
		}
	}
}

// ProcessLivenessCheck is a basic health check for process liveness.
type ProcessLivenessCheck struct {
	name string
}

func NewProcessLivenessCheck() *ProcessLivenessCheck {
	return &ProcessLivenessCheck{name: "process-liveness"}
}

func (plc *ProcessLivenessCheck) Name() string {
	return plc.name
}

func (plc *ProcessLivenessCheck) Check(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	procs := runtime.NumGoroutine()
	if procs > 10000 {
		return fmt.Errorf("too many goroutines: %d", procs)
	}

	return nil
}
