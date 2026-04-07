package task_orchestrator

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"
)

type stubWorker struct {
	res   []byte
	err   error
	delay time.Duration
}

func (s stubWorker) Execute(ctx context.Context, taskID string, payload []byte) ([]byte, error) {
	if s.delay > 0 {
		select {
		case <-time.After(s.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if s.err != nil {
		return nil, s.err
	}
	return s.res, nil
}

func mpLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

func TestMultiPathExecutorMajorityWins(t *testing.T) {
	workers := []Worker{
		stubWorker{res: []byte("ok")},
		stubWorker{res: []byte("ok")},
		stubWorker{res: []byte("bad")},
	}
	exec := NewMultiPathExecutor(mpLogger(), workers, 2)
	res, err := exec.Execute(context.Background(), "task1", []byte("payload"), 3)
	if err != nil {
		t.Fatalf("expected quorum, got error: %v", err)
	}
	if string(res) != "ok" {
		t.Fatalf("expected majority result 'ok', got %q", string(res))
	}
}

func TestMultiPathExecutorFailsWithoutQuorum(t *testing.T) {
	workers := []Worker{
		stubWorker{res: []byte("a")},
		stubWorker{res: []byte("b")},
		stubWorker{err: errors.New("boom")},
	}
	exec := NewMultiPathExecutor(mpLogger(), workers, 2)
	if _, err := exec.Execute(context.Background(), "task2", []byte("payload"), 3); err == nil {
		t.Fatalf("expected quorum failure")
	}
}

func TestMultiPathExecutorTimeout(t *testing.T) {
	workers := []Worker{
		stubWorker{res: []byte("slow"), delay: 200 * time.Millisecond},
		stubWorker{res: []byte("slow"), delay: 200 * time.Millisecond},
	}
	exec := NewMultiPathExecutor(mpLogger(), workers, 2)
	if _, err := exec.WithTimeout("task3", []byte("payload"), 2, 50*time.Millisecond); err == nil {
		t.Fatalf("expected timeout error")
	}
}
