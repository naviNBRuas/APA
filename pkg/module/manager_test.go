package module

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestModule(t *testing.T, dir, name string) {
	moduleDir := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(moduleDir, 0755))

	// Create dummy wasm file
	wasmPath := filepath.Join(moduleDir, name+".wasm")
	// A minimal valid WASM module: (module)
	wasmBytes := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
	require.NoError(t, os.WriteFile(wasmPath, wasmBytes, 0644))

	// Create manifest
	manifest := Manifest{
		Name:     name,
		Version:  "v1.0.0",
		WasmFile: name + ".wasm",
		Hash:     "...", // Use placeholder for simplicity
	}
	manifestPath := filepath.Join(moduleDir, "manifest.json")
	manifestBytes, err := json.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(manifestPath, manifestBytes, 0644))
}

func TestManager_LoadModulesFromDir(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	createTestModule(t, tempDir, "test-module-1")
	createTestModule(t, tempDir, "test-module-2")

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	manager, err := NewManager(context.Background(), logger, tempDir)
	require.NoError(t, err)
	defer manager.Shutdown(context.Background())

	// Execute
	err = manager.LoadModulesFromDir()
	require.NoError(t, err)

	// Verify
	loadedModules := manager.ListModules()
	assert.Len(t, loadedModules, 2)

	// Check if modules are loaded correctly
	names := make(map[string]bool)
	for _, m := range loadedModules {
		names[m.Name] = true
	}
	assert.True(t, names["test-module-1"])
	assert.True(t, names["test-module-2"])
}
