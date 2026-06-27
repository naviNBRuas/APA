package selfhealing

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/naviNBRuas/APA/pkg/health"
)

// HealingFramework manages self-healing strategies
type HealingFramework struct {
	logger        *slog.Logger
	strategies    map[string]HealingStrategy
	strategyMutex sync.RWMutex
	healthChecker HealthChecker
	eventHandler  EventHandler
	configuration map[string]interface{}
}

// HealthChecker defines the interface for checking system health
type HealthChecker interface {
	CheckHealth(ctx context.Context) ([]*health.CheckResult, error)
}

// EventHandler defines the interface for handling healing events
type EventHandler interface {
	OnHealingAttempt(issue *HealthIssue, strategy HealingStrategy, result *HealingResult)
	OnHealingFailure(issue *HealthIssue, strategy HealingStrategy, err error)
	OnHealingSuccess(issue *HealthIssue, strategy HealingStrategy, result *HealingResult)
}

// NewHealingFramework creates a new healing framework
func NewHealingFramework(
	logger *slog.Logger,
	healthChecker HealthChecker,
	eventHandler EventHandler,
) *HealingFramework {
	return &HealingFramework{
		logger:        logger,
		strategies:    make(map[string]HealingStrategy),
		healthChecker: healthChecker,
		eventHandler:  eventHandler,
		configuration: make(map[string]interface{}),
	}
}

// RegisterStrategy registers a new healing strategy
func (hf *HealingFramework) RegisterStrategy(strategy HealingStrategy) error {
	hf.strategyMutex.Lock()
	defer hf.strategyMutex.Unlock()

	name := strategy.Name()
	if _, exists := hf.strategies[name]; exists {
		return fmt.Errorf("strategy with name '%s' already registered", name)
	}

	hf.strategies[name] = strategy
	hf.logger.Info("Registered healing strategy", "name", name, "description", strategy.Description())
	return nil
}

// UnregisterStrategy unregisters a healing strategy
func (hf *HealingFramework) UnregisterStrategy(name string) error {
	hf.strategyMutex.Lock()
	defer hf.strategyMutex.Unlock()

	if _, exists := hf.strategies[name]; !exists {
		return fmt.Errorf("strategy with name '%s' not found", name)
	}

	delete(hf.strategies, name)
	hf.logger.Info("Unregistered healing strategy", "name", name)
	return nil
}

// ListStrategies returns a list of all registered strategies
func (hf *HealingFramework) ListStrategies() []string {
	hf.strategyMutex.RLock()
	defer hf.strategyMutex.RUnlock()

	names := make([]string, 0, len(hf.strategies))
	for name := range hf.strategies {
		names = append(names, name)
	}

	return names
}

// ConfigureStrategy configures a specific strategy
func (hf *HealingFramework) ConfigureStrategy(name string, config map[string]interface{}) error {
	hf.strategyMutex.RLock()
	strategy, exists := hf.strategies[name]
	hf.strategyMutex.RUnlock()

	if !exists {
		return fmt.Errorf("strategy with name '%s' not found", name)
	}

	return strategy.Configure(config)
}

// DetectAndHeal detects health issues and applies appropriate healing strategies
func (hf *HealingFramework) DetectAndHeal(ctx context.Context) error {
	hf.logger.Info("Starting health detection and healing cycle")

	results, err := hf.healthChecker.CheckHealth(ctx)
	if err != nil {
		return fmt.Errorf("failed to check health: %w", err)
	}

	hf.logger.Info("Health check completed", "result_count", len(results))

	issues := hf.convertCheckResultsToIssues(results)

	for _, issue := range issues {
		if err := hf.applyHealingStrategies(ctx, issue); err != nil {
			hf.logger.Error("Failed to apply healing strategies for issue",
				"issue_id", issue.ID,
				"error", err)
		}
	}

	hf.logger.Info("Health detection and healing cycle completed")
	return nil
}

// convertCheckResultsToIssues converts health check results to health issues
func (hf *HealingFramework) convertCheckResultsToIssues(results []*health.CheckResult) []*HealthIssue {
	var issues []*HealthIssue

	for _, result := range results {
		if result.Status == health.StatusFailed || result.Status == health.StatusWarning {
			issue := &HealthIssue{
				ID:          fmt.Sprintf("issue-%d", time.Now().UnixNano()),
				Type:        result.Component,
				Severity:    hf.mapHealthStatusToSeverity(result.Status),
				Description: result.Message,
				Component:   result.Component,
				Timestamp:   time.Now(),
				Metrics:     result.Metrics,
				Context:     make(map[string]interface{}),
			}

			issues = append(issues, issue)
		}
	}

	return issues
}

// mapHealthStatusToSeverity maps health status to severity level
func (hf *HealingFramework) mapHealthStatusToSeverity(status health.Status) string {
	switch status {
	case health.StatusFailed:
		return "critical"
	case health.StatusWarning:
		return "high"
	case health.StatusDegraded:
		return "medium"
	default:
		return "low"
	}
}

// applyHealingStrategies applies appropriate healing strategies to resolve an issue
func (hf *HealingFramework) applyHealingStrategies(ctx context.Context, issue *HealthIssue) error {
	hf.logger.Info("Applying healing strategies for issue",
		"issue_id", issue.ID,
		"type", issue.Type,
		"severity", issue.Severity)

	applicableStrategies := hf.getApplicableStrategies(issue)

	if len(applicableStrategies) == 0 {
		hf.logger.Warn("No applicable healing strategies found for issue",
			"issue_id", issue.ID,
			"type", issue.Type)
		return nil
	}

	hf.logger.Info("Found applicable healing strategies",
		"issue_id", issue.ID,
		"strategy_count", len(applicableStrategies))

	for _, strategy := range applicableStrategies {
		hf.logger.Info("Attempting to apply healing strategy",
			"issue_id", issue.ID,
			"strategy", strategy.Name())

		startTime := time.Now()
		result, err := strategy.Apply(ctx, issue)
		duration := time.Since(startTime)

		if result != nil {
			result.Duration = duration
		}

		if err != nil {
			hf.logger.Error("Healing strategy failed",
				"issue_id", issue.ID,
				"strategy", strategy.Name(),
				"error", err,
				"duration", duration)

			if hf.eventHandler != nil {
				hf.eventHandler.OnHealingFailure(issue, strategy, err)
			}

			continue
		}

		if result.Success {
			hf.logger.Info("Healing strategy succeeded",
				"issue_id", issue.ID,
				"strategy", strategy.Name(),
				"action", result.ActionTaken,
				"duration", duration)

			if hf.eventHandler != nil {
				hf.eventHandler.OnHealingSuccess(issue, strategy, result)
			}

			if result.RetryNeeded {
				hf.logger.Info("Healing strategy indicates retry is needed",
					"issue_id", issue.ID,
					"strategy", strategy.Name())
			}

			return nil
		} else {
			hf.logger.Warn("Healing strategy reported failure",
				"issue_id", issue.ID,
				"strategy", strategy.Name(),
				"message", result.Message,
				"duration", duration)

			if hf.eventHandler != nil {
				hf.eventHandler.OnHealingAttempt(issue, strategy, result)
			}
		}
	}

	hf.logger.Warn("All healing strategies failed for issue",
		"issue_id", issue.ID,
		"type", issue.Type)

	return fmt.Errorf("all healing strategies failed for issue %s", issue.ID)
}

// getApplicableStrategies returns strategies that can handle the given issue, sorted by priority
func (hf *HealingFramework) getApplicableStrategies(issue *HealthIssue) []HealingStrategy {
	hf.strategyMutex.RLock()
	defer hf.strategyMutex.RUnlock()

	var applicable []HealingStrategy
	for _, strategy := range hf.strategies {
		if strategy.CanHandle(issue) {
			applicable = append(applicable, strategy)
		}
	}

	for i := 0; i < len(applicable)-1; i++ {
		for j := 0; j < len(applicable)-i-1; j++ {
			if applicable[j].Priority() < applicable[j+1].Priority() {
				applicable[j], applicable[j+1] = applicable[j+1], applicable[j]
			}
		}
	}

	return applicable
}

// SchedulePeriodicHealing schedules automatic healing cycles at regular intervals
func (hf *HealingFramework) SchedulePeriodicHealing(ctx context.Context, interval time.Duration) {
	hf.logger.Info("Scheduling periodic healing cycles", "interval", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			hf.logger.Info("Periodic healing scheduler stopped")
			return
		case <-ticker.C:
			if err := hf.DetectAndHeal(ctx); err != nil {
				hf.logger.Error("Periodic healing cycle failed", "error", err)
			}
		}
	}
}
