package driver

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestDriverManager tests the driver manager functionality
func TestDriverManager(t *testing.T) {
	// Create a temporary directory for drivers
	tempDir, err := os.MkdirTemp("", "driver_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a logger
	logger := slog.Default()

	// Create a driver manager
	manager := NewManager(logger, tempDir)

	// Test adding a trusted key
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	manager.AddTrustedKey("test-key", publicKey)

	// Create a test driver binary
	driverContent := []byte("#!/bin/sh\necho 'Test Driver'")
	driverHash := sha256.Sum256(driverContent)
	driverHashHex := hex.EncodeToString(driverHash[:])

	// Create a signature
	signature := ed25519.Sign(privateKey, driverContent)
	signatureHex := hex.EncodeToString(signature)

	// Create a test server to serve the manifest and driver
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/manifest.json" {
			// Create manifest with correct URLs
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
			if err != nil {
				t.Fatal(err)
			}
			
			w.Header().Set("Content-Type", "application/json")
			w.Write(manifestJSON)
		} else if r.URL.Path == "/test-driver" {
			w.Write(driverContent)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	// Test fetching and verifying a driver
	ctx := context.Background()
	fetchedManifest, driverBytes, err := manager.FetchAndVerify(ctx, server.URL+"/manifest.json")
	if err != nil {
		t.Fatalf("Failed to fetch and verify driver: %v", err)
	}

	if fetchedManifest.Name != "test-driver" {
		t.Errorf("Expected driver name 'test-driver', got '%s'", fetchedManifest.Name)
	}

	if len(driverBytes) != len(driverContent) {
		t.Errorf("Expected driver content length %d, got %d", len(driverContent), len(driverBytes))
	}

	// Test listing drivers (should be empty since we haven't loaded any)
	drivers := manager.ListDrivers()
	if len(drivers) != 0 {
		t.Errorf("Expected 0 drivers, got %d", len(drivers))
	}

	// Test creating a base driver
	baseDriver := NewBaseDriver(manager, "test-driver", "1.0.0", "test", "A test driver")
	if baseDriver.Name() != "test-driver" {
		t.Errorf("Expected driver name 'test-driver', got '%s'", baseDriver.Name())
	}

	if baseDriver.Version() != "1.0.0" {
		t.Errorf("Expected driver version '1.0.0', got '%s'", baseDriver.Version())
	}

	if baseDriver.Type() != "test" {
		t.Errorf("Expected driver type 'test', got '%s'", baseDriver.Type())
	}

	if baseDriver.Description() != "A test driver" {
		t.Errorf("Expected driver description 'A test driver', got '%s'", baseDriver.Description())
	}

	// Test loading a driver
	manager.LoadDriver(baseDriver)

	// Test listing drivers (should now have 1)
	drivers = manager.ListDrivers()
	if len(drivers) != 1 {
		t.Errorf("Expected 1 driver, got %d", len(drivers))
	}

	// Test getting a driver
	driver, exists := manager.GetDriver("test-driver")
	if !exists {
		t.Error("Expected driver to exist")
	}

	if driver.Name() != "test-driver" {
		t.Errorf("Expected driver name 'test-driver', got '%s'", driver.Name())
	}

	// Test unloading a driver
	err = manager.UnloadDriver("test-driver")
	if err != nil {
		t.Errorf("Failed to unload driver: %v", err)
	}

	// Test listing drivers (should be empty again)
	drivers = manager.ListDrivers()
	if len(drivers) != 0 {
		t.Errorf("Expected 0 drivers, got %d", len(drivers))
	}
}

// TestDriverManagerInvalidSignature tests signature verification failure
func TestDriverManagerInvalidSignature(t *testing.T) {
	// Create a temporary directory for drivers
	tempDir, err := os.MkdirTemp("", "driver_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a logger
	logger := slog.Default()

	// Create a driver manager
	manager := NewManager(logger, tempDir)

	// Create a test driver binary
	driverContent := []byte("#!/bin/sh\necho 'Test Driver'")
	driverHash := sha256.Sum256(driverContent)
	driverHashHex := hex.EncodeToString(driverHash[:])

	// Create an invalid signature (signed with a different key)
	_, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	signature := ed25519.Sign(privateKey, driverContent)
	signatureHex := hex.EncodeToString(signature)

	// Create a test server to serve the manifest and driver
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/manifest.json" {
			// Create manifest with correct URLs
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
			if err != nil {
				t.Fatal(err)
			}
			
			w.Header().Set("Content-Type", "application/json")
			w.Write(manifestJSON)
		} else if r.URL.Path == "/test-driver" {
			w.Write(driverContent)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	// Add a different trusted key (so signature verification will fail)
	publicKey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	manager.AddTrustedKey("test-key", publicKey)

	// Test fetching and verifying a driver with invalid signature
	ctx := context.Background()
	_, _, err = manager.FetchAndVerify(ctx, server.URL+"/manifest.json")
	if err == nil {
		t.Error("Expected signature verification to fail, but it succeeded")
	}

	if fmt.Sprintf("%v", err) != "signature verification failed: no valid signatures found" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}