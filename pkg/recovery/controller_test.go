package recovery

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	controllerManifest "github.com/naviNBRuas/APA/pkg/controller/manifest"
	"github.com/naviNBRuas/APA/pkg/module"
	"github.com/libp2p/go-libp2p/core/peer"
)

// MockP2P is a mock implementation of the P2PService interface.
type MockP2P struct {
	mock.Mock
}

func (m *MockP2P) FetchModule(ctx context.Context, peerID peer.ID, name, version string) (*module.Manifest, []byte, error) {
	args := m.Called(ctx, peerID, name, version)
	return args.Get(0).(*module.Manifest), args.Get(1).([]byte), args.Error(2)
}

func (m *MockP2P) ClosePeer(peerID peer.ID) error {
	args := m.Called(peerID)
	return args.Error(0)
}

// MockModuleManager is a mock implementation of the ModuleManagerService.
type MockModuleManager struct {
	mock.Mock
}

func (m *MockModuleManager) SaveAndLoadModule(manifest *module.Manifest, wasmBytes []byte) error {
	args := m.Called(manifest, wasmBytes)
	return args.Error(0)
}

func (m *MockModuleManager) ListModules() []*module.Manifest {
	args := m.Called()
	return args.Get(0).([]*module.Manifest)
}

func (m *MockModuleManager) StopModule(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

// MockControllerManager is a mock implementation of the ControllerManagerService.
type MockControllerManager struct {
	mock.Mock
}

func (m *MockControllerManager) ListControllers() []*controllerManifest.Manifest {
	args := m.Called()
	return args.Get(0).([]*controllerManifest.Manifest)
}

func (m *MockControllerManager) StopController(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

// MockPolicyEnforcer is a mock implementation of the PolicyEnforcer.
type MockPolicyEnforcer struct {
	mock.Mock
}

func (m *MockPolicyEnforcer) Authorize(ctx context.Context, subject string, action string, resource string) (bool, string, error) {
	args := m.Called(ctx, subject, action, resource)
	return args.Bool(0), args.String(1), args.Error(2)
}

// TestCreateAndRestoreSnapshot tests the CreateSnapshot and RestoreSnapshot functions.
func TestCreateAndRestoreSnapshot(t *testing.T) {
	assert := assert.New(t)

	// Create a temporary logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Mock dependencies
	mockP2P := new(MockP2P)
	mockModuleManager := new(MockModuleManager)
	mockControllerManager := new(MockControllerManager)

	// Mock config data
	originalConfig := map[string]string{"key": "value", "another": "config"}
	var capturedConfigData []byte
	applyConfigFunc := func(configData []byte) error {
		capturedConfigData = configData
		return nil
	}

	// Create RecoveryController
	rc := NewRecoveryController(logger, originalConfig, applyConfigFunc, mockP2P, mockModuleManager, mockControllerManager)

	ctx := context.Background()

	// Test CreateSnapshot
	err := rc.CreateSnapshot(ctx)
	assert.NoError(err)
	assert.FileExists("agent-snapshot.yaml")

	// Test RestoreSnapshot
	err = rc.RestoreSnapshot(ctx)
	assert.NoError(err)
	assert.NotNil(capturedConfigData)

	// Clean up
	os.Remove("agent-snapshot.yaml")
}
