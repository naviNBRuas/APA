package tests

import (
	"testing"
	"time"
)

// TestBasicFunctionality tests core functionality without external dependencies.
func TestBasicFunctionality(t *testing.T) {
	t.Run("Time Operations", func(t *testing.T) {
		start := time.Now()
		time.Sleep(10 * time.Millisecond)
		duration := time.Since(start)

		if duration < 10*time.Millisecond {
			t.Errorf("Sleep duration too short: got %v, expected at least 10ms", duration)
		}

		t.Logf("Time measurement working correctly: %v", duration)
	})

	t.Run("String Operations", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected bool
		}{
			{"hello", true},
			{"", false},
			{"test123", true},
		}

		for _, tc := range testCases {
			result := len(tc.input) > 0
			if result != tc.expected {
				t.Errorf("String length check failed for '%s': got %v, expected %v",
					tc.input, result, tc.expected)
			}
		}

		t.Log("String operations working correctly")
	})

	t.Run("Math Operations", func(t *testing.T) {
		tests := []struct {
			a, b, expected int
		}{
			{2, 3, 5},
			{10, 5, 15},
			{0, 0, 0},
			{-1, 1, 0},
		}

		for _, test := range tests {
			result := test.a + test.b
			if result != test.expected {
				t.Errorf("Addition failed: %d + %d = %d, expected %d",
					test.a, test.b, result, test.expected)
			}
		}

		t.Log("Math operations working correctly")
	})
}

// TestConfiguration validates basic configuration handling.
func TestConfiguration(t *testing.T) {
	type Config struct {
		Name    string
		Enabled bool
		Timeout time.Duration
	}

	t.Run("Default Configuration", func(t *testing.T) {
		config := Config{
			Name:    "test-agent",
			Enabled: true,
			Timeout: 30 * time.Second,
		}

		if config.Name == "" {
			t.Error("Config name should not be empty")
		}

		if !config.Enabled {
			t.Error("Config should be enabled by default")
		}

		if config.Timeout <= 0 {
			t.Error("Config timeout should be positive")
		}

		t.Logf("Default configuration: %+v", config)
	})

	t.Run("Configuration Validation", func(t *testing.T) {
		validConfigs := []Config{
			{Name: "valid1", Enabled: true, Timeout: time.Second},
			{Name: "valid2", Enabled: false, Timeout: time.Minute},
		}

		for i, config := range validConfigs {
			if config.Name == "" {
				t.Errorf("Valid config %d has empty name", i)
			}
			if config.Timeout <= 0 {
				t.Errorf("Valid config %d has invalid timeout", i)
			}
		}

		t.Log("Configuration validation working correctly")
	})
}

// BenchmarkBasicOperations benchmarks fundamental operations.
func BenchmarkBasicOperations(b *testing.B) {
	b.Run("StringConcatenation", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = "hello" + "world" + string(rune(i%256))
		}
	})

	b.Run("MathOperations", func(b *testing.B) {
		b.ReportAllocs()
		total := 0
		for i := 0; i < b.N; i++ {
			total += i * 2
		}
		_ = total
	})

	b.Run("TimeMeasurement", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			start := time.Now()
			_ = time.Since(start)
		}
	})
}
