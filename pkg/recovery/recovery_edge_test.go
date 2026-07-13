package recovery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	controllerManifest "github.com/naviNBRuas/APA/pkg/controller/manifest"
	"github.com/naviNBRuas/APA/pkg/module"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func validHash(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func TestRecoveryController_New(t *testing.T) {
	rc := NewRecoveryController(slog.Default(), nil, nil, nil, nil, nil)
	require.NotNil(t, rc)
	assert.NotNil(t, rc.quarantineList)
	assert.Equal(t, "agent-snapshot.yaml", rc.snapshotPath)
}

func TestVerifyModule_NilManifest(t *testing.T) {
	rc := NewRecoveryController(slog.Default(), nil, nil, nil, nil, nil)
	err := rc.verifyModule(nil, []byte("data"))
	assert.EqualError(t, err, "module manifest is nil")
}

func TestVerifyModule_EmptyPayload(t *testing.T) {
	rc := NewRecoveryController(slog.Default(), nil, nil, nil, nil, nil)
	err := rc.verifyModule(&module.Manifest{Name: "m", Version: "1", Hash: validHash([]byte("d"))}, nil)
	assert.EqualError(t, err, "module payload is empty")
}

func TestVerifyModule_EmptyName(t *testing.T) {
	rc := NewRecoveryController(slog.Default(), nil, nil, nil, nil, nil)
	err := rc.verifyModule(&module.Manifest{Name: "", Version: "1", Hash: validHash([]byte("d"))}, []byte("data"))
	assert.EqualError(t, err, "module manifest missing name")
}

func TestVerifyModule_EmptyVersion(t *testing.T) {
	rc := NewRecoveryController(slog.Default(), nil, nil, nil, nil, nil)
	err := rc.verifyModule(&module.Manifest{Name: "m", Version: "", Hash: validHash([]byte("d"))}, []byte("data"))
	assert.EqualError(t, err, "module manifest missing version")
}

func TestVerifyModule_EmptyHash(t *testing.T) {
	rc := NewRecoveryController(slog.Default(), nil, nil, nil, nil, nil)
	err := rc.verifyModule(&module.Manifest{Name: "m", Version: "1", Hash: ""}, []byte("data"))
	assert.EqualError(t, err, "module manifest missing hash")
}

func TestVerifyModule_InvalidHashLength(t *testing.T) {
	rc := NewRecoveryController(slog.Default(), nil, nil, nil, nil, nil)
	err := rc.verifyModule(&module.Manifest{Name: "m", Version: "1", Hash: "tooshort"}, []byte("data"))
	assert.EqualError(t, err, "module hash is not a valid SHA-256 hex string")
}

func TestVerifyModule_HashMismatch(t *testing.T) {
	rc := NewRecoveryController(slog.Default(), nil, nil, nil, nil, nil)
	data := []byte("hello")
	wrongHash := strings.Repeat("a", 64)
	err := rc.verifyModule(&module.Manifest{Name: "m", Version: "1", Hash: wrongHash}, data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "module hash mismatch")
}

func TestVerifyModule_MissingSignatures(t *testing.T) {
	rc := NewRecoveryController(slog.Default(), nil, nil, nil, nil, nil)
	data := []byte("hello")
	err := rc.verifyModule(&module.Manifest{Name: "m", Version: "1", Hash: validHash(data)}, data)
	assert.NoError(t, err, "missing signatures should only warn, not fail")
}

func TestVerifyModule_Success(t *testing.T) {
	rc := NewRecoveryController(slog.Default(), nil, nil, nil, nil, nil)
	data := []byte("valid module data")
	err := rc.verifyModule(&module.Manifest{Name: "m", Version: "1", Hash: validHash(data), Signatures: []string{"sig1"}}, data)
	assert.NoError(t, err)
}

func TestRequestPeerCopy_Success(t *testing.T) {
	data := []byte("module wasm bytes")
	m := &module.Manifest{Name: "mod1", Version: "1.0", Hash: validHash(data), Signatures: []string{"sig"}}

	mockP2P := new(MockP2P)
	mockMM := new(MockModuleManager)

	peerIDStr := "QmZoiJNAvCffeEeF7K1PE3sS1TYNzsLpFn1gtJ59eM8Nks"
	decodedID, _ := peer.Decode(peerIDStr)

	mockP2P.On("FetchModule", mock.Anything, decodedID, "mod1", "latest").Return(m, data, nil)
	mockMM.On("SaveAndLoadModule", m, data).Return(nil)

	rc := NewRecoveryController(slog.Default(), nil, nil, mockP2P, mockMM, nil)
	err := rc.RequestPeerCopy(context.Background(), "mod1", peerIDStr)
	assert.NoError(t, err)
	mockP2P.AssertExpectations(t)
	mockMM.AssertExpectations(t)
}

func TestRequestPeerCopy_NilP2P(t *testing.T) {
	rc := NewRecoveryController(slog.Default(), nil, nil, nil, new(MockModuleManager), nil)
	err := rc.RequestPeerCopy(context.Background(), "mod1", "peerID")
	assert.EqualError(t, err, "p2p service not configured")
}

func TestRequestPeerCopy_NilModuleManager(t *testing.T) {
	rc := NewRecoveryController(slog.Default(), nil, nil, new(MockP2P), nil, nil)
	err := rc.RequestPeerCopy(context.Background(), "mod1", "peerID")
	assert.EqualError(t, err, "module manager not configured")
}

func TestRequestPeerCopy_InvalidPeerID(t *testing.T) {
	rc := NewRecoveryController(slog.Default(), nil, nil, new(MockP2P), new(MockModuleManager), nil)
	err := rc.RequestPeerCopy(context.Background(), "mod1", "invalid-peer-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode peer ID")
}

func TestRequestPeerCopy_FetchError(t *testing.T) {
	mockP2P := new(MockP2P)
	mockMM := new(MockModuleManager)
	peerIDStr := "QmZoiJNAvCffeEeF7K1PE3sS1TYNzsLpFn1gtJ59eM8Nks"
	decodedID, _ := peer.Decode(peerIDStr)

	mockP2P.On("FetchModule", mock.Anything, decodedID, "mod1", "latest").Return((*module.Manifest)(nil), []byte{}, fmt.Errorf("network error"))

	rc := NewRecoveryController(slog.Default(), nil, nil, mockP2P, mockMM, nil)
	err := rc.RequestPeerCopy(context.Background(), "mod1", peerIDStr)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch module from peer")
}

func TestRequestPeerCopy_VerifyError(t *testing.T) {
	mockP2P := new(MockP2P)
	mockMM := new(MockModuleManager)
	peerIDStr := "QmZoiJNAvCffeEeF7K1PE3sS1TYNzsLpFn1gtJ59eM8Nks"
	decodedID, _ := peer.Decode(peerIDStr)

	mockP2P.On("FetchModule", mock.Anything, decodedID, "mod1", "latest").Return((*module.Manifest)(nil), []byte{}, nil)

	rc := NewRecoveryController(slog.Default(), nil, nil, mockP2P, mockMM, nil)
	err := rc.RequestPeerCopy(context.Background(), "mod1", peerIDStr)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "module verification failed")
}

func TestRequestPeerCopy_SaveError(t *testing.T) {
	data := []byte("module wasm bytes")
	m := &module.Manifest{Name: "mod1", Version: "1.0", Hash: validHash(data)}

	mockP2P := new(MockP2P)
	mockMM := new(MockModuleManager)
	peerIDStr := "QmZoiJNAvCffeEeF7K1PE3sS1TYNzsLpFn1gtJ59eM8Nks"
	decodedID, _ := peer.Decode(peerIDStr)

	mockP2P.On("FetchModule", mock.Anything, decodedID, "mod1", "latest").Return(m, data, nil)
	mockMM.On("SaveAndLoadModule", m, data).Return(fmt.Errorf("disk full"))

	rc := NewRecoveryController(slog.Default(), nil, nil, mockP2P, mockMM, nil)
	err := rc.RequestPeerCopy(context.Background(), "mod1", peerIDStr)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save and load fetched module")
}

func TestQuarantineNode_Success(t *testing.T) {
	mockP2P := new(MockP2P)
	mockMM := new(MockModuleManager)
	mockCM := new(MockControllerManager)
	peerIDStr := "QmZoiJNAvCffeEeF7K1PE3sS1TYNzsLpFn1gtJ59eM8Nks"
	decodedID, _ := peer.Decode(peerIDStr)

	mockP2P.On("ClosePeer", decodedID).Return(nil)
	mockMM.On("ListModules").Return([]*module.Manifest{})
	mockCM.On("ListControllers").Return([]*controllerManifest.Manifest{})

	rc := NewRecoveryController(slog.Default(), nil, nil, mockP2P, mockMM, mockCM)
	err := rc.QuarantineNode(context.Background(), peerIDStr)
	assert.NoError(t, err)
	assert.True(t, rc.IsQuarantined(peerIDStr))
	mockP2P.AssertExpectations(t)
}

func TestQuarantineNode_AlreadyQuarantined(t *testing.T) {
	mockP2P := new(MockP2P)
	mockMM := new(MockModuleManager)
	mockCM := new(MockControllerManager)
	peerIDStr := "QmZoiJNAvCffeEeF7K1PE3sS1TYNzsLpFn1gtJ59eM8Nks"
	decodedID, _ := peer.Decode(peerIDStr)

	mockP2P.On("ClosePeer", decodedID).Return(nil)
	mockMM.On("ListModules").Return([]*module.Manifest{})
	mockCM.On("ListControllers").Return([]*controllerManifest.Manifest{})

	rc := NewRecoveryController(slog.Default(), nil, nil, mockP2P, mockMM, mockCM)
	err := rc.QuarantineNode(context.Background(), peerIDStr)
	require.NoError(t, err)

	err = rc.QuarantineNode(context.Background(), peerIDStr)
	assert.NoError(t, err)
}

func TestQuarantineNode_InvalidPeerID(t *testing.T) {
	rc := NewRecoveryController(slog.Default(), nil, nil, nil, nil, nil)
	err := rc.QuarantineNode(context.Background(), "not-a-peer-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode peer ID")
}

func TestQuarantineNode_NilServices(t *testing.T) {
	rc := NewRecoveryController(slog.Default(), nil, nil, nil, nil, nil)
	err := rc.QuarantineNode(context.Background(), "QmZoiJNAvCffeEeF7K1PE3sS1TYNzsLpFn1gtJ59eM8Nks")
	assert.NoError(t, err)
}

func TestIsQuarantined(t *testing.T) {
	rc := NewRecoveryController(slog.Default(), nil, nil, nil, nil, nil)
	assert.False(t, rc.IsQuarantined("node1"))
	rc.quarantineList["node1"] = time.Now()
	assert.True(t, rc.IsQuarantined("node1"))
}

func TestListQuarantined_Empty(t *testing.T) {
	rc := NewRecoveryController(slog.Default(), nil, nil, nil, nil, nil)
	assert.Empty(t, rc.ListQuarantined())
}

func TestListQuarantined_WithItems(t *testing.T) {
	rc := NewRecoveryController(slog.Default(), nil, nil, nil, nil, nil)
	rc.quarantineList["node1"] = time.Now()
	rc.quarantineList["node2"] = time.Now()
	list := rc.ListQuarantined()
	assert.ElementsMatch(t, []string{"node1", "node2"}, list)
}

func TestCreateSnapshot_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "snap-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	config := map[string]string{"key": "value"}
	rc := NewRecoveryController(slog.Default(), config, nil, nil, nil, nil)
	err = rc.CreateSnapshot(context.Background())
	assert.NoError(t, err)
	assert.FileExists(t, filepath.Join(tmpDir, "agent-snapshot.yaml"))
}

func TestRestoreSnapshot_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "snap-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	var captured []byte
	config := map[string]string{"key": "value"}
	rc := NewRecoveryController(slog.Default(), config, func(d []byte) error {
		captured = d
		return nil
	}, nil, nil, nil)

	err = rc.CreateSnapshot(context.Background())
	require.NoError(t, err)

	err = rc.RestoreSnapshot(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, captured)
}

func TestRestoreSnapshot_NilApplyConfig(t *testing.T) {
	rc := NewRecoveryController(slog.Default(), nil, nil, nil, nil, nil)
	// Create the snapshot file so RestoreSnapshot reaches the applyConfigFunc check
	err := os.WriteFile("agent-snapshot.yaml", []byte("key: value\n"), 0600)
	require.NoError(t, err)
	defer os.Remove("agent-snapshot.yaml")

	err = rc.RestoreSnapshot(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "applyConfigFunc is not set")
}

func TestRestoreSnapshot_MissingFile(t *testing.T) {
	rc := NewRecoveryController(slog.Default(), nil, nil, nil, nil, nil)
	err := rc.RestoreSnapshot(context.Background())
	assert.Error(t, err)
}

func TestDistributeRecoveryResources(t *testing.T) {
	mockP2PSvc := new(MockP2PService)
	mockMMSvc := new(MockModuleManagerService)
	mockCMSvc := new(MockControllerManagerService)
	mockBackup := new(MockBackupManagerService)
	mockRep := new(MockReputationService)

	prm := NewP2PRecoveryManager(slog.Default(), mockP2PSvc, mockMMSvc, mockCMSvc, mockBackup, mockRep)

	peer1, _ := peer.Decode("QmPeer1")
	peer2, _ := peer.Decode("QmPeer2")

	err := prm.DistributeRecoveryResources(context.Background(), []peer.ID{peer1, peer2})
	assert.NoError(t, err)
}

func TestDistributeRecoveryResources_EmptyPeers(t *testing.T) {
	mockP2PSvc := new(MockP2PService)
	mockMMSvc := new(MockModuleManagerService)
	mockCMSvc := new(MockControllerManagerService)
	mockBackup := new(MockBackupManagerService)
	mockRep := new(MockReputationService)

	prm := NewP2PRecoveryManager(slog.Default(), mockP2PSvc, mockMMSvc, mockCMSvc, mockBackup, mockRep)

	err := prm.DistributeRecoveryResources(context.Background(), []peer.ID{})
	assert.NoError(t, err)
}

func TestDeleteSnapshot_OutsideStoragePath(t *testing.T) {
	logger := slog.Default()
	controller := NewExtendedRecoveryController(logger, nil, nil, nil, nil, nil, "/tmp/snapshots")

	err := controller.DeleteSnapshot("/etc/passwd")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not within the snapshot storage path")
}

func TestRestoreFromComprehensiveSnapshot_ChecksumMismatch(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "snapshots")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	controller := NewExtendedRecoveryController(slog.Default(), nil, func(d []byte) error { return nil }, nil, nil, nil, tmpDir)

	snapshot := SnapshotData{
		Timestamp: time.Now(), Version: "1.0", AgentID: "agent1",
		Configuration: map[string]string{"key": "value"},
		Checksum:      "badchecksum",
	}

	data, _ := json.Marshal(snapshot)
	snapPath := filepath.Join(tmpDir, "snapshot_agent1_20240101_000000.json")
	os.WriteFile(snapPath, data, 0600)

	ctx := context.Background()
	err = controller.RestoreFromComprehensiveSnapshot(ctx, snapPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "checksum verification failed")
}

func TestSchedulePeriodicSnapshots(t *testing.T) {
	controller := NewExtendedRecoveryController(slog.Default(), nil, nil, nil, nil, nil, "/tmp")
	controller.SchedulePeriodicSnapshots(context.Background(), "agent1", time.Minute)
}

func TestCreateComprehensiveSnapshot_WriteError(t *testing.T) {
	controller := NewExtendedRecoveryController(slog.Default(), nil, nil, nil, nil, nil, "/nonexistent/path/that/should/not/exist")
	_, err := controller.CreateComprehensiveSnapshot(context.Background(), "agent1")
	assert.Error(t, err)
}

func TestRecoveryControllerInterface(t *testing.T) {
	var _ P2PService = (*MockP2P)(nil)
	var _ P2PService = (*MockP2PService)(nil)
	var _ ModuleManagerService = (*MockModuleManager)(nil)
	var _ ModuleManagerService = (*MockModuleManagerService)(nil)
	var _ ControllerManagerService = (*MockControllerManager)(nil)
	var _ ControllerManagerService = (*MockControllerManagerService)(nil)
}
