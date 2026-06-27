package module

import (
	"fmt"
	"io"
	"log/slog"
	"testing"
)

func newBenchManager(n int) *Manager {
	modules := make(map[string]Module, n)
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("bench-module-%d", i)
		modules[name] = &WasmModule{
			manifest: &Manifest{
				Name:    name,
				Version: "v1.0.0",
			},
		}
	}
	return &Manager{
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		modules: modules,
	}
}

func BenchmarkListModules(b *testing.B) {
	m := newBenchManager(100)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.ListModules()
	}
}

func BenchmarkHasModule(b *testing.B) {
	m := newBenchManager(100)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.HasModule("bench-module-1", "v1.0.0")
	}
}
