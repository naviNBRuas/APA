package networking

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// TaskEnvelope represents a hop-by-hop store-and-forward instruction.
type TaskEnvelope struct {
	ID        string
	Payload   []byte
	ExpiresAt time.Time
	Hops      int
	MaxHops   int
}

// StoreAndForward manages delay-tolerant, hop-by-hop task distribution.
type StoreAndForward struct {
	logger   *slog.Logger
	maxCache int
	ttl      time.Duration

	decider ForwardDecider

	mu    sync.Mutex
	tasks map[string]*TaskEnvelope
	seen  map[string]map[peer.ID]struct{} // taskID -> peers that have seen it
}

// ForwardDecider can veto forwarding based on local policies (reputation, budget, network conditions).
type ForwardDecider interface {
	AllowForward(target peer.ID, payloadBytes int) bool
}

// NewStoreAndForward creates a store-and-forward manager.
func NewStoreAndForward(logger *slog.Logger, maxCache int, ttl time.Duration) *StoreAndForward {
	if maxCache <= 0 {
		maxCache = 512
	}
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	return &StoreAndForward{
		logger:   logger,
		maxCache: maxCache,
		ttl:      ttl,
		tasks:    make(map[string]*TaskEnvelope),
		seen:     make(map[string]map[peer.ID]struct{}),
	}
}

// SetDecider installs a forwarding decider. If nil, all forwards are allowed.
func (s *StoreAndForward) SetDecider(decider ForwardDecider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.decider = decider
}

// Enqueue creates a new local task envelope and stores it for forwarding.
func (s *StoreAndForward) Enqueue(payload []byte, maxHops int) TaskEnvelope {
	if maxHops <= 0 {
		maxHops = 8
	}
	id := s.newID()
	env := TaskEnvelope{
		ID:        id,
		Payload:   append([]byte(nil), payload...),
		ExpiresAt: time.Now().Add(s.ttl),
		Hops:      0,
		MaxHops:   maxHops,
	}
	s.mu.Lock()
	s.evictIfNeeded()
	s.tasks[id] = &env
	s.mu.Unlock()
	return env
}

// Receive stores a task received from another peer if valid and not expired.
func (s *StoreAndForward) Receive(env TaskEnvelope, from peer.ID) {
	if time.Now().After(env.ExpiresAt) {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.tasks[env.ID]; ok {
		// Merge hop count to keep the higher value, but preserve earliest expiry.
		if env.Hops > existing.Hops {
			existing.Hops = env.Hops
		}
		if env.ExpiresAt.Before(existing.ExpiresAt) {
			existing.ExpiresAt = env.ExpiresAt
		}
	} else {
		s.evictIfNeeded()
		copyEnv := env
		copyEnv.Payload = append([]byte(nil), env.Payload...)
		s.tasks[env.ID] = &copyEnv
	}

	if from != "" {
		s.markSeen(env.ID, from)
	}
}

// NextForPeer returns up to limit envelopes not yet seen by the target peer and within hop limits.
// It increments hop counts in the returned copies and marks the peer as having seen them.
func (s *StoreAndForward) NextForPeer(target peer.ID, limit int) []TaskEnvelope {
	if limit <= 0 {
		limit = 10
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.pruneLocked()

	results := make([]TaskEnvelope, 0, limit)
	for _, env := range s.tasks {
		if len(results) >= limit {
			break
		}
		if s.hasSeen(env.ID, target) {
			continue
		}
		if env.Hops >= env.MaxHops {
			continue
		}

		if s.decider != nil {
			if !s.decider.AllowForward(target, len(env.Payload)) {
				continue
			}
		}

		copyEnv := *env
		copyEnv.Hops++
		results = append(results, copyEnv)
		s.markSeen(env.ID, target)
	}
	return results
}

// MarkDelivered removes a task once confidently delivered end-to-end.
func (s *StoreAndForward) MarkDelivered(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tasks, id)
	delete(s.seen, id)
}

// newID creates a pseudorandom stable-looking identifier.
func (s *StoreAndForward) newID() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		now := time.Now().UnixNano()
		for i := range buf {
			buf[i] = byte(now >> (uint(i) % 8))
		}
	}
	sum := sha256.Sum256(buf)
	return hex.EncodeToString(sum[:])
}

func (s *StoreAndForward) markSeen(id string, pid peer.ID) {
	if pid == "" {
		return
	}
	m, ok := s.seen[id]
	if !ok {
		m = make(map[peer.ID]struct{})
		s.seen[id] = m
	}
	m[pid] = struct{}{}
}

func (s *StoreAndForward) hasSeen(id string, pid peer.ID) bool {
	m, ok := s.seen[id]
	if !ok {
		return false
	}
	_, seen := m[pid]
	return seen
}

func (s *StoreAndForward) pruneLocked() {
	now := time.Now()
	for id, env := range s.tasks {
		if now.After(env.ExpiresAt) {
			delete(s.tasks, id)
			delete(s.seen, id)
		}
	}
}

func (s *StoreAndForward) evictIfNeeded() {
	if len(s.tasks) < s.maxCache {
		return
	}
	s.pruneLocked()
	// If still over limit, evict oldest expiry first.
	if len(s.tasks) < s.maxCache {
		return
	}
	var oldestID string
	var oldestExp time.Time
	first := true
	for id, env := range s.tasks {
		if first || env.ExpiresAt.Before(oldestExp) {
			oldestID = id
			oldestExp = env.ExpiresAt
			first = false
		}
	}
	if oldestID != "" {
		delete(s.tasks, oldestID)
		delete(s.seen, oldestID)
		s.logger.Warn("store-and-forward cache evicted oldest task", "task_id", oldestID)
	}
}
