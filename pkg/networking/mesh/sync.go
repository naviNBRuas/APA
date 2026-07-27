package mesh

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// SyncEngine handles data synchronisation across the mesh (code, state, updates).
type SyncEngine struct {
	logger     *slog.Logger
	nodeID     string
	interval   time.Duration
	mu         sync.RWMutex
	codeStore  map[string]*CodeArtifact
	stateStore map[string]*StateEntry
	published  map[string]time.Time
	running    bool
	eventCh    chan SyncEvent
	ctx        context.Context
	cancel     context.CancelFunc
}

// CodeArtifact represents distributable code/data in the mesh.
type CodeArtifact struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Hash      string `json:"hash"`
	Data      []byte `json:"-"`
	Size      int    `json:"size"`
	Publisher string `json:"publisher"`
	Timestamp int64  `json:"timestamp"`
	Signature []byte `json:"signature,omitempty"`
}

// StateEntry represents a key-value state update.
type StateEntry struct {
	Key       string `json:"key"`
	Value     []byte `json:"value"`
	Version   int64  `json:"version"`
	Publisher string `json:"publisher"`
	Timestamp int64  `json:"timestamp"`
}

// SyncEvent is emitted when sync operations complete.
type SyncEvent struct {
	Type   string
	Name   string
	Error  error
}

const (
	SyncEventCodeReceived  = "code_received"
	SyncEventStateUpdated  = "state_updated"
	SyncEventSyncCompleted = "sync_completed"
	SyncEventError         = "sync_error"
)

// NewSyncEngine creates a new sync engine.
func NewSyncEngine(logger *slog.Logger, nodeID string, interval time.Duration) *SyncEngine {
	ctx, cancel := context.WithCancel(context.Background())
	return &SyncEngine{
		logger:     logger,
		nodeID:     nodeID,
		interval:   interval,
		codeStore:  make(map[string]*CodeArtifact),
		stateStore: make(map[string]*StateEntry),
		published:  make(map[string]time.Time),
		ctx:        ctx,
		cancel:     cancel,
		eventCh:    make(chan SyncEvent, 64),
	}
}

// Start begins the sync loop.
func (se *SyncEngine) Start(ctx context.Context) {
	se.mu.Lock()
	defer se.mu.Unlock()
	if se.running {
		return
	}
	se.running = true
	se.logger.Info("Sync engine started", "interval", se.interval)
}

// Stop stops the sync engine.
func (se *SyncEngine) Stop() {
	se.mu.Lock()
	defer se.mu.Unlock()
	se.running = false
	se.cancel()
}

// PublishCode publishes a code artifact to the mesh.
func (se *SyncEngine) PublishCode(ctx context.Context, name, version string, data []byte) error {
	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])

	artifact := &CodeArtifact{
		Name:      name,
		Version:   version,
		Hash:      hashStr,
		Data:      data,
		Size:      len(data),
		Publisher: se.nodeID,
		Timestamp: time.Now().UTC().UnixNano(),
	}

	key := codeKey(name, version)

	se.mu.Lock()
	se.codeStore[key] = artifact
	se.published[key] = time.Now()
	se.mu.Unlock()

	se.logger.Info("Code published to mesh",
		"name", name,
		"version", version,
		"size", len(data),
		"hash", truncateID(hashStr),
	)
	return nil
}

// PublishState publishes a state update to the mesh.
func (se *SyncEngine) PublishState(ctx context.Context, key string, value []byte) error {
	entry := &StateEntry{
		Key:       key,
		Value:     value,
		Version:   time.Now().UTC().UnixNano(),
		Publisher: se.nodeID,
		Timestamp: time.Now().UTC().UnixNano(),
	}

	se.mu.Lock()
	se.stateStore[key] = entry
	se.mu.Unlock()

	return nil
}

// ReceiveCode handles an incoming code artifact from a peer.
func (se *SyncEngine) ReceiveCode(artifact *CodeArtifact) error {
	if artifact == nil {
		return fmt.Errorf("nil artifact")
	}
	key := codeKey(artifact.Name, artifact.Version)

	se.mu.Lock()
	existing, has := se.codeStore[key]
	if has && existing.Timestamp >= artifact.Timestamp {
		se.mu.Unlock()
		return nil
	}
	se.codeStore[key] = artifact
	se.mu.Unlock()

	se.emit(SyncEvent{Type: SyncEventCodeReceived, Name: artifact.Name})
	publisher := artifact.Publisher
	if len(publisher) > 12 {
		publisher = publisher[:12]
	}
	se.logger.Info("Code received from mesh",
		"name", artifact.Name,
		"version", artifact.Version,
		"publisher", publisher,
		"size", artifact.Size,
	)
	return nil
}

// ReceiveState handles an incoming state update from a peer.
func (se *SyncEngine) ReceiveState(entry *StateEntry) error {
	if entry == nil {
		return fmt.Errorf("nil state entry")
	}

	se.mu.Lock()
	existing, has := se.stateStore[entry.Key]
	if has && existing.Version >= entry.Version {
		se.mu.Unlock()
		return nil
	}
	se.stateStore[entry.Key] = entry
	se.mu.Unlock()

	se.emit(SyncEvent{Type: SyncEventStateUpdated, Name: entry.Key})
	return nil
}

// GetCode retrieves a code artifact by name and version.
func (se *SyncEngine) GetCode(name, version string) *CodeArtifact {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return se.codeStore[codeKey(name, version)]
}

// GetState retrieves a state entry by key.
func (se *SyncEngine) GetState(key string) *StateEntry {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return se.stateStore[key]
}

// HasCode checks if a code artifact exists.
func (se *SyncEngine) HasCode(name, version string) bool {
	se.mu.RLock()
	defer se.mu.RUnlock()
	_, ok := se.codeStore[codeKey(name, version)]
	return ok
}

// AllCode returns all code artifacts.
func (se *SyncEngine) AllCode() []*CodeArtifact {
	se.mu.RLock()
	defer se.mu.RUnlock()
	out := make([]*CodeArtifact, 0, len(se.codeStore))
	for _, v := range se.codeStore {
		out = append(out, v)
	}
	return out
}

// AllState returns all state entries.
func (se *SyncEngine) AllState() map[string]*StateEntry {
	se.mu.RLock()
	defer se.mu.RUnlock()
	out := make(map[string]*StateEntry, len(se.stateStore))
	for k, v := range se.stateStore {
		out[k] = v
	}
	return out
}

// Events returns the sync event channel.
func (se *SyncEngine) Events() <-chan SyncEvent {
	return se.eventCh
}

// Stats returns sync engine statistics.
func (se *SyncEngine) Stats() map[string]interface{} {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return map[string]interface{}{
		"code_artifacts": len(se.codeStore),
		"state_entries":  len(se.stateStore),
		"published":      len(se.published),
		"running":        se.running,
	}
}

func (se *SyncEngine) emit(ev SyncEvent) {
	select {
	case se.eventCh <- ev:
	default:
	}
}

func codeKey(name, version string) string {
	return fmt.Sprintf("%s@%s", name, version)
}

// SyncMessage wraps data for mesh sync transport.
type SyncMessage struct {
	Type     string          `json:"type"`
	Sender   string          `json:"sender"`
	Payload  json.RawMessage `json:"payload"`
}

func init() {
	_ = json.Marshal
}
