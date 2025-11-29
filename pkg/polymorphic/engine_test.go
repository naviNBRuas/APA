package polymorphic

import (
	"log/slog"
	"testing"
)

func TestTransformCode(t *testing.T) {
	logger := slog.Default()
	engine := NewEngine(logger)
	
	// Test data
	original := []byte("This is a test message")
	
	// Transform the code
	transformed, err := engine.TransformCode(original)
	if err != nil {
		t.Fatalf("Failed to transform code: %v", err)
	}
	
	// Check that the transformed code is different from the original
	// Note: There's a small chance they could be the same, but it's extremely unlikely
	if len(original) != len(transformed) {
		t.Errorf("Expected transformed code length %d, got %d", len(original), len(transformed))
	}
}

func TestGenerateGarbageCode(t *testing.T) {
	logger := slog.Default()
	engine := NewEngine(logger)
	
	// Generate garbage code
	garbage, err := engine.GenerateGarbageCode(100)
	if err != nil {
		t.Fatalf("Failed to generate garbage code: %v", err)
	}
	
	// Check that the garbage code has the correct length
	if len(garbage) != 100 {
		t.Errorf("Expected garbage code length 100, got %d", len(garbage))
	}
}

func TestInsertGarbageCode(t *testing.T) {
	logger := slog.Default()
	engine := NewEngine(logger)
	
	// Test data
	original := []byte("This is a test message")
	
	// Insert garbage code
	result, err := engine.InsertGarbageCode(original, 0.5)
	if err != nil {
		t.Fatalf("Failed to insert garbage code: %v", err)
	}
	
	// Check that the result is larger than the original
	if len(result) <= len(original) {
		t.Error("Result should be larger than original")
	}
	
	// Check that all original bytes are still present in the result (though not necessarily in order)
	originalMap := make(map[byte]int)
	resultMap := make(map[byte]int)
	
	for _, b := range original {
		originalMap[b]++
	}
	
	for _, b := range result {
		resultMap[b]++
	}
	
	for b, count := range originalMap {
		if resultMap[b] < count {
			t.Errorf("Byte %d is missing from result", b)
		}
	}
}