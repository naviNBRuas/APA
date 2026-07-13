package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateKey(t *testing.T) {
	tests := []struct {
		name    string
		size    int
		wantErr bool
	}{
		{"16 bytes (AES-128)", 16, false},
		{"24 bytes (AES-192)", 24, false},
		{"32 bytes (AES-256)", 32, false},
		{"zero size", 0, true},
		{"negative size", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := GenerateKey(tt.size)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, key)
				return
			}
			require.NoError(t, err)
			assert.Len(t, key, tt.size)
		})
	}
}

func TestGenerateKey_Unique(t *testing.T) {
	key1, err := GenerateKey(32)
	require.NoError(t, err)
	key2, err := GenerateKey(32)
	require.NoError(t, err)

	assert.NotEqual(t, key1, key2, "consecutive keys should be unique")
}

func TestEncryptDecrypt(t *testing.T) {
	key, err := GenerateKey(32)
	require.NoError(t, err)

	tests := []struct {
		name      string
		plaintext []byte
	}{
		{"small message", []byte("hello")},
		{"large message", []byte("The quick brown fox jumps over the lazy dog. This is a longer message to test AES-GCM encryption with various sizes.")},
		{"binary data", []byte{0x00, 0xFF, 0xFE, 0x01, 0x7F, 0x80}},
		{"unicode text", []byte("Hello, 世界! 🌍")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ciphertext, err := EncryptData(key, tt.plaintext)
			require.NoError(t, err)
			require.NotNil(t, ciphertext)
			assert.NotEqual(t, tt.plaintext, ciphertext, "ciphertext should differ from plaintext")

			decrypted, err := DecryptData(key, ciphertext)
			require.NoError(t, err)
			assert.Equal(t, len(tt.plaintext), len(decrypted))
			assert.Equal(t, tt.plaintext, decrypted)
		})
	}
}

func TestEncryptDecrypt_WrongKey(t *testing.T) {
	key, err := GenerateKey(32)
	require.NoError(t, err)

	wrongKey, err := GenerateKey(32)
	require.NoError(t, err)

	plaintext := []byte("secret data")
	ciphertext, err := EncryptData(key, plaintext)
	require.NoError(t, err)

	_, err = DecryptData(wrongKey, ciphertext)
	assert.Error(t, err, "decryption with wrong key should fail")
}

func TestEncryptDecrypt_TamperedCiphertext(t *testing.T) {
	key, err := GenerateKey(32)
	require.NoError(t, err)

	plaintext := []byte("secret data")
	ciphertext, err := EncryptData(key, plaintext)
	require.NoError(t, err)

	tampered := make([]byte, len(ciphertext))
	copy(tampered, ciphertext)
	if len(tampered) > 0 {
		tampered[len(tampered)-1] ^= 0xFF
	}

	_, err = DecryptData(key, tampered)
	assert.Error(t, err, "decryption of tampered ciphertext should fail")
}

func TestEncryptDecrypt_ShortCiphertext(t *testing.T) {
	key, err := GenerateKey(32)
	require.NoError(t, err)

	_, err = DecryptData(key, []byte{})
	assert.Error(t, err)

	block, err := aes.NewCipher(key)
	require.NoError(t, err)
	gcm, err := cipher.NewGCM(block)
	require.NoError(t, err)

	short := make([]byte, gcm.NonceSize()-1)
	_, err = DecryptData(key, short)
	assert.Error(t, err, "ciphertext shorter than nonce should fail")
}

func TestEncryptDecrypt_MultipleKeys(t *testing.T) {
	plaintext := []byte("test data across multiple keys")

	tests := []struct {
		name string
		size int
	}{
		{"AES-128", 16},
		{"AES-192", 24},
		{"AES-256", 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := GenerateKey(tt.size)
			require.NoError(t, err)

			ciphertext, err := EncryptData(key, plaintext)
			require.NoError(t, err)

			decrypted, err := DecryptData(key, ciphertext)
			require.NoError(t, err)
			assert.Equal(t, plaintext, decrypted)
		})
	}
}

func TestGenerateKey_InvalidSizes(t *testing.T) {
	_, err := GenerateKey(0)
	assert.Error(t, err)

	_, err = GenerateKey(-1)
	assert.Error(t, err)
}

func TestEncryptWithNilData(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	ciphertext, err := EncryptData(key, nil)
	require.NoError(t, err)

	decrypted, err := DecryptData(key, ciphertext)
	require.NoError(t, err)
	assert.Nil(t, decrypted)
}

func TestDecryptWithNilData(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	_, err := DecryptData(key, nil)
	assert.Error(t, err)
}
