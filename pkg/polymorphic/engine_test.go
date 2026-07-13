package polymorphic

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransformCode(t *testing.T) {
	logger := slog.Default()
	engine := NewEngine(logger)

	// Test data
	original := []byte("This is a test message")

	transformed, err := engine.TransformCode(original)
	require.NoError(t, err, "Failed to transform code: %v", err)
	require.NotEmpty(t, transformed, "Transformed code should not be empty")
	assert.GreaterOrEqual(t, len(transformed), len(original), "Expected transformed code length to be >= %d, got %d", len(original), len(transformed))
}

func TestGenerateGarbageCode(t *testing.T) {
	logger := slog.Default()
	engine := NewEngine(logger)

	// Generate garbage code
	garbage, err := engine.GenerateGarbageCode(100)
	require.NoError(t, err, "Failed to generate garbage code: %v", err)
	assert.Equal(t, 100, len(garbage), "Expected garbage code length 100, got %d", len(garbage))
}

func TestInsertGarbageCode(t *testing.T) {
	logger := slog.Default()
	engine := NewEngine(logger)

	// Test data
	original := []byte("This is a test message")

	// Insert garbage code
	result, err := engine.InsertGarbageCode(original, 0.5)
	require.NoError(t, err, "Failed to insert garbage code: %v", err)

	assert.Greater(t, len(result), len(original), "Result should be larger than original")

	originalMap := make(map[byte]int)
	resultMap := make(map[byte]int)

	for _, b := range original {
		originalMap[b]++
	}

	for _, b := range result {
		resultMap[b]++
	}

	for b, count := range originalMap {
		assert.GreaterOrEqual(t, resultMap[b], count, "Byte %d is missing from result", b)
	}
}
