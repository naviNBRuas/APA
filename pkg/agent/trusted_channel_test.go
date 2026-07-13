package agent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrustedChannel(t *testing.T) {
	ch := NewTrustedChannel([]byte("secret"))
	data := []byte("payload")
	sig := ch.Sign(data)
	require.NoError(t, ch.Verify(data, sig), "verify failed")
	require.Error(t, ch.Verify([]byte("tamper"), sig), "expected mismatch")
}
