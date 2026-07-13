package networking

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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

	require.Equal(t, len(a), len(b), "candidate length mismatch")
	require.Equal(t, 3, len(a), "unexpected candidate count: a=%d b=%d", len(a), len(b))

	for i := range a {
		require.Equal(t, b[i], a[i], "candidates not deterministic at %d: %s vs %s", i, a[i], b[i])
	}
}

func TestDeterministicCandidatesVaryAcrossEpochs(t *testing.T) {
	namer := NewDeterministicNamer(DGAConfig{Seed: "unit-test", Count: 2, Epoch: 1 * time.Hour})
	entropy := []byte{0x0a, 0x0b}

	t1 := time.Date(2026, 1, 9, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 9, 11, 0, 0, 0, time.UTC)

	a := namer.Candidates(t1, entropy)
	b := namer.Candidates(t2, entropy)

	require.False(t, a[0] == b[0] && a[1] == b[1], "expected candidates to change across epochs")
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
	require.NoError(t, err, "expected success, got error: %v", err)
	require.Equal(t, names[1], gotName, "expected second candidate to succeed, got %s", gotName)
	require.Len(t, addrs, 1, "unexpected addresses: %v", addrs)
	require.Equal(t, "203.0.113.10", addrs[0], "unexpected address")
}

func TestBinaryPivotNonZero(t *testing.T) {
	var zero [32]byte
	require.NotZero(t, binaryPivot(zero), "expected non-zero pivot")
}
