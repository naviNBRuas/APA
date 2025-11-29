package security

import (
	"bytes"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	// Generate a key
	key, err := GenerateKey(32) // 256-bit key
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Test data
	plaintext := []byte("This is a secret message")

	// Encrypt the data
	ciphertext, err := EncryptData(key, plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt data: %v", err)
	}

	// Decrypt the data
	decrypted, err := DecryptData(key, ciphertext)
	if err != nil {
		t.Fatalf("Failed to decrypt data: %v", err)
	}

	// Check that the decrypted data matches the original
	if !bytes.Equal(plaintext, decrypted) {
		t.Errorf("Decrypted data does not match original. Expected: %s, Got: %s", plaintext, decrypted)
	}
}

func TestGenerateKey(t *testing.T) {
	key, err := GenerateKey(32)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	if len(key) != 32 {
		t.Errorf("Expected key length 32, got %d", len(key))
	}
}