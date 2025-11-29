package polymorphic

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"log/slog"
	"math/big"
	
	"golang.org/x/crypto/chacha20poly1305"
)

// Engine represents a polymorphic engine for code transformation
type Engine struct {
	logger *slog.Logger
}

// NewEngine creates a new polymorphic engine
func NewEngine(logger *slog.Logger) *Engine {
	return &Engine{
		logger: logger,
	}
}

// TransformCode applies polymorphic transformations to code
func (e *Engine) TransformCode(code []byte) ([]byte, error) {
	// Apply multiple layers of transformation for stronger polymorphism
	
	// 1. Apply XOR transformation with a random key
	xorTransformed, err := e.applyXORTransformation(code)
	if err != nil {
		return nil, fmt.Errorf("failed to apply XOR transformation: %w", err)
	}
	
	// 2. Apply AES encryption with a random key
	aesEncrypted, err := e.applyAESEncryption(xorTransformed)
	if err != nil {
		return nil, fmt.Errorf("failed to apply AES encryption: %w", err)
	}
	
	// 3. Apply ChaCha20-Poly1305 encryption
	chachaEncrypted, err := e.applyChaCha20Poly1305Encryption(aesEncrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to apply ChaCha20-Poly1305 encryption: %w", err)
	}
	
	// 4. Insert garbage code at random positions
	finalCode, err := e.InsertGarbageCode(chachaEncrypted, 0.1) // 10% garbage ratio
	if err != nil {
		return nil, fmt.Errorf("failed to insert garbage code: %w", err)
	}
	
	e.logger.Info("Applied polymorphic transformation", 
		"original_size", len(code), 
		"transformed_size", len(finalCode))
	
	return finalCode, nil
}

// applyXORTransformation applies XOR transformation with a random key
func (e *Engine) applyXORTransformation(code []byte) ([]byte, error) {
	key, err := e.generateRandomKey(1)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}
	
	transformed := make([]byte, len(code))
	for i, b := range code {
		transformed[i] = b ^ key[0]
	}
	
	return transformed, nil
}

// applyAESEncryption applies AES encryption with a random key
func (e *Engine) applyAESEncryption(code []byte) ([]byte, error) {
	// Generate a random 256-bit key
	key, err := e.generateRandomKey(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate AES key: %w", err)
	}
	
	// Generate a random 128-bit IV
	iv, err := e.generateRandomKey(16)
	if err != nil {
		return nil, fmt.Errorf("failed to generate IV: %w", err)
	}
	
	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}
	
	// Pad the code to be a multiple of the block size
	padding := aes.BlockSize - len(code)%aes.BlockSize
	paddedCode := make([]byte, len(code)+padding)
	copy(paddedCode, code)
	for i := len(code); i < len(paddedCode); i++ {
		paddedCode[i] = byte(padding)
	}
	
	// Encrypt the padded code
	encrypted := make([]byte, len(paddedCode))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(encrypted, paddedCode)
	
	// Prepend IV to the encrypted data
	result := make([]byte, len(iv)+len(encrypted))
	copy(result, iv)
	copy(result[len(iv):], encrypted)
	
	return result, nil
}

// applyChaCha20Poly1305Encryption applies ChaCha20-Poly1305 encryption
func (e *Engine) applyChaCha20Poly1305Encryption(code []byte) ([]byte, error) {
	// Generate a random key
	key, err := e.generateRandomKey(chacha20poly1305.KeySize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ChaCha20-Poly1305 key: %w", err)
	}
	
	// Create ChaCha20-Poly1305 AEAD
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create ChaCha20-Poly1305 AEAD: %w", err)
	}
	
	// Generate a random nonce
	nonce, err := e.generateRandomKey(chacha20poly1305.NonceSize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	
	// Encrypt the code
	encrypted := aead.Seal(nil, nonce, code, nil)
	
	// Prepend nonce to the encrypted data
	result := make([]byte, len(nonce)+len(encrypted))
	copy(result, nonce)
	copy(result[len(nonce):], encrypted)
	
	return result, nil
}

// generateRandomKey generates a random key of the specified length
func (e *Engine) generateRandomKey(length int) ([]byte, error) {
	key := make([]byte, length)
	_, err := rand.Read(key)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}
	return key, nil
}

// GenerateGarbageCode generates garbage code to obfuscate the original code
func (e *Engine) GenerateGarbageCode(size int) ([]byte, error) {
	garbage := make([]byte, size)
	_, err := rand.Read(garbage)
	if err != nil {
		return nil, fmt.Errorf("failed to generate garbage code: %w", err)
	}
	return garbage, nil
}

// InsertGarbageCode inserts garbage code at random positions in the original code
func (e *Engine) InsertGarbageCode(code []byte, garbageRatio float64) ([]byte, error) {
	if garbageRatio <= 0 || garbageRatio > 1 {
		return nil, fmt.Errorf("garbage ratio must be between 0 and 1")
	}
	
	garbageSize := int(float64(len(code)) * garbageRatio)
	garbage, err := e.GenerateGarbageCode(garbageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate garbage code: %w", err)
	}
	
	// Create a buffer to hold the result
	var result []byte
	
	// Start with the original code
	result = append(result, code...)
	
	// Insert garbage at random positions
	for i := 0; i < garbageSize; i++ {
		// Generate a random position to insert garbage
		pos, err := rand.Int(rand.Reader, big.NewInt(int64(len(result)+1))) // +1 to allow insertion at the end
		if err != nil {
			return nil, fmt.Errorf("failed to generate random position: %w", err)
		}
		
		// Insert garbage byte at position
		result = append(result[:pos.Int64()], append([]byte{garbage[i]}, result[pos.Int64():]...)...)
	}
	
	e.logger.Info("Inserted garbage code", 
		"original_size", len(code), 
		"garbage_size", garbageSize, 
		"result_size", len(result))
	
	return result, nil
}

// ReverseTransformation reverses the polymorphic transformation
func (e *Engine) ReverseTransformation(code []byte) ([]byte, error) {
	// In a real implementation, this would reverse the transformations
	// For now, we'll just return the code as-is
	e.logger.Info("Reversing polymorphic transformation", "size", len(code))
	return code, nil
}