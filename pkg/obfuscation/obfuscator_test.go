package obfuscation

import (
	"log/slog"
	"testing"
)

func TestObfuscate(t *testing.T) {
	logger := slog.Default()
	obfuscator := NewObfuscator(logger)
	
	// Test data
	original := []byte("This is a test message for obfuscation")
	
	// Obfuscate the code
	obfuscated, err := obfuscator.Obfuscate(original)
	if err != nil {
		t.Fatalf("Failed to obfuscate code: %v", err)
	}
	
	// Check that the obfuscated code has the correct length
	// We expect it to be larger due to the dummy bytes we insert
	if len(obfuscated) <= len(original) {
		t.Errorf("Expected obfuscated code to be larger than original, got %d <= %d", len(obfuscated), len(original))
	}
}

func TestAntiAnalysis(t *testing.T) {
	logger := slog.Default()
	antiAnalysis := NewAntiAnalysis(logger)
	
	// Test debugger detection
	debuggerDetected := antiAnalysis.DetectDebugger()
	if debuggerDetected {
		t.Error("Debugger detection should return false in test environment")
	}
	
	// Test sandbox detection
	sandboxDetected := antiAnalysis.DetectSandbox()
	if sandboxDetected {
		t.Error("Sandbox detection should return false in test environment")
	}
}

func TestAntiTampering(t *testing.T) {
	logger := slog.Default()
	antiTampering := NewAntiTampering(logger)
	
	// Test integrity verification
	code := []byte("This is a test message")
	integrityOK := antiTampering.VerifyIntegrity(code)
	if !integrityOK {
		t.Error("Integrity verification should return true in test environment")
	}
	
	// Test memory protection
	err := antiTampering.ProtectMemory()
	if err != nil {
		t.Errorf("Memory protection should not fail in test environment: %v", err)
	}
}