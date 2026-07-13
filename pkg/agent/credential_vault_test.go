package agent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCredentialVaultStoresAndExpires(t *testing.T) {
	v := NewCredentialVault()
	v.Put("api", "token123", 10*time.Millisecond)
	cred, ok := v.Get("api")
	require.True(t, ok, "expected token")
	require.Equal(t, "token123", cred.Token, "expected token")
	time.Sleep(15 * time.Millisecond)
	_, ok = v.Get("api")
	require.False(t, ok, "expected expiry")
}
