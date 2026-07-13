package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	manifest "github.com/naviNBRuas/APA/pkg/controller/manifest"
	"github.com/naviNBRuas/APA/pkg/networking"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestControllerInterface(t *testing.T) {
	var _ Controller = (*GoBinaryController)(nil)
	var _ Controller = (*DummyController)(nil)
}

func TestNewGoBinaryController(t *testing.T) {
	manifest := &manifest.Manifest{Name: "test-ctl", Path: "/bin/echo", Version: "1.0"}
	ctrl := NewGoBinaryController(testLogger(), manifest)
	require.NotNil(t, ctrl)
	assert.Equal(t, "test-ctl", ctrl.Name())
	assert.Equal(t, "/bin/echo", ctrl.Manifest.Path)
	assert.NotEmpty(t, ctrl.configFilePath)
	assert.NotEmpty(t, ctrl.messageFilePath)
	assert.NotNil(t, ctrl.CommandFactory)

	os.Remove(ctrl.configFilePath)
	os.Remove(ctrl.messageFilePath)
}

func TestGoBinaryController_Name(t *testing.T) {
	m := &manifest.Manifest{Name: "my-ctl"}
	gbc := NewGoBinaryController(testLogger(), m)
	assert.Equal(t, "my-ctl", gbc.Name())
	os.Remove(gbc.configFilePath)
	os.Remove(gbc.messageFilePath)
}

func TestGoBinaryController_SetSandboxOptions(t *testing.T) {
	gbc := NewGoBinaryController(testLogger(), &manifest.Manifest{Name: "test"})
	defer os.Remove(gbc.configFilePath)
	defer os.Remove(gbc.messageFilePath)

	opts := SandboxOptions{WorkingDir: "/tmp", Env: []string{"FOO=bar"}, Nice: 5}
	gbc.SetSandboxOptions(opts)
	assert.Equal(t, "/tmp", gbc.sandbox.WorkingDir)
	assert.Equal(t, []string{"FOO=bar"}, gbc.sandbox.Env)
	assert.Equal(t, 5, gbc.sandbox.Nice)
}

func TestGoBinaryController_Status_NotStarted(t *testing.T) {
	gbc := NewGoBinaryController(testLogger(), &manifest.Manifest{Name: "test"})
	defer os.Remove(gbc.configFilePath)
	defer os.Remove(gbc.messageFilePath)

	status, err := gbc.Status()
	require.NoError(t, err)
	assert.Equal(t, "not_started", status["status"])
}

func TestGoBinaryController_Configure_NotRunning(t *testing.T) {
	gbc := NewGoBinaryController(testLogger(), &manifest.Manifest{Name: "test"})
	defer os.Remove(gbc.configFilePath)
	defer os.Remove(gbc.messageFilePath)

	err := gbc.Configure([]byte(`{"key":"value"}`))
	assert.ErrorContains(t, err, "not running")
}

func TestGoBinaryController_HandleMessage_NotRunning(t *testing.T) {
	gbc := NewGoBinaryController(testLogger(), &manifest.Manifest{Name: "test"})
	defer os.Remove(gbc.configFilePath)
	defer os.Remove(gbc.messageFilePath)

	err := gbc.HandleMessage(context.Background(), networking.ControllerMessage{})
	assert.ErrorContains(t, err, "not running")
}

func TestGoBinaryController_Stop_NotStarted(t *testing.T) {
	gbc := NewGoBinaryController(testLogger(), &manifest.Manifest{Name: "test"})
	defer os.Remove(gbc.configFilePath)
	defer os.Remove(gbc.messageFilePath)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := gbc.Stop(ctx)
	require.NoError(t, err)
}

func TestGoBinaryController_Stop_WithCancel(t *testing.T) {
	gbc := NewGoBinaryController(testLogger(), &manifest.Manifest{Name: "test"})
	defer os.Remove(gbc.configFilePath)
	defer os.Remove(gbc.messageFilePath)

	cancelCtx, cancel := context.WithCancel(context.Background())
	gbc.cancel = cancel

	ctx, timeout := context.WithTimeout(context.Background(), time.Second)
	defer timeout()
	err := gbc.Stop(ctx)
	require.NoError(t, err)
	assert.True(t, cancelCtx.Err() != nil)
}

func TestNewDummyController(t *testing.T) {
	dc := NewDummyController("dummy", testLogger(), &manifest.Manifest{Name: "dummy"})
	require.NotNil(t, dc)
	assert.Equal(t, "dummy", dc.Name())
}

func TestDummyController_StartStop(t *testing.T) {
	dc := NewDummyController("dummy", testLogger(), &manifest.Manifest{Name: "dummy"})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := dc.Start(ctx)
	require.NoError(t, err)

	status, err := dc.Status()
	require.NoError(t, err)
	assert.Equal(t, "running", status["status"])

	err = dc.Stop(ctx)
	require.NoError(t, err)

	status, err = dc.Status()
	require.NoError(t, err)
	assert.Equal(t, "stopped", status["status"])
}

func TestDummyController_Configure(t *testing.T) {
	dc := NewDummyController("old-name", testLogger(), &manifest.Manifest{Name: "dummy"})

	err := dc.Configure([]byte(`{"name":"new-name"}`))
	require.NoError(t, err)
	assert.Equal(t, "new-name", dc.Name())
}

func TestDummyController_Configure_Empty(t *testing.T) {
	dc := NewDummyController("name", testLogger(), &manifest.Manifest{Name: "dummy"})

	err := dc.Configure(nil)
	require.NoError(t, err)
	assert.Equal(t, "name", dc.Name())

	err = dc.Configure([]byte{})
	require.NoError(t, err)
	assert.Equal(t, "name", dc.Name())
}

func TestDummyController_Configure_InvalidJSON(t *testing.T) {
	dc := NewDummyController("name", testLogger(), &manifest.Manifest{Name: "dummy"})

	err := dc.Configure([]byte(`invalid json`))
	assert.ErrorContains(t, err, "dummy configure")
}

func TestDummyController_Status(t *testing.T) {
	dc := NewDummyController("dummy", testLogger(), &manifest.Manifest{Name: "dummy"})
	status, err := dc.Status()
	require.NoError(t, err)
	assert.Equal(t, "initialized", status["status"])
	assert.Equal(t, "dummy", status["name"])
}

func TestDummyController_HandleMessage(t *testing.T) {
	dc := NewDummyController("dummy", testLogger(), &manifest.Manifest{Name: "dummy"})
	msg := networking.ControllerMessage{
		Type:         "ping",
		Data:         json.RawMessage(`"hello"`),
		SenderPeerID: "peer1",
	}
	err := dc.HandleMessage(context.Background(), msg)
	require.NoError(t, err)
}

func TestLogWriter(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	lw := newLogWriter(logger, slog.LevelInfo, "test-ctl")

	n, err := lw.Write([]byte("hello world"))
	require.NoError(t, err)
	assert.Equal(t, 11, n)
	assert.Contains(t, buf.String(), "test-ctl: hello world")
}

func TestOsExecCommand_Implementation(t *testing.T) {
	ctx := context.Background()
	cmd := DefaultCommandFactory(ctx, "sh", "-c", "echo hello")
	assert.NotNil(t, cmd)

	var stdout bytes.Buffer
	cmd.SetStdout(&stdout)
	cmd.SetStderr(io.Discard)

	err := cmd.Start()
	require.NoError(t, err)

	err = cmd.Wait()
	require.NoError(t, err)

	assert.Contains(t, stdout.String(), "hello")
	assert.NotNil(t, cmd.Process())
}


