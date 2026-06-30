package store

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

type Store struct {
	path     string
	logger   *slog.Logger
	mu       sync.RWMutex
	data     map[string]json.RawMessage
	modified bool
}

func New(path string, logger *slog.Logger) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("store mkdir: %w", err)
	}
	s := &Store{
		path:   path,
		logger: logger,
		data:   make(map[string]json.RawMessage),
	}
	if err := s.load(); err != nil {
		logger.Warn("Store: no existing data, starting fresh", "path", path, "error", err)
	}
	return s, nil
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.data)
}

func (s *Store) save() error {
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("store marshal: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("store write tmp: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("store rename: %w", err)
	}
	return nil
}

func (s *Store) Get(key string, value interface{}) error {
	s.mu.RLock()
	raw, ok := s.data[key]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("store: key %q not found", key)
	}
	return json.Unmarshal(raw, value)
}

func (s *Store) Set(key string, value interface{}) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("store marshal value: %w", err)
	}
	s.mu.Lock()
	s.data[key] = raw
	s.modified = true
	s.mu.Unlock()
	return nil
}

func (s *Store) SetAndSave(key string, value interface{}) error {
	if err := s.Set(key, value); err != nil {
		return err
	}
	return s.Flush()
}

func (s *Store) Delete(key string) {
	s.mu.Lock()
	delete(s.data, key)
	s.modified = true
	s.mu.Unlock()
}

func (s *Store) Exists(key string) bool {
	s.mu.RLock()
	_, ok := s.data[key]
	s.mu.RUnlock()
	return ok
}

func (s *Store) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]string, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}
	return keys
}

func (s *Store) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.modified {
		return nil
	}
	if err := s.save(); err != nil {
		return err
	}
	s.modified = false
	return nil
}

func (s *Store) Close() error {
	return s.Flush()
}
