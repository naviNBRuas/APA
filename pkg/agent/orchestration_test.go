package agent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNativeOrchestratorDryRunAndAllow(t *testing.T) {
	orch := NewNativeOrchestrator(nil)
	chain := []CommandSpec{{Exec: "sh", Args: []string{"-c", "echo hello"}}}
	res, err := orch.RunChain(context.Background(), chain, true)
	require.NoError(t, err, "dry run should not error")
	require.Len(t, res, 1, "expected one result")
}

func TestNativeOrchestratorExecutesAllowed(t *testing.T) {
	orch := NewNativeOrchestrator([]string{"echo"})
	chain := []CommandSpec{{Exec: "echo", Args: []string{"hi"}}}
	res, err := orch.RunChain(context.Background(), chain, false)
	require.NoError(t, err, "execution failed")
	require.NotEmpty(t, string(res[0].Stdout), "expected stdout content")
}

func TestNativeOrchestratorBlocksDisallowed(t *testing.T) {
	orch := NewNativeOrchestrator([]string{"echo"})
	_, err := orch.RunChain(context.Background(), []CommandSpec{{Exec: "curl"}}, true)
	require.Error(t, err, "expected allowlist error")
}
