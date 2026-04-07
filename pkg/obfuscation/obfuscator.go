package obfuscation

import (
	"bufio"
	"bytes"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	mrand "math/rand"
	"os"
	"runtime"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

// Obfuscator provides code obfuscation capabilities
type Obfuscator struct {
	logger *slog.Logger
}

// NewObfuscator creates a new obfuscator
func NewObfuscator(logger *slog.Logger) *Obfuscator {
	return &Obfuscator{
		logger: logger,
	}
}

// Obfuscate applies multiple obfuscation techniques to the code
func (o *Obfuscator) Obfuscate(code []byte) ([]byte, error) {
	// Apply multiple obfuscation techniques
	obfuscated := make([]byte, len(code))
	copy(obfuscated, code)

	// 1. Control flow obfuscation (inserting dummy branches / NOP sleds)
	obfuscated, err := o.obfuscateControlFlow(obfuscated)
	if err != nil {
		return nil, fmt.Errorf("failed to obfuscate control flow: %w", err)
	}

	// 2. String encryption (keyed XOR with embedded key header)
	obfuscated, err = o.encryptStrings(obfuscated)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt strings: %w", err)
	}

	// 3. Instruction substitution (byte-level substitutions)
	obfuscated, err = o.substituteInstructions(obfuscated)
	if err != nil {
		return nil, fmt.Errorf("failed to substitute instructions: %w", err)
	}

	o.logger.Info("Applied code obfuscation", "original_size", len(code), "obfuscated_size", len(obfuscated))
	return obfuscated, nil
}

// obfuscateControlFlow inserts dummy control flow branches
func (o *Obfuscator) obfuscateControlFlow(code []byte) ([]byte, error) {
	if len(code) == 0 {
		return code, nil
	}
	mrand.Seed(time.Now().UnixNano())

	// Insert NOP-sled style sequences and opaque predicate markers.
	insertions := len(code) / 8
	dummySeq := []byte{0x90, 0x66, 0x90, 0x0F, 0x1F, 0x00} // mixed NOPs and filler
	result := make([]byte, len(code))
	copy(result, code)

	for i := 0; i < insertions; i++ {
		pos := mrand.Intn(len(result) + 1)
		seq := make([]byte, len(dummySeq))
		copy(seq, dummySeq)
		// Add a simple opaque predicate marker byte
		seq[mrand.Intn(len(seq))] ^= byte(mrand.Intn(255))
		result = append(result[:pos], append(seq, result[pos:]...)...)
	}

	return result, nil
}

// encryptStrings encrypts string literals in the code
func (o *Obfuscator) encryptStrings(code []byte) ([]byte, error) {
	key := make([]byte, 8)
	if _, err := crand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate encryption key: %w", err)
	}

	result := make([]byte, 0, len(code)+len(key)+2)
	// Embed key length and key so the transformation can be reversed downstream.
	result = append(result, byte(len(key)))
	result = append(result, key...)

	for i, b := range code {
		result = append(result, b^key[i%len(key)])
	}

	return result, nil
}

// substituteInstructions substitutes instructions with equivalent ones
func (o *Obfuscator) substituteInstructions(code []byte) ([]byte, error) {
	if len(code) == 0 {
		return code, nil
	}

	// Perform reversible nibble substitution to break common byte patterns.
	subst := func(b byte) byte {
		high := (b & 0xF0) >> 4
		low := b & 0x0F
		// simple reversible swap/rotate
		high = ((high << 1) | (high >> 3)) & 0x0F
		low = ((low >> 1) | (low << 3)) & 0x0F
		return (high << 4) | low
	}

	result := make([]byte, len(code))
	for i, b := range code {
		if i%7 == 0 {
			result[i] = ^b // periodic inversion for entropy
			continue
		}
		result[i] = subst(b)
	}
	return result, nil
}

// AntiAnalysis provides anti-analysis capabilities
type AntiAnalysis struct {
	logger *slog.Logger
}

// NewAntiAnalysis creates a new anti-analysis component
func NewAntiAnalysis(logger *slog.Logger) *AntiAnalysis {
	return &AntiAnalysis{
		logger: logger,
	}
}

// DetectDebugger checks for the presence of a debugger
func (a *AntiAnalysis) DetectDebugger() bool {
	// Heuristics: TracerPid in /proc, well-known env vars, parent cmdline hints
	if runtime.GOOS == "windows" {
		return false
	}

	if tracer, err := readTracerPID("/proc/self/status"); err == nil && tracer != 0 {
		a.logger.Warn("Debugger detected via TracerPid", "pid", tracer)
		return true
	}

	for _, key := range []string{"LD_PRELOAD", "GODEBUG", "DEBUG"} {
		if v := os.Getenv(key); v != "" {
			a.logger.Warn("Debugger/analysis env var detected", "key", key)
			return true
		}
	}

	if cmdline, err := os.ReadFile("/proc/self/cmdline"); err == nil {
		cl := strings.ToLower(string(cmdline))
		if strings.Contains(cl, "dlv") || strings.Contains(cl, "gdb") || strings.Contains(cl, "lldb") {
			a.logger.Warn("Debugger signature in cmdline")
			return true
		}
	}

	return false
}

// DetectSandbox checks for the presence of a sandbox
func (a *AntiAnalysis) DetectSandbox() bool {
	if runtime.GOOS != "linux" {
		return false
	}

	// Check cgroup signatures for containers
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		lc := strings.ToLower(string(data))
		if strings.Contains(lc, "docker") || strings.Contains(lc, "kubepods") || strings.Contains(lc, "lxc") {
			a.logger.Warn("Sandbox/container indicators in cgroup")
			return true
		}
	}

	// Check common virtualization product names
	if data, err := os.ReadFile("/sys/class/dmi/id/product_name"); err == nil {
		lc := strings.ToLower(string(data))
		if strings.Contains(lc, "virtual") || strings.Contains(lc, "kvm") || strings.Contains(lc, "vmware") {
			a.logger.Warn("Sandbox indicator from DMI product name", "product", strings.TrimSpace(lc))
			return true
		}
	}

	// Minimal memory footprint (<2GB) can hint sandbox
	if mem, err := readMemTotal("/proc/meminfo"); err == nil && mem < 2*1024*1024*1024 {
		a.logger.Warn("Low memory environment detected", "bytes", mem)
		return true
	}

	return false
}

// AntiTampering provides anti-tampering capabilities
type AntiTampering struct {
	logger       *slog.Logger
	baselineHash []byte
}

// NewAntiTampering creates a new anti-tampering component
func NewAntiTampering(logger *slog.Logger) *AntiTampering {
	return &AntiTampering{
		logger: logger,
	}
}

// VerifyIntegrity verifies the integrity of the code
func (a *AntiTampering) VerifyIntegrity(code []byte) bool {
	hash := sha256.Sum256(code)

	// If no baseline set, record and trust first known-good hash
	if len(a.baselineHash) == 0 {
		a.baselineHash = hash[:]
		a.logger.Info("Anti-tampering baseline established", "hash", hex.EncodeToString(a.baselineHash))
		return true
	}

	match := bytes.Equal(hash[:], a.baselineHash)
	if !match {
		a.logger.Error("Code integrity check failed", "expected", hex.EncodeToString(a.baselineHash), "actual", hex.EncodeToString(hash[:]))
	}
	return match
}

// ProtectMemory protects memory from unauthorized access
func (a *AntiTampering) ProtectMemory() error {
	// Best-effort mlockall on supported Unix platforms to reduce swapping and scraping.
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		if err := unix.Mlockall(unix.MCL_CURRENT | unix.MCL_FUTURE); err != nil {
			a.logger.Warn("mlockall failed", "error", err)
			// Do not treat this as fatal in environments where locking is restricted.
			return nil
		}
		return nil
	}

	return nil
}

// SetBaselineDigest sets an expected hash for integrity checks.
func (a *AntiTampering) SetBaselineDigest(hash []byte) {
	a.baselineHash = make([]byte, len(hash))
	copy(a.baselineHash, hash)
}

// Utility functions

func readTracerPID(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		if strings.HasPrefix(line, "TracerPid:") {
			var pid int
			_, err := fmt.Sscanf(line, "TracerPid:\t%d", &pid)
			return pid, err
		}
	}
	return 0, errors.New("TracerPid not found")
}

func readMemTotal(path string) (uint64, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var value uint64
	_, err = fmt.Sscanf(string(b), "MemTotal: %d kB", &value)
	if err != nil {
		return 0, err
	}
	return value * 1024, nil
}
