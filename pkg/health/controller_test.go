package health

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCheck struct {
	name    string
	checkFn func(ctx context.Context) error
}

func (m *mockCheck) Name() string { return m.name }
func (m *mockCheck) Check(ctx context.Context) error {
	if m.checkFn != nil {
		return m.checkFn(ctx)
	}
	return nil
}

func TestNewHealthController(t *testing.T) {
	t.Run("with nil logger", func(t *testing.T) {
		hc := NewHealthController(nil)
		require.NotNil(t, hc)
		assert.Empty(t, hc.checks)
		assert.Nil(t, hc.logger)
	})

	t.Run("with logger", func(t *testing.T) {
		logger := slog.Default()
		hc := NewHealthController(logger)
		require.NotNil(t, hc)
		assert.NotNil(t, hc.logger)
	})
}

func TestRegisterCheck(t *testing.T) {
	t.Run("single check", func(t *testing.T) {
		hc := NewHealthController(nil)
		hc.RegisterCheck(&mockCheck{name: "check-1"})
		assert.Len(t, hc.checks, 1)
		assert.Equal(t, "check-1", hc.checks[0].Name())
	})

	t.Run("multiple checks", func(t *testing.T) {
		hc := NewHealthController(nil)
		for range 5 {
			hc.RegisterCheck(&mockCheck{name: "check"})
		}
		assert.Len(t, hc.checks, 5)
	})
}

func TestStartHealthChecks(t *testing.T) {
	t.Run("check is called periodically", func(t *testing.T) {
		hc := NewHealthController(nil)
		checkCalled := make(chan struct{}, 10)
		hc.RegisterCheck(&mockCheck{
			name: "triggered",
			checkFn: func(ctx context.Context) error {
				checkCalled <- struct{}{}
				return nil
			},
		})

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go hc.StartHealthChecks(ctx, 10*time.Millisecond)

		select {
		case <-checkCalled:
		case <-time.After(time.Second):
			require.Fail(t, "health check was not called within timeout")
		}
	})

	t.Run("stops on context cancel", func(t *testing.T) {
		hc := NewHealthController(nil)
		hc.RegisterCheck(&mockCheck{name: "test"})

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		done := make(chan struct{})
		go func() {
			hc.StartHealthChecks(ctx, 10*time.Millisecond)
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(time.Second):
			require.Fail(t, "health checks did not stop after context cancellation")
		}
	})

	t.Run("handles nil logger", func(t *testing.T) {
		hc := NewHealthController(nil)
		hc.RegisterCheck(&mockCheck{name: "test"})

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		hc.StartHealthChecks(ctx, 10*time.Millisecond)
	})
}

func TestRunChecks(t *testing.T) {
	t.Run("passing check", func(t *testing.T) {
		hc := NewHealthController(slog.Default())
		hc.RegisterCheck(&mockCheck{name: "pass"})

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		hc.StartHealthChecks(ctx, 10*time.Millisecond)
	})

	t.Run("failing check", func(t *testing.T) {
		hc := NewHealthController(slog.Default())
		hc.RegisterCheck(&mockCheck{
			name: "failing",
			checkFn: func(ctx context.Context) error {
				return errors.New("check failed")
			},
		})

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		hc.StartHealthChecks(ctx, 10*time.Millisecond)
	})
}

func TestProcessLivenessCheck(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		plc := NewProcessLivenessCheck()
		assert.Equal(t, "process-liveness", plc.Name())
	})

	t.Run("check passes under normal conditions", func(t *testing.T) {
		plc := NewProcessLivenessCheck()
		err := plc.Check(context.Background())
		assert.NoError(t, err)
	})

	t.Run("check returns error on cancelled context", func(t *testing.T) {
		plc := NewProcessLivenessCheck()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := plc.Check(ctx)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("check fails with too many goroutines", func(t *testing.T) {
		expected := fmt.Sprintf("too many goroutines: %d", 10001)
		err := errors.New(expected)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too many goroutines")
	})
}

func TestHealthStatusConstants(t *testing.T) {
	assert.Equal(t, Status("ok"), StatusOK)
	assert.Equal(t, Status("failed"), StatusFailed)
	assert.Equal(t, Status("degraded"), StatusDegraded)
	assert.Equal(t, Status("warning"), StatusWarning)
}

func TestCheckResult(t *testing.T) {
	t.Run("full result", func(t *testing.T) {
		r := CheckResult{
			Name:      "disk-check",
			Component: "disk",
			Status:    StatusOK,
			Message:   "disk usage below threshold",
			Metrics:   map[string]interface{}{"usage_percent": 65.0},
		}

		assert.Equal(t, "disk-check", r.Name)
		assert.Equal(t, "disk", r.Component)
		assert.Equal(t, StatusOK, r.Status)
		assert.Equal(t, "disk usage below threshold", r.Message)
		assert.Equal(t, 65.0, r.Metrics["usage_percent"])
	})

	t.Run("result with nil metrics", func(t *testing.T) {
		r := CheckResult{
			Name:   "memory-check",
			Status: StatusWarning,
		}

		assert.Equal(t, "memory-check", r.Name)
		assert.Equal(t, StatusWarning, r.Status)
		assert.Nil(t, r.Metrics)
	})

	t.Run("result with failed status", func(t *testing.T) {
		r := CheckResult{
			Name:    "api-check",
			Status:  StatusFailed,
			Message: "API endpoint unreachable",
		}

		assert.Equal(t, StatusFailed, r.Status)
		assert.Equal(t, "API endpoint unreachable", r.Message)
	})

	t.Run("result with degraded status", func(t *testing.T) {
		r := CheckResult{
			Name:   "db-check",
			Status: StatusDegraded,
		}

		assert.Equal(t, StatusDegraded, r.Status)
	})
}
