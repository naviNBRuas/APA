package agent

import (
	"context"
	"testing"
)

func TestNativeOrchestratorDryRunAndAllow(t *testing.T) {
	orch := NewNativeOrchestrator(nil)
	chain := []CommandSpec{{Exec: "sh", Args: []string{"-c", "echo hello"}}}
	res, err := orch.RunChain(context.Background(), chain, true)
	if err != nil {
		t.Fatalf("dry run should not error: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected one result")
	}
}

func TestNativeOrchestratorExecutesAllowed(t *testing.T) {
	orch := NewNativeOrchestrator([]string{"echo"})
	chain := []CommandSpec{{Exec: "echo", Args: []string{"hi"}}}
	res, err := orch.RunChain(context.Background(), chain, false)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}
	if string(res[0].Stdout) == "" {
		t.Fatalf("expected stdout content")
	}
}

func TestNativeOrchestratorBlocksDisallowed(t *testing.T) {
	orch := NewNativeOrchestrator([]string{"echo"})
	_, err := orch.RunChain(context.Background(), []CommandSpec{{Exec: "curl"}}, true)
	if err == nil {
		t.Fatalf("expected allowlist error")
	}
}
