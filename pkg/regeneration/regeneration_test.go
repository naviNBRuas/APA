package regeneration

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func nilRegenerator() *Regenerator {
	return nil
}

func TestStopAgent(t *testing.T) {
	r := &Regenerator{logger: slog.Default(), config: &Config{BinaryPath: "/test/agentd"}}
	err := r.stopAgent()
	assert.NoError(t, err)

	err = nilRegenerator().stopAgent()
	assert.Error(t, err)
}

func TestStartAgent(t *testing.T) {
	r := &Regenerator{logger: slog.Default(), config: &Config{BinaryPath: "/test/agentd"}}
	err := r.startAgent()
	assert.NoError(t, err)

	err = nilRegenerator().startAgent()
	assert.Error(t, err)
}

func TestIsRespondingToHealthChecks(t *testing.T) {
	r := &Regenerator{logger: slog.Default(), config: &Config{}}
	assert.True(t, r.isRespondingToHealthChecks(context.Background()))
}

func TestIsBinaryExecutable(t *testing.T) {
	r := &Regenerator{logger: slog.Default(), config: &Config{}}

	// Nil regenerator
	err := nilRegenerator().isBinaryExecutable()
	assert.Error(t, err)

	// Non-existent path
	r.config.BinaryPath = "/nonexistent/binary"
	err = r.isBinaryExecutable()
	assert.Error(t, err)

	// Create a temp executable file
	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "test_agentd")
	err = os.WriteFile(execPath, []byte("#!/bin/sh\necho test"), 0755)
	require.NoError(t, err)

	r.config.BinaryPath = execPath
	err = r.isBinaryExecutable()
	assert.NoError(t, err)
}

func TestIsBinaryExecutable_NonExecutable(t *testing.T) {
	tmpDir := t.TempDir()
	nonExecPath := filepath.Join(tmpDir, "non_executable")
	err := os.WriteFile(nonExecPath, []byte("data"), 0644)
	require.NoError(t, err)

	r := &Regenerator{logger: slog.Default(), config: &Config{BinaryPath: nonExecPath}}
	err = r.isBinaryExecutable()
	assert.Error(t, err)
}

func TestIsBinaryIntact_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "agentd")
	err := os.WriteFile(binaryPath, []byte("binary content"), 0644)
	require.NoError(t, err)

	r := &Regenerator{logger: slog.Default(), config: &Config{BinaryPath: binaryPath}}
	assert.True(t, r.isBinaryIntact())
}

func TestIsBinaryIntact_Nil(t *testing.T) {
	r := nilRegenerator()
	assert.False(t, r.isBinaryIntact())
}

func TestIsAgentHealthy(t *testing.T) {
	r := &Regenerator{
		logger: slog.Default(),
		config: &Config{BinaryPath: "/nonexistent"},
	}
	assert.False(t, r.isAgentHealthy(context.Background()))
}

func TestIsAgentHealthy_Nil(t *testing.T) {
	assert.False(t, nilRegenerator().isAgentHealthy(context.Background()))
}

func TestIsProcessRunning_Nil(t *testing.T) {
	assert.False(t, nilRegenerator().isProcessRunning())
}

func TestTriggerRegeneration_Nil(t *testing.T) {
	err := nilRegenerator().TriggerRegeneration(context.Background())
	assert.Error(t, err)
}

func TestTriggerRegeneration_Valid(t *testing.T) {
	r := &Regenerator{
		logger: slog.Default(),
		config: &Config{BinaryPath: "/test/agentd"},
	}
	err := r.TriggerRegeneration(context.Background())
	assert.NoError(t, err)
}

func TestStart_NilRegenerator(t *testing.T) {
	nilRegenerator().Start(context.Background())
}

func TestCheckAndRegenerate_Nil(t *testing.T) {
	nilRegenerator().checkAndRegenerate(context.Background())
}

func TestCheckAndRegenerate_PreventsDouble(t *testing.T) {
	r := &Regenerator{
		logger: slog.Default(),
		config: &Config{
			BinaryPath: "/nonexistent",
		},
	}

	ctx := context.Background()
	r.checkAndRegenerate(ctx)
	assert.False(t, r.isRegenerating)
}

func TestHandleRegeneration(t *testing.T) {
	r := &Regenerator{
		logger: slog.Default(),
		config: &Config{BinaryPath: "/test/agentd"},
	}
	r.handleRegeneration(context.Background())
}

func TestPerformProcessInjection_Nil(t *testing.T) {
	nilRegenerator().performProcessInjection()
}

func TestPerformProcessInjection_NoInjectors(t *testing.T) {
	r := &Regenerator{
		logger: slog.Default(),
		config: &Config{BinaryPath: "/test/agentd"},
	}
	r.performProcessInjection()
}

func TestRegenerateAgent_Nil(t *testing.T) {
	err := nilRegenerator().regenerateAgent(context.Background())
	assert.Error(t, err)
}

func TestRegenerateAgent_NoBackup(t *testing.T) {
	r := &Regenerator{
		logger: slog.Default(),
		config: &Config{
			BinaryPath: "/nonexistent/agentd",
			BackupPath: t.TempDir(),
		},
	}
	err := r.regenerateAgent(context.Background())
	assert.Error(t, err)
}

func TestRegenerateAgent_WithBackup(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backup")
	err := os.MkdirAll(backupDir, 0755)
	require.NoError(t, err)

	backupPath := filepath.Join(backupDir, "agent_backup")
	err = os.WriteFile(backupPath, []byte("backup content"), 0644)
	require.NoError(t, err)

	binaryPath := filepath.Join(tmpDir, "agentd")
	r := &Regenerator{
		logger: slog.Default(),
		config: &Config{
			BinaryPath: binaryPath,
			BackupPath: backupDir,
		},
	}
	err = r.regenerateAgent(context.Background())
	assert.NoError(t, err)

	data, err := os.ReadFile(binaryPath)
	require.NoError(t, err)
	assert.Equal(t, "backup content", string(data))
}

func TestRestoreAgentFromBackup_Nil(t *testing.T) {
	err := nilRegenerator().restoreAgentFromBackup(context.Background())
	assert.Error(t, err)
}

func TestRestoreAgentFromBackup_NoBackupDir(t *testing.T) {
	r := &Regenerator{
		logger: slog.Default(),
		config: &Config{
			BinaryPath: "/nonexistent/agentd",
			BackupPath: "/nonexistent/backup",
		},
	}
	err := r.restoreAgentFromBackup(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "backup not found")
}

func TestRestoreAgentFromBackup_Success(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backup")
	err := os.MkdirAll(backupDir, 0755)
	require.NoError(t, err)

	backupPath := filepath.Join(backupDir, "agent_backup")
	err = os.WriteFile(backupPath, []byte("restore test"), 0644)
	require.NoError(t, err)

	binaryPath := filepath.Join(tmpDir, "agentd")
	r := &Regenerator{
		logger: slog.Default(),
		config: &Config{
			BinaryPath: binaryPath,
			BackupPath: backupDir,
		},
	}
	err = r.restoreAgentFromBackup(context.Background())
	assert.NoError(t, err)

	data, err := os.ReadFile(binaryPath)
	require.NoError(t, err)
	assert.Equal(t, "restore test", string(data))
}

func TestVerifyRestoredAgent_Nil(t *testing.T) {
	err := nilRegenerator().verifyRestoredAgent()
	assert.Error(t, err)
}

func TestVerifyRestoredAgent_NonExistent(t *testing.T) {
	r := &Regenerator{
		logger: slog.Default(),
		config: &Config{BinaryPath: "/nonexistent/agentd"},
	}
	err := r.verifyRestoredAgent()
	assert.Error(t, err)
}

func TestVerifyRestoredAgent_Success(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "agentd")
	err := os.WriteFile(binaryPath, []byte("binary content"), 0755)
	require.NoError(t, err)

	r := &Regenerator{
		logger: slog.Default(),
		config: &Config{BinaryPath: binaryPath},
	}
	err = r.verifyRestoredAgent()
	assert.NoError(t, err)
}

func TestCopyFile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src")
	dst := filepath.Join(tmpDir, "dst")
	err := os.WriteFile(src, []byte("copy test"), 0644)
	require.NoError(t, err)

	err = copyFile(src, dst)
	assert.NoError(t, err)

	data, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "copy test", string(data))
}

func TestMonitorLoop_Cancellation(t *testing.T) {
	r := &Regenerator{
		logger: slog.Default(),
		config: &Config{
			BinaryPath:            "/test/agentd",
			RegenerationInterval:  10 * time.Millisecond,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		r.monitorLoop(ctx)
		close(done)
	}()

	time.Sleep(15 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("monitorLoop did not stop after context cancellation")
	}
}

func TestMonitorLoop_Nil(t *testing.T) {
	nilRegenerator().monitorLoop(context.Background())
}

func TestProcessInjectionLoop_Cancellation(t *testing.T) {
	r := &Regenerator{
		logger: slog.Default(),
		config: &Config{
			BinaryPath:           "/test/agentd",
			RegenerationInterval: 10 * time.Millisecond,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		r.processInjectionLoop(ctx)
		close(done)
	}()

	time.Sleep(15 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("processInjectionLoop did not stop after context cancellation")
	}
}

func TestProcessInjectionLoop_Nil(t *testing.T) {
	nilRegenerator().processInjectionLoop(context.Background())
}

func TestPerformPeriodicInjection(t *testing.T) {
	r := &Regenerator{
		logger: slog.Default(),
		config: &Config{BinaryPath: "/test/agentd"},
	}
	r.performPeriodicInjection(context.Background())
}
