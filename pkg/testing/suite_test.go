//go:build enhanced

package testing

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTestSuite(t *testing.T) {
	ts, err := NewTestSuite(slog.Default(), TestConfig{})
	require.NoError(t, err, "NewTestSuite failed: %v", err)
	require.NotNil(t, ts, "NewTestSuite returned nil")
}

func TestTestSuiteRun(t *testing.T) {
	ts, err := NewTestSuite(slog.Default(), TestConfig{
		EnableUnitTests: true,
		TestTimeout:     10 * time.Second,
	})
	require.NoError(t, err, "NewTestSuite failed: %v", err)

	summary, err := ts.Run()
	require.NoError(t, err, "Run failed: %v", err)
	require.NotNil(t, summary, "Run returned nil summary")
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
			require.NoError(t, err, "NewTestSuite failed: %v", err)
			_, err = ts.Run()
			if tc.wantErr {
				assert.Error(t, err, "expected error, got nil")
			} else {
				assert.NoError(t, err, "unexpected error: %v", err)
			}
		})
	}
}
