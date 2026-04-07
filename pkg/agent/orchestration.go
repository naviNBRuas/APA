package agent

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"path/filepath"
)

// CommandSpec represents a single native utility invocation.
type CommandSpec struct {
	Exec string
	Args []string
}

// CommandResult captures stdout/stderr for a native invocation.
type CommandResult struct {
	Spec   CommandSpec
	Stdout []byte
	Stderr []byte
	Err    error
}

// NativeOrchestrator composes built-in OS utilities into execution chains without introducing extra binaries.
// It enforces an allowlist to keep execution constrained to native interpreters and management interfaces.
type NativeOrchestrator struct {
	allowlist map[string]struct{}
}

// Default allowlist covers common native interpreters and schedulers.
var defaultAllowed = []string{"sh", "bash", "dash", "cmd", "powershell", "pwsh", "systemctl", "crontab", "at", "schtasks", "launchctl", "echo"}

// NewNativeOrchestrator builds an orchestrator with the provided allowlist (falls back to defaults).
func NewNativeOrchestrator(allowed []string) *NativeOrchestrator {
	if len(allowed) == 0 {
		allowed = defaultAllowed
	}
	m := make(map[string]struct{}, len(allowed))
	for _, a := range allowed {
		m[a] = struct{}{}
	}
	return &NativeOrchestrator{allowlist: m}
}

// RunChain executes the chain in order. When dryRun is true, commands are validated but not executed.
func (o *NativeOrchestrator) RunChain(ctx context.Context, chain []CommandSpec, dryRun bool) ([]CommandResult, error) {
	if o == nil {
		return nil, errors.New("orchestrator not initialized")
	}
	results := make([]CommandResult, 0, len(chain))
	for _, spec := range chain {
		if !o.allowed(spec.Exec) {
			return results, errors.New("command not allowed: " + spec.Exec)
		}
		if dryRun {
			results = append(results, CommandResult{Spec: spec})
			continue
		}

		cmd := exec.CommandContext(ctx, spec.Exec, spec.Args...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		results = append(results, CommandResult{Spec: spec, Stdout: stdout.Bytes(), Stderr: stderr.Bytes(), Err: err})
		if err != nil {
			return results, err
		}
	}
	return results, nil
}

func (o *NativeOrchestrator) allowed(execName string) bool {
	if execName == "" {
		return false
	}
	base := filepath.Base(execName)
	_, ok := o.allowlist[base]
	return ok
}
