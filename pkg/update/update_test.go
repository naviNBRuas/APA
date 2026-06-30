package update

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/mod/semver"
)

func newTestManager(t *testing.T, version string) *Manager {
	t.Helper()
	logger := slog.Default()
	cfg := Config{
		ServerURL:     "http://example.com/update",
		CheckInterval: 1 * time.Hour,
		PublicKey:     "0000000000000000000000000000000000000000000000000000000000000000",
		EnableP2P:     true,
	}
	m, err := NewManager(logger, cfg, version)
	require.NoError(t, err)
	require.NotNil(t, m)
	return m
}

func TestBackupCurrentBinary(t *testing.T) {
	origDir, err := os.Getwd()
	require.NoError(t, err)
	tmpDir := t.TempDir()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	m := newTestManager(t, "v1.0.0")
	err = m.backupCurrentBinary()
	require.NoError(t, err)

	_, err = os.Stat("agentd.rollback")
	require.NoError(t, err)
}

func TestRollback(t *testing.T) {
	origDir, err := os.Getwd()
	require.NoError(t, err)
	tmpDir := t.TempDir()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	data := []byte("binary to rollback to")
	err = os.WriteFile("agentd.rollback", data, 0755)
	require.NoError(t, err)

	m := newTestManager(t, "v1.0.0")
	err = m.Rollback()
	require.NoError(t, err)

	restored, err := os.ReadFile("agentd.new")
	require.NoError(t, err)
	assert.Equal(t, data, restored)
}

func TestRollback_NoBackup(t *testing.T) {
	origDir, err := os.Getwd()
	require.NoError(t, err)
	tmpDir := t.TempDir()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	m := newTestManager(t, "v1.0.0")
	err = m.Rollback()
	assert.Error(t, err)
}

func TestEnsureSemverPrefix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "v0.0.0"},
		{"v1.0.0", "v1.0.0"},
		{"1.0.0", "v1.0.0"},
		{"v2.3.4-alpha", "v2.3.4-alpha"},
		{"2.3.4-beta", "v2.3.4-beta"},
	}
	for _, tc := range tests {
		got := ensureSemverPrefix(tc.input)
		assert.Equal(t, tc.expected, got, "ensureSemverPrefix(%q)", tc.input)
	}
}

func TestSemverCompare(t *testing.T) {
	tests := []struct {
		current string
		release string
		newer   bool
	}{
		{"v1.0.0", "v1.0.0", false},
		{"v1.0.0", "v1.0.1", true},
		{"v1.0.0", "v0.9.9", false},
		{"v2.0.0", "v1.9.9", false},
		{"v1.0.0", "v1.0.0-alpha", false},
		{"v1.0.0", "v1.0.1-beta", true},
		{"", "v1.0.0", true},
		{"v1.0.0", "", false},
	}
	for _, tc := range tests {
		var gotNewer bool
		cur := ensureSemverPrefix(tc.current)
		rel := ensureSemverPrefix(tc.release)
		if semver.IsValid(rel) {
			if semver.IsValid(cur) {
				gotNewer = semver.Compare(rel, cur) > 0
			} else {
				gotNewer = rel > cur
			}
		} else if !semver.IsValid(cur) {
			gotNewer = tc.release > tc.current
		}
		assert.Equal(t, tc.newer, gotNewer, "current=%q release=%q", tc.current, tc.release)
	}
}

func TestCheckForUpdate_AlreadyUpToDate(t *testing.T) {
	m := newTestManager(t, "v1.0.0")

	updateCalled := false
	m.OnUpdateReady = func() { updateCalled = true }

	mockP2P := new(MockP2PNetwork)
	mockP2P.On("GetConnectedPeers").Return([]peer.ID{})
	m.SetP2PNetwork(mockP2P)

	m.CheckForUpdate()
	assert.False(t, updateCalled, "should not trigger callback when up-to-date")
}

func TestDownloadFile(t *testing.T) {
	expectedData := []byte("mock binary content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(expectedData)
	}))
	defer server.Close()

	m := newTestManager(t, "v1.0.0")
	data, err := m.downloadFile(context.Background(), server.URL)
	require.NoError(t, err)
	assert.Equal(t, expectedData, data)
}

func TestDownloadFile_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	m := newTestManager(t, "v1.0.0")
	_, err := m.downloadFile(context.Background(), server.URL)
	assert.Error(t, err)
}

func TestDownloadFile_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	m := newTestManager(t, "v1.0.0")
	_, err := m.downloadFile(context.Background(), server.URL)
	assert.Error(t, err)
}

func TestPerformUpdate(t *testing.T) {
	origDir, err := os.Getwd()
	require.NoError(t, err)
	tmpDir := t.TempDir()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	err = os.WriteFile("agentd", []byte("current binary"), 0755)
	require.NoError(t, err)

	newBinary := []byte("new binary v1.0.1")
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	pubKeyHex := fmt.Sprintf("%x", pubKey)

	hash := sha256.Sum256(newBinary)
	sig := ed25519.Sign(privKey, hash[:])
	sigHex := fmt.Sprintf("%x", sig)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(newBinary)
	}))
	defer server.Close()

	logger := slog.Default()
	cfg := Config{
		ServerURL:     server.URL,
		CheckInterval: 1 * time.Hour,
		PublicKey:     pubKeyHex,
		EnableP2P:     false,
	}
	m, err := NewManager(logger, cfg, "v1.0.0")
	require.NoError(t, err)

	platform := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
	release := &ReleaseInfo{
		Version: "v1.0.1",
		Artifacts: map[string]ArtifactInfo{
			platform: {URL: server.URL, Signature: sigHex},
		},
	}

	err = m.performUpdate(context.Background(), release)
	require.NoError(t, err)

	_, err = os.Stat("agentd.rollback")
	require.NoError(t, err, "rollback file should exist")

	staged, err := os.ReadFile("agentd.new")
	require.NoError(t, err)
	assert.Equal(t, newBinary, staged)
}

func TestPerformUpdate_MissingArtifact(t *testing.T) {
	m := newTestManager(t, "v1.0.0")
	release := &ReleaseInfo{
		Version:   "v1.0.1",
		Artifacts: map[string]ArtifactInfo{},
	}
	err := m.performUpdate(context.Background(), release)
	assert.Error(t, err)
}

func TestPerformUpdate_InvalidSignature(t *testing.T) {
	origDir, err := os.Getwd()
	require.NoError(t, err)
	tmpDir := t.TempDir()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	err = os.WriteFile("agentd", []byte("current binary"), 0755)
	require.NoError(t, err)

	pubKey, _, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	logger := slog.Default()
	cfg := Config{
		ServerURL:     "http://example.com",
		CheckInterval: 1 * time.Hour,
		PublicKey:     fmt.Sprintf("%x", pubKey),
		EnableP2P:     false,
	}
	m, err := NewManager(logger, cfg, "v1.0.0")
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("tampered binary"))
	}))
	defer server.Close()

	platform := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
	release := &ReleaseInfo{
		Version: "v1.0.1",
		Artifacts: map[string]ArtifactInfo{
			platform: {URL: server.URL, Signature: "badbadbad"},
		},
	}

	err = m.performUpdate(context.Background(), release)
	assert.Error(t, err)
}

func TestStartPeriodicCheck_Cancellation(t *testing.T) {
	m := newTestManager(t, "v1.0.0")

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		m.StartPeriodicCheck(ctx, 100*time.Millisecond)
		close(done)
	}()

	time.Sleep(150 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("StartPeriodicCheck did not return after cancellation")
	}
}

func TestCheckForP2PUpdate_NoPeers(t *testing.T) {
	m := newTestManager(t, "v1.0.0")
	mockP2P := new(MockP2PNetwork)
	m.SetP2PNetwork(mockP2P)

	mockP2P.On("GetConnectedPeers").Return([]peer.ID{})
	ctx := context.Background()
	release, data, err := m.checkForP2PUpdate(ctx)
	assert.Error(t, err)
	assert.Nil(t, release)
	assert.Nil(t, data)
	mockP2P.AssertExpectations(t)
}

func TestReleaseInfoArtifactLookup(t *testing.T) {
	release := &ReleaseInfo{
		Version: "v1.0.1",
		Artifacts: map[string]ArtifactInfo{
			"linux/amd64":   {URL: "https://example.com/linux-amd64", Signature: "sig1"},
			"linux/arm64":   {URL: "https://example.com/linux-arm64", Signature: "sig2"},
			"darwin/amd64":  {URL: "https://example.com/darwin-amd64", Signature: "sig3"},
			"windows/amd64": {URL: "https://example.com/windows-amd64", Signature: "sig4"},
		},
	}
	assert.Len(t, release.Artifacts, 4)
}

func TestCurrentVersion(t *testing.T) {
	m := newTestManager(t, "v2.0.0")
	assert.Equal(t, "v2.0.0", m.CurrentVersion())
}