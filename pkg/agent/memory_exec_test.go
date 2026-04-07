package agent

import (
	"context"
	"testing"
)

func TestMemoryExecutorRunsPayload(t *testing.T) {
	m := MemoryExecutor{}
	out, _, err := m.ExecShellPayload(context.Background(), []byte("echo memtest"))
	if err != nil {
		t.Fatalf("exec failed: %v", err)
	}
	if len(out) == 0 {
		t.Fatalf("expected output")
	}
}

func TestMemoryExecutorInjectsCallback(t *testing.T) {
	m := MemoryExecutor{}
	called := false
	err := m.InjectIntoProcess(context.Background(), 123, func(context.Context) error {
		called = true
		return nil
	})
	if err != nil || !called {
		t.Fatalf("expected callback to run, err=%v", err)
	}
}
