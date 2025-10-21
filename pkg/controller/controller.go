package controller

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"time"
	"io"

	manifest "github.com/naviNBRuas/APA/pkg/controller/manifest"
)

// Command is an interface for os/exec.Cmd, allowing it to be mocked.
type Command interface {
	Start() error
	Wait() error
	Process() *os.Process
	SetStdout(w io.Writer)
	SetStderr(w io.Writer)
}

// CommandFactory is a function that creates a Command.
type CommandFactory func(ctx context.Context, name string, arg ...string) Command

// DefaultCommandFactory is the default CommandFactory using os/exec.CommandContext.
type osExecCommand struct {
	cmd *exec.Cmd
}

func (o *osExecCommand) Start() error {
	return o.cmd.Start()
}

func (o *osExecCommand) Wait() error {
	return o.cmd.Wait()
}

func (o *osExecCommand) Process() *os.Process {
	return o.cmd.Process
}

func (o *osExecCommand) SetStdout(w io.Writer) {
	o.cmd.Stdout = w
}

func (o *osExecCommand) SetStderr(w io.Writer) {
	o.cmd.Stderr = w
}

func DefaultCommandFactory(ctx context.Context, name string, arg ...string) Command {
	cmd := exec.CommandContext(ctx, name, arg...)
	return &osExecCommand{cmd: cmd}
}

// Controller defines the interface for a decentralized controller module.
type Controller interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Configure(configData []byte) error // New method for configuration
	Status() (map[string]string, error) // New method for status reporting
}

// GoBinaryController implements the Controller interface for an external Go binary.
type GoBinaryController struct {
	name          string
	logger        *slog.Logger
	Manifest      *manifest.Manifest
	cmd           Command // Changed from *exec.Cmd
	cancel        context.CancelFunc
	CommandFactory CommandFactory // New field
}

// NewGoBinaryController creates a new GoBinaryController.
func NewGoBinaryController(logger *slog.Logger, manifest *manifest.Manifest) *GoBinaryController {
	return &GoBinaryController{
		name:          manifest.Name,
		logger:        logger,
		Manifest:      manifest,
		CommandFactory: DefaultCommandFactory, // Use default factory
	}
}

// Name returns the name of the controller.
func (gbc *GoBinaryController) Name() string {
	return gbc.name
}

// Start starts the external Go binary controller.
func (gbc *GoBinaryController) Start(ctx context.Context) error {
	gbc.logger.Info("Starting GoBinaryController", "name", gbc.name, "path", gbc.Manifest.Path)

	ctrlCtx, cancel := context.WithCancel(context.Background())
	gbc.cancel = cancel

	gbc.cmd = gbc.CommandFactory(ctrlCtx, gbc.Manifest.Path) // Use command factory
	gbc.cmd.SetStdout(newLogWriter(gbc.logger, slog.LevelInfo, gbc.name))
	gbc.cmd.SetStderr(newLogWriter(gbc.logger, slog.LevelError, gbc.name))

	if err := gbc.cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("failed to start controller binary '%s': %w", gbc.name, err)
	}

	go func() {
		<-ctrlCtx.Done()
		gbc.logger.Info("GoBinaryController context cancelled, stopping process", "name", gbc.name)
		if gbc.cmd.Process() != nil {
			gbc.cmd.Process().Kill()
		}
	}()

	go func() {
		if err := gbc.cmd.Wait(); err != nil {
			gbc.logger.Error("GoBinaryController process exited with error", "name", gbc.name, "error", err)
		}
		cancel() // Ensure context is cancelled if process exits
		gbc.logger.Info("GoBinaryController process exited", "name", gbc.name)
	}()

	return nil
}

// Stop stops the external Go binary controller.
func (gbc *GoBinaryController) Stop(ctx context.Context) error {
	gbc.logger.Info("Stopping GoBinaryController", "name", gbc.name)
	if gbc.cancel != nil {
		gbc.cancel()
	}

	// Wait for the process to actually stop
	done := make(chan struct{}) 
	go func() {
		if gbc.cmd != nil && gbc.cmd.Process() != nil {
			_ = gbc.cmd.Wait() // Wait for the process to exit after kill
		}
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("timeout stopping controller '%s': %w", gbc.name, ctx.Err())
	}
}

// Configure is not yet implemented for GoBinaryController.
func (gbc *GoBinaryController) Configure(configData []byte) error {
	gbc.logger.Warn("Configure method not implemented for GoBinaryController", "name", gbc.name)
	return fmt.Errorf("configure method not implemented for GoBinaryController")
}

// Status returns a basic status for GoBinaryController.
func (gbc *GoBinaryController) Status() (map[string]string, error) {
	status := make(map[string]string)
	status["status"] = "running" // Placeholder
	status["pid"] = fmt.Sprintf("%d", gbc.cmd.Process().Pid)
	return status, nil
}

// logWriter is an io.Writer that writes to slog.Logger.
type logWriter struct {
	logger *slog.Logger
	level  slog.Level
	prefix string
}

func newLogWriter(logger *slog.Logger, level slog.Level, prefix string) *logWriter {
	return &logWriter{logger: logger, level: level, prefix: prefix}
}

func (lw *logWriter) Write(p []byte) (n int, err error) {
	lw.logger.Log(context.Background(), lw.level, lw.prefix+": "+string(p))
	return len(p), nil
}

// DummyController is a placeholder implementation of the Controller interface.
type DummyController struct {
	name   string
	logger *slog.Logger
	Manifest *manifest.Manifest
}
// NewDummyController creates a new DummyController.
func NewDummyController(name string, logger *slog.Logger, manifest *manifest.Manifest) *DummyController {
	return &DummyController{
		name:   name,
		logger: logger,
		Manifest: manifest,
	}
}

// Name returns the name of the controller.
func (dc *DummyController) Name() string {
	return dc.name
}

// Start simulates starting the controller.
func (dc *DummyController) Start(ctx context.Context) error {
	dc.logger.Info("DummyController started", "name", dc.name)
	// Simulate some work
	go func() {
		select {
		case <-ctx.Done():
			dc.logger.Info("DummyController context cancelled", "name", dc.name)
			return
		case <-time.After(10 * time.Second):
			dc.logger.Info("DummyController still running after 10s", "name", dc.name)
		}
	}()
	return nil
}

// Stop simulates stopping the controller.
func (dc *DummyController) Stop(ctx context.Context) error {
	dc.logger.Info("DummyController stopped", "name", dc.name)
	return nil
}

// Configure is not yet implemented for DummyController.
func (dc *DummyController) Configure(configData []byte) error {
	dc.logger.Info("DummyController Configure method called (no-op)", "name", dc.name)
	return nil
}

// Status returns a basic status for DummyController.
func (dc *DummyController) Status() (map[string]string, error) {
	status := make(map[string]string)
	status["status"] = "dummy_running"
	return status, nil
}
