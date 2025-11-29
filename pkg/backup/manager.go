package backup

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

// BackupManager handles encrypted backups of agent data
type BackupManager struct {
	logger     *slog.Logger
	backupDir  string
	passphrase string
}

// BackupData represents the structure of backed up data
type BackupData struct {
	Timestamp   time.Time              `json:"timestamp"`
	Version     string                 `json:"version"`
	Config      map[string]interface{} `json:"config"`
	State       map[string]interface{} `json:"state"`
	CriticalData map[string]interface{} `json:"critical_data"`
	Checksum    string                 `json:"checksum"`
}

// NewBackupManager creates a new backup manager
func NewBackupManager(logger *slog.Logger, backupDir, passphrase string) *BackupManager {
	return &BackupManager{
		logger:     logger,
		backupDir:  backupDir,
		passphrase: passphrase,
	}
}

// CreateBackup creates an encrypted backup of agent data
func (bm *BackupManager) CreateBackup(config, state, criticalData map[string]interface{}) (string, error) {
	bm.logger.Info("Creating encrypted backup")

	// Create backup data structure
	backupData := &BackupData{
		Timestamp:    time.Now(),
		Version:      "1.0",
		Config:       config,
		State:        state,
		CriticalData: criticalData,
	}

	// Serialize backup data
	data, err := json.Marshal(backupData)
	if err != nil {
		return "", fmt.Errorf("failed to serialize backup data: %w", err)
	}

	// Calculate checksum
	checksum := bm.calculateChecksum(data)
	backupData.Checksum = checksum

	// Re-serialize with checksum
	data, err = json.Marshal(backupData)
	if err != nil {
		return "", fmt.Errorf("failed to serialize backup data with checksum: %w", err)
	}

	// Encrypt the data
	encryptedData, err := bm.encryptData(data)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt backup data: %w", err)
	}

	// Create backup filename
	filename := fmt.Sprintf("backup_%s.enc", time.Now().Format("20060102_150405"))
	filepath := filepath.Join(bm.backupDir, filename)

	// Write encrypted data to file
	err = os.WriteFile(filepath, encryptedData, 0600)
	if err != nil {
		return "", fmt.Errorf("failed to write backup file: %w", err)
	}

	bm.logger.Info("Backup created successfully", "file", filepath)
	return filepath, nil
}

// RestoreBackup restores agent data from an encrypted backup
func (bm *BackupManager) RestoreBackup(backupFile string) (*BackupData, error) {
	bm.logger.Info("Restoring from encrypted backup", "file", backupFile)

	// Read encrypted data from file
	encryptedData, err := os.ReadFile(backupFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup file: %w", err)
	}

	// Decrypt the data
	data, err := bm.decryptData(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt backup data: %w", err)
	}

	// Deserialize backup data
	var backupData BackupData
	err = json.Unmarshal(data, &backupData)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize backup data: %w", err)
	}

	// Verify checksum
	if !bm.verifyChecksum(data, backupData.Checksum) {
		return nil, fmt.Errorf("backup checksum verification failed")
	}

	bm.logger.Info("Backup restored successfully", "timestamp", backupData.Timestamp)
	return &backupData, nil
}

// ListBackups lists all available backups
func (bm *BackupManager) ListBackups() ([]string, error) {
	bm.logger.Info("Listing available backups")

	// Read directory
	entries, err := os.ReadDir(bm.backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	// Filter for backup files
	var backups []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".enc" {
			backups = append(backups, filepath.Join(bm.backupDir, entry.Name()))
		}
	}

	bm.logger.Info("Found backups", "count", len(backups))
	return backups, nil
}

// DeleteBackup deletes a backup file
func (bm *BackupManager) DeleteBackup(backupFile string) error {
	bm.logger.Info("Deleting backup", "file", backupFile)

	err := os.Remove(backupFile)
	if err != nil {
		return fmt.Errorf("failed to delete backup file: %w", err)
	}

	bm.logger.Info("Backup deleted successfully", "file", backupFile)
	return nil
}

// encryptData encrypts data using AES-GCM with a passphrase-derived key
func (bm *BackupManager) encryptData(data []byte) ([]byte, error) {
	// Derive key from passphrase
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	key := pbkdf2.Key([]byte(bm.passphrase), salt, 10000, 32, sha256.New)

	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt data
	ciphertext := gcm.Seal(nonce, nonce, data, nil)

	// Prepend salt to ciphertext
	result := append(salt, ciphertext...)

	return result, nil
}

// decryptData decrypts data using AES-GCM with a passphrase-derived key
func (bm *BackupManager) decryptData(encryptedData []byte) ([]byte, error) {
	// Extract salt (first 16 bytes)
	if len(encryptedData) < 16 {
		return nil, fmt.Errorf("encrypted data too short")
	}

	salt := encryptedData[:16]
	ciphertext := encryptedData[16:]

	// Derive key from passphrase
	key := pbkdf2.Key([]byte(bm.passphrase), salt, 10000, 32, sha256.New)

	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract nonce
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt data
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	return plaintext, nil
}

// calculateChecksum calculates SHA-256 checksum of data
func (bm *BackupManager) calculateChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// verifyChecksum verifies the checksum of data
func (bm *BackupManager) verifyChecksum(data []byte, expectedChecksum string) bool {
	actualChecksum := bm.calculateChecksum(data)
	return actualChecksum == expectedChecksum
}

// ScheduleAutomaticBackups schedules automatic backups at regular intervals
func (bm *BackupManager) ScheduleAutomaticBackups(interval time.Duration, config, state, criticalData map[string]interface{}) {
	bm.logger.Info("Scheduling automatic backups", "interval", interval)

	// In a real implementation, this would:
	// 1. Create a ticker for the specified interval
	// 2. Create backups periodically
	// 3. Handle cleanup of old backups
	
	// For now, we'll just log the action
	bm.logger.Info("Would schedule automatic backups every", "interval", interval)
}