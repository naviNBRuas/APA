package manager

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/naviNBRuas/APA/pkg/controller"
	"github.com/naviNBRuas/APA/pkg/controller/manifest"
	"github.com/naviNBRuas/APA/pkg/networking"
	"github.com/naviNBRuas/APA/pkg/policy"
	"io"
)

// Ensure that MockController implements controller.Controller
var _ controller.Controller = (*MockController)(nil)
// Ensure that manifest is used
var _ manifest.Manifest
// Ensure that MockPolicyEnforcer implements policy.PolicyEnforcer
var _ policy.PolicyEnforcer = (*MockPolicyEnforcer)(nil)

// MockPolicyEnforcer is a mock implementation of the PolicyEnforcer.
type MockPolicyEnforcer struct {
	mock.Mock
}

func (m *MockPolicyEnforcer) Authorize(ctx context.Context, subject string, action string, resource string) (bool, string, error) {
	args := m.Called(ctx, subject, action, resource)
	return args.Bool(0), args.String(1), args.Error(2)
}

// MockCommand is a mock implementation of the controller.Command interface.
type MockCommand struct {
	mock.Mock
}

func (m *MockCommand) Start() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockCommand) Wait() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockCommand) Process() *os.Process {
	args := m.Called()
	return args.Get(0).(*os.Process)
}

func (m *MockCommand) SetStdout(w io.Writer) {
	m.Called(w)
}

func (m *MockCommand) SetStderr(w io.Writer) {
	m.Called(w)
}

// MockController is a mock implementation of the controller.Controller interface.
type MockController struct {
	mock.Mock
}

func (m *MockController) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockController) Start(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockController) Stop(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockController) Configure(configData []byte) error {
	args := m.Called(configData)
	return args.Error(0)
}

func (m *MockController) Status() (map[string]string, error) {
	args := m.Called()
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockController) HandleMessage(ctx context.Context, message networking.ControllerMessage) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}



func TestManager_LoadControllersFromDir(t *testing.T) {
	assert := assert.New(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create a temporary directory for controllers
	controllerDir, err := os.MkdirTemp("", "controller_test")
	assert.NoError(err)
	defer os.RemoveAll(controllerDir)

	// Create a mock policy enforcer
	mockPolicyEnforcer := new(MockPolicyEnforcer)
	mockPolicyEnforcer.On("Authorize", mock.Anything, "test-controller", "load_controller", "").Return(true, "", nil)

	// Create a manifest file
	manifestContent := `{
		"name": "test-controller",
		"version": "v1.0.0",
		"path": "./test-controller",
		"hash": "...",
		"capabilities": [],
		"policy": ""
	}`
	controllerSubDir := filepath.Join(controllerDir, "test-controller")
	err = os.MkdirAll(controllerSubDir, 0755)
	assert.NoError(err)
	err = os.WriteFile(filepath.Join(controllerSubDir, "manifest.json"), []byte(manifestContent), 0644)
	assert.NoError(err)

	// Create a dummy controller binary (placeholder for hash verification)
	err = os.WriteFile(filepath.Join(controllerSubDir, "test-controller"), []byte("dummy binary"), 0755)
	assert.NoError(err)

	// Create manager
	m := NewManager(logger, controllerDir, mockPolicyEnforcer)

	// Load controllers
	err = m.LoadControllersFromDir(context.Background())
	assert.NoError(err)

	// Verify controller is loaded
	controllers := m.ListControllers()
	assert.Len(controllers, 1)
	assert.Equal("test-controller", controllers[0].Name)

	mockPolicyEnforcer.AssertExpectations(t)
}

func TestManager_StartStopController(t *testing.T) {
	assert := assert.New(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create a temporary directory for controllers
	controllerDir, err := os.MkdirTemp("", "controller_test")
	assert.NoError(err)
	defer os.RemoveAll(controllerDir)

	// Create a mock policy enforcer
	mockPolicyEnforcer := new(MockPolicyEnforcer)
	mockPolicyEnforcer.On("Authorize", mock.Anything, "test-controller", "load_controller", "").Return(true, "", nil)

	// Create a manifest file
	manifestContent := `{
		"name": "test-controller",
		"version": "v1.0.0",
		"path": "./test-controller",
		"hash": "...",
		"capabilities": [],
		"policy": ""
	}`
	controllerSubDir := filepath.Join(controllerDir, "test-controller")
	err = os.MkdirAll(controllerSubDir, 0755)
	assert.NoError(err)
	err = os.WriteFile(filepath.Join(controllerSubDir, "manifest.json"), []byte(manifestContent), 0644)
	assert.NoError(err)

	// Create a dummy controller binary (placeholder for hash verification)
	err = os.WriteFile(filepath.Join(controllerSubDir, "test-controller"), []byte("dummy binary"), 0755)
	assert.NoError(err)

	// Create manager
	m := NewManager(logger, controllerDir, mockPolicyEnforcer)

	// Load controllers
	err = m.LoadControllersFromDir(context.Background())
	assert.NoError(err)

	// Get the loaded controller
	loadedControllers := m.ListControllers()
	assert.Len(loadedControllers, 1)
	gbc, ok := m.controllers[loadedControllers[0].Name].(*controller.GoBinaryController)
	assert.True(ok)

	// Create a mock command
	mockCommand := new(MockCommand)
	mockCommand.On("Start").Return(nil)
	mockCommand.On("Wait").Return(nil)
	mockCommand.On("Process").Return(&os.Process{}) // Return a dummy process
	mockCommand.On("SetStdout", mock.Anything).Return()
	mockCommand.On("SetStderr", mock.Anything).Return()

	// Override the command factory for the test
	gbc.CommandFactory = func(ctx context.Context, name string, arg ...string) controller.Command {
		return mockCommand
	}

	// Start controller
	err = m.StartController(context.Background(), "test-controller")
	assert.NoError(err)

	// Stop controller
	err = m.StopController(context.Background(), "test-controller")
	assert.NoError(err)

	mockPolicyEnforcer.AssertExpectations(t)
	mockCommand.AssertExpectations(t)
}