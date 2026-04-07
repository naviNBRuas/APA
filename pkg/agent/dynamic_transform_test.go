package agent

import (
	"log/slog"
	"testing"

	"github.com/naviNBRuas/APA/pkg/polymorphic"
)

func TestTransformationManagerRoundTrip(t *testing.T) {
	eng := polymorphic.NewEngine(slog.Default())
	tm := NewTransformationManager(eng, slog.Default())
	original := []byte("print('hi')")
	variant1, fp1, err := tm.NextVariant(original)
	if err != nil || len(variant1) == 0 || fp1 == "" {
		t.Fatalf("variant failed: %v", err)
	}
	variant2, fp2, err := tm.NextVariant(original)
	if err != nil {
		t.Fatalf("variant2 failed: %v", err)
	}
	if fp1 == fp2 || len(variant2) == 0 {
		t.Fatalf("expected differing fingerprints")
	}
	recovered, err := tm.ReverseVariant(variant1)
	if err != nil {
		t.Fatalf("reverse failed: %v", err)
	}
	if string(recovered) != string(original) {
		t.Fatalf("recovered mismatch")
	}
}
