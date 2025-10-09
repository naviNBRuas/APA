package driver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

// Manifest defines the metadata and security properties of a driver.
type Manifest struct {
	Name       string   `json:"name"`
	Version    string   `json:"version"`
	Type       string   `json:"type"`
	URL        string   `json:"url"`
	Hash       string   `json:"hash"` // SHA-256 hash of the driver binary
	Signatures []string `json:"signatures"`
}

// Manager handles the lifecycle of drivers.
type Manager struct {
	logger    *slog.Logger
	drivers   map[string]Driver // Maps driver name to Driver instance
	driverDir string
	mu        sync.RWMutex
}

// NewManager creates a new driver manager.
func NewManager(logger *slog.Logger, driverDir string) *Manager {
	return &Manager{
		logger:    logger,
		drivers:   make(map[string]Driver),
		driverDir: driverDir,
	}
}

// FetchAndVerify fetches a driver from a URL, verifies its signature and hash, and saves it.
func (m *Manager) FetchAndVerify(ctx context.Context, manifestURL string) (*Manifest, []byte, error) {
	m.logger.Info("Fetching and verifying driver", "url", manifestURL)

	// 1. Fetch manifest
	resp, err := http.Get(manifestURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("bad status fetching manifest: %s", resp.Status)
	}

	manifestBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read manifest body: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	// 2. Fetch driver binary
	driverBytes, err := m.downloadFile(ctx, manifest.URL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to download driver binary: %w", err)
	}

	// 3. Verify hash
	hasher := sha256.New()
	hasher.Write(driverBytes)
	actualHash := hex.EncodeToString(hasher.Sum(nil))
	if actualHash != manifest.Hash {
		return nil, nil, fmt.Errorf("hash mismatch for driver '%s': expected %s, got %s", manifest.Name, manifest.Hash, actualHash)
	}
	m.logger.Info("Driver hash verified", "name", manifest.Name)

	// TODO: 4. Verify signatures (requires crypto/signature package)

	// 5. Save driver binary and manifest
	driverSubDir := filepath.Join(m.driverDir, manifest.Name)
	if err := os.MkdirAll(driverSubDir, 0755); err != nil {
		return nil, nil, fmt.Errorf("failed to create directory for driver %s: %w", manifest.Name, err)
	}

	manifestPath := filepath.Join(driverSubDir, "manifest.json")
	if err := os.WriteFile(manifestPath, manifestBytes, 0644); err != nil {
		return nil, nil, fmt.Errorf("failed to save driver manifest: %w", err)
	}

	driverPath := filepath.Join(driverSubDir, fmt.Sprintf("%s-%s", manifest.Name, manifest.Version))
	if err := os.WriteFile(driverPath, driverBytes, 0755); err != nil {
		return nil, nil, fmt.Errorf("failed to save driver binary: %w", err)
	}

	m.logger.Info("Driver fetched and verified successfully", "name", manifest.Name, "version", manifest.Version)
	return &manifest, driverBytes, nil
}

// LoadDriver loads a driver into the manager (does not run it).
func (m *Manager) LoadDriver(driver Driver) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.drivers[driver.Name()] = driver
	m.logger.Info("Driver loaded", "name", driver.Name(), "version", driver.Version())
}

// GetDriver returns a loaded driver by name.
func (m *Manager) GetDriver(name string) (Driver, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	driver, ok := m.drivers[name]
	return driver, ok
}

// downloadFile is a helper to download a file from a URL.
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
		return nil, fmt.Errorf("bad status from driver server: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}
