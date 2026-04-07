package polymorphic

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"

	"golang.org/x/crypto/chacha20poly1305"
)

// Engine represents a polymorphic engine for code transformation
type Engine struct {
	logger *slog.Logger
}

// transformationEnvelope carries all randomness needed to reverse the polymorphic transformation.
type transformationEnvelope struct {
	XORKey           byte   `json:"xor_key"`
	AESKey           []byte `json:"aes_key"`
	AESIV            []byte `json:"aes_iv"`
	ChaChaKey        []byte `json:"chacha_key"`
	ChaChaNonce      []byte `json:"chacha_nonce"`
	GarbagePositions []int  `json:"garbage_positions"`
	GarbageBytes     []byte `json:"garbage_bytes"`
}

const polymorphicMagic = "PMOR1"

// NewEngine creates a new polymorphic engine
func NewEngine(logger *slog.Logger) *Engine {
	return &Engine{
		logger: logger,
	}
}

// TransformCode applies polymorphic transformations to code
func (e *Engine) TransformCode(code []byte) ([]byte, error) {
	// Apply multiple layers of transformation for stronger polymorphism while keeping
	// enough metadata to make the transformation reversible.

	// 1. XOR layer
	xorKey, err := e.generateRandomKey(1)
	if err != nil {
		return nil, fmt.Errorf("failed to generate XOR key: %w", err)
	}
	xorTransformed := e.applyXORTransformation(code, xorKey[0])

	// 2. AES-CBC layer
	aesKey, err := e.generateRandomKey(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate AES key: %w", err)
	}
	aesIV, err := e.generateRandomKey(aes.BlockSize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate AES IV: %w", err)
	}
	aesEncrypted, err := e.applyAESEncryption(xorTransformed, aesKey, aesIV)
	if err != nil {
		return nil, fmt.Errorf("failed to apply AES encryption: %w", err)
	}

	// 3. ChaCha20-Poly1305 layer
	chachaKey, err := e.generateRandomKey(chacha20poly1305.KeySize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ChaCha20-Poly1305 key: %w", err)
	}
	chachaNonce, err := e.generateRandomKey(chacha20poly1305.NonceSize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ChaCha20-Poly1305 nonce: %w", err)
	}
	chachaEncrypted, err := e.applyChaCha20Poly1305Encryption(aesEncrypted, chachaKey, chachaNonce)
	if err != nil {
		return nil, fmt.Errorf("failed to apply ChaCha20-Poly1305 encryption: %w", err)
	}

	// 4. Insert garbage with tracked positions
	garbageInserted, positions, garbageBytes, err := e.insertGarbageCodeWithMap(chachaEncrypted, 0.10) // 10% ratio
	if err != nil {
		return nil, fmt.Errorf("failed to insert garbage code: %w", err)
	}

	meta := transformationEnvelope{
		XORKey:           xorKey[0],
		AESKey:           aesKey,
		AESIV:            aesIV,
		ChaChaKey:        chachaKey,
		ChaChaNonce:      chachaNonce,
		GarbagePositions: positions,
		GarbageBytes:     garbageBytes,
	}

	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transformation metadata: %w", err)
	}

	// Envelope format: magic (5 bytes) | metaLen (uint32 BE) | meta | payload
	buf := &bytes.Buffer{}
	buf.WriteString(polymorphicMagic)
	lenField := make([]byte, 4)
	binary.BigEndian.PutUint32(lenField, uint32(len(metaBytes)))
	buf.Write(lenField)
	buf.Write(metaBytes)
	buf.Write(garbageInserted)

	e.logger.Info("Applied polymorphic transformation",
		"original_size", len(code),
		"transformed_size", buf.Len())

	return buf.Bytes(), nil
}

// applyXORTransformation applies XOR transformation with a random key
func (e *Engine) applyXORTransformation(code []byte, key byte) []byte {
	transformed := make([]byte, len(code))
	for i, b := range code {
		transformed[i] = b ^ key
	}
	return transformed
}

// applyAESEncryption applies AES encryption with the provided key and IV (CBC/PKCS7)
func (e *Engine) applyAESEncryption(code []byte, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	padding := aes.BlockSize - len(code)%aes.BlockSize
	padded := make([]byte, len(code)+padding)
	copy(padded, code)
	for i := len(code); i < len(padded); i++ {
		padded[i] = byte(padding)
	}

	encrypted := make([]byte, len(padded))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(encrypted, padded)
	return encrypted, nil
}

// applyAESDecryption reverses AES-CBC encryption with PKCS7 unpadding
func (e *Engine) applyAESDecryption(code, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}
	if len(code)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("ciphertext is not a multiple of block size")
	}

	decrypted := make([]byte, len(code))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(decrypted, code)

	if len(decrypted) == 0 {
		return nil, errors.New("decrypted data empty")
	}
	padding := int(decrypted[len(decrypted)-1])
	if padding == 0 || padding > aes.BlockSize || padding > len(decrypted) {
		return nil, fmt.Errorf("invalid PKCS7 padding")
	}
	return decrypted[:len(decrypted)-padding], nil
}

// applyChaCha20Poly1305Encryption applies ChaCha20-Poly1305 encryption with supplied key/nonce
func (e *Engine) applyChaCha20Poly1305Encryption(code []byte, key, nonce []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create ChaCha20-Poly1305 AEAD: %w", err)
	}
	return aead.Seal(nil, nonce, code, nil), nil
}

// applyChaCha20Poly1305Decryption reverses ChaCha20-Poly1305 encryption
func (e *Engine) applyChaCha20Poly1305Decryption(code []byte, key, nonce []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create ChaCha20-Poly1305 AEAD: %w", err)
	}
	return aead.Open(nil, nonce, code, nil)
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
	res, _, _, err := e.insertGarbageCodeWithMap(code, garbageRatio)
	return res, err
}

// ReverseTransformation reverses the polymorphic transformation
func (e *Engine) ReverseTransformation(code []byte) ([]byte, error) {
	if len(code) < len(polymorphicMagic)+4 {
		return nil, fmt.Errorf("payload too small to contain polymorphic header")
	}
	if string(code[:len(polymorphicMagic)]) != polymorphicMagic {
		return nil, fmt.Errorf("invalid polymorphic magic")
	}
	metaLen := binary.BigEndian.Uint32(code[len(polymorphicMagic) : len(polymorphicMagic)+4])
	metaStart := len(polymorphicMagic) + 4
	metaEnd := metaStart + int(metaLen)
	if metaEnd > len(code) {
		return nil, fmt.Errorf("metadata length exceeds payload")
	}
	metaBytes := code[metaStart:metaEnd]
	payload := code[metaEnd:]

	var meta transformationEnvelope
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	// Remove garbage in reverse order (positions were recorded during insertion)
	cleaned, err := e.removeGarbageWithVerification(payload, meta.GarbagePositions, meta.GarbageBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to strip garbage: %w", err)
	}

	// Reverse ChaCha20-Poly1305
	chachaPlain, err := e.applyChaCha20Poly1305Decryption(cleaned, meta.ChaChaKey, meta.ChaChaNonce)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt chacha layer: %w", err)
	}

	// Reverse AES
	aesPlain, err := e.applyAESDecryption(chachaPlain, meta.AESKey, meta.AESIV)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt AES layer: %w", err)
	}

	// Reverse XOR
	original := e.applyXORTransformation(aesPlain, meta.XORKey)
	e.logger.Info("Reversed polymorphic transformation", "original_size", len(original))
	return original, nil
}

// insertGarbageCodeWithMap inserts garbage bytes and returns their positions for reversibility.
func (e *Engine) insertGarbageCodeWithMap(code []byte, garbageRatio float64) ([]byte, []int, []byte, error) {
	if garbageRatio <= 0 || garbageRatio > 1 {
		return nil, nil, nil, fmt.Errorf("garbage ratio must be between 0 and 1")
	}

	garbageSize := int(float64(len(code)) * garbageRatio)
	garbage, err := e.GenerateGarbageCode(garbageSize)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate garbage code: %w", err)
	}

	result := make([]byte, len(code))
	copy(result, code)
	positions := make([]int, 0, garbageSize)

	for i := 0; i < garbageSize; i++ {
		pos, err := rand.Int(rand.Reader, big.NewInt(int64(len(result)+1)))
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to generate random position: %w", err)
		}
		idx := int(pos.Int64())
		positions = append(positions, idx)
		result = append(result[:idx], append([]byte{garbage[i]}, result[idx:]...)...)
	}

	e.logger.Info("Inserted garbage code",
		"original_size", len(code),
		"garbage_size", garbageSize,
		"result_size", len(result))

	return result, positions, garbage, nil
}

// removeGarbageWithVerification removes garbage bytes by recorded positions and verifies integrity.
func (e *Engine) removeGarbageWithVerification(code []byte, positions []int, garbage []byte) ([]byte, error) {
	if len(positions) != len(garbage) {
		return nil, fmt.Errorf("garbage metadata mismatch: positions=%d bytes=%d", len(positions), len(garbage))
	}
	if len(positions) == 0 {
		return code, nil
	}

	// Remove from the end to keep indices valid
	clean := make([]byte, len(code))
	copy(clean, code)
	for i := len(positions) - 1; i >= 0; i-- {
		pos := positions[i]
		if pos < 0 || pos >= len(clean) {
			return nil, fmt.Errorf("garbage position %d out of range", pos)
		}
		if clean[pos] != garbage[i] {
			return nil, fmt.Errorf("garbage byte mismatch at pos %d", pos)
		}
		clean = append(clean[:pos], clean[pos+1:]...)
	}
	return clean, nil
}
