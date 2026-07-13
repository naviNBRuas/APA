package backup

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackupManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "backup_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir) //nolint:errcheck

	logger := slog.Default()
	passphrase := "test-passphrase"
	backupManager := NewBackupManager(logger, tempDir, passphrase)

	require.NotNil(t, backupManager, "Failed to create backup manager")
	assert.Equal(t, tempDir, backupManager.backupDir)
	assert.Equal(t, passphrase, backupManager.passphrase)
}

func TestCreateRestoreBackup(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "backup_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir) //nolint:errcheck

	logger := slog.Default()
	passphrase := "test-passphrase"
	backupManager := NewBackupManager(logger, tempDir, passphrase)

	config := map[string]interface{}{
		"agent_name": "test-agent",
		"version":    "1.0.0",
	}

	state := map[string]interface{}{
		"modules_loaded":  5,
		"peers_connected": 10,
	}

	criticalData := map[string]interface{}{
		"private_key": "secret-key-data",
		"credentials": "sensitive-credentials",
	}

	backupFile, err := backupManager.CreateBackup(config, state, criticalData)
	require.NoError(t, err, "Failed to create backup")

	assert.FileExists(t, backupFile, "Backup file was not created")

	backups, err := backupManager.ListBackups()
	assert.NoError(t, err, "Failed to list backups")
	assert.Equal(t, 1, len(backups))

	restoredData, err := backupManager.RestoreBackup(backupFile)
	require.NoError(t, err, "Failed to restore backup")

	assert.Equal(t, "1.0", restoredData.Version)
	assert.Equal(t, "test-agent", restoredData.Config["agent_name"])

	err = backupManager.DeleteBackup(backupFile)
	assert.NoError(t, err, "Failed to delete backup")

	if _, err := os.Stat(backupFile); !os.IsNotExist(err) {
		assert.True(t, os.IsNotExist(err), "Backup file was not deleted")
	}
}

func TestListBackups(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "backup_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir) //nolint:errcheck

	logger := slog.Default()
	passphrase := "test-passphrase"
	backupManager := NewBackupManager(logger, tempDir, passphrase)

	backups, err := backupManager.ListBackups()
	assert.NoError(t, err, "Failed to list backups")
	assert.Equal(t, 0, len(backups))

	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	backupFile := filepath.Join(tempDir, "backup_20230101_120000.enc")
	err = os.WriteFile(backupFile, []byte("encrypted backup data"), 0600)
	require.NoError(t, err)

	backups, err = backupManager.ListBackups()
	assert.NoError(t, err, "Failed to list backups")
	assert.Equal(t, 1, len(backups))
	assert.Equal(t, backupFile, backups[0])
}

func TestEncryptionDecryption(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "backup_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir) //nolint:errcheck

	logger := slog.Default()
	passphrase := "test-passphrase"
	backupManager := NewBackupManager(logger, tempDir, passphrase)

	testData := []byte("This is test data to encrypt and decrypt")

	encryptedData, err := backupManager.encryptData(testData)
	require.NoError(t, err, "Failed to encrypt data")

	assert.Greater(t, len(encryptedData), len(testData), "Encrypted data should be longer than original data")

	decryptedData, err := backupManager.decryptData(encryptedData)
	require.NoError(t, err, "Failed to decrypt data")

	assert.Equal(t, string(testData), string(decryptedData), "Decrypted data does not match original")
}

func TestChecksum(t *testing.T) {
	logger := slog.Default()
	passphrase := "test-passphrase"
	backupManager := NewBackupManager(logger, "/tmp", passphrase)

	testData := []byte("This is test data for checksum")

	checksum := backupManager.calculateChecksum(testData)
	assert.NotEmpty(t, checksum, "Checksum should not be empty")

	assert.True(t, backupManager.verifyChecksum(testData, checksum), "Checksum verification should pass for correct data")

	wrongData := []byte("This is wrong data")
	assert.False(t, backupManager.verifyChecksum(wrongData, checksum), "Checksum verification should fail for incorrect data")
}
