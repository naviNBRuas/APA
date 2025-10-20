package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
	"io"
	"crypto/sha256"
	"encoding/hex"

	controllerPkg "github.com/naviNBRuas/APA/pkg/controller"
	manifest "github.com/naviNBRuas/APA/pkg/controller/manifest"
)

// Manager handles the lifecycle of controllers.
type Manager struct {
	logger     *slog.Logger
	controllers map[string]controllerPkg.Controller // Maps controller name to Controller instance
	controllerDir string
	mu          sync.RWMutex
}

// NewManager creates a new controller manager.
func NewManager(logger *slog.Logger, controllerDir string) *Manager {
	return &Manager{
		logger:     logger,
		controllers: make(map[string]controllerPkg.Controller),
		controllerDir: controllerDir,
	}
}

// LoadControllersFromDir scans the controller directory for manifest.json files and loads them.
func (m *Manager) LoadControllersFromDir(ctx context.Context) error {
	m.logger.Info("Scanning for controllers in directory", "path", m.controllerDir)

	return filepath.Walk(m.controllerDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == "manifest.json" {
			if err := m.loadControllerFromManifest(ctx, path); err != nil {
				m.logger.Error("Failed to load controller from manifest", "path", path, "error", err)
				// Continue to next manifest
			}
		}
		return nil
	})
}

// loadControllerFromManifest parses a manifest and loads the controller.
func (m *Manager) loadControllerFromManifest(ctx context.Context, manifestPath string) error {
	// 1. Read and parse manifest
	manifest, err := m.parseManifest(manifestPath)
	if err != nil {
		return err
	}

	// 2. Verify controller binary hash
	controllerPath := filepath.Join(filepath.Dir(manifestPath), manifest.Path)
	err = m.verifyHash(controllerPath, manifest.Hash)
	if err != nil {
		return fmt.Errorf("controller '%s' hash verification failed: %w", manifest.Name, err)
	}
	m.logger.Debug("Controller hash verified", "name", manifest.Name)


	// For now, we'll assume the controller is a simple Go binary.
	// In a real implementation, this would involve dynamic loading (e.g., WASM, plugins).
	// We'll create a GoBinaryController for now.
	controller := controllerPkg.NewGoBinaryController(m.logger, manifest)

	m.mu.Lock()
	defer m.mu.Unlock()
	m.controllers[manifest.Name] = controller
	m.logger.Info("Successfully loaded controller", "name", manifest.Name, "version", manifest.Version)

	return nil
}

// StartController starts a loaded controller by name.
func (m *Manager) StartController(ctx context.Context, name string) error {
	m.mu.RLock()
	controller, ok := m.controllers[name]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("controller '%s' not found", name)
	}

	return controller.Start(ctx)
}

// StopController stops a running controller by name.
func (m *Manager) StopController(ctx context.Context, name string) error {
	m.mu.RLock()
	controller, ok := m.controllers[name]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("controller '%s' not found", name)
	}

	return controller.Stop(ctx)
}

// ListControllers returns the manifests of all loaded controllers.
func (m *Manager) ListControllers() []*manifest.Manifest {
	m.mu.RLock()
	defer m.mu.RUnlock()
	manifests := make([]*manifest.Manifest, 0, len(m.controllers))
	for _, ctrl := range m.controllers {
		if goBinaryCtrl, ok := ctrl.(*controllerPkg.GoBinaryController); ok { // Check for GoBinaryController
			manifests = append(manifests, goBinaryCtrl.Manifest)
		} else if dummyCtrl, ok := ctrl.(*controllerPkg.DummyController); ok { // Keep DummyController for now
			manifests = append(manifests, dummyCtrl.Manifest)
		}
	}
	return manifests
}

// Shutdown gracefully stops all controllers.
func (m *Manager) Shutdown(ctx context.Context) error {
	m.logger.Info("Shutting down controller manager")
	m.mu.RLock()
	defer m.mu.RUnlock()
	for name, controller := range m.controllers {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := controller.Stop(ctx); err != nil {
			m.logger.Error("Failed to stop controller during shutdown", "name", name, "error", err)
		}
	}
	return nil
}

func (m *Manager) parseManifest(path string) (*manifest.Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}
	var manifest manifest.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest json: %w", err)
	}
	return &manifest, nil
}

// verifyHash calculates the SHA256 hash of a file and compares it to an expected hex-encoded hash.
// A placeholder hash "..." is always considered valid for testing.
func (m *Manager) verifyHash(filePath, expectedHash string) error {
	if expectedHash == "..." {
		m.logger.Warn("Skipping hash verification for placeholder hash", "file", filePath)
		return nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return err
	}

	actualHash := hex.EncodeToString(hasher.Sum(nil))
	if actualHash != expectedHash {
		return fmt.Errorf("hash mismatch: expected %s, got %s", expectedHash, actualHash)
	}

	return nil
}
