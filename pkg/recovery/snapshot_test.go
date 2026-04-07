package recovery

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	controllerManifest "github.com/naviNBRuas/APA/pkg/controller/manifest"
	"github.com/naviNBRuas/APA/pkg/module"
	"github.com/stretchr/testify/assert"
)

func TestNewExtendedRecoveryController(t *testing.T) {
	logger := slog.Default()
	mockP2P := new(MockP2PService)
	mockModuleManager := new(MockModuleManagerService)
	mockControllerManager := new(MockControllerManagerService)
	config := map[string]interface{}{"test": "config"}
	applyConfigFunc := func(configData []byte) error { return nil }
	snapshotStoragePath := "/tmp/snapshots"

	controller := NewExtendedRecoveryController(
		logger,
		config,
		applyConfigFunc,
		mockP2P,
		mockModuleManager,
		mockControllerManager,
		snapshotStoragePath,
	)

	assert.NotNil(t, controller)
	assert.Equal(t, snapshotStoragePath, controller.snapshotStoragePath)
}

func TestCreateComprehensiveSnapshot(t *testing.T) {
	// Create temporary directory for snapshots
	tempDir, err := os.MkdirTemp("", "snapshots")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	logger := slog.Default()
	mockP2P := new(MockP2PService)
	mockModuleManager := new(MockModuleManagerService)
	mockControllerManager := new(MockControllerManagerService)
	config := map[string]interface{}{"test": "config"}
	applyConfigFunc := func(configData []byte) error { return nil }

	controller := NewExtendedRecoveryController(
		logger,
		config,
		applyConfigFunc,
		mockP2P,
		mockModuleManager,
		mockControllerManager,
		tempDir,
	)

	// Mock module and controller managers
	modules := []*module.Manifest{
		{Name: "test-module", Version: "1.0", Hash: "abc123"},
	}
	controllers := []*controllerManifest.Manifest{
		{Name: "test-controller", Version: "1.0", Path: "/path/to/controller"},
	}

	mockModuleManager.On("ListModules").Return(modules)
	mockControllerManager.On("ListControllers").Return(controllers)

	// Test creating a snapshot
	ctx := context.Background()
	agentID := "test-agent-123"
	snapshotPath, err := controller.CreateComprehensiveSnapshot(ctx, agentID)

	assert.NoError(t, err)
	assert.NotEmpty(t, snapshotPath)
	assert.FileExists(t, snapshotPath)

	// Verify the snapshot file name format
	filename := filepath.Base(snapshotPath)
	expectedPrefix := "snapshot_" + agentID
	assert.True(t, len(filename) > len(expectedPrefix) && filename[:len(expectedPrefix)] == expectedPrefix)

	mockModuleManager.AssertExpectations(t)
	mockControllerManager.AssertExpectations(t)
}

func TestRestoreFromComprehensiveSnapshot(t *testing.T) {
	// Create temporary directory for snapshots
	tempDir, err := os.MkdirTemp("", "snapshots")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	logger := slog.Default()
	mockP2P := new(MockP2PService)
	mockModuleManager := new(MockModuleManagerService)
	mockControllerManager := new(MockControllerManagerService)
	config := map[string]interface{}{"test": "config"}
	applyConfigFunc := func(configData []byte) error { return nil }

	controller := NewExtendedRecoveryController(
		logger,
		config,
		applyConfigFunc,
		mockP2P,
		mockModuleManager,
		mockControllerManager,
		tempDir,
	)

	// First create a snapshot
	modules := []*module.Manifest{
		{Name: "test-module", Version: "1.0", Hash: "abc123"},
	}
	controllers := []*controllerManifest.Manifest{
		{Name: "test-controller", Version: "1.0", Path: "/path/to/controller"},
	}

	mockModuleManager.On("ListModules").Return(modules)
	mockControllerManager.On("ListControllers").Return(controllers)

	ctx := context.Background()
	agentID := "test-agent-123"
	snapshotPath, err := controller.CreateComprehensiveSnapshot(ctx, agentID)
	assert.NoError(t, err)
	assert.NotEmpty(t, snapshotPath)

	// Reset mocks for restore operation
	mockModuleManager.ExpectedCalls = nil
	mockControllerManager.ExpectedCalls = nil

	// Test restoring from the snapshot
	err = controller.RestoreFromComprehensiveSnapshot(ctx, snapshotPath)
	assert.NoError(t, err)

	mockModuleManager.AssertExpectations(t)
	mockControllerManager.AssertExpectations(t)
}

func TestListSnapshots(t *testing.T) {
	// Create temporary directory for snapshots
	tempDir, err := os.MkdirTemp("", "snapshots")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	logger := slog.Default()
	mockP2P := new(MockP2PService)
	mockModuleManager := new(MockModuleManagerService)
	mockControllerManager := new(MockControllerManagerService)
	config := map[string]interface{}{"test": "config"}
	applyConfigFunc := func(configData []byte) error { return nil }

	controller := NewExtendedRecoveryController(
		logger,
		config,
		applyConfigFunc,
		mockP2P,
		mockModuleManager,
		mockControllerManager,
		tempDir,
	)

	// Test listing snapshots when directory is empty
	snapshots, err := controller.ListSnapshots()
	assert.NoError(t, err)
	assert.Empty(t, snapshots)

	// Create a snapshot
	modules := []*module.Manifest{{Name: "test-module", Version: "1.0", Hash: "abc123"}}
	controllers := []*controllerManifest.Manifest{{Name: "test-controller", Version: "1.0", Path: "/path/to/controller"}}

	mockModuleManager.On("ListModules").Return(modules)
	mockControllerManager.On("ListControllers").Return(controllers)

	ctx := context.Background()
	agentID := "test-agent-123"
	_, err = controller.CreateComprehensiveSnapshot(ctx, agentID)
	assert.NoError(t, err)

	// Test listing snapshots after creating one
	snapshots, err = controller.ListSnapshots()
	assert.NoError(t, err)
	assert.Len(t, snapshots, 1)
	assert.Contains(t, snapshots[0], "snapshot_"+agentID)

	mockModuleManager.AssertExpectations(t)
	mockControllerManager.AssertExpectations(t)
}

func TestDeleteSnapshot(t *testing.T) {
	// Create temporary directory for snapshots
	tempDir, err := os.MkdirTemp("", "snapshots")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	logger := slog.Default()
	mockP2P := new(MockP2PService)
	mockModuleManager := new(MockModuleManagerService)
	mockControllerManager := new(MockControllerManagerService)
	config := map[string]interface{}{"test": "config"}
	applyConfigFunc := func(configData []byte) error { return nil }

	controller := NewExtendedRecoveryController(
		logger,
		config,
		applyConfigFunc,
		mockP2P,
		mockModuleManager,
		mockControllerManager,
		tempDir,
	)

	// Create a snapshot
	modules := []*module.Manifest{{Name: "test-module", Version: "1.0", Hash: "abc123"}}
	controllers := []*controllerManifest.Manifest{{Name: "test-controller", Version: "1.0", Path: "/path/to/controller"}}

	mockModuleManager.On("ListModules").Return(modules)
	mockControllerManager.On("ListControllers").Return(controllers)

	ctx := context.Background()
	agentID := "test-agent-123"
	snapshotPath, err := controller.CreateComprehensiveSnapshot(ctx, agentID)
	assert.NoError(t, err)
	assert.FileExists(t, snapshotPath)

	// Test deleting the snapshot
	err = controller.DeleteSnapshot(snapshotPath)
	assert.NoError(t, err)
	assert.NoFileExists(t, snapshotPath)

	// Test deleting non-existent snapshot
	err = controller.DeleteSnapshot("/non/existent/snapshot.json")
	assert.Error(t, err)

	mockModuleManager.AssertExpectations(t)
	mockControllerManager.AssertExpectations(t)
}

func TestCalculateSnapshotChecksum(t *testing.T) {
	logger := slog.Default()
	mockP2P := new(MockP2PService)
	mockModuleManager := new(MockModuleManagerService)
	mockControllerManager := new(MockControllerManagerService)
	config := map[string]interface{}{"test": "config"}
	applyConfigFunc := func(configData []byte) error { return nil }
	snapshotStoragePath := "/tmp/snapshots"

	controller := NewExtendedRecoveryController(
		logger,
		config,
		applyConfigFunc,
		mockP2P,
		mockModuleManager,
		mockControllerManager,
		snapshotStoragePath,
	)

	snapshot := &SnapshotData{
		Timestamp: time.Now(),
		Version:   "1.0",
		AgentID:   "test-agent",
		Configuration: map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		},
		Modules: []*module.Manifest{
			{Name: "module1", Version: "1.0"},
		},
		Controllers: []*controllerManifest.Manifest{
			{Name: "controller1", Version: "1.0"},
		},
		OperationalState: map[string]interface{}{
			"state_key": "state_value",
		},
	}

	// Calculate checksum twice - should be the same
	checksum1, err := controller.calculateSnapshotChecksum(snapshot)
	assert.NoError(t, err)
	assert.NotEmpty(t, checksum1)

	checksum2, err := controller.calculateSnapshotChecksum(snapshot)
	assert.NoError(t, err)
	assert.Equal(t, checksum1, checksum2)

	// Modify snapshot and verify checksum changes
	snapshot.Version = "2.0"
	checksum3, err := controller.calculateSnapshotChecksum(snapshot)
	assert.NoError(t, err)
	assert.NotEqual(t, checksum1, checksum3)
}
