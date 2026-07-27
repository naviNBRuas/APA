package mesh

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func testKey() ed25519.PrivateKey {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	return priv
}

// ---------------------------------------------------------------------------
// MeshNode tests
// ---------------------------------------------------------------------------

func TestNewMeshNode_Clone(t *testing.T) {
	t.Parallel()
	pubKey := make([]byte, 32)
	for i := range pubKey {
		pubKey[i] = byte(i)
	}
	n := &MeshNode{
		ID:        "test-node",
		PublicKey: pubKey,
		VirtualIP: []byte{0xfd, 0x00, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		Name:      "test",
		Version:   "1.0",
		Status:    NodeStatusOnline,
		Services:  map[string]string{"http": "8080"},
		Addresses: []string{"192.168.1.1"},
	}
	clone := n.Clone()
	require.NotNil(t, clone)
	assert.Equal(t, n.ID, clone.ID)
	assert.Equal(t, n.Name, clone.Name)
	assert.Equal(t, n.Version, clone.Version)
	assert.Equal(t, n.VirtualIP, clone.VirtualIP)
	assert.Equal(t, n.PublicKey, clone.PublicKey)
	assert.Equal(t, n.Services, clone.Services)

	clone.Services["new"] = "added"
	assert.NotContains(t, n.Services, "new")
}

func TestMeshNode_MarshalUnmarshal(t *testing.T) {
	t.Parallel()
	n := &MeshNode{
		ID:       "test-id",
		Name:     "test-node",
		Version:  "1.0.0",
		Status:   NodeStatusOnline,
		Services: map[string]string{"api": "9000"},
	}
	data, err := n.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(data), "test-id")

	n2 := &MeshNode{}
	err = n2.UnmarshalJSON(data)
	require.NoError(t, err)
	assert.Equal(t, n.ID, n2.ID)
	assert.Equal(t, n.Name, n2.Name)
}

// ---------------------------------------------------------------------------
// MeshNetwork tests
// ---------------------------------------------------------------------------

func TestNewMeshNetwork(t *testing.T) {
	t.Parallel()
	cfg := DefaultMeshConfig()
	cfg.EnableRelay = false
	cfg.EnableLAN = false

	mn, err := NewMeshNetwork(testLogger(), cfg, testKey())
	require.NoError(t, err)
	require.NotNil(t, mn)
	assert.NotNil(t, mn.Self())
	assert.NotEmpty(t, mn.Self().ID)
	assert.NotNil(t, mn.Self().VirtualIP)
	assert.Equal(t, NodeStatusOnline, mn.Self().Status)
}

func TestMeshNetwork_Self(t *testing.T) {
	t.Parallel()
	cfg := DefaultMeshConfig()
	cfg.EnableRelay = false
	cfg.EnableLAN = false
	cfg.NodeName = "my-node"

	mn, err := NewMeshNetwork(testLogger(), cfg, testKey())
	require.NoError(t, err)
	self := mn.Self()
	assert.Equal(t, "my-node", self.Name)
	assert.Equal(t, Version, self.Version)
}

func TestMeshNetwork_StartStop(t *testing.T) {
	cfg := DefaultMeshConfig()
	cfg.EnableRelay = false
	cfg.EnableLAN = false
	cfg.ListenPort = 0

	mn, err := NewMeshNetwork(testLogger(), cfg, testKey())
	require.NoError(t, err)

	err = mn.Start()
	require.NoError(t, err)

	mn.Stop()
}

func TestMeshNetwork_PeerManagement(t *testing.T) {
	t.Parallel()
	cfg := DefaultMeshConfig()
	cfg.EnableRelay = false
	cfg.EnableLAN = false

	mn, err := NewMeshNetwork(testLogger(), cfg, testKey())
	require.NoError(t, err)

	assert.Equal(t, 0, mn.PeerCount())
	assert.Empty(t, mn.Peers())
	assert.Nil(t, mn.GetPeer("nonexistent"))

	peerPriv := testKey()
	peerPub := peerPriv.Public().(ed25519.PublicKey)
	peerSM := NewSecurityManager(testLogger(), peerPriv, peerPub)
	signedInfo := peerSM.SignNodeInfo(&NodeInfo{
		ID:        "peer-1",
		Name:      "peer-node",
		Version:   "1.0",
		PublicKey: peerPub,
		Capabilities: CapabilitySet{SupportsCodeSync: true},
	})
	_, err = mn.handleIncomingConnection("192.168.1.1:9000", signedInfo)
	require.NoError(t, err)
	assert.Equal(t, 1, mn.PeerCount())

	peer := mn.GetPeer("peer-1")
	require.NotNil(t, peer)
	assert.Equal(t, "peer-node", peer.Name)
}

func TestMeshNetwork_HandleIncoming_RejectsNil(t *testing.T) {
	t.Parallel()
	cfg := DefaultMeshConfig()
	cfg.EnableRelay = false
	cfg.EnableLAN = false

	mn, _ := NewMeshNetwork(testLogger(), cfg, testKey())
	_, err := mn.handleIncomingConnection("addr", nil)
	assert.Error(t, err)
}

func TestMeshNetwork_Info(t *testing.T) {
	t.Parallel()
	cfg := DefaultMeshConfig()
	cfg.EnableRelay = false
	cfg.EnableLAN = false

	mn, _ := NewMeshNetwork(testLogger(), cfg, testKey())
	info := mn.Info()
	assert.NotNil(t, info)
	assert.Contains(t, info, "node_id")
	assert.Contains(t, info, "node_name")
	assert.Contains(t, info, "virtual_ip")
	assert.Contains(t, info, "peer_count")
}

func TestMeshNetwork_Events(t *testing.T) {
	cfg := DefaultMeshConfig()
	cfg.EnableRelay = false
	cfg.EnableLAN = false
	cfg.ListenPort = 0

	mn, _ := NewMeshNetwork(testLogger(), cfg, testKey())
	mn.Start()
	defer mn.Stop()

	events := mn.Events()
	assert.NotNil(t, events)
}

func TestMeshNetwork_SyncCode(t *testing.T) {
	t.Parallel()
	cfg := DefaultMeshConfig()
	cfg.EnableRelay = false
	cfg.EnableLAN = false

	mn, _ := NewMeshNetwork(testLogger(), cfg, testKey())
	err := mn.SyncCode("test-module", "1.0.0", []byte("module data"))
	assert.NoError(t, err)
}

func TestMeshNetwork_SyncState(t *testing.T) {
	t.Parallel()
	cfg := DefaultMeshConfig()
	cfg.EnableRelay = false
	cfg.EnableLAN = false

	mn, _ := NewMeshNetwork(testLogger(), cfg, testKey())
	err := mn.SyncState("config/key", []byte("value"))
	assert.NoError(t, err)
}

func TestMeshNetwork_Broadcast(t *testing.T) {
	cfg := DefaultMeshConfig()
	cfg.EnableRelay = false
	cfg.EnableLAN = false
	cfg.ListenPort = 0

	mn, _ := NewMeshNetwork(testLogger(), cfg, testKey())
	mn.Start()
	defer mn.Stop()

	err := mn.Broadcast("test", []byte("hello"))
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// SecurityManager tests
// ---------------------------------------------------------------------------

func TestNewSecurityManager(t *testing.T) {
	t.Parallel()
	priv := testKey()
	pub := priv.Public().(ed25519.PublicKey)
	sm := NewSecurityManager(testLogger(), priv, pub)
	require.NotNil(t, sm)
}

func TestSecurityManager_SignAndVerify(t *testing.T) {
	t.Parallel()
	priv := testKey()
	pub := priv.Public().(ed25519.PublicKey)
	sm := NewSecurityManager(testLogger(), priv, pub)

	data := []byte("test message")
	sig := sm.Sign(data)
	assert.Len(t, sig, ed25519.SignatureSize)

	assert.True(t, sm.Verify(data, sig, pub))
	assert.False(t, sm.Verify([]byte("wrong data"), sig, pub))
}

func TestSecurityManager_Authenticate_Valid(t *testing.T) {
	t.Parallel()
	priv := testKey()
	pub := priv.Public().(ed25519.PublicKey)
	sm := NewSecurityManager(testLogger(), priv, pub)

	info := &NodeInfo{ID: "test", PublicKey: pub}
	info = sm.SignNodeInfo(info)

	sm2 := NewSecurityManager(testLogger(), nil, nil)
	authed := sm2.Authenticate(info)
	assert.True(t, authed)
}

func TestSecurityManager_Authenticate_Banned(t *testing.T) {
	t.Parallel()
	priv := testKey()
	pub := priv.Public().(ed25519.PublicKey)
	sm := NewSecurityManager(testLogger(), priv, pub)
	sm.BanPeer("banned-id", "test ban")

	sm2 := NewSecurityManager(testLogger(), priv, pub)
	info := &NodeInfo{ID: "banned-id", PublicKey: pub}
	info = sm2.SignNodeInfo(info)

	authed := sm.Authenticate(info)
	assert.False(t, authed)
}

func TestSecurityManager_Authenticate_NilInfo(t *testing.T) {
	t.Parallel()
	sm := NewSecurityManager(testLogger(), nil, nil)
	assert.False(t, sm.Authenticate(nil))
}

func TestSecurityManager_Verify_WrongKeySize(t *testing.T) {
	t.Parallel()
	sm := NewSecurityManager(testLogger(), nil, nil)
	assert.False(t, sm.Verify([]byte("data"), []byte("sig"), make([]byte, 16)))
}

func TestSecurityManager_AdmitAndBan(t *testing.T) {
	t.Parallel()
	sm := NewSecurityManager(testLogger(), nil, nil)
	sm.AdmitPeer("peer-1")
	assert.True(t, sm.IsAdmitted("peer-1"))
	assert.False(t, sm.IsBanned("peer-1"))

	sm.BanPeer("peer-1", "bad behavior")
	assert.True(t, sm.IsBanned("peer-1"))
	assert.False(t, sm.IsAdmitted("peer-1"))
}

func TestSecurityManager_GenerateSessionKey(t *testing.T) {
	t.Parallel()
	sm := NewSecurityManager(testLogger(), nil, nil)
	key1 := sm.GenerateSessionKey(make([]byte, 32))
	key2 := sm.GenerateSessionKey(make([]byte, 32))
	assert.Len(t, key1, 32)
	assert.Len(t, key2, 32)
}

func TestSecurityManager_EncryptDecryptRoundTrip(t *testing.T) {
	t.Parallel()
	sm := NewSecurityManager(testLogger(), nil, nil)
	key := sm.GenerateSessionKey(make([]byte, 32))
	plain := []byte("mesh-secret-payload")

	ct, err := sm.EncryptPayload(key, plain)
	require.NoError(t, err)
	assert.NotEqual(t, plain, ct)

	got, err := sm.DecryptPayload(key, ct)
	require.NoError(t, err)
	assert.Equal(t, plain, got)
}

func TestSecurityManager_EncryptPayload_BadKey(t *testing.T) {
	t.Parallel()
	sm := NewSecurityManager(testLogger(), nil, nil)
	_, err := sm.EncryptPayload([]byte("short"), []byte("data"))
	assert.Error(t, err)
}

func TestNewMeshNetwork_ZeroIntervalsUseDefaults(t *testing.T) {
	t.Parallel()
	cfg := MeshConfig{NodeName: "defaults-test", EnableRelay: false, EnableLAN: false}
	mn, err := NewMeshNetwork(testLogger(), cfg, testKey())
	require.NoError(t, err)
	require.NoError(t, mn.Start())
	defer mn.Stop()
	assert.Greater(t, mn.config.HeartbeatInterval, time.Duration(0))
	assert.Greater(t, mn.config.SyncInterval, time.Duration(0))
}

// ---------------------------------------------------------------------------
// TransportChain tests
// ---------------------------------------------------------------------------

func TestNewTransportChain(t *testing.T) {
	t.Parallel()
	cfg := DefaultMeshConfig()
	cfg.EnableRelay = false
	cfg.EnableLAN = false
	tc := NewTransportChain(testLogger(), cfg)
	require.NotNil(t, tc)
	assert.NotNil(t, tc.Receive())
}

func TestTransportChain_Available(t *testing.T) {
	t.Parallel()
	cfg := DefaultMeshConfig()
	cfg.EnableRelay = false
	cfg.EnableLAN = false
	cfg.ListenPort = 0
	tc := NewTransportChain(testLogger(), cfg)
	ctx := context.Background()
	tc.Start(ctx)
	defer tc.Stop()
	avail := tc.Available()
	assert.NotNil(t, avail)
}

func TestTransportChain_StartStop(t *testing.T) {
	cfg := DefaultMeshConfig()
	cfg.EnableRelay = false
	cfg.EnableLAN = false
	cfg.ListenPort = 0

	tc := NewTransportChain(testLogger(), cfg)
	ctx := context.Background()
	err := tc.Start(ctx)
	require.NoError(t, err)
	tc.Stop()
}

func TestTransportChain_Broadcast(t *testing.T) {
	cfg := DefaultMeshConfig()
	cfg.EnableRelay = false
	cfg.EnableLAN = false
	cfg.ListenPort = 0

	tc := NewTransportChain(testLogger(), cfg)
	ctx := context.Background()
	tc.Start(ctx)
	defer tc.Stop()

	err := tc.Broadcast("test", []byte("hello"))
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// DiscoveryEngine tests
// ---------------------------------------------------------------------------

func TestNewDiscoveryEngine(t *testing.T) {
	t.Parallel()
	de := NewDiscoveryEngine(testLogger(), DefaultMeshConfig())
	require.NotNil(t, de)
}

func TestDiscoveryEngine_StartStop(t *testing.T) {
	cfg := DefaultMeshConfig()
	cfg.EnableRelay = false
	cfg.EnableLAN = false
	cfg.ListenPort = 0

	de := NewDiscoveryEngine(testLogger(), cfg)
	tc := NewTransportChain(testLogger(), cfg)

	mn, _ := NewMeshNetwork(testLogger(), cfg, testKey())

	ctx := context.Background()
	err := de.Start(ctx, mn.Self(), tc)
	require.NoError(t, err)
	de.Stop()
}

func TestDiscoveryEngine_Events(t *testing.T) {
	t.Parallel()
	de := NewDiscoveryEngine(testLogger(), DefaultMeshConfig())
	events := de.Events()
	assert.NotNil(t, events)
}

// ---------------------------------------------------------------------------
// SyncEngine tests
// ---------------------------------------------------------------------------

func TestNewSyncEngine(t *testing.T) {
	t.Parallel()
	se := NewSyncEngine(testLogger(), "test-node", 30*time.Second)
	require.NotNil(t, se)
	assert.NotNil(t, se.Events())
}

func TestSyncEngine_PublishAndGetCode(t *testing.T) {
	t.Parallel()
	se := NewSyncEngine(testLogger(), "test-node", 30*time.Second)
	ctx := context.Background()

	err := se.PublishCode(ctx, "my-module", "1.0.0", []byte("wasm data"))
	require.NoError(t, err)

	artifact := se.GetCode("my-module", "1.0.0")
	require.NotNil(t, artifact)
	assert.Equal(t, "my-module", artifact.Name)
	assert.Equal(t, "1.0.0", artifact.Version)
	assert.Equal(t, []byte("wasm data"), artifact.Data)
	assert.Equal(t, 9, artifact.Size)
	assert.NotEmpty(t, artifact.Hash)
}

func TestSyncEngine_ReceiveCode(t *testing.T) {
	t.Parallel()
	se := NewSyncEngine(testLogger(), "test-node", 30*time.Second)

	err := se.ReceiveCode(&CodeArtifact{
		Name:      "remote-mod",
		Version:   "2.0.0",
		Data:      []byte("remote data"),
		Publisher: "peer-1",
		Timestamp: time.Now().UTC().UnixNano(),
	})
	require.NoError(t, err)

	assert.True(t, se.HasCode("remote-mod", "2.0.0"))
	artifact := se.GetCode("remote-mod", "2.0.0")
	require.NotNil(t, artifact)
	assert.Equal(t, "peer-1", artifact.Publisher)
}

func TestSyncEngine_ReceiveCode_Nil(t *testing.T) {
	t.Parallel()
	se := NewSyncEngine(testLogger(), "test-node", 30*time.Second)
	err := se.ReceiveCode(nil)
	assert.Error(t, err)
}

func TestSyncEngine_ReceiveCode_OlderVersion(t *testing.T) {
	t.Parallel()
	se := NewSyncEngine(testLogger(), "test-node", 30*time.Second)
	ts := time.Now().UTC().UnixNano()

	se.ReceiveCode(&CodeArtifact{Name: "mod", Version: "1.0", Data: []byte("v1"), Timestamp: ts})
	err := se.ReceiveCode(&CodeArtifact{Name: "mod", Version: "1.0", Data: []byte("v1-older"), Timestamp: ts - 1})
	require.NoError(t, err)

	artifact := se.GetCode("mod", "1.0")
	assert.Equal(t, []byte("v1"), artifact.Data)
}

func TestSyncEngine_PublishAndGetState(t *testing.T) {
	t.Parallel()
	se := NewSyncEngine(testLogger(), "test-node", 30*time.Second)
	ctx := context.Background()

	err := se.PublishState(ctx, "agent/config", []byte(`{"key":"value"}`))
	require.NoError(t, err)

	entry := se.GetState("agent/config")
	require.NotNil(t, entry)
	assert.Equal(t, "agent/config", entry.Key)
	assert.Equal(t, `{"key":"value"}`, string(entry.Value))
}

func TestSyncEngine_ReceiveState_OlderVersion(t *testing.T) {
	t.Parallel()
	se := NewSyncEngine(testLogger(), "test-node", 30*time.Second)

	se.ReceiveState(&StateEntry{Key: "k", Value: []byte("v2"), Version: 100})
	err := se.ReceiveState(&StateEntry{Key: "k", Value: []byte("v1"), Version: 50})
	require.NoError(t, err)

	entry := se.GetState("k")
	assert.Equal(t, []byte("v2"), entry.Value)
}

func TestSyncEngine_AllCode(t *testing.T) {
	t.Parallel()
	se := NewSyncEngine(testLogger(), "test-node", 30*time.Second)
	se.ReceiveCode(&CodeArtifact{Name: "a", Version: "1", Data: []byte("a")})
	se.ReceiveCode(&CodeArtifact{Name: "b", Version: "1", Data: []byte("b")})
	assert.Len(t, se.AllCode(), 2)
}

func TestSyncEngine_AllState(t *testing.T) {
	t.Parallel()
	se := NewSyncEngine(testLogger(), "test-node", 30*time.Second)
	se.ReceiveState(&StateEntry{Key: "k1", Value: []byte("v1"), Version: 1})
	se.ReceiveState(&StateEntry{Key: "k2", Value: []byte("v2"), Version: 2})
	assert.Len(t, se.AllState(), 2)
}

func TestSyncEngine_HasCode(t *testing.T) {
	t.Parallel()
	se := NewSyncEngine(testLogger(), "test-node", 30*time.Second)
	assert.False(t, se.HasCode("nonexistent", "1.0"))

	se.ReceiveCode(&CodeArtifact{Name: "exists", Version: "1.0", Data: []byte("data")})
	assert.True(t, se.HasCode("exists", "1.0"))
}

func TestSyncEngine_Stats(t *testing.T) {
	t.Parallel()
	se := NewSyncEngine(testLogger(), "test-node", 30*time.Second)
	se.ReceiveCode(&CodeArtifact{Name: "m", Version: "1", Data: []byte("d")})
	stats := se.Stats()
	assert.Equal(t, 1, stats["code_artifacts"])
	assert.Equal(t, 0, stats["state_entries"])
	assert.Equal(t, false, stats["running"])
}

func TestSyncEngine_StartStop(t *testing.T) {
	t.Parallel()
	se := NewSyncEngine(testLogger(), "test-node", 30*time.Second)
	ctx := context.Background()
	se.Start(ctx)
	assert.True(t, se.running)
	se.Stop()
	assert.False(t, se.running)
}

// ---------------------------------------------------------------------------
// NodeInfo / NewNodeInfo tests
// ---------------------------------------------------------------------------

func TestNodeInfo_Signing(t *testing.T) {
	t.Parallel()
	priv := testKey()
	pub := priv.Public().(ed25519.PublicKey)
	sm := NewSecurityManager(testLogger(), priv, pub)

	node := &MeshNode{
		ID:        "node-1",
		PublicKey: pub,
		Name:      "test-node",
		Version:   "1.0",
		Services:  map[string]string{"http": "80"},
	}
	info := NewNodeInfo(node, sm)
	require.NotNil(t, info)
	assert.Equal(t, "node-1", info.ID)
	assert.Len(t, info.Signature, ed25519.SignatureSize)
}

func TestNewSession(t *testing.T) {
	t.Parallel()
	s1 := NewSession()
	s2 := NewSession()
	assert.NotEqual(t, s1, s2)
	assert.NotEmpty(t, s1)
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestDeriveVirtualIP(t *testing.T) {
	t.Parallel()
	pubKey := make([]byte, 32)
	for i := range pubKey {
		pubKey[i] = byte(i)
	}
	ip := deriveVirtualIP(pubKey, "fd00:apa::/48")
	require.NotNil(t, ip)
	assert.True(t, len(ip) >= 16)
	assert.Equal(t, byte(0xfd), ip[0])
}

func TestMeshNetwork_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	cfg := DefaultMeshConfig()
	cfg.EnableRelay = false
	cfg.EnableLAN = false
	cfg.ListenPort = 0

	mn, _ := NewMeshNetwork(testLogger(), cfg, testKey())

	done := make(chan struct{})
	go func() {
		for i := 0; i < 50; i++ {
		_, _ = mn.handleIncomingConnection("addr", &NodeInfo{
			ID:        fmt.Sprintf("peer-%d", i),
			PublicKey: make([]byte, 32),
		})
		}
		done <- struct{}{}
	}()

	go func() {
		for i := 0; i < 50; i++ {
			mn.PeerCount()
			mn.Peers()
			mn.GetPeer(fmt.Sprintf("peer-%d", i))
		}
		done <- struct{}{}
	}()

	<-done
	<-done
}

func TestRelayServer_StartStop(t *testing.T) {
	rs := NewRelayServer(testLogger(), 0)
	err := rs.Start()
	require.NoError(t, err)
	assert.True(t, rs.Addr() != "")
	assert.Equal(t, 0, rs.ClientCount())
	rs.Stop()
	assert.Equal(t, 0, rs.ClientCount())
}

func TestRelayServer_StartTwice(t *testing.T) {
	t.Parallel()
	rs := NewRelayServer(testLogger(), 0)
	err := rs.Start()
	require.NoError(t, err)
	err = rs.Start()
	require.NoError(t, err)
	rs.Stop()
}

func TestRelayServer_ConnectAndRelay(t *testing.T) {
	rs := NewRelayServer(testLogger(), 0)
	err := rs.Start()
	require.NoError(t, err)
	defer rs.Stop()

	conn, err := net.Dial("tcp", rs.Addr())
	require.NoError(t, err)
	defer conn.Close()

	enc := json.NewEncoder(conn)
	dec := json.NewDecoder(conn)

	regMsg := RelayMessage{
		Type:    "relay_register",
		Payload: mustMarshal(map[string]string{"id": "test-node-1"}),
	}
	err = enc.Encode(regMsg)
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, rs.ClientCount())

	listMsg := RelayMessage{Type: "relay_list"}
	err = enc.Encode(listMsg)
	require.NoError(t, err)

	var resp RelayMessage
	dec.Decode(&resp)
	assert.Equal(t, "relay_list_response", resp.Type)

	pingMsg := RelayMessage{Type: "relay_ping"}
	err = enc.Encode(pingMsg)
	require.NoError(t, err)

	var pong RelayMessage
	dec.Decode(&pong)
	assert.Equal(t, "relay_pong", pong.Type)
}

func TestRelayServer_ForwardBetweenClients(t *testing.T) {
	rs := NewRelayServer(testLogger(), 0)
	err := rs.Start()
	require.NoError(t, err)
	defer rs.Stop()

	conn1, err := net.Dial("tcp", rs.Addr())
	require.NoError(t, err)
	defer conn1.Close()
	enc1 := json.NewEncoder(conn1)
	enc1.Encode(RelayMessage{
		Type:    "relay_register",
		Payload: mustMarshal(map[string]string{"id": "node-a"}),
	})

	conn2, err := net.Dial("tcp", rs.Addr())
	require.NoError(t, err)
	defer conn2.Close()
	enc2 := json.NewEncoder(conn2)
	dec2 := json.NewDecoder(conn2)
	enc2.Encode(RelayMessage{
		Type:    "relay_register",
		Payload: mustMarshal(map[string]string{"id": "node-b"}),
	})

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 2, rs.ClientCount())

	err = enc1.Encode(RelayMessage{
		Type:    "relay_forward",
		To:      "node-b",
		Payload: []byte(`hello-from-a`),
	})
	require.NoError(t, err)

	var received RelayMessage
	dec2.Decode(&received)
	assert.Equal(t, "relay_deliver", received.Type)
	assert.Contains(t, string(received.Payload), "hello-from-a")
}

func TestRelayTransport_RelayMessageJSON(t *testing.T) {
	t.Parallel()
	orig := RelayMessage{
		Type:    "test",
		From:    "node-1",
		To:      "node-2",
		Payload: mustMarshal(map[string]string{"key": "value"}),
	}
	data, err := json.Marshal(orig)
	require.NoError(t, err)

	var decoded RelayMessage
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, orig.Type, decoded.Type)
	assert.Equal(t, orig.From, decoded.From)
	assert.Equal(t, orig.To, decoded.To)
}

func TestRelayServer_Broadcast(t *testing.T) {
	rs := NewRelayServer(testLogger(), 0)
	err := rs.Start()
	require.NoError(t, err)
	defer rs.Stop()

	conn1, err := net.Dial("tcp", rs.Addr())
	require.NoError(t, err)
	defer conn1.Close()
	enc1 := json.NewEncoder(conn1)

	conn2, err := net.Dial("tcp", rs.Addr())
	require.NoError(t, err)
	defer conn2.Close()
	enc2 := json.NewEncoder(conn2)
	dec2 := json.NewDecoder(conn2)

	enc1.Encode(RelayMessage{
		Type:    "relay_register",
		Payload: mustMarshal(map[string]string{"id": "broadcaster"}),
	})
	enc2.Encode(RelayMessage{
		Type:    "relay_register",
		Payload: mustMarshal(map[string]string{"id": "listener"}),
	})
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 2, rs.ClientCount())

	enc1.Encode(RelayMessage{
		Type:    "relay_broadcast",
		Payload: []byte(`broadcast-test`),
	})

	var received RelayMessage
	dec2.Decode(&received)
	assert.Equal(t, "relay_deliver", received.Type)
	assert.Contains(t, string(received.Payload), "broadcast-test")
}

func TestEnvelope_JSON(t *testing.T) {
	t.Parallel()
	env := Envelope{
		Type:    "heartbeat",
		From:    "node-1",
		To:      "node-2",
		Payload: []byte(`{"key":"value"}`),
		SentAt:  time.Now().UTC().UnixNano(),
	}
	data, err := json.Marshal(env)
	require.NoError(t, err)

	var env2 Envelope
	err = json.Unmarshal(data, &env2)
	require.NoError(t, err)
	assert.Equal(t, env.Type, env2.Type)
	assert.Equal(t, env.From, env2.From)
}
