package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"
)

// AuditEntry is a single append-only audit record with hash chaining.
type AuditEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Actor     string                 `json:"actor"`
	Action    string                 `json:"action"`
	Path      string                 `json:"path"`
	Method    string                 `json:"method"`
	PeerID    string                 `json:"peer_id,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	PrevHash  string                 `json:"prev_hash"`
	Hash      string                 `json:"hash"`
}

// AuditLogger writes chained JSONL audit records to a file.
type AuditLogger struct {
	logger   *slog.Logger
	path     string
	mu       sync.Mutex
	lastHash string
}

func NewAuditLogger(logger *slog.Logger, path string) *AuditLogger {
	return &AuditLogger{logger: logger, path: path}
}

// Append writes a new audit entry, chaining from the previous hash.
func (al *AuditLogger) Append(entry AuditEntry) error {
	al.mu.Lock()
	defer al.mu.Unlock()

	entry.Timestamp = time.Now().UTC()
	entry.PrevHash = al.lastHash

	h := sha256.New()
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal audit entry: %w", err)
	}
	h.Write(data)
	entry.Hash = hex.EncodeToString(h.Sum(nil))
	if al.lastHash == "" {
		entry.PrevHash = entry.Hash // genesis links to self
	}

	data, err = json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal audit entry with hash: %w", err)
	}

	f, err := os.OpenFile(al.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open audit log: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write audit entry: %w", err)
	}

	al.lastHash = entry.Hash
	al.logger.Debug("Audit entry appended", "action", entry.Action, "hash", entry.Hash)
	return nil
}
