package swarm

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func discoveryTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func newTestResourceManager() *ResourceManager {
	logger := discoveryTestLogger()
	rep := NewReputationSystem(logger)
	routing := NewRoutingManager(logger, rep)
	tm := NewTopologyManager(logger, rep, routing)
	return NewResourceManager(logger, rep, routing, tm)
}

func TestNewResourceManager(t *testing.T) {
	rm := newTestResourceManager()
	require.NotNil(t, rm)
}

func TestResourceManager_AnnounceResource(t *testing.T) {
	rm := newTestResourceManager()
	ctx := context.Background()

	res := &ResourceInfo{
		ID:      "res-1",
		Name:    "test-resource",
		Type:    "module",
		Version: "1.0.0",
		Size:    1024,
		Hash:    "abc123",
	}
	rm.AnnounceResource(ctx, res)

	got := rm.GetResource(ctx, "res-1")
	require.NotNil(t, got)
	assert.Equal(t, "test-resource", got.Name)
	assert.Equal(t, "1.0.0", got.Version)
}

func TestResourceManager_GetResource_NotFound(t *testing.T) {
	rm := newTestResourceManager()
	got := rm.GetResource(context.Background(), "nonexistent")
	assert.Nil(t, got)
}

func TestResourceManager_GetResource_ReturnsCopy(t *testing.T) {
	rm := newTestResourceManager()
	ctx := context.Background()

	rm.AnnounceResource(ctx, &ResourceInfo{ID: "res-1", Name: "original", Tags: []string{"a", "b"}})
	got := rm.GetResource(ctx, "res-1")
	got.Name = "modified"
	got.Tags[0] = "x"

	got2 := rm.GetResource(ctx, "res-1")
	assert.Equal(t, "original", got2.Name)
	assert.Equal(t, "a", got2.Tags[0])
}

func TestResourceManager_HandleResourceAnnouncement(t *testing.T) {
	rm := newTestResourceManager()
	ctx := context.Background()
	pid := peer.ID("peer-1")

	rm.HandleResourceAnnouncement(ctx, &ResourceAnnouncement{
		PeerID:     pid,
		Resource:   &ResourceInfo{ID: "res-1", Name: "remote-resource", Type: "module", Version: "2.0.0"},
		Timestamp:  time.Now(),
		Expiration: time.Now().Add(time.Hour),
	})

	got := rm.GetResource(ctx, "res-1")
	require.NotNil(t, got)
	assert.Equal(t, "remote-resource", got.Name)

	owners := rm.GetResourceOwners(ctx, "res-1")
	require.Len(t, owners, 1)
	assert.Equal(t, pid, owners[0])
}

func TestResourceManager_HandleResourceAnnouncement_AddsRoute(t *testing.T) {
	rm := newTestResourceManager()
	ctx := context.Background()
	pid := peer.ID("peer-1")

	rm.HandleResourceAnnouncement(ctx, &ResourceAnnouncement{
		PeerID:   pid,
		Resource: &ResourceInfo{ID: "res-1"},
	})

	route := rm.routing.GetBestRoute(ctx, "res-1")
	require.NotNil(t, route)
	assert.Equal(t, pid, route.PeerID)
}

func TestResourceManager_HandleResourceAnnouncement_NoDuplicateOwner(t *testing.T) {
	rm := newTestResourceManager()
	ctx := context.Background()
	pid := peer.ID("peer-1")

	ann := &ResourceAnnouncement{PeerID: pid, Resource: &ResourceInfo{ID: "res-1"}}
	rm.HandleResourceAnnouncement(ctx, ann)
	rm.HandleResourceAnnouncement(ctx, ann)

	owners := rm.GetResourceOwners(ctx, "res-1")
	require.Len(t, owners, 1)
}

func TestResourceManager_GetResourceOwners_NotFound(t *testing.T) {
	rm := newTestResourceManager()
	owners := rm.GetResourceOwners(context.Background(), "unknown")
	assert.Nil(t, owners)
}

func TestResourceManager_GetResourceOwners_ReturnsCopy(t *testing.T) {
	rm := newTestResourceManager()
	ctx := context.Background()
	pid := peer.ID("peer-1")

	rm.HandleResourceAnnouncement(ctx, &ResourceAnnouncement{PeerID: pid, Resource: &ResourceInfo{ID: "res-1"}})

	owners := rm.GetResourceOwners(ctx, "res-1")
	owners[0] = peer.ID("hacker")

	owners2 := rm.GetResourceOwners(ctx, "res-1")
	assert.Equal(t, pid, owners2[0])
}

func TestResourceManager_GetResourcesByPeer(t *testing.T) {
	rm := newTestResourceManager()
	ctx := context.Background()
	pid := peer.ID("peer-1")
	other := peer.ID("peer-2")

	rm.HandleResourceAnnouncement(ctx, &ResourceAnnouncement{PeerID: pid, Resource: &ResourceInfo{ID: "res-1", Name: "alpha"}})
	rm.HandleResourceAnnouncement(ctx, &ResourceAnnouncement{PeerID: other, Resource: &ResourceInfo{ID: "res-2", Name: "beta"}})

	resources := rm.GetResourcesByPeer(ctx, pid)
	require.Len(t, resources, 1)
	assert.Equal(t, "alpha", resources[0].Name)
}

func TestResourceManager_GetResourcesByPeer_Empty(t *testing.T) {
	rm := newTestResourceManager()
	resources := rm.GetResourcesByPeer(context.Background(), peer.ID("nobody"))
	assert.Empty(t, resources)
}

func TestResourceManager_GetAllResources(t *testing.T) {
	rm := newTestResourceManager()
	ctx := context.Background()

	rm.AnnounceResource(ctx, &ResourceInfo{ID: "res-1", Name: "alpha"})
	rm.AnnounceResource(ctx, &ResourceInfo{ID: "res-2", Name: "beta"})

	resources := rm.GetAllResources(ctx)
	require.Len(t, resources, 2)
}

func TestResourceManager_GetResourceCount(t *testing.T) {
	rm := newTestResourceManager()
	assert.Equal(t, 0, rm.GetResourceCount(context.Background()))

	rm.AnnounceResource(context.Background(), &ResourceInfo{ID: "res-1"})
	assert.Equal(t, 1, rm.GetResourceCount(context.Background()))
}

func TestResourceManager_FindResources_NameMatch(t *testing.T) {
	rm := newTestResourceManager()
	ctx := context.Background()

	rm.AnnounceResource(ctx, &ResourceInfo{ID: "res-1", Name: "foo", Type: "module", Version: "1.0"})
	rm.AnnounceResource(ctx, &ResourceInfo{ID: "res-2", Name: "bar", Type: "module", Version: "2.0"})

	results := rm.FindResources(ctx, &ResourceQuery{Name: "foo"})
	require.Len(t, results, 1)
	assert.Equal(t, "res-1", results[0].ID)
}

func TestResourceManager_FindResources_TypeMatch(t *testing.T) {
	rm := newTestResourceManager()
	ctx := context.Background()

	rm.AnnounceResource(ctx, &ResourceInfo{ID: "res-1", Name: "a", Type: "module"})
	rm.AnnounceResource(ctx, &ResourceInfo{ID: "res-2", Name: "b", Type: "service"})

	results := rm.FindResources(ctx, &ResourceQuery{Type: "module"})
	require.Len(t, results, 1)
	assert.Equal(t, "res-1", results[0].ID)
}

func TestResourceManager_FindResources_VersionMatch(t *testing.T) {
	rm := newTestResourceManager()
	ctx := context.Background()

	rm.AnnounceResource(ctx, &ResourceInfo{ID: "res-1", Name: "a", Version: "1.0"})
	rm.AnnounceResource(ctx, &ResourceInfo{ID: "res-2", Name: "a", Version: "2.0"})

	results := rm.FindResources(ctx, &ResourceQuery{Name: "a", Version: "2.0"})
	require.Len(t, results, 1)
	assert.Equal(t, "res-2", results[0].ID)
}

func TestResourceManager_FindResources_TagMatch(t *testing.T) {
	rm := newTestResourceManager()
	ctx := context.Background()

	rm.AnnounceResource(ctx, &ResourceInfo{ID: "res-1", Name: "a", Tags: []string{"gpu", "ml"}})
	rm.AnnounceResource(ctx, &ResourceInfo{ID: "res-2", Name: "b", Tags: []string{"cpu"}})

	results := rm.FindResources(ctx, &ResourceQuery{Tags: []string{"gpu", "ml"}})
	require.Len(t, results, 1)
	assert.Equal(t, "res-1", results[0].ID)
}

func TestResourceManager_FindResources_TagNoMatch(t *testing.T) {
	rm := newTestResourceManager()
	ctx := context.Background()

	rm.AnnounceResource(ctx, &ResourceInfo{ID: "res-1", Name: "a", Tags: []string{"gpu"}})
	rm.AnnounceResource(ctx, &ResourceInfo{ID: "res-2", Name: "b", Tags: []string{"cpu"}})

	results := rm.FindResources(ctx, &ResourceQuery{Tags: []string{"gpu", "ml"}})
	assert.Empty(t, results)
}

func TestResourceManager_FindResources_EmptyQuery(t *testing.T) {
	rm := newTestResourceManager()
	ctx := context.Background()

	rm.AnnounceResource(ctx, &ResourceInfo{ID: "res-1", Name: "a"})
	rm.AnnounceResource(ctx, &ResourceInfo{ID: "res-2", Name: "b"})

	results := rm.FindResources(ctx, &ResourceQuery{})
	require.Len(t, results, 2)
}

func TestResourceManager_GetTrustedResourceOwners(t *testing.T) {
	rm := newTestResourceManager()
	ctx := context.Background()
	trusted := peer.ID("trusted-peer")
	untrusted := peer.ID("untrusted-peer")

	rm.reputation.RecordInteraction(string(trusted), ModuleTransfer, Success)
	rm.reputation.RecordInteraction(string(trusted), ModuleTransfer, Success)

	rm.HandleResourceAnnouncement(ctx, &ResourceAnnouncement{PeerID: trusted, Resource: &ResourceInfo{ID: "res-1"}})
	rm.HandleResourceAnnouncement(ctx, &ResourceAnnouncement{PeerID: untrusted, Resource: &ResourceInfo{ID: "res-1"}})

	owners := rm.GetTrustedResourceOwners(ctx, "res-1", 55.0)
	require.Len(t, owners, 1)
	assert.Equal(t, trusted, owners[0])
}

func TestResourceManager_GetTrustedResourceOwners_NotFound(t *testing.T) {
	rm := newTestResourceManager()
	owners := rm.GetTrustedResourceOwners(context.Background(), "unknown", 60.0)
	assert.Nil(t, owners)
}

func TestResourceManager_GetTrustedResourceOwners_AllUntrusted(t *testing.T) {
	rm := newTestResourceManager()
	ctx := context.Background()

	rm.HandleResourceAnnouncement(ctx, &ResourceAnnouncement{PeerID: peer.ID("low-rep"), Resource: &ResourceInfo{ID: "res-1"}})

	owners := rm.GetTrustedResourceOwners(ctx, "res-1", 90.0)
	assert.Empty(t, owners)
}

func TestResourceManager_RemoveExpiredResources(t *testing.T) {
	rm := newTestResourceManager()
	ctx := context.Background()

	rm.AnnounceResource(ctx, &ResourceInfo{
		ID: "res-old", Name: "old", UpdatedAt: time.Now().Add(-48 * time.Hour),
	})
	rm.AnnounceResource(ctx, &ResourceInfo{
		ID: "res-fresh", Name: "fresh", UpdatedAt: time.Now(),
	})

	rm.RemoveExpiredResources(ctx)

	assert.Nil(t, rm.GetResource(ctx, "res-old"))
	assert.NotNil(t, rm.GetResource(ctx, "res-fresh"))
}

func TestResourceManager_RemoveExpiredResources_NoneExpired(t *testing.T) {
	rm := newTestResourceManager()
	ctx := context.Background()

	rm.AnnounceResource(ctx, &ResourceInfo{ID: "res-1", Name: "a", UpdatedAt: time.Now()})
	rm.RemoveExpiredResources(ctx)

	assert.Equal(t, 1, rm.GetResourceCount(ctx))
}

func TestResourceManager_RemoveExpiredResources_RemovesOwners(t *testing.T) {
	rm := newTestResourceManager()
	ctx := context.Background()
	pid := peer.ID("peer-1")

	rm.HandleResourceAnnouncement(ctx, &ResourceAnnouncement{
		PeerID:   pid,
		Resource: &ResourceInfo{ID: "res-old", UpdatedAt: time.Now().Add(-48 * time.Hour)},
	})

	rm.RemoveExpiredResources(ctx)

	assert.Nil(t, rm.GetResourceOwners(ctx, "res-old"))
}
