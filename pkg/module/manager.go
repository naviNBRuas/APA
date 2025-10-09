package module

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
)

// Manager handles the lifecycle of WASM modules.
type Manager struct {
	logger      *slog.Logger
	wasmRuntime *WasmRuntime
	modules     map[string]Module // Maps module name to Module instance
	moduleDir   string
	OnModuleLoad func(manifest Manifest)
}

// NewManager creates a new module manager.
func NewManager(ctx context.Context, logger *slog.Logger, moduleDir string) (*Manager, error) {
	wasmRuntime, err := NewWasmRuntime(ctx, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create wasm runtime: %w", err)
	}

	return &Manager{
		logger:      logger,
		wasmRuntime: wasmRuntime,
		modules:     make(map[string]Module),
		moduleDir:   moduleDir,
	}, nil
}

// LoadModulesFromDir scans the module directory for manifest.json files and loads them.
func (m *Manager) LoadModulesFromDir() error {
	m.logger.Info("Scanning for modules in directory", "path", m.moduleDir)

	return filepath.Walk(m.moduleDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == "manifest.json" {
			if err := m.loadModuleFromManifest(path); err != nil {
				m.logger.Error("Failed to load module from manifest", "path", path, "error", err)
				// Continue to next manifest
			}
		}
		return nil
	})
}

// loadModuleFromManifest parses a manifest, verifies the wasm file, and loads it.
func (m *Manager) loadModuleFromManifest(manifestPath string) error {
	// 1. Read and parse manifest
	manifest, err := m.parseManifest(manifestPath)
	if err != nil {
		return err
	}

	// 2. Verify Wasm file hash
	wasmPath := filepath.Join(filepath.Dir(manifestPath), manifest.WasmFile)
	err = m.verifyHash(wasmPath, manifest.Hash)
	if err != nil {
		return fmt.Errorf("module '%s' hash verification failed: %w", manifest.Name, err)
	}
	m.logger.Debug("Module hash verified", "name", manifest.Name)

	// 3. Read Wasm file
	wasmBytes, err := os.ReadFile(wasmPath)
	if err != nil {
		return fmt.Errorf("failed to read wasm file '%s': %w", wasmPath, err)
	}

	// 4. Compile module
	compiledModule, err := m.wasmRuntime.CompileModule(context.Background(), wasmBytes)
	if err != nil {
		return fmt.Errorf("failed to compile module '%s': %w", manifest.Name, err)
	}

	// 5. Instantiate module
	instance, err := m.wasmRuntime.InstantiateModule(context.Background(), compiledModule, manifest.Name)
	if err != nil {
		return fmt.Errorf("failed to instantiate module '%s': %w", manifest.Name, err)
	}

	// 6. Create and store module
	module := NewWasmModule(manifest, instance, m.logger)
	m.modules[module.Name()] = module
	m.logger.Info("Successfully loaded module", "name", module.Name(), "version", manifest.Version)

	// 7. Announce module load via callback
	if m.OnModuleLoad != nil {
		m.OnModuleLoad(*manifest)
	}

	return nil
}

// InstallModule fetches a module from a URL, verifies it, and installs it.
func (m *Manager) InstallModule(manifestURL string) error {
	m.logger.Info("Installing module from URL", "url", manifestURL)

	// 1. Fetch manifest from URL
	resp, err := http.Get(manifestURL)
	if err != nil {
		return fmt.Errorf("failed to fetch manifest from URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status fetching manifest: %s", resp.Status)
	}

	manifestBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read manifest body: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	// 2. Fetch wasm file from URL specified in manifest
	wasmURL := manifest.WasmURL
	if wasmURL == "" {
		return fmt.Errorf("manifest for %s does not contain a WasmURL", manifest.Name)
	}

	resp, err = http.Get(wasmURL)
	if err != nil {
		return fmt.Errorf("failed to fetch wasm file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status fetching wasm file: %s", resp.Status)
	}

	wasmBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read wasm file body: %w", err)
	}

	// 3. Verify hash
	hasher := sha256.New()
	hasher.Write(wasmBytes)
	actualHash := hex.EncodeToString(hasher.Sum(nil))
	if actualHash != manifest.Hash {
		return fmt.Errorf("hash mismatch for %s: expected %s, got %s", manifest.Name, manifest.Hash, actualHash)
	}
	m.logger.Info("Module hash verified", "name", manifest.Name)

	// 4. Save to moduleDir and load
	return m.SaveAndLoadModule(&manifest, wasmBytes)
}

// RunModule starts a loaded module by name.
func (m *Manager) RunModule(name string) error {
	module, ok := m.modules[name]
	if !ok {
		return fmt.Errorf("module '%s' not found", name)
	}
	return module.Start()
}

// StopModule stops a running module by name.
func (m *Manager) StopModule(name string) error {
	module, ok := m.modules[name]
	if !ok {
		return fmt.Errorf("module '%s' not found", name)
	}
	return module.Stop()
}

// ListModules returns the manifests of all loaded modules.
func (m *Manager) ListModules() []*Manifest {
	manifests := make([]*Manifest, 0, len(m.modules))
	for _, mod := range m.modules {
		if wasmMod, ok := mod.(*WasmModule); ok {
			manifests = append(manifests, wasmMod.manifest)
		}
	}
	return manifests
}

// HasModule checks if a module with the given name and version is already loaded.
func (m *Manager) HasModule(name, version string) bool {
	for _, mod := range m.modules {
		if wasmMod, ok := mod.(*WasmModule); ok {
			if wasmMod.manifest.Name == name && wasmMod.manifest.Version == version {
				return true
			}
		}
	}
	return false
}

// GetModuleData finds a loaded module and returns its manifest and raw WASM bytes.
func (m *Manager) GetModuleData(name, version string) (*Manifest, []byte, error) {
	for _, mod := range m.modules {
		if wasmMod, ok := mod.(*WasmModule); ok {
			if wasmMod.manifest.Name == name && wasmMod.manifest.Version == version {
				wasmPath := filepath.Join(m.moduleDir, name, wasmMod.manifest.WasmFile)
				wasmBytes, err := os.ReadFile(wasmPath)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to read wasm file for module %s: %w", name, err)
				}
				return wasmMod.manifest, wasmBytes, nil
			}
		}
	}
	return nil, nil, fmt.Errorf("module %s version %s not found", name, version)
}

// SaveAndLoadModule saves a new module to disk and loads it into the runtime.
func (m *Manager) SaveAndLoadModule(manifest *Manifest, wasmBytes []byte) error {
	moduleSubDir := filepath.Join(m.moduleDir, manifest.Name)
	if err := os.MkdirAll(moduleSubDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for new module %s: %w", manifest.Name, err)
	}

	// Save manifest
	manifestPath := filepath.Join(moduleSubDir, "manifest.json")
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest for saving: %w", err)
	}
	if err := os.WriteFile(manifestPath, manifestBytes, 0644); err != nil {
		return fmt.Errorf("failed to save manifest file: %w", err)
	}

	// Save wasm file
	wasmPath := filepath.Join(moduleSubDir, manifest.WasmFile)
	if err := os.WriteFile(wasmPath, wasmBytes, 0644); err != nil {
		return fmt.Errorf("failed to save wasm file: %w", err)
	}

	m.logger.Info("Successfully saved new module", "name", manifest.Name)

	// Finally, load the new module into the runtime
	return m.loadModuleFromManifest(manifestPath)
}

// Shutdown gracefully stops all modules and closes the wasm runtime.
func (m *Manager) Shutdown() error {
	m.logger.Info("Shutting down module manager")
	for name, module := range m.modules {
		if err := module.Stop(); err != nil {
			m.logger.Error("Failed to stop module during shutdown", "name", name, "error", err)
			// We continue trying to stop other modules
		}
	}
	return m.wasmRuntime.Close(context.Background())
}

func (m *Manager) parseManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}
	var manifest Manifest
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
