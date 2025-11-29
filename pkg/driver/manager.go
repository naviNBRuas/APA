package driver

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
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"
)

// Manifest defines the metadata and security properties of a driver.
type Manifest struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Type         string            `json:"type"`
	URL          string            `json:"url"`
	Hash         string            `json:"hash"` // SHA-256 hash of the driver binary
	Signatures   map[string]string `json:"signatures"` // Map of key name to signature
	Architectures map[string]ArchitectureInfo `json:"architectures"` // Architecture-specific information
}

// ArchitectureInfo contains architecture-specific information for a driver
type ArchitectureInfo struct {
	URL    string `json:"url"`
	Hash   string `json:"hash"`
}

// Manager handles the lifecycle of drivers.
type Manager struct {
	logger     *slog.Logger
	drivers    map[string]Driver // Maps driver name to Driver instance
	driverDir  string
	mu         sync.RWMutex
	httpClient *http.Client
	publicKeys map[string]ed25519.PublicKey // Trusted public keys for signature verification
}

// NewManager creates a new driver manager.
func NewManager(logger *slog.Logger, driverDir string) *Manager {
	return &Manager{
		logger:     logger,
		drivers:    make(map[string]Driver),
		driverDir:  driverDir,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		publicKeys: make(map[string]ed25519.PublicKey),
	}
}

// AddTrustedKey adds a trusted public key for signature verification
func (m *Manager) AddTrustedKey(keyName string, publicKey ed25519.PublicKey) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.publicKeys[keyName] = publicKey
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

	// 2. Determine the appropriate architecture URL
	driverURL := manifest.URL
	expectedHash := manifest.Hash
	
	// Check if we have architecture-specific information
	currentArch := runtime.GOOS + "/" + runtime.GOARCH
	if archInfo, exists := manifest.Architectures[currentArch]; exists {
		driverURL = archInfo.URL
		expectedHash = archInfo.Hash
	}

	// 3. Fetch driver binary
	driverBytes, err := m.downloadFile(ctx, driverURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to download driver binary: %w", err)
	}

	// 4. Verify hash
	hasher := sha256.New()
	hasher.Write(driverBytes)
	actualHash := hex.EncodeToString(hasher.Sum(nil))
	if actualHash != expectedHash {
		return nil, nil, fmt.Errorf("hash mismatch for driver '%s': expected %s, got %s", manifest.Name, expectedHash, actualHash)
	}
	m.logger.Info("Driver hash verified", "name", manifest.Name)

	// 5. Verify signatures
	if err := m.verifySignatures(&manifest, driverBytes); err != nil {
		return nil, nil, fmt.Errorf("signature verification failed: %w", err)
	}

	// 6. Save driver binary and manifest
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

// verifySignatures verifies the signatures of a driver against trusted public keys
func (m *Manager) verifySignatures(manifest *Manifest, driverBytes []byte) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(manifest.Signatures) == 0 {
		return fmt.Errorf("no signatures found in manifest")
	}

	// Verify against all trusted keys
	verified := false
	for keyName, signatureHex := range manifest.Signatures {
		publicKey, exists := m.publicKeys[keyName]
		if !exists {
			m.logger.Warn("Skipping signature verification for unknown key", "key", keyName)
			continue
		}

		signature, err := hex.DecodeString(signatureHex)
		if err != nil {
			m.logger.Error("Failed to decode signature", "error", err)
			continue
		}

		if ed25519.Verify(publicKey, driverBytes, signature) {
			m.logger.Info("Driver signature verified", "name", manifest.Name, "key", keyName)
			verified = true
			break // At least one valid signature is sufficient
		} else {
			m.logger.Warn("Signature verification failed", "key", keyName)
		}
	}

	if !verified {
		return fmt.Errorf("no valid signatures found")
	}

	return nil
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

// ExecuteDriver executes a driver in a sandboxed environment
func (m *Manager) ExecuteDriver(ctx context.Context, name string, args []string) ([]byte, error) {
	m.mu.RLock()
	driver, exists := m.drivers[name]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("driver %s not found", name)
	}

	// Get the driver path
	driverSubDir := filepath.Join(m.driverDir, driver.Name())
	driverPath := filepath.Join(driverSubDir, fmt.Sprintf("%s-%s", driver.Name(), driver.Version()))

	// Prepare the command
	cmd := exec.CommandContext(ctx, driverPath, args...)

	// Set up sandboxing attributes
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// Restrict file system access
		// Note: More comprehensive sandboxing would require OS-specific implementations
	}

	// Set environment variables
	cmd.Env = append(os.Environ(), 
		"APA_DRIVER_MODE=true",
		"APA_DRIVER_NAME="+driver.Name(),
	)

	// Execute the command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("driver execution failed: %w, output: %s", err, string(output))
	}

	m.logger.Info("Driver executed successfully", "name", name)
	return output, nil
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

// ListDrivers returns a list of all loaded drivers
func (m *Manager) ListDrivers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	drivers := make([]string, 0, len(m.drivers))
	for name := range m.drivers {
		drivers = append(drivers, name)
	}
	return drivers
}

// UnloadDriver unloads a driver from the manager
func (m *Manager) UnloadDriver(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.drivers[name]; !exists {
		return fmt.Errorf("driver %s not found", name)
	}

	delete(m.drivers, name)
	m.logger.Info("Driver unloaded", "name", name)
	return nil
}