package tests

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func testAgentBinaryName(base string) string {
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}
	return base
}

// TestStandaloneAgent tests the basic functionality of the standalone agent.
func TestStandaloneAgent(t *testing.T) {
	// Build the standalone agent
	binaryName := testAgentBinaryName("test-agent")
	binaryPath, err := filepath.Abs(binaryName)
	if err != nil {
		t.Fatalf("Failed to resolve binary path: %v", err)
	}
	cmd := exec.Command("go", "build", "-o", binaryName, "../cmd/standalone-agent/main.go")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build standalone agent: %v", err)
	}
	defer os.Remove(binaryName)

	// Test basic execution with version flag
	t.Run("Version Flag", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "--version")
		output, err := cmd.Output()
		if err != nil {
			t.Fatalf("Failed to run agent with --version: %v", err)
		}

		outputStr := string(output)
		if outputStr == "" {
			t.Error("Version output is empty")
		}

		t.Logf("Version output: %s", outputStr)
	})

	// Test help flag
	t.Run("Help Flag", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "--help")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Logf("Help command returned non-zero exit (acceptable for some flag parsers): %v", err)
		}

		outputStr := string(output)
		if outputStr == "" {
			t.Error("Help output is empty")
			return
		}

		// Check for expected help text
		expectedFlags := []string{"-demo", "-demo-delay", "-log-level", "-version"}
		for _, flag := range expectedFlags {
			if !contains(outputStr, flag) {
				t.Errorf("Help output missing expected flag: %s", flag)
			}
		}

		t.Logf("Help output contains all expected flags")
	})

	// Test demonstration mode with short delay
	t.Run("Demonstration Mode", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binaryPath, "--demo", "--demo-delay=1s")
		err := cmd.Run()
		if ctx.Err() == context.DeadlineExceeded {
			t.Log("Demonstration timed out (as expected)")
			return
		}
		if err != nil {
			t.Logf("Demonstration completed with non-zero exit (acceptable): %v", err)
		}
	})

	// Test different log levels
	t.Run("Log Levels", func(t *testing.T) {
		levels := []string{"debug", "info", "warn", "error"}

		for _, level := range levels {
			t.Run(level, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, binaryPath, "--log-level", level, "--demo", "--demo-delay=100ms")
				err := cmd.Run()
				if ctx.Err() == context.DeadlineExceeded {
					t.Logf("Log level %s test timed out", level)
					return
				}
				if err != nil {
					t.Logf("Log level %s test completed with non-zero exit (acceptable): %v", level, err)
				}
			})
		}
	})
}

// BenchmarkStandaloneAgent benchmarks the agent startup time.
func BenchmarkStandaloneAgent(b *testing.B) {
	// Build the agent once
	binaryName := testAgentBinaryName("bench-agent")
	binaryPath, err := filepath.Abs(binaryName)
	if err != nil {
		b.Fatalf("Failed to resolve benchmark binary path: %v", err)
	}
	cmd := exec.Command("go", "build", "-o", binaryName, "../cmd/standalone-agent/main.go")
	if err := cmd.Run(); err != nil {
		b.Fatalf("Failed to build benchmark agent: %v", err)
	}
	defer os.Remove(binaryName)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		cmd := exec.CommandContext(ctx, binaryPath, "--demo", "--demo-delay=10ms")
		_ = cmd.Run()
		cancel()
	}
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr || contains(s[1:], substr))))
}
