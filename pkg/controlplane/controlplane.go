package controlplane

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"sort"
	"sync"
	"time"
)

const (
	controlTopicDefault  = "apa/controlplane/1.0.0"
	electionTopicDefault = "apa/controlplane/election/1.0.0"
)

// Config controls behaviour of the decentralized control plane.
//
// Mode options:
//   - "leaderless" (default): every node gossips updates directly.
//   - "elected": nodes perform lightweight leader election and non-leaders
//     forward update requests to the elected leader for canonical rebroadcast.
//
// Partial state is bounded by PartialStateLimit entries and entries expire after EntryTTL.
type Config struct {
	Mode              string        `yaml:"mode"`
	GossipTopic       string        `yaml:"gossip_topic"`
	ElectionTopic     string        `yaml:"election_topic"`
	PartialStateLimit int           `yaml:"partial_state_limit"`
	EntryTTL          time.Duration `yaml:"entry_ttl"`
	SyncInterval      time.Duration `yaml:"sync_interval"`
	ReplicationFactor int           `yaml:"replication_factor"`
}

// Transport is a minimal abstraction over the underlying message bus (e.g., libp2p pubsub).
type Transport interface {
	Publish(ctx context.Context, topic string, payload []byte) error
	Subscribe(ctx context.Context, topic string) (<-chan []byte, error)
	LocalID() string
}

type stateEntry struct {
	Key       string
	Value     []byte
	Version   uint64
	Owner     string
	ExpiresAt time.Time
	UpdatedAt time.Time
}

type controlMessage struct {
	Key     string    `json:"key"`
	Value   []byte    `json:"value"`
	Version uint64    `json:"version"`
	Owner   string    `json:"owner"`
	TTLMs   int64     `json:"ttl_ms"`
	SentAt  time.Time `json:"sent_at"`
	Type    string    `json:"type"` // update | request
}

type electionMessage struct {
	Candidate string    `json:"candidate"`
	Rank      int64     `json:"rank"`
	SentAt    time.Time `json:"sent_at"`
}

type ControlPlane struct {
	logger    *slog.Logger
	cfg       Config
	transport Transport

	mu         sync.RWMutex
	store      map[string]*stateEntry
	order      []string
	localVers  map[string]uint64
	isLeader   bool
	leaderRank int64
	cancel     context.CancelFunc
}

func New(logger *slog.Logger, transport Transport, cfg Config) *ControlPlane {
	if cfg.GossipTopic == "" {
		cfg.GossipTopic = controlTopicDefault
	}
	if cfg.ElectionTopic == "" {
		cfg.ElectionTopic = electionTopicDefault
	}
	if cfg.PartialStateLimit <= 0 {
		cfg.PartialStateLimit = 256
	}
	if cfg.EntryTTL <= 0 {
		cfg.EntryTTL = 5 * time.Minute
	}
	if cfg.SyncInterval <= 0 {
		cfg.SyncInterval = 30 * time.Second
	}
	if cfg.ReplicationFactor <= 0 {
		cfg.ReplicationFactor = 3
	}
	mode := cfg.Mode
	if mode == "" {
		mode = "leaderless"
		cfg.Mode = mode
	}

	cp := &ControlPlane{
		logger:    logger,
		cfg:       cfg,
		transport: transport,
		store:     make(map[string]*stateEntry),
		order:     make([]string, 0, cfg.PartialStateLimit),
		localVers: make(map[string]uint64),
	}
	// Default leader assumption for leaderless.
	cp.isLeader = mode == "leaderless"
	if mode == "elected" {
		cp.leaderRank = rand.New(rand.NewSource(time.Now().UnixNano())).Int63()
	}
	return cp
}

// SetLeaderRank overrides the randomly chosen election rank (intended for tests or tuning).
func (c *ControlPlane) SetLeaderRank(rank int64) {
	c.leaderRank = rank
}

// Start begins gossip and optional leader election.
func (c *ControlPlane) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	// Subscribe control messages
	controlCh, err := c.transport.Subscribe(ctx, c.cfg.GossipTopic)
	if err != nil {
		return fmt.Errorf("subscribe control: %w", err)
	}
	go c.consumeControl(ctx, controlCh)

	if c.cfg.Mode == "elected" {
		electCh, err := c.transport.Subscribe(ctx, c.cfg.ElectionTopic)
		if err != nil {
			return fmt.Errorf("subscribe election: %w", err)
		}
		go c.consumeElection(ctx, electCh)
		go c.electionLoop(ctx)
	}

	go c.pruneLoop(ctx)
	return nil
}

func (c *ControlPlane) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
}

// Set records a state update and gossips according to the configured mode.
func (c *ControlPlane) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = c.cfg.EntryTTL
	}
	msg := controlMessage{
		Key:    key,
		Value:  value,
		Owner:  c.transport.LocalID(),
		TTLMs:  ttl.Milliseconds(),
		SentAt: time.Now().UTC(),
		Type:   "update",
	}

	c.mu.Lock()
	version := c.localVers[key] + 1
	c.localVers[key] = version
	msg.Version = version
	c.applyLocked(msg)
	c.mu.Unlock()

	if c.cfg.Mode == "elected" && !c.isLeader {
		msg.Type = "request"
	}
	return c.broadcast(ctx, msg)
}

// Get returns the value if present and not expired.
func (c *ControlPlane) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.store[key]
	if !ok || time.Now().After(entry.ExpiresAt) {
		return nil, false
	}
	return entry.Value, true
}

// Snapshot returns a shallow copy of current non-expired entries.
type SnapshotEntry struct {
	Key     string
	Value   []byte
	Version uint64
	Owner   string
	Expires time.Time
}

func (c *ControlPlane) Snapshot() []SnapshotEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	now := time.Now()
	res := make([]SnapshotEntry, 0, len(c.store))
	for _, e := range c.store {
		if now.After(e.ExpiresAt) {
			continue
		}
		res = append(res, SnapshotEntry{Key: e.Key, Value: append([]byte(nil), e.Value...), Version: e.Version, Owner: e.Owner, Expires: e.ExpiresAt})
	}
	return res
}

func (c *ControlPlane) consumeControl(ctx context.Context, ch <-chan []byte) {
	for {
		select {
		case <-ctx.Done():
			return
		case payload, ok := <-ch:
			if !ok {
				return
			}
			var msg controlMessage
			if err := json.Unmarshal(payload, &msg); err != nil {
				c.logger.Error("controlplane: decode message", "error", err)
				continue
			}
			if msg.Key == "" {
				continue
			}
			c.handleControlMessage(ctx, msg)
		}
	}
}

func (c *ControlPlane) handleControlMessage(ctx context.Context, msg controlMessage) {
	// Leader-mode routing: requests handled by leader only
	if c.cfg.Mode == "elected" {
		if msg.Type == "request" {
			if c.isLeader {
				msg.Type = "update"
				c.handleControlMessage(ctx, msg)
			}
			return
		}
	}

	c.mu.Lock()
	applied := c.applyLocked(msg)
	c.mu.Unlock()

	if applied && msg.Owner != c.transport.LocalID() {
		// Forward to improve reach if partial state limit allows.
		_ = c.broadcast(ctx, msg)
	}
}

func (c *ControlPlane) broadcast(ctx context.Context, msg controlMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.transport.Publish(ctx, c.cfg.GossipTopic, data)
}

func (c *ControlPlane) applyLocked(msg controlMessage) bool {
	effectiveTTL := time.Duration(msg.TTLMs) * time.Millisecond
	if effectiveTTL <= 0 {
		effectiveTTL = c.cfg.EntryTTL
	}
	expires := time.Now().Add(effectiveTTL)
	existing, ok := c.store[msg.Key]
	if ok {
		if msg.Version <= existing.Version {
			return false
		}
	}

	entry := &stateEntry{
		Key:       msg.Key,
		Value:     append([]byte(nil), msg.Value...),
		Version:   msg.Version,
		Owner:     msg.Owner,
		ExpiresAt: expires,
		UpdatedAt: time.Now(),
	}
	c.store[msg.Key] = entry
	c.bumpOrder(msg.Key)
	c.enforceLimit()
	return true
}

func (c *ControlPlane) bumpOrder(key string) {
	// simple order list (remove existing then append)
	for i, k := range c.order {
		if k == key {
			c.order = append(c.order[:i], c.order[i+1:]...)
			break
		}
	}
	c.order = append(c.order, key)
}

func (c *ControlPlane) enforceLimit() {
	if len(c.store) <= c.cfg.PartialStateLimit {
		return
	}
	// remove oldest by UpdatedAt
	type kv struct {
		key string
		at  time.Time
	}
	items := make([]kv, 0, len(c.store))
	for k, v := range c.store {
		items = append(items, kv{key: k, at: v.UpdatedAt})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].at.Before(items[j].at) })
	overflow := len(c.store) - c.cfg.PartialStateLimit
	for i := 0; i < overflow; i++ {
		delete(c.store, items[i].key)
	}
}

func (c *ControlPlane) pruneLoop(ctx context.Context) {
	ticker := time.NewTicker(c.cfg.EntryTTL / 2)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			c.mu.Lock()
			for k, v := range c.store {
				if now.After(v.ExpiresAt) {
					delete(c.store, k)
				}
			}
			c.mu.Unlock()
		}
	}
}

func (c *ControlPlane) electionLoop(ctx context.Context) {
	ticker := time.NewTicker(c.cfg.SyncInterval)
	defer ticker.Stop()
	self := c.transport.LocalID()
	// announce immediately
	_ = c.publishElection(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = c.publishElection(ctx)
			// log under lock to avoid races
			c.mu.RLock()
			leader := c.isLeader
			rank := c.leaderRank
			c.mu.RUnlock()
			c.logger.Debug("controlplane election status", "leader", leader, "self", self, "rank", rank)
		}
	}
}

func (c *ControlPlane) publishElection(ctx context.Context) error {
	c.mu.RLock()
	rank := c.leaderRank
	c.mu.RUnlock()
	msg := electionMessage{Candidate: c.transport.LocalID(), Rank: rank, SentAt: time.Now().UTC()}
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.transport.Publish(ctx, c.cfg.ElectionTopic, payload)
}

func (c *ControlPlane) consumeElection(ctx context.Context, ch <-chan []byte) {
	bestCandidate := c.transport.LocalID()
	c.mu.RLock()
	bestRank := c.leaderRank
	c.mu.RUnlock()
	// assume leadership until a superior candidate is seen
	c.mu.Lock()
	c.isLeader = true
	c.mu.Unlock()
	for {
		select {
		case <-ctx.Done():
			return
		case payload, ok := <-ch:
			if !ok {
				return
			}
			var msg electionMessage
			if err := json.Unmarshal(payload, &msg); err != nil {
				c.logger.Error("controlplane: decode election", "error", err)
				continue
			}
			if msg.Candidate == "" {
				continue
			}
			if msg.Rank > bestRank || (msg.Rank == bestRank && msg.Candidate < bestCandidate) {
				bestRank = msg.Rank
				bestCandidate = msg.Candidate
			}
			c.mu.Lock()
			c.isLeader = bestCandidate == c.transport.LocalID()
			// leave c.leaderRank unchanged; it denotes our own election rank
			c.mu.Unlock()
		}
	}
}

func (c *ControlPlane) IsLeader() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isLeader
}
