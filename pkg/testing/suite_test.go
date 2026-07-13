//go:build enhanced

package testing

import (
	"log/slog"
	"testing"
	"time"
)

func TestNewTestSuite(t *testing.T) {
	ts, err := NewTestSuite(slog.Default(), TestConfig{})
	if err != nil {
		t.Fatalf("NewTestSuite failed: %v", err)
	}
	if ts == nil {
		t.Fatal("NewTestSuite returned nil")
	}
}

func TestTestSuiteRun(t *testing.T) {
	ts, err := NewTestSuite(slog.Default(), TestConfig{
		EnableUnitTests: true,
		TestTimeout:     10 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewTestSuite failed: %v", err)
	}

	summary, err := ts.Run()
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if summary == nil {
		t.Fatal("Run returned nil summary")
	}
}

func TestTestSuiteConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  TestConfig
		wantErr bool
	}{
		{"default config", TestConfig{}, false},
		{"unit tests only", TestConfig{EnableUnitTests: true}, false},
		{"all enabled", TestConfig{
			EnableUnitTests:          true,
			EnableIntegrationTests:   true,
			EnablePerformanceTests:   true,
			EnableStressTests:        true,
			EnableCompatibilityTests: true,
		}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts, err := NewTestSuite(slog.Default(), tc.config)
			if err != nil {
				t.Fatalf("NewTestSuite failed: %v", err)
			}
			_, err = ts.Run()
			if tc.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
