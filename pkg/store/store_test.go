package store

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.json")
	s, err := New(path, slog.Default())
	require.NoError(t, err)
	require.NotNil(t, s)
	defer func() { _ = s.Close() }()
}

func TestSetAndGet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.json")
	s, err := New(path, slog.Default())
	require.NoError(t, err)
	defer func() { _ = s.Close() }()

	err = s.Set("key1", "value1")
	require.NoError(t, err)

	var got string
	err = s.Get("key1", &got)
	require.NoError(t, err)
	assert.Equal(t, "value1", got)
}

func TestGetNonExistent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.json")
	s, err := New(path, slog.Default())
	require.NoError(t, err)
	defer func() { _ = s.Close() }()

	var v string
	err = s.Get("nonexistent", &v)
	assert.Error(t, err)
}

func TestSetAndSavePersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.json")
	s, err := New(path, slog.Default())
	require.NoError(t, err)

	err = s.SetAndSave("greeting", "hello")
	require.NoError(t, err)
	err = s.Close()
	require.NoError(t, err)

	s2, err := New(path, slog.Default())
	require.NoError(t, err)
	defer func() { _ = s2.Close() }()

	var got string
	err = s2.Get("greeting", &got)
	require.NoError(t, err)
	assert.Equal(t, "hello", got)
}

func TestDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.json")
	s, err := New(path, slog.Default())
	require.NoError(t, err)
	defer func() { _ = s.Close() }()

	_ = s.Set("key1", "value1")
	assert.True(t, s.Exists("key1"))

	s.Delete("key1")
	assert.False(t, s.Exists("key1"))
}

func TestKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.json")
	s, err := New(path, slog.Default())
	require.NoError(t, err)
	defer func() { _ = s.Close() }()

	_ = s.Set("a", 1)
	_ = s.Set("b", 2)
	_ = s.Set("c", 3)

	keys := s.Keys()
	assert.Len(t, keys, 3)
	assert.Contains(t, keys, "a")
	assert.Contains(t, keys, "b")
	assert.Contains(t, keys, "c")
}

func TestStructValues(t *testing.T) {
	type Config struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	path := filepath.Join(t.TempDir(), "test.json")
	s, err := New(path, slog.Default())
	require.NoError(t, err)
	defer func() { _ = s.Close() }()

	cfg := Config{Name: "test", Count: 42}
	err = s.Set("config", cfg)
	require.NoError(t, err)

	var loaded Config
	err = s.Get("config", &loaded)
	require.NoError(t, err)
	assert.Equal(t, "test", loaded.Name)
	assert.Equal(t, 42, loaded.Count)
}

func TestConcurrentAccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.json")
	s, err := New(path, slog.Default())
	require.NoError(t, err)
	defer func() { _ = s.Close() }()

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func(n int) {
			_ = s.Set("key", n)
			var v int
			_ = s.Get("key", &v)
			done <- struct{}{}
		}(i)
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestFlushIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.json")
	s, err := New(path, slog.Default())
	require.NoError(t, err)

	err = s.Flush()
	require.NoError(t, err)

	_ = s.Set("x", 1)
	err = s.Flush()
	require.NoError(t, err)

	err = s.Flush()
	require.NoError(t, err)
	_ = s.Close()
}

func TestLoadExistingFile(t *testing.T) {
	originalContent := `{"name": "apa-agent", "version": "1.0.0"}`
	path := filepath.Join(t.TempDir(), "test.json")
	err := os.WriteFile(path, []byte(originalContent), 0644)
	require.NoError(t, err)

	s, err := New(path, slog.Default())
	require.NoError(t, err)
	defer func() { _ = s.Close() }()

	var name string
	err = s.Get("name", &name)
	require.NoError(t, err)
	assert.Equal(t, "apa-agent", name)

	var version string
	err = s.Get("version", &version)
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", version)
}
