package backup

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBackupManager(t *testing.T) {
	// Create a temporary directory for backups
	tempDir, err := os.MkdirTemp("", "backup_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	logger := slog.Default()
	passphrase := "test-passphrase"
	backupManager := NewBackupManager(logger, tempDir, passphrase)

	// Test creating a backup manager
	if backupManager == nil {
		t.Fatal("Failed to create backup manager")
	}

	// Test that fields are initialized
	if backupManager.backupDir != tempDir {
		t.Errorf("Expected backupDir %s, got %s", tempDir, backupManager.backupDir)
	}

	if backupManager.passphrase != passphrase {
		t.Errorf("Expected passphrase %s, got %s", passphrase, backupManager.passphrase)
	}
}

func TestCreateRestoreBackup(t *testing.T) {
	// Create a temporary directory for backups
	tempDir, err := os.MkdirTemp("", "backup_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	logger := slog.Default()
	passphrase := "test-passphrase"
	backupManager := NewBackupManager(logger, tempDir, passphrase)

	// Test data
	config := map[string]interface{}{
		"agent_name": "test-agent",
		"version":    "1.0.0",
	}

	state := map[string]interface{}{
		"modules_loaded": 5,
		"peers_connected": 10,
	}

	criticalData := map[string]interface{}{
		"private_key": "secret-key-data",
		"credentials": "sensitive-credentials",
	}

	// Test creating a backup
	backupFile, err := backupManager.CreateBackup(config, state, criticalData)
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Check that backup file exists
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		t.Error("Backup file was not created")
	}

	// Test listing backups
	backups, err := backupManager.ListBackups()
	if err != nil {
		t.Errorf("Failed to list backups: %v", err)
	}

	if len(backups) != 1 {
		t.Errorf("Expected 1 backup, got %d", len(backups))
	}

	// Test restoring from backup
	restoredData, err := backupManager.RestoreBackup(backupFile)
	if err != nil {
		t.Fatalf("Failed to restore backup: %v", err)
	}

	// Check restored data
	if restoredData.Version != "1.0" {
		t.Errorf("Expected version 1.0, got %s", restoredData.Version)
	}

	if restoredData.Config["agent_name"] != "test-agent" {
		t.Errorf("Expected agent_name test-agent, got %v", restoredData.Config["agent_name"])
	}

	// Test deleting backup
	err = backupManager.DeleteBackup(backupFile)
	if err != nil {
		t.Errorf("Failed to delete backup: %v", err)
	}

	// Check that backup file no longer exists
	if _, err := os.Stat(backupFile); !os.IsNotExist(err) {
		t.Error("Backup file was not deleted")
	}
}

func TestListBackups(t *testing.T) {
	// Create a temporary directory for backups
	tempDir, err := os.MkdirTemp("", "backup_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	logger := slog.Default()
	passphrase := "test-passphrase"
	backupManager := NewBackupManager(logger, tempDir, passphrase)

	// Test listing backups with empty directory
	backups, err := backupManager.ListBackups()
	if err != nil {
		t.Errorf("Failed to list backups: %v", err)
	}

	if len(backups) != 0 {
		t.Errorf("Expected 0 backups, got %d", len(backups))
	}

	// Create some test files (non-backup files)
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create a backup file
	backupFile := filepath.Join(tempDir, "backup_20230101_120000.enc")
	err = os.WriteFile(backupFile, []byte("encrypted backup data"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	// Test listing backups again
	backups, err = backupManager.ListBackups()
	if err != nil {
		t.Errorf("Failed to list backups: %v", err)
	}

	if len(backups) != 1 {
		t.Errorf("Expected 1 backup, got %d", len(backups))
	}

	if backups[0] != backupFile {
		t.Errorf("Expected backup file %s, got %s", backupFile, backups[0])
	}
}

func TestEncryptionDecryption(t *testing.T) {
	// Create a temporary directory for backups
	tempDir, err := os.MkdirTemp("", "backup_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	logger := slog.Default()
	passphrase := "test-passphrase"
	backupManager := NewBackupManager(logger, tempDir, passphrase)

	// Test data
	testData := []byte("This is test data to encrypt and decrypt")

	// Test encryption
	encryptedData, err := backupManager.encryptData(testData)
	if err != nil {
		t.Fatalf("Failed to encrypt data: %v", err)
	}

	if len(encryptedData) <= len(testData) {
		t.Error("Encrypted data should be longer than original data")
	}

	// Test decryption
	decryptedData, err := backupManager.decryptData(encryptedData)
	if err != nil {
		t.Fatalf("Failed to decrypt data: %v", err)
	}

	if string(decryptedData) != string(testData) {
		t.Errorf("Decrypted data does not match original. Expected: %s, Got: %s", testData, decryptedData)
	}
}

func TestChecksum(t *testing.T) {
	logger := slog.Default()
	passphrase := "test-passphrase"
	backupManager := NewBackupManager(logger, "/tmp", passphrase)

	// Test data
	testData := []byte("This is test data for checksum")

	// Test calculating checksum
	checksum := backupManager.calculateChecksum(testData)
	if checksum == "" {
		t.Error("Checksum should not be empty")
	}

	// Test verifying checksum
	if !backupManager.verifyChecksum(testData, checksum) {
		t.Error("Checksum verification should pass for correct data")
	}

	// Test verifying checksum with incorrect data
	wrongData := []byte("This is wrong data")
	if backupManager.verifyChecksum(wrongData, checksum) {
		t.Error("Checksum verification should fail for incorrect data")
	}
}