package networking

import (
	"bytes"
	"testing"
)

func TestEncryptedMessengerRoundTrip(t *testing.T) {
	m, err := NewEncryptedMessenger(bytes.Repeat([]byte{0x01}, 32))
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}
	nonce, ct, err := m.Seal([]byte("hello"))
	if err != nil {
		t.Fatalf("seal failed: %v", err)
	}
	plain, err := m.Open(nonce, ct)
	if err != nil || string(plain) != "hello" {
		t.Fatalf("open failed: %v", err)
	}
}

func TestSelectTransport(t *testing.T) {
	opt, err := SelectTransport([]TransportOption{{Name: "http", Available: false}, {Name: "ws", Available: true}})
	if err != nil || opt.Name != "ws" {
		t.Fatalf("selection failed: %v %v", opt, err)
	}
}
