package networking

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"
)

type stubResolver struct {
	successes map[string][]string
}

func (s *stubResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	if addrs, ok := s.successes[host]; ok {
		return addrs, nil
	}
	return nil, errors.New("no record")
}

func dgaLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestDeterministicCandidatesStable(t *testing.T) {
	namer := NewDeterministicNamer(DGAConfig{Seed: "unit-test", Count: 3})
	ts := time.Date(2026, 1, 9, 15, 4, 0, 0, time.UTC)
	entropy := []byte{0x01, 0x02, 0x03}

	a := namer.Candidates(ts, entropy)
	b := namer.Candidates(ts, entropy)

	if len(a) != len(b) || len(a) != 3 {
		t.Fatalf("unexpected candidate count: a=%d b=%d", len(a), len(b))
	}

	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("candidates not deterministic at %d: %s vs %s", i, a[i], b[i])
		}
	}
}

func TestDeterministicCandidatesVaryAcrossEpochs(t *testing.T) {
	namer := NewDeterministicNamer(DGAConfig{Seed: "unit-test", Count: 2, Epoch: 1 * time.Hour})
	entropy := []byte{0x0a, 0x0b}

	t1 := time.Date(2026, 1, 9, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 9, 11, 0, 0, 0, time.UTC)

	a := namer.Candidates(t1, entropy)
	b := namer.Candidates(t2, entropy)

	if a[0] == b[0] && a[1] == b[1] {
		t.Fatalf("expected candidates to change across epochs")
	}
}

func TestResolveFirstTriesMultipleCandidates(t *testing.T) {
	namer := NewDeterministicNamer(DGAConfig{Seed: "unit-test", Count: 3})
	ts := time.Date(2026, 1, 9, 15, 0, 0, 0, time.UTC)
	entropy := []byte{0xAA, 0xBB}
	names := namer.Candidates(ts, entropy)

	// Only the second candidate will resolve
	resolver := &stubResolver{successes: map[string][]string{
		names[1]: {"203.0.113.10"},
	}}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	gotName, addrs, err := ResolveFirst(ctx, resolver, names)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if gotName != names[1] {
		t.Fatalf("expected second candidate to succeed, got %s", gotName)
	}
	if len(addrs) != 1 || addrs[0] != "203.0.113.10" {
		t.Fatalf("unexpected addresses: %v", addrs)
	}
}

func TestBinaryPivotNonZero(t *testing.T) {
	var zero [32]byte
	if v := binaryPivot(zero); v == 0 {
		t.Fatalf("expected non-zero pivot")
	}
}
