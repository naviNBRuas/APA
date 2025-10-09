package update

import (
	"context"
	"crypto"
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
}

// Config holds the configuration for the update manager.
type Config struct {
	ServerURL      string        `yaml:"server_url"`
	CheckInterval  time.Duration `yaml:"check_interval"`
	PublicKey      string        `yaml:"public_key"`
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

// StartPeriodicCheck begins a loop to check for updates.
func (m *Manager) StartPeriodicCheck(ctx context.Context, interval time.Duration) {
	m.logger.Info("Update checker started", "interval", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Check immediately on start, then on each tick.
	m.CheckForUpdate(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.CheckForUpdate(ctx)
		}
	}
}

func (m *Manager) CheckForUpdate(ctx context.Context) {
	m.logger.Info("Checking for agent updates", "url", m.updateURL)

	// 1. Fetch release info
	release, err := m.fetchReleaseInfo(ctx)
	if err != nil {
		m.logger.Error("Failed to fetch release info", "error", err)
		return
	}

	// 2. Compare versions
	if release.Version <= m.currentVersion {
		m.logger.Info("Agent is up to date", "current_version", m.currentVersion)
		return
	}
	m.logger.Info("New agent version available", "new_version", release.Version)

	// 3. Perform the update
	if err := m.performUpdate(ctx, release); err != nil {
		m.logger.Error("Failed to perform update", "error", err)
	} else {
		m.logger.Info("Update downloaded and verified. Triggering shutdown to apply.")
		if m.OnUpdateReady != nil {
			m.OnUpdateReady()
		}
	}
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
	sig, err := hex.DecodeString(artifact.Signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}
	hash := sha256.Sum256(newBinary)
	if !ed25519.Verify(m.publicKey, hash[:], sig) {
		return fmt.Errorf("signature verification failed")
	}
	m.logger.Info("New binary signature verified successfully")

	// 4. Save the new binary to a temporary location
	if err := os.WriteFile("agentd.new", newBinary, 0755); err != nil {
		return fmt.Errorf("failed to write new binary: %w", err)
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
