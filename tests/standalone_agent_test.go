package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func testAgentBinaryName(base string) string {
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}
	return base
}

func TestStandaloneAgentVersion(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go binary not found")
	}

	binaryName := testAgentBinaryName("test-agent")
	binaryPath, _ := filepath.Abs(binaryName)
	cmd := exec.Command("go", "build", "-o", binaryName, "../cmd/standalone-agent/main.go")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build standalone agent: %v", err)
	}
	defer os.Remove(binaryName)

	cmd = exec.Command(binaryPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to run agent with --version: %v", err)
	}
	if string(output) == "" {
		t.Error("Version output is empty")
	}
}
