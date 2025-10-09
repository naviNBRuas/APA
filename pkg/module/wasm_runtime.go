package module

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// WasmRuntime encapsulates the wazero runtime and provides a safe execution environment.
type WasmRuntime struct {
	logger  *slog.Logger
	runtime wazero.Runtime
}

// NewWasmRuntime creates a new WASM runtime environment.
func NewWasmRuntime(ctx context.Context, logger *slog.Logger) (*WasmRuntime, error) {
	runtime := wazero.NewRuntime(ctx)

	// Instantiate WASI, which is required for many modules.
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, runtime); err != nil {
		runtime.Close(ctx)
		return nil, fmt.Errorf("failed to instantiate WASI: %w", err)
	}

	// TODO: Instantiate host APIs (e.g., metrics, KV store) and add them to the runtime.

	return &WasmRuntime{
		logger:  logger,
		runtime: runtime,
	}, nil
}

// CompileModule compiles a WASM module but does not instantiate it.
func (r *WasmRuntime) CompileModule(ctx context.Context, wasmBytes []byte) (wazero.CompiledModule, error) {
	compiledModule, err := r.runtime.CompileModule(ctx, wasmBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to compile wasm module: %w", err)
	}
	return compiledModule, nil
}

// InstantiateModule creates a new instance of a compiled module.
func (r *WasmRuntime) InstantiateModule(ctx context.Context, compiledModule wazero.CompiledModule, moduleName string) (api.Module, error) {
	moduleInstance, err := r.runtime.InstantiateModule(ctx, compiledModule, wazero.NewModuleConfig().WithName(moduleName))
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate module '%s': %w", moduleName, err)
	}
	return moduleInstance, nil
}

// Close shuts down the wazero runtime.
func (r *WasmRuntime) Close(ctx context.Context) error {
	return r.runtime.Close(ctx)
}

// WasmModule is a concrete implementation of the Module interface for a WASM module.
type WasmModule struct {
	manifest *Manifest
	instance api.Module
	logger   *slog.Logger
}

// NewWasmModule creates a new WasmModule instance.
func NewWasmModule(manifest *Manifest, instance api.Module, logger *slog.Logger) *WasmModule {
	return &WasmModule{
		manifest: manifest,
		instance: instance,
		logger:   logger,
	}
}

func (m *WasmModule) Name() string {
	return m.manifest.Name
}

// Start runs the module's entrypoint function.
func (m *WasmModule) Start() error {
	m.logger.Info("Starting module", "name", m.Name(), "entry", m.manifest.Entry)
	entryFunc := m.instance.ExportedFunction(m.manifest.Entry)
	if entryFunc == nil {
		return fmt.Errorf("entry function '%s' not found in module '%s'", m.manifest.Entry, m.Name())
	}

	// The _start function in WASI has a specific signature (no params, no results).
	_, err := entryFunc.Call(context.Background())
	if err != nil {
		return fmt.Errorf("module '%s' execution failed: %w", m.Name(), err)
	}

	return nil
}

func (m *WasmModule) Stop() error {
	m.logger.Info("Stopping module", "name", m.Name())
	// Closing the instance effectively stops it.
	return m.instance.Close(context.Background())
}
