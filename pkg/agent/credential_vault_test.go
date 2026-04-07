package agent

import (
	"testing"
	"time"
)

func TestCredentialVaultStoresAndExpires(t *testing.T) {
	v := NewCredentialVault()
	v.Put("api", "token123", 10*time.Millisecond)
	if cred, ok := v.Get("api"); !ok || cred.Token != "token123" {
		t.Fatalf("expected token")
	}
	time.Sleep(15 * time.Millisecond)
	if _, ok := v.Get("api"); ok {
		t.Fatalf("expected expiry")
	}
}
