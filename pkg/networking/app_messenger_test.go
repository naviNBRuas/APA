package networking

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncryptedMessengerRoundTrip(t *testing.T) {
	m, err := NewEncryptedMessenger(bytes.Repeat([]byte{0x01}, 32))
	require.NoError(t, err, "init failed: %v", err)
	nonce, ct, err := m.Seal([]byte("hello"))
	require.NoError(t, err, "seal failed: %v", err)
	plain, err := m.Open(nonce, ct)
	require.NoError(t, err, "open failed: %v", err)
	require.Equal(t, "hello", string(plain))
}

func TestSelectTransport(t *testing.T) {
	opt, err := SelectTransport([]TransportOption{{Name: "http", Available: false}, {Name: "ws", Available: true}})
	require.NoError(t, err, "selection failed: %v %v", opt, err)
	require.Equal(t, "ws", opt.Name, "selection failed")
}
