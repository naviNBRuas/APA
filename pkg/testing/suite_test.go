//go:build enhanced

package testing

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

func TestNewTestSuite(t *testing.T) {
	ts := NewTestSuite(slog.Default(), TestConfig{})
	if ts == nil {
		t.Fatal("NewTestSuite returned nil")
	}
}

func TestTestSuiteStartStop(t *testing.T) {
	ts := NewTestSuite(slog.Default(), TestConfig{
		UnitTestsConfig: UnitTestsConfig{Enabled: true},
	})
	if err := ts.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	ts.Stop()
}

func TestTestSuiteRunUnitTests(t *testing.T) {
	ts := NewTestSuite(slog.Default(), TestConfig{
		EnableUnitTests: true,
		TestTimeout:     10 * time.Second,
	})
	if err := ts.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer ts.Stop()

	results, err := ts.RunAllTests(context.Background())
	if err != nil {
		t.Fatalf("RunAllTests failed: %v", err)
	}
	if results == nil {
		t.Fatal("RunAllTests returned nil results")
	}
}

func TestTestSuiteGetSummary(t *testing.T) {
	ts := NewTestSuite(slog.Default(), TestConfig{})
	summary := ts.GetSummary()
	if summary == nil {
		t.Fatal("GetSummary returned nil")
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
			ts := NewTestSuite(slog.Default(), tc.config)
			err := ts.Start()
			if tc.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			ts.Stop()
		})
	}
}
