package agent

import (
	"log/slog"
	"testing"

	"github.com/naviNBRuas/APA/pkg/polymorphic"
	"github.com/stretchr/testify/require"
)

func TestTransformationManagerRoundTrip(t *testing.T) {
	eng := polymorphic.NewEngine(slog.Default())
	tm := NewTransformationManager(eng, slog.Default())
	original := []byte("print('hi')")
	variant1, fp1, err := tm.NextVariant(original)
	require.NoError(t, err, "variant failed")
	require.NotEmpty(t, variant1, "variant should have content")
	require.NotEmpty(t, fp1, "variant should have fingerprint")
	variant2, fp2, err := tm.NextVariant(original)
	require.NoError(t, err, "variant2 failed")
	require.NotEqual(t, fp1, fp2, "expected differing fingerprints")
	require.NotEmpty(t, variant2, "variant2 should have content")
	recovered, err := tm.ReverseVariant(variant1)
	require.NoError(t, err, "reverse failed")
	require.Equal(t, original, recovered, "recovered mismatch")
}
