package agent

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
)

// MemoryExecutor runs payloads directly from memory without persisting to disk.
type MemoryExecutor struct{}

// ExecShellPayload executes a shell script provided as bytes via stdin to the system shell.
func (m MemoryExecutor) ExecShellPayload(ctx context.Context, payload []byte) ([]byte, []byte, error) {
	if len(payload) == 0 {
		return nil, nil, errors.New("empty payload")
	}
	cmd := exec.CommandContext(ctx, "sh")
	cmd.Stdin = bytes.NewReader(payload)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	err := cmd.Run()
	return out.Bytes(), errBuf.Bytes(), err
}

// InjectIntoProcess simulates in-memory execution against a running process by invoking the provided callback.
// It avoids disk writes and lets callers hook into live processes with custom logic.
func (m MemoryExecutor) InjectIntoProcess(ctx context.Context, pid int, fn func(context.Context) error) error {
	if pid <= 0 {
		return errors.New("invalid pid")
	}
	if fn == nil {
		return errors.New("nil callback")
	}
	return fn(ctx)
}
