package agent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMemoryExecutorRunsPayload(t *testing.T) {
	m := MemoryExecutor{}
	out, _, err := m.ExecShellPayload(context.Background(), []byte("echo memtest"))
	require.NoError(t, err, "exec failed")
	require.NotEmpty(t, out, "expected output")
}

func TestMemoryExecutorInjectsCallback(t *testing.T) {
	m := MemoryExecutor{}
	called := false
	err := m.InjectIntoProcess(context.Background(), 123, func(context.Context) error {
		called = true
		return nil
	})
	require.NoError(t, err, "expected callback to run")
	require.True(t, called, "expected callback to run")
}
