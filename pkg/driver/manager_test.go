package driver

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDriverManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "driver_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir) //nolint:errcheck

	logger := slog.Default()

	manager := NewManager(logger, tempDir)

	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	manager.AddTrustedKey("test-key", publicKey)

	driverContent := []byte("#!/bin/sh\necho 'Test Driver'")
	driverHash := sha256.Sum256(driverContent)
	driverHashHex := hex.EncodeToString(driverHash[:])

	signature := ed25519.Sign(privateKey, driverContent)
	signatureHex := hex.EncodeToString(signature)

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/manifest.json" {
			manifest := Manifest{
				Name:    "test-driver",
				Version: "1.0.0",
				Type:    "test",
				URL:     server.URL + "/test-driver",
				Hash:    driverHashHex,
				Signatures: map[string]string{
					"test-key": signatureHex,
				},
			}

			manifestJSON, err := json.Marshal(manifest)
			require.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write(manifestJSON); err != nil {
				assert.NoError(t, err, "Failed to write manifest")
			}
		} else if r.URL.Path == "/test-driver" {
			if _, err := w.Write(driverContent); err != nil {
				assert.NoError(t, err, "Failed to write driver")
			}
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	fetchedManifest, driverBytes, err := manager.FetchAndVerify(ctx, server.URL+"/manifest.json")
	require.NoError(t, err, "Failed to fetch and verify driver")

	assert.Equal(t, "test-driver", fetchedManifest.Name)
	assert.Equal(t, len(driverContent), len(driverBytes))

	drivers := manager.ListDrivers()
	assert.Equal(t, 0, len(drivers))

	baseDriver := NewBaseDriver(manager, "test-driver", "1.0.0", "test", "A test driver")
	assert.Equal(t, "test-driver", baseDriver.Name())
	assert.Equal(t, "1.0.0", baseDriver.Version())
	assert.Equal(t, "test", baseDriver.Type())
	assert.Equal(t, "A test driver", baseDriver.Description())

	manager.LoadDriver(baseDriver)

	drivers = manager.ListDrivers()
	assert.Equal(t, 1, len(drivers))

	driver, exists := manager.GetDriver("test-driver")
	assert.True(t, exists, "Expected driver to exist")
	assert.Equal(t, "test-driver", driver.Name())

	err = manager.UnloadDriver("test-driver")
	assert.NoError(t, err, "Failed to unload driver")

	drivers = manager.ListDrivers()
	assert.Equal(t, 0, len(drivers))
}

func TestDriverManagerInvalidSignature(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "driver_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir) //nolint:errcheck

	logger := slog.Default()

	manager := NewManager(logger, tempDir)

	driverContent := []byte("#!/bin/sh\necho 'Test Driver'")
	driverHash := sha256.Sum256(driverContent)
	driverHashHex := hex.EncodeToString(driverHash[:])

	_, privateKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	signature := ed25519.Sign(privateKey, driverContent)
	signatureHex := hex.EncodeToString(signature)

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/manifest.json" {
			manifest := Manifest{
				Name:    "test-driver",
				Version: "1.0.0",
				Type:    "test",
				URL:     server.URL + "/test-driver",
				Hash:    driverHashHex,
				Signatures: map[string]string{
					"test-key": signatureHex,
				},
			}

			manifestJSON, err := json.Marshal(manifest)
			require.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write(manifestJSON); err != nil {
				assert.NoError(t, err, "Failed to write manifest")
			}
		} else if r.URL.Path == "/test-driver" {
			if _, err := w.Write(driverContent); err != nil {
				assert.NoError(t, err, "Failed to write driver")
			}
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	publicKey, _, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	manager.AddTrustedKey("test-key", publicKey)

	ctx := context.Background()
	_, _, err = manager.FetchAndVerify(ctx, server.URL+"/manifest.json")
	require.Error(t, err, "Expected signature verification to fail, but it succeeded")
	assert.EqualError(t, err, "signature verification failed: no valid signatures found")
}
