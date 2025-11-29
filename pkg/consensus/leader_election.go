package consensus

import (
	"context"
	"log/slog"
	"math/rand"
	"sync"
	"time"
)

// LeaderElection implements a simple leader election consensus algorithm
type LeaderElection struct {
	*BaseConsensus
	mu           sync.RWMutex
	values       map[string]interface{}
	electionTerm int64
	votedFor     string
}

// NewLeaderElection creates a new LeaderElection consensus algorithm
func NewLeaderElection(logger *slog.Logger, config *Config) *LeaderElection {
	return &LeaderElection{
		BaseConsensus: NewBaseConsensus(logger, config),
		values:        make(map[string]interface{}),
		electionTerm:  0,
		votedFor:      "",
	}
}

// Start begins the leader election process
func (le *LeaderElection) Start(ctx context.Context) error {
	le.logger.Info("Starting leader election consensus")
	
	// Start the election timer
	go le.runElectionTimer(ctx)
	
	return nil
}

// Stop halts the consensus process
func (le *LeaderElection) Stop() error {
	le.logger.Info("Stopping leader election consensus")
	return nil
}

// ProposeValue proposes a value for consensus
func (le *LeaderElection) ProposeValue(ctx context.Context, key string, value interface{}) error {
	le.mu.Lock()
	defer le.mu.Unlock()
	
	// Only the leader can propose values
	if !le.IsLeader() {
		return nil // Non-leader nodes don't propose values in this simple implementation
	}
	
	le.values[key] = value
	le.logger.Info("Proposed value for key", "key", key, "value", value)
	
	return nil
}

// GetValue retrieves the agreed-upon value for a key
func (le *LeaderElection) GetValue(ctx context.Context, key string) (interface{}, error) {
	le.mu.RLock()
	defer le.mu.RUnlock()
	
	value, exists := le.values[key]
	if !exists {
		return nil, nil
	}
	
	return value, nil
}

// runElectionTimer runs the election timer
func (le *LeaderElection) runElectionTimer(ctx context.Context) {
	// Randomize the election timeout between 150-300ms
	timeout := time.Duration(150+rand.Intn(150)) * time.Millisecond
	ticker := time.NewTicker(timeout)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			le.startElection()
			// Reset the ticker with a new random timeout
			timeout = time.Duration(150+rand.Intn(150)) * time.Millisecond
			ticker.Reset(timeout)
		}
	}
}

// startElection starts a new election
func (le *LeaderElection) startElection() {
	le.mu.Lock()
	defer le.mu.Unlock()
	
	// Increment the election term
	le.electionTerm++
	le.votedFor = le.config.NodeID
	
	le.logger.Info("Starting new election", "term", le.electionTerm)
	
	// In a real implementation, we would send vote requests to other nodes
	// For this simple implementation, we'll just assume we win the election
	le.SetLeader(true, le.config.NodeID)
}