package agent

import (
	"testing"
	"time"
)

func TestEnvInspectorProfiles(t *testing.T) {
	// override to deterministic values
	uptimeReader = func() (time.Duration, bool) { return 42 * time.Second, true }
	entropyCheck = func() bool { return true }
	virtCheck = func() bool { return true }
	hwHintReader = func() string { return "TestMachine" }

	prof := EnvInspector{}.Inspect()
	if prof.Uptime != 42*time.Second || !prof.Virtualized || !prof.EntropyAvailable {
		t.Fatalf("unexpected profile: %+v", prof)
	}
	if !prof.ShouldPreferLowProfile() {
		t.Fatalf("expected low-profile preference")
	}
}
