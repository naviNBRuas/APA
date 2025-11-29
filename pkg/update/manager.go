package update

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// ReleaseInfo describes a new agent release.
type ReleaseInfo struct {
	Version   string `json:"version"`
	Artifacts map[string]ArtifactInfo `json:"artifacts"` // Keyed by "os/arch"
}

// ArtifactInfo contains the URL and signature for a specific binary.
type ArtifactInfo struct {
	URL       string `json:"url"`
	Signature string `json:"signature"`
}

// Manager handles the agent's self-update process.
type Manager struct {
	logger         *slog.Logger
	httpClient     *http.Client
	updateURL      string
	publicKey      ed25519.PublicKey
	currentVersion string
	OnUpdateReady  func() // Callback to trigger graceful shutdown
	p2pNetwork     P2PNetworkInterface // Interface for P2P network operations
}

// P2PNetworkInterface defines the interface for P2P network operations
type P2PNetworkInterface interface {
	FetchUpdateFromPeer(ctx context.Context, peerID peer.ID, version string) (*ReleaseInfo, []byte, error)
	GetConnectedPeers() []peer.ID
}

// Config holds the configuration for the update manager.
type Config struct {
	ServerURL      string        `yaml:"server_url"`
	CheckInterval  time.Duration `yaml:"check_interval"`
	PublicKey      string        `yaml:"public_key"`
	EnableP2P      bool          `yaml:"enable_p2p"` // Enable P2P update functionality
}

// NewManager creates a new update manager.
func NewManager(logger *slog.Logger, cfg Config, currentVersion string) (*Manager, error) {
	pubKeyBytes, err := hex.DecodeString(cfg.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key: %w", err)
	}
	if len(pubKeyBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key size")
	}

	return &Manager{
		logger:         logger,
		httpClient:     &http.Client{Timeout: 1 * time.Minute},
		updateURL:      cfg.ServerURL,
		publicKey:      pubKeyBytes,
		currentVersion: currentVersion,
	}, nil
}

// SetP2PNetwork sets the P2P network interface for the update manager
func (m *Manager) SetP2PNetwork(p2p P2PNetworkInterface) {
	m.p2pNetwork = p2p
}

// CurrentVersion returns the agent's current version string.
func (m *Manager) CurrentVersion() string {
	return m.currentVersion
}

// GetCurrentRelease returns the current release information.
// This is a placeholder implementation that would need to be enhanced
// to actually provide the current release data.
func (m *Manager) GetCurrentRelease() (*ReleaseInfo, []byte, error) {
	// In a real implementation, this would return the actual current release data
	// For now, we'll return a minimal release info
	release := &ReleaseInfo{
		Version: m.currentVersion,
		Artifacts: make(map[string]ArtifactInfo),
	}
	
	// Return empty data for now
	return release, []byte{}, nil
}

// StartPeriodicCheck begins a loop to check for updates.
func (m *Manager) StartPeriodicCheck(ctx context.Context, interval time.Duration) {
	m.logger.Info("Update checker started", "interval", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Wait a bit before the first check to allow the agent to settle.
	time.Sleep(5 * time.Second)
	m.CheckForUpdate()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.CheckForUpdate()
		}
	}
}

// CheckForUpdate fetches the latest release information and, if a newer version
// is available, downloads and verifies the update.
func (m *Manager) CheckForUpdate() {
	m.logger.Info("Checking for agent updates", "url", m.updateURL)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 1. Try to fetch release info from P2P network if enabled
	var release *ReleaseInfo
	var releaseData []byte
	var err error

	if m.p2pNetwork != nil {
		release, releaseData, err = m.checkForP2PUpdate(ctx)
		if err != nil {
			m.logger.Warn("Failed to check for P2P update, falling back to server", "error", err)
		}
	}

	// 2. If P2P check failed or P2P is not enabled, fetch from server
	if release == nil {
		release, err = m.fetchReleaseInfo(ctx)
		if err != nil {
			m.logger.Error("Failed to fetch release info", "error", err)
			return
		}
	}

	// 3. Compare versions
	if release.Version <= m.currentVersion {
		m.logger.Info("Agent is up to date", "current_version", m.currentVersion)
		return
	}
	m.logger.Info("New agent version available", "new_version", release.Version)

	// 4. Perform the update
	if releaseData != nil {
		// P2P update
		if err := m.performP2PUpdate(ctx, release, releaseData); err != nil {
			m.logger.Error("Failed to perform P2P update", "error", err)
		} else {
			m.logger.Info("P2P update downloaded and verified. Triggering shutdown to apply.")
			if m.OnUpdateReady != nil {
				m.OnUpdateReady()
			}
		}
	} else {
		// Server update
		if err := m.performUpdate(ctx, release); err != nil {
			m.logger.Error("Failed to perform update", "error", err)
		} else {
			m.logger.Info("Update downloaded and verified. Triggering shutdown to apply.")
			if m.OnUpdateReady != nil {
				m.OnUpdateReady()
			}
		}
	}
}

// checkForP2PUpdate checks for updates from connected peers
func (m *Manager) checkForP2PUpdate(ctx context.Context) (*ReleaseInfo, []byte, error) {
	if m.p2pNetwork == nil {
		return nil, nil, fmt.Errorf("P2P network not available")
	}

	// Get connected peers
	peers := m.p2pNetwork.GetConnectedPeers()
	if len(peers) == 0 {
		return nil, nil, fmt.Errorf("no connected peers")
	}

	// Try to fetch update from each peer
	for _, peerID := range peers {
		release, data, err := m.p2pNetwork.FetchUpdateFromPeer(ctx, peerID, "latest")
		if err != nil {
			m.logger.Warn("Failed to fetch update from peer", "peer", peerID, "error", err)
			continue
		}

		// Verify the release
		if err := m.verifyRelease(release, data); err != nil {
			m.logger.Warn("Failed to verify release from peer", "peer", peerID, "error", err)
			continue
		}

		// If we found a newer version, return it
		if release.Version > m.currentVersion {
			return release, data, nil
		}
	}

	return nil, nil, fmt.Errorf("no newer version found from peers")
}

func (m *Manager) performUpdate(ctx context.Context, release *ReleaseInfo) error {
	// 1. Find artifact for our platform
	key := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
	artifact, ok := release.Artifacts[key]
	if !ok {
		return fmt.Errorf("no artifact found for current platform: %s", key)
	}

	// 2. Download the new binary
	newBinary, err := m.downloadFile(ctx, artifact.URL)
	if err != nil {
		return fmt.Errorf("failed to download new binary: %w", err)
	}

	// 3. Verify the signature
	if err := m.verifyArtifact(artifact, newBinary); err != nil {
		return fmt.Errorf("artifact verification failed: %w", err)
	}
	m.logger.Info("New binary signature verified successfully")

	// 4. Save the new binary to a temporary location
	if err := os.WriteFile("agentd.new", newBinary, 0755); err != nil {
		return fmt.Errorf("failed to write new binary: %w", err)
	}

	return nil
}

// performP2PUpdate performs an update using data received from a peer
func (m *Manager) performP2PUpdate(ctx context.Context, release *ReleaseInfo, data []byte) error {
	// 1. Find artifact for our platform
	key := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
	artifact, ok := release.Artifacts[key]
	if !ok {
		return fmt.Errorf("no artifact found for current platform: %s", key)
	}

	// 2. Verify the signature
	if err := m.verifyArtifact(artifact, data); err != nil {
		return fmt.Errorf("artifact verification failed: %w", err)
	}
	m.logger.Info("New binary signature verified successfully")

	// 3. Save the new binary to a temporary location
	if err := os.WriteFile("agentd.new", data, 0755); err != nil {
		return fmt.Errorf("failed to write new binary: %w", err)
	}

	return nil
}

// verifyArtifact verifies the signature of an artifact
func (m *Manager) verifyArtifact(artifact ArtifactInfo, data []byte) error {
	sig, err := hex.DecodeString(artifact.Signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}
	hash := sha256.Sum256(data)
	if !ed25519.Verify(m.publicKey, hash[:], sig) {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}

// verifyRelease verifies a release and its artifacts
func (m *Manager) verifyRelease(release *ReleaseInfo, data []byte) error {
	// Find artifact for our platform
	key := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
	artifact, ok := release.Artifacts[key]
	if !ok {
		return fmt.Errorf("no artifact found for current platform: %s", key)
	}

	// Verify the artifact
	if err := m.verifyArtifact(artifact, data); err != nil {
		return fmt.Errorf("artifact verification failed: %w", err)
	}

	return nil
}

func (m *Manager) fetchReleaseInfo(ctx context.Context) (*ReleaseInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.updateURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status from update server: %s", resp.Status)
	}

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

func (m *Manager) downloadFile(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status from artifact server: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}
