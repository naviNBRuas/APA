package task_orchestrator

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Worker executes a task and returns its result bytes.
type Worker interface {
	Execute(ctx context.Context, taskID string, payload []byte) ([]byte, error)
}

// MultiPathExecutor fans tasks out to multiple workers and requires a quorum on the outputs.
// Outlier results are discarded, enabling fault tolerance and integrity checks.
type MultiPathExecutor struct {
	logger  *slog.Logger
	workers []Worker
	quorum  int
}

// NewMultiPathExecutor constructs an executor with the given workers and quorum.
func NewMultiPathExecutor(logger *slog.Logger, workers []Worker, quorum int) *MultiPathExecutor {
	if quorum <= 0 {
		quorum = (len(workers) / 2) + 1
	}
	return &MultiPathExecutor{logger: logger, workers: workers, quorum: quorum}
}

// Execute fan-outs a task to up to replicas workers (or all if replicas<=0) and validates outputs.
func (m *MultiPathExecutor) Execute(ctx context.Context, taskID string, payload []byte, replicas int) ([]byte, error) {
	if len(m.workers) == 0 {
		return nil, errors.New("no workers registered")
	}
	if replicas <= 0 || replicas > len(m.workers) {
		replicas = len(m.workers)
	}

	results := make(chan []byte, replicas)
	errs := make(chan error, replicas)
	var wg sync.WaitGroup
	for i := 0; i < replicas; i++ {
		w := m.workers[i]
		wg.Add(1)
		go func(w Worker) {
			defer wg.Done()
			res, err := w.Execute(ctx, taskID, payload)
			if err != nil {
				errs <- err
				return
			}
			results <- res
		}(w)
	}

	go func() {
		wg.Wait()
		close(results)
		close(errs)
	}()

	votes := make(map[[32]byte][][]byte)
	for res := range results {
		digest := sha256.Sum256(res)
		votes[digest] = append(votes[digest], res)
	}

	// wait for errors channel to drain to avoid goroutine leaks
	for range errs {
	}

	bestDigest := [32]byte{}
	bestCount := 0
	var bestPayload []byte
	for digest, payloads := range votes {
		if len(payloads) > bestCount {
			bestDigest = digest
			bestCount = len(payloads)
			bestPayload = payloads[0]
		}
	}

	if bestCount >= m.quorum {
		m.logger.Info("multi-path quorum reached", "task", taskID, "votes", bestCount, "digest", fmt.Sprintf("%x", bestDigest))
		return bestPayload, nil
	}
	return nil, fmt.Errorf("quorum not reached: %d/%d", bestCount, m.quorum)
}

// WithTimeout executes with a bounded duration, cancelling workers exceeding the limit.
func (m *MultiPathExecutor) WithTimeout(taskID string, payload []byte, replicas int, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return m.Execute(ctx, taskID, payload, replicas)
}

// NewCommandWorker wraps a TaskOrchestrator to act as a Worker for multipath execution.
func NewCommandWorker(to *TaskOrchestrator) Worker {
	return commandWorker{to: to}
}

type commandWorker struct {
	to *TaskOrchestrator
}

func (cw commandWorker) Execute(ctx context.Context, taskID string, payload []byte) ([]byte, error) {
	var cmd TaskCommand
	if err := json.Unmarshal(payload, &cmd); err != nil {
		return nil, fmt.Errorf("decode command: %w", err)
	}
	// Use runCommand to capture output.
	return cw.to.runCommand(ctx, cmd)
}
