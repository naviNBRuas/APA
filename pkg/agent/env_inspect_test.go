package agent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEnvInspectorProfiles(t *testing.T) {
	// override to deterministic values
	uptimeReader = func() (time.Duration, bool) { return 42 * time.Second, true }
	entropyCheck = func() bool { return true }
	virtCheck = func() bool { return true }
	hwHintReader = func() string { return "TestMachine" }

	prof := EnvInspector{}.Inspect()
	require.Equal(t, 42*time.Second, prof.Uptime, "unexpected profile")
	require.True(t, prof.Virtualized, "unexpected profile")
	require.True(t, prof.EntropyAvailable, "unexpected profile")
	require.True(t, prof.ShouldPreferLowProfile(), "expected low-profile preference")
}
