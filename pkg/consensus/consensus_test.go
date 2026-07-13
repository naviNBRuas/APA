package consensus

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsensusInterface(t *testing.T) {
	var _ Consensus = (*LeaderElection)(nil)
}

func TestNewBaseConsensus(t *testing.T) {
	config := &Config{NodeID: "node1", Algorithm: "raft"}
	bc := NewBaseConsensus(slog.Default(), config)
	require.NotNil(t, bc)
	assert.False(t, bc.IsLeader())
	assert.Empty(t, bc.GetLeaderID())
}

func TestBaseConsensus_NilLogger(t *testing.T) {
	bc := NewBaseConsensus(nil, &Config{NodeID: "node1"})
	require.NotNil(t, bc)
	assert.False(t, bc.IsLeader())
}

func TestSetLeader_BecomesLeader(t *testing.T) {
	bc := NewBaseConsensus(slog.Default(), &Config{NodeID: "node1"})
	bc.SetLeader(true, "node1")
	assert.True(t, bc.IsLeader())
	assert.Equal(t, "node1", bc.GetLeaderID())
}

func TestSetLeader_BecomesFollower(t *testing.T) {
	bc := NewBaseConsensus(slog.Default(), &Config{NodeID: "node1"})
	bc.SetLeader(true, "node1")
	bc.SetLeader(false, "node2")
	assert.False(t, bc.IsLeader())
	assert.Equal(t, "node2", bc.GetLeaderID())
}

func TestSetLeader_EmptyLeaderID(t *testing.T) {
	bc := NewBaseConsensus(slog.Default(), &Config{NodeID: "node1"})
	bc.SetLeader(false, "")
	assert.False(t, bc.IsLeader())
	assert.Empty(t, bc.GetLeaderID())
}

func TestBaseConsensus_ConcurrentAccess(t *testing.T) {
	bc := NewBaseConsensus(slog.Default(), &Config{NodeID: "node1"})
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				bc.SetLeader(true, "node1")
			} else {
				_ = bc.IsLeader()
				_ = bc.GetLeaderID()
			}
		}(i)
	}
	wg.Wait()
}

func TestNewLeaderElection(t *testing.T) {
	config := &Config{NodeID: "node1", PeerIDs: []string{"peer1", "peer2"}, Algorithm: "leader-election"}
	le := NewLeaderElection(slog.Default(), config)
	require.NotNil(t, le)
	assert.NotNil(t, le.values)
	assert.Equal(t, int64(0), le.electionTerm)
	assert.Empty(t, le.votedFor)
}

func TestLeaderElection_StartStop(t *testing.T) {
	le := NewLeaderElection(slog.Default(), &Config{NodeID: "node1"})
	ctx := context.Background()

	err := le.Start(ctx)
	require.NoError(t, err)

	err = le.Stop()
	require.NoError(t, err)
}

func TestLeaderElection_ProposeValue_NonLeader(t *testing.T) {
	le := NewLeaderElection(slog.Default(), &Config{NodeID: "node1"})

	err := le.ProposeValue(context.Background(), "key1", "value1")
	require.NoError(t, err)

	val, err := le.GetValue(context.Background(), "key1")
	require.NoError(t, err)
	assert.Nil(t, val)
}

func TestLeaderElection_ProposeValue_AsLeader(t *testing.T) {
	le := NewLeaderElection(slog.Default(), &Config{NodeID: "node1"})
	le.SetLeader(true, "node1")

	err := le.ProposeValue(context.Background(), "key1", "value1")
	require.NoError(t, err)

	val, err := le.GetValue(context.Background(), "key1")
	require.NoError(t, err)
	assert.Equal(t, "value1", val)
}

func TestLeaderElection_ProposeValue_MultipleKeys(t *testing.T) {
	le := NewLeaderElection(slog.Default(), &Config{NodeID: "node1"})
	le.SetLeader(true, "node1")

	assert.NoError(t, le.ProposeValue(context.Background(), "k1", "v1"))
	assert.NoError(t, le.ProposeValue(context.Background(), "k2", "v2"))

	v1, _ := le.GetValue(context.Background(), "k1")
	assert.Equal(t, "v1", v1)
	v2, _ := le.GetValue(context.Background(), "k2")
	assert.Equal(t, "v2", v2)
}

func TestLeaderElection_GetValue_NotFound(t *testing.T) {
	le := NewLeaderElection(slog.Default(), &Config{NodeID: "node1"})

	val, err := le.GetValue(context.Background(), "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, val)
}

func TestLeaderElection_GetValue_EmptyKey(t *testing.T) {
	le := NewLeaderElection(slog.Default(), &Config{NodeID: "node1"})
	le.SetLeader(true, "node1")

	le.ProposeValue(context.Background(), "", "empty-key-value")

	val, err := le.GetValue(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, "empty-key-value", val)
}

func TestLeaderElection_Start_CancelledContext(t *testing.T) {
	le := NewLeaderElection(slog.Default(), &Config{NodeID: "node1"})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := le.Start(ctx)
	require.NoError(t, err)
	defer le.Stop()

	time.Sleep(50 * time.Millisecond)
}

func TestLeaderElection_Start_ElectionEventually(t *testing.T) {
	le := NewLeaderElection(slog.Default(), &Config{NodeID: "node1"})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := le.Start(ctx)
	require.NoError(t, err)
	defer le.Stop()

	assert.Eventually(t, func() bool {
		return le.IsLeader()
	}, 2*time.Second, 50*time.Millisecond)

	assert.Equal(t, "node1", le.GetLeaderID())
	assert.Equal(t, int64(1), le.electionTerm)
	assert.Equal(t, "node1", le.votedFor)
}

func TestLeaderElection_ProposeValue_Concurrent(t *testing.T) {
	le := NewLeaderElection(slog.Default(), &Config{NodeID: "node1"})
	le.SetLeader(true, "node1")

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := "key"
			le.ProposeValue(context.Background(), key, i)
		}(i)
	}
	wg.Wait()
}

func TestNewConfig(t *testing.T) {
	config := &Config{
		NodeID:     "test-node",
		PeerIDs:    []string{"peer1", "peer2"},
		Algorithm:  "raft",
		ListenAddr: ":9090",
	}
	assert.Equal(t, "test-node", config.NodeID)
	assert.Equal(t, []string{"peer1", "peer2"}, config.PeerIDs)
	assert.Equal(t, "raft", config.Algorithm)
	assert.Equal(t, ":9090", config.ListenAddr)
}
