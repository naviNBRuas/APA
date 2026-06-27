package health

import (
	"context"
	"errors"
	"testing"
	"time"
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
	hc := NewHealthController(nil)
	if hc == nil {
		t.Fatal("expected non-nil controller")
	}
	if len(hc.checks) != 0 {
		t.Fatalf("expected 0 checks, got %d", len(hc.checks))
	}
}

func TestRegisterCheck(t *testing.T) {
	hc := NewHealthController(nil)
	hc.RegisterCheck(&mockCheck{name: "test-check"})
	if len(hc.checks) != 1 {
		t.Fatalf("expected 1 check, got %d", len(hc.checks))
	}
	if hc.checks[0].Name() != "test-check" {
		t.Fatalf("expected check name 'test-check', got %s", hc.checks[0].Name())
	}
}

func TestMultipleChecks(t *testing.T) {
	hc := NewHealthController(nil)
	for i := 0; i < 5; i++ {
		hc.RegisterCheck(&mockCheck{name: "check"})
	}
	if len(hc.checks) != 5 {
		t.Fatalf("expected 5 checks, got %d", len(hc.checks))
	}
}

func TestStartHealthChecks(t *testing.T) {
	hc := NewHealthController(nil)
	checkCalled := make(chan struct{}, 1)
	hc.RegisterCheck(&mockCheck{
		name: "triggered",
		checkFn: func(ctx context.Context) error {
			select {
			case checkCalled <- struct{}{}:
			default:
			}
			return nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	go hc.StartHealthChecks(ctx, 10*time.Millisecond)

	select {
	case <-checkCalled:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("health check was not called within timeout")
	}
	cancel()
}

func TestStartHealthChecksStopsOnCancel(t *testing.T) {
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
	case <-time.After(500 * time.Millisecond):
		t.Fatal("health checks did not stop after context cancellation")
	}
}

func TestRunChecksHandlesErrors(t *testing.T) {
	hc := NewHealthController(nil)
	hc.RegisterCheck(&mockCheck{
		name: "failing",
		checkFn: func(ctx context.Context) error {
			return errors.New("check failed")
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	hc.StartHealthChecks(ctx, 10*time.Millisecond)
}

func TestProcessLivenessCheck(t *testing.T) {
	plc := NewProcessLivenessCheck()
	if plc.Name() != "process-liveness" {
		t.Fatalf("expected name 'process-liveness', got %s", plc.Name())
	}
	if err := plc.Check(context.Background()); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestHealthStatusConstants(t *testing.T) {
	if StatusOK != "ok" {
		t.Fatalf("expected StatusOK to be 'ok', got %s", StatusOK)
	}
	if StatusFailed != "failed" {
		t.Fatalf("expected StatusFailed to be 'failed', got %s", StatusFailed)
	}
	if StatusDegraded != "degraded" {
		t.Fatalf("expected StatusDegraded to be 'degraded', got %s", StatusDegraded)
	}
	if StatusWarning != "warning" {
		t.Fatalf("expected StatusWarning to be 'warning', got %s", StatusWarning)
	}
}

func TestCheckResult(t *testing.T) {
	r := CheckResult{
		Name:      "test",
		Component: "component-a",
		Status:    StatusOK,
		Message:   "all good",
		Metrics:   map[string]interface{}{"cpu": 0.5},
	}
	if r.Name != "test" || r.Status != StatusOK {
		t.Fatalf("unexpected CheckResult fields: %+v", r)
	}
}
