package consensus

import (
	"context"
	"log/slog"
)

// Consensus defines the interface for consensus algorithms
type Consensus interface {
	// Start begins the consensus process
	Start(ctx context.Context) error
	
	// Stop halts the consensus process
	Stop() error
	
	// ProposeValue proposes a value for consensus
	ProposeValue(ctx context.Context, key string, value interface{}) error
	
	// GetValue retrieves the agreed-upon value for a key
	GetValue(ctx context.Context, key string) (interface{}, error)
	
	// IsLeader returns whether this node is the leader
	IsLeader() bool
	
	// GetLeaderID returns the ID of the current leader
	GetLeaderID() string
}

// Config holds configuration for the consensus algorithm
type Config struct {
	NodeID     string   `yaml:"node_id"`
	PeerIDs    []string `yaml:"peer_ids"`
	Algorithm  string   `yaml:"algorithm"` // "raft", "paxos", etc.
	ListenAddr string   `yaml:"listen_addr"`
}

// BaseConsensus provides a base implementation for consensus algorithms
type BaseConsensus struct {
	logger  *slog.Logger
	config  *Config
	isLeader bool
	leaderID string
}

// NewBaseConsensus creates a new BaseConsensus
func NewBaseConsensus(logger *slog.Logger, config *Config) *BaseConsensus {
	return &BaseConsensus{
		logger:  logger,
		config:  config,
		isLeader: false,
		leaderID: "",
	}
}

// IsLeader returns whether this node is the leader
func (c *BaseConsensus) IsLeader() bool {
	return c.isLeader
}

// GetLeaderID returns the ID of the current leader
func (c *BaseConsensus) GetLeaderID() string {
	return c.leaderID
}

// SetLeader sets the leader status for this node
func (c *BaseConsensus) SetLeader(isLeader bool, leaderID string) {
	c.isLeader = isLeader
	c.leaderID = leaderID
	
	if isLeader {
		c.logger.Info("This node is now the leader")
	} else {
		c.logger.Info("This node is no longer the leader", "leader_id", leaderID)
	}
}