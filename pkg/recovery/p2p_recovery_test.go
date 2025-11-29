package recovery

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockP2PService is a mock implementation of the P2PService interface
type MockP2PService struct {
	mock.Mock
}

func (m *MockP2PService) FetchModule(ctx context.Context, peerID peer.ID, name, version string) (*module.Manifest, []byte, error) {
	args := m.Called(ctx, peerID, name, version)
	return args.Get(0).(*module.Manifest), args.Get(1).([]byte), args.Error(2)
}

func (m *MockP2PService) ClosePeer(peerID peer.ID) error {
	args := m.Called(peerID)
	return args.Error(0)
}

// MockModuleManagerService is a mock implementation of the ModuleManagerService interface
type MockModuleManagerService struct {
	mock.Mock
}

func (m *MockModuleManagerService) SaveAndLoadModule(manifest *module.Manifest, wasmBytes []byte) error {
	args := m.Called(manifest, wasmBytes)
	return args.Error(0)
}

func (m *MockModuleManagerService) ListModules() []*module.Manifest {
	args := m.Called()
	return args.Get(0).([]*module.Manifest)
}

func (m *MockModuleManagerService) StopModule(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

// MockControllerManagerService is a mock implementation of the ControllerManagerService interface
type MockControllerManagerService struct {
	mock.Mock
}

func (m *MockControllerManagerService) ListControllers() []*controllerManifest.Manifest {
	args := m.Called()
	return args.Get(0).([]*controllerManifest.Manifest)
}

func (m *MockControllerManagerService) StopController(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

// MockBackupManagerService is a mock implementation of the BackupManagerService interface
type MockBackupManagerService struct {
	mock.Mock
}

func (m *MockBackupManagerService) CreateBackup(config, state, criticalData map[string]interface{}) (string, error) {
	args := m.Called(config, state, criticalData)
	return args.String(0), args.Error(1)
}

func (m *MockBackupManagerService) RestoreBackup(backupFile string) (interface{}, error) {
	args := m.Called(backupFile)
	return args.Get(0), args.Error(1)
}

func (m *MockBackupManagerService) ListBackups() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}

// MockReputationService is a mock implementation of the ReputationService interface
type MockReputationService struct {
	mock.Mock
}

func (m *MockReputationService) GetPeerScore(peerID string) float64 {
	args := m.Called(peerID)
	return args.Get(0).(float64)
}

func (m *MockReputationService) IsTrustedPeer(peerID string, threshold float64) bool {
	args := m.Called(peerID, threshold)
	return args.Bool(0)
}

func TestNewP2PRecoveryManager(t *testing.T) {
	logger := slog.Default()
	mockP2P := new(MockP2PService)
	mockModuleManager := new(MockModuleManagerService)
	mockControllerManager := new(MockControllerManagerService)
	mockBackupManager := new(MockBackupManagerService)
	mockReputation := new(MockReputationService)

	manager := NewP2PRecoveryManager(
		logger,
		mockP2P,
		mockModuleManager,
		mockControllerManager,
		mockBackupManager,
		mockReputation,
	)

	assert.NotNil(t, manager)
	assert.Equal(t, 3, manager.maxRetries)
	assert.Equal(t, 5*time.Second, manager.retryDelay)
	assert.Equal(t, 80.0, manager.trustThreshold)
}

func TestRequestRecoveryFromPeer_TrustedPeer(t *testing.T) {
	logger := slog.Default()
	mockP2P := new(MockP2PService)
	mockModuleManager := new(MockModuleManagerService)
	mockControllerManager := new(MockControllerManagerService)
	mockBackupManager := new(MockBackupManagerService)
	mockReputation := new(MockReputationService)

	manager := NewP2PRecoveryManager(
		logger,
		mockP2P,
		mockModuleManager,
		mockControllerManager,
		mockBackupManager,
		mockReputation,
	)

	// Mock a trusted peer
	requesterID, _ := peer.Decode("QmPeer1")
	targetID, _ := peer.Decode("QmPeer2")
	
	req := &RecoveryRequest{
		RequesterID:  requesterID,
		TargetID:     targetID,
		ResourceType: "module",
		ResourceName: "test-module",
		Version:      "1.0",
		Timestamp:    time.Now(),
		Priority:     "medium",
	}

	mockReputation.On("IsTrustedPeer", targetID.String(), 80.0).Return(true)
	
	// Test successful recovery
	ctx := context.Background()
	response, err := manager.RequestRecoveryFromPeer(ctx, req)
	
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.True(t, response.Success)
	mockReputation.AssertExpectations(t)
}

func TestRequestRecoveryFromPeer_UntrustedPeer(t *testing.T) {
	logger := slog.Default()
	mockP2P := new(MockP2PService)
	mockModuleManager := new(MockModuleManagerService)
	mockControllerManager := new(MockControllerManagerService)
	mockBackupManager := new(MockBackupManagerService)
	mockReputation := new(MockReputationService)

	manager := NewP2PRecoveryManager(
		logger,
		mockP2P,
		mockModuleManager,
		mockControllerManager,
		mockBackupManager,
		mockReputation,
	)

	// Mock an untrusted peer
	requesterID, _ := peer.Decode("QmPeer1")
	targetID, _ := peer.Decode("QmPeer2")
	
	req := &RecoveryRequest{
		RequesterID:  requesterID,
		TargetID:     targetID,
		ResourceType: "module",
		ResourceName: "test-module",
		Version:      "1.0",
		Timestamp:    time.Now(),
		Priority:     "medium",
	}

	mockReputation.On("IsTrustedPeer", targetID.String(), 80.0).Return(false)
	
	// Test recovery with untrusted peer
	ctx := context.Background()
	response, err := manager.RequestRecoveryFromPeer(ctx, req)
	
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "not trusted")
	mockReputation.AssertExpectations(t)
}

func TestRecoverAgentState(t *testing.T) {
	logger := slog.Default()
	mockP2P := new(MockP2PService)
	mockModuleManager := new(MockModuleManagerService)
	mockControllerManager := new(MockControllerManagerService)
	mockBackupManager := new(MockBackupManagerService)
	mockReputation := new(MockReputationService)

	manager := NewP2PRecoveryManager(
		logger,
		mockP2P,
		mockModuleManager,
		mockControllerManager,
		mockBackupManager,
		mockReputation,
	)

	// Create peer IDs
	peer1ID, _ := peer.Decode("QmPeer1")
	peer2ID, _ := peer.Decode("QmPeer2")
	peerIDs := []peer.ID{peer1ID, peer2ID}
	
	// Mock trusted peers
	mockReputation.On("IsTrustedPeer", peer1ID.String(), 80.0).Return(true)
	mockReputation.On("IsTrustedPeer", peer2ID.String(), 80.0).Return(true)
	
	// Test agent state recovery
	ctx := context.Background()
	err := manager.RecoverAgentState(ctx, peerIDs)
	
	assert.NoError(t, err)
	mockReputation.AssertExpectations(t)
}

func TestRecoverAgentState_NoTrustedPeers(t *testing.T) {
	logger := slog.Default()
	mockP2P := new(MockP2PService)
	mockModuleManager := new(MockModuleManagerService)
	mockControllerManager := new(MockControllerManagerService)
	mockBackupManager := new(MockBackupManagerService)
	mockReputation := new(MockReputationService)

	manager := NewP2PRecoveryManager(
		logger,
		mockP2P,
		mockModuleManager,
		mockControllerManager,
		mockBackupManager,
		mockReputation,
	)

	// Create peer IDs
	peer1ID, _ := peer.Decode("QmPeer1")
	peer2ID, _ := peer.Decode("QmPeer2")
	peerIDs := []peer.ID{peer1ID, peer2ID}
	
	// Mock untrusted peers
	mockReputation.On("IsTrustedPeer", peer1ID.String(), 80.0).Return(false)
	mockReputation.On("IsTrustedPeer", peer2ID.String(), 80.0).Return(false)
	
	// Test agent state recovery with no trusted peers
	ctx := context.Background()
	err := manager.RecoverAgentState(ctx, peerIDs)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no trusted peers")
	mockReputation.AssertExpectations(t)
}