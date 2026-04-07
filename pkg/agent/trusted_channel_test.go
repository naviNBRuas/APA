package agent

import "testing"

func TestTrustedChannel(t *testing.T) {
	ch := NewTrustedChannel([]byte("secret"))
	data := []byte("payload")
	sig := ch.Sign(data)
	if err := ch.Verify(data, sig); err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if err := ch.Verify([]byte("tamper"), sig); err == nil {
		t.Fatalf("expected mismatch")
	}
}
