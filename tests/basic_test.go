package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/naviNBRuas/APA/pkg/agent"
)

func writeMinimalConfig(t *testing.T, dir string) string {
	t.Helper()

	dirs := []string{"modules", "policies", "controllers", "identities"}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(dir, d), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	identityFile := filepath.Join(dir, "identities", "key.pem")

	policyFile := filepath.Join(dir, "policies", "policy.yaml")
	if err := os.WriteFile(policyFile, []byte("trusted_authors: []\n"), 0644); err != nil {
		t.Fatalf("write policy: %v", err)
	}

	configPath := filepath.Join(dir, "config.yaml")
	configContent := `
admin_listen_address: 127.0.0.1:0
module_path: ` + dir + `/modules
identity_file_path: ` + identityFile + `
policy_path: ` + policyFile + `
controller_path: ` + dir + `/controllers
update:
  public_key: "92aaba2155699b6691c41270c98d4570a96716a1d5f98e44b556958e352270a0"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return configPath
}

func TestNewRuntime(t *testing.T) {
	dir := t.TempDir()
	configPath := writeMinimalConfig(t, dir)

	rt, err := agent.NewRuntime(configPath, "test")
	if err != nil {
		t.Fatalf("NewRuntime failed: %v", err)
	}
	if rt == nil {
		t.Fatal("NewRuntime returned nil")
	}
}

func TestNewRuntimeInvalidConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(configPath, []byte("{invalid: yaml: ["), 0644); err != nil {
		t.Fatalf("write bad config: %v", err)
	}

	if _, err := agent.NewRuntime(configPath, "test"); err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestNewRuntimeMissingConfig(t *testing.T) {
	if _, err := agent.NewRuntime("/nonexistent/path/config.yaml", "test"); err == nil {
		t.Error("expected error for nonexistent config")
	}
}

func TestApplyConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := writeMinimalConfig(t, dir)

	rt, err := agent.NewRuntime(configPath, "test")
	if err != nil {
		t.Fatalf("NewRuntime failed: %v", err)
	}

	newConfig := "admin_listen_address: 127.0.0.1:0\nlog_level: debug\n"
	if err := rt.ApplyConfig([]byte(newConfig)); err != nil {
		t.Errorf("ApplyConfig failed: %v", err)
	}
}

func TestStopWithoutStart(t *testing.T) {
	dir := t.TempDir()
	configPath := writeMinimalConfig(t, dir)

	rt, err := agent.NewRuntime(configPath, "test")
	if err != nil {
		t.Fatalf("NewRuntime failed: %v", err)
	}
	rt.Stop()
}

func TestGetCurrentRelease(t *testing.T) {
	dir := t.TempDir()
	configPath := writeMinimalConfig(t, dir)

	rt, err := agent.NewRuntime(configPath, "test")
	if err != nil {
		t.Fatalf("NewRuntime failed: %v", err)
	}

	release, data, err := rt.GetCurrentRelease()
	if err == nil {
		t.Logf("release: %+v", release)
		_ = data
	} else {
		t.Logf("GetCurrentRelease returned expected error: %v", err)
	}
}
