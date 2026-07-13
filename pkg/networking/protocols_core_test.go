package networking

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/naviNBRuas/APA/pkg/update"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMultiProtocolManager_NilLogger(t *testing.T) {
	mpm, err := NewMultiProtocolManager(nil, MultiProtocolConfig{})
	assert.Error(t, err)
	assert.Nil(t, mpm)
}

func TestNewMultiProtocolManager_EmptyProtocols(t *testing.T) {
	mpm, err := NewMultiProtocolManager(slog.Default(), MultiProtocolConfig{})
	require.NoError(t, err)
	require.NotNil(t, mpm)
	assert.NotNil(t, mpm.protocols)
	assert.NotNil(t, mpm.protocolHealth)
}

func TestGetActiveProtocol(t *testing.T) {
	mpm, err := NewMultiProtocolManager(slog.Default(), MultiProtocolConfig{})
	require.NoError(t, err)
	assert.Equal(t, ProtocolLibP2P, mpm.GetActiveProtocol())
}

func TestGetProtocolHealth(t *testing.T) {
	mpm, err := NewMultiProtocolManager(slog.Default(), MultiProtocolConfig{})
	require.NoError(t, err)
	health := mpm.GetProtocolHealth()
	assert.NotNil(t, health)
}

func TestStartStop(t *testing.T) {
	mpm, err := NewMultiProtocolManager(slog.Default(), MultiProtocolConfig{})
	require.NoError(t, err)
	assert.False(t, mpm.isRunning)

	err = mpm.Start()
	assert.NoError(t, err)
	assert.True(t, mpm.isRunning)

	err = mpm.Start()
	assert.Error(t, err)

	mpm.Stop()
	assert.False(t, mpm.isRunning)

	mpm.Stop()
}

func TestReceiveMessages(t *testing.T) {
	mpm, err := NewMultiProtocolManager(slog.Default(), MultiProtocolConfig{
		EnabledProtocols: []ProtocolType{ProtocolTCP},
	})
	require.NoError(t, err)
	ch := mpm.ReceiveMessages()
	assert.NotNil(t, ch)
	mpm.Stop()
}

func TestNewProtocolFailureDetector(t *testing.T) {
	logger := slog.Default()
	pfd := NewProtocolFailureDetector(logger, 5*time.Second)
	require.NotNil(t, pfd)
	assert.NotNil(t, pfd.failureHistory)
}

func TestNewProtocolSwitchingEngine(t *testing.T) {
	logger := slog.Default()
	pse := NewProtocolSwitchingEngine(logger, true)
	require.NotNil(t, pse)
	assert.NotNil(t, pse.switchingHistory)
}

func TestRecordSwitch(t *testing.T) {
	pse := NewProtocolSwitchingEngine(slog.Default(), false)
	event := &SwitchEvent{
		Timestamp:    time.Now(),
		FromProtocol: ProtocolTCP,
		ToProtocol:   ProtocolUDP,
		Reason:       "latency",
		Success:      true,
	}
	pse.RecordSwitch(event)
	require.Len(t, pse.switchingHistory, 1)
	assert.Equal(t, ProtocolUDP, pse.currentProtocol)
}

func TestRecordFailure(t *testing.T) {
	pfd := NewProtocolFailureDetector(slog.Default(), 5*time.Second)
	event := &FailureEvent{
		Timestamp:   time.Now(),
		Protocol:    ProtocolTCP,
		FailureType: "connection_refused",
		Description: "connection refused",
		Severity:    SeverityHigh,
	}
	pfd.RecordFailure(event)
	require.Len(t, pfd.failureHistory[ProtocolTCP], 1)
}

func TestSelectBestRoute_NoRoutes(t *testing.T) {
	ire := NewIntelligentRoutingEngine(slog.Default())
	route := ire.SelectBestRoute("unknown-peer", &NetworkMessage{})
	assert.Nil(t, route)
}

func TestSelectBestRoute_WithRoutes(t *testing.T) {
	ire := NewIntelligentRoutingEngine(slog.Default())
	ire.routingTable["peer1"] = []RouteOption{
		{
			Protocol:    ProtocolTCP,
			Reliability: 0.9,
			Bandwidth:   100,
			Latency:     10 * time.Millisecond,
			Cost:        1.0,
		},
		{
			Protocol:    ProtocolUDP,
			Reliability: 0.8,
			Bandwidth:   200,
			Latency:     5 * time.Millisecond,
			Cost:        2.0,
		},
	}
	route := ire.SelectBestRoute("peer1", &NetworkMessage{})
	require.NotNil(t, route)
	assert.Equal(t, ProtocolUDP, route.Protocol)
}

func TestSelectBestRoute_EmptyRouteList(t *testing.T) {
	ire := NewIntelligentRoutingEngine(slog.Default())
	ire.routingTable["peer2"] = []RouteOption{}
	route := ire.SelectBestRoute("peer2", &NetworkMessage{})
	assert.Nil(t, route)
}

func TestRegisterPropagationHandler(t *testing.T) {
	p2p := &P2P{}
	handler := func(ctx context.Context, id peer.ID, payload PropagationPayload) error { return nil }
	p2p.RegisterPropagationHandler(handler)
	assert.NotNil(t, p2p.propagationHandler)
}

func TestSetFetchUpdateHandler(t *testing.T) {
	p2p := &P2P{}
	handler := func(version string) (*update.ReleaseInfo, []byte, error) { return nil, nil, nil }
	p2p.SetFetchUpdateHandler(handler)
	assert.NotNil(t, p2p.FetchUpdateHandler)
}

type protocolInfo struct {
	name     string
	factory  func() (interface{}, error)
	protoTyp ProtocolType
}

func TestNewTCPProtocol(t *testing.T) {
	p, err := NewTCPProtocol(slog.Default())
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.NotNil(t, p.messageChan)
	assert.NotNil(t, p.peers)
	assert.NotNil(t, p.healthMetrics)
}

func TestTCPProtocol_ReceiveMessages(t *testing.T) {
	p, err := NewTCPProtocol(slog.Default())
	require.NoError(t, err)
	ch := p.ReceiveMessages()
	assert.NotNil(t, ch)
}

func TestTCPProtocol_GetConnectionInfo(t *testing.T) {
	p, err := NewTCPProtocol(slog.Default())
	require.NoError(t, err)
	info := p.GetConnectionInfo()
	assert.NotNil(t, info)
	assert.Equal(t, ConnectionConnected, info.Status)
}

func TestTCPProtocol_GetHealthMetrics(t *testing.T) {
	p, err := NewTCPProtocol(slog.Default())
	require.NoError(t, err)
	metrics := p.GetHealthMetrics()
	assert.NotNil(t, metrics)
}

func TestTCPProtocol_RegisterPeerEndpoint(t *testing.T) {
	p, err := NewTCPProtocol(slog.Default())
	require.NoError(t, err)
	p.RegisterPeerEndpoint("peer1", "127.0.0.1:9001")
	addr, ok := p.peers["peer1"]
	assert.True(t, ok)
	assert.Equal(t, "127.0.0.1:9001", addr)
}

func TestTCPProtocol_Close(t *testing.T) {
	p, err := NewTCPProtocol(slog.Default())
	require.NoError(t, err)
	err = p.Close()
	assert.NoError(t, err)
}

func TestNewUDPProtocol(t *testing.T) {
	p, err := NewUDPProtocol(slog.Default())
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.NotNil(t, p.messageChan)
	assert.NotNil(t, p.peers)
	assert.NotNil(t, p.healthMetrics)
}

func TestUDPProtocol_ReceiveMessages(t *testing.T) {
	p, err := NewUDPProtocol(slog.Default())
	require.NoError(t, err)
	ch := p.ReceiveMessages()
	assert.NotNil(t, ch)
}

func TestUDPProtocol_GetConnectionInfo(t *testing.T) {
	p, err := NewUDPProtocol(slog.Default())
	require.NoError(t, err)
	info := p.GetConnectionInfo()
	assert.NotNil(t, info)
	assert.Equal(t, ConnectionConnected, info.Status)
}

func TestUDPProtocol_GetHealthMetrics(t *testing.T) {
	p, err := NewUDPProtocol(slog.Default())
	require.NoError(t, err)
	metrics := p.GetHealthMetrics()
	assert.NotNil(t, metrics)
}

func TestUDPProtocol_RegisterPeerEndpoint(t *testing.T) {
	p, err := NewUDPProtocol(slog.Default())
	require.NoError(t, err)
	p.RegisterPeerEndpoint("peer1", "127.0.0.1:9002")
	addr, ok := p.peers["peer1"]
	assert.True(t, ok)
	assert.Equal(t, "127.0.0.1:9002", addr.String())
}

func TestUDPProtocol_Close(t *testing.T) {
	p, err := NewUDPProtocol(slog.Default())
	require.NoError(t, err)
	err = p.Close()
	assert.NoError(t, err)
}

func TestNewHTTPProtocol(t *testing.T) {
	p, err := NewHTTPProtocol(slog.Default())
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.NotNil(t, p.messageChan)
	assert.NotNil(t, p.endpoints)
	assert.NotNil(t, p.client)
}

func TestHTTPProtocol_ReceiveMessages(t *testing.T) {
	p, err := NewHTTPProtocol(slog.Default())
	require.NoError(t, err)
	ch := p.ReceiveMessages()
	assert.NotNil(t, ch)
}

func TestHTTPProtocol_GetConnectionInfo(t *testing.T) {
	p, err := NewHTTPProtocol(slog.Default())
	require.NoError(t, err)
	info := p.GetConnectionInfo()
	assert.NotNil(t, info)
	assert.Equal(t, ConnectionConnected, info.Status)
}

func TestHTTPProtocol_GetHealthMetrics(t *testing.T) {
	p, err := NewHTTPProtocol(slog.Default())
	require.NoError(t, err)
	metrics := p.GetHealthMetrics()
	assert.NotNil(t, metrics)
}

func TestHTTPProtocol_RegisterPeerEndpoint(t *testing.T) {
	p, err := NewHTTPProtocol(slog.Default())
	require.NoError(t, err)
	p.RegisterPeerEndpoint("peer1", "http://127.0.0.1:9003")
	addr, ok := p.endpoints["peer1"]
	assert.True(t, ok)
	assert.Equal(t, "http://127.0.0.1:9003", addr)
}

func TestHTTPProtocol_Close(t *testing.T) {
	p, err := NewHTTPProtocol(slog.Default())
	require.NoError(t, err)
	err = p.Close()
	assert.NoError(t, err)
}

func TestNewWebSocketProtocol(t *testing.T) {
	p, err := NewWebSocketProtocol(slog.Default(), WebSocketConfig{})
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.NotNil(t, p.messageChan)
	assert.NotNil(t, p.connections)
	assert.NotNil(t, p.peers)
}

func TestWebSocketProtocol_ReceiveMessages(t *testing.T) {
	p, err := NewWebSocketProtocol(slog.Default(), WebSocketConfig{})
	require.NoError(t, err)
	ch := p.ReceiveMessages()
	assert.NotNil(t, ch)
}

func TestWebSocketProtocol_GetConnectionInfo(t *testing.T) {
	p, err := NewWebSocketProtocol(slog.Default(), WebSocketConfig{})
	require.NoError(t, err)
	info := p.GetConnectionInfo()
	assert.NotNil(t, info)
	assert.Equal(t, ConnectionStatus(""), info.Status)
}

func TestWebSocketProtocol_GetHealthMetrics(t *testing.T) {
	p, err := NewWebSocketProtocol(slog.Default(), WebSocketConfig{})
	require.NoError(t, err)
	metrics := p.GetHealthMetrics()
	assert.NotNil(t, metrics)
}

func TestWebSocketProtocol_RegisterPeerEndpoint(t *testing.T) {
	p, err := NewWebSocketProtocol(slog.Default(), WebSocketConfig{})
	require.NoError(t, err)
	p.RegisterPeerEndpoint("peer1", "ws://127.0.0.1:9004")
	addr, ok := p.peers["peer1"]
	assert.True(t, ok)
	assert.Equal(t, "ws://127.0.0.1:9004", addr)
}

func TestWebSocketProtocol_Close(t *testing.T) {
	p, err := NewWebSocketProtocol(slog.Default(), WebSocketConfig{})
	require.NoError(t, err)
	err = p.Close()
	assert.NoError(t, err)
}

func TestNewDNSProtocol(t *testing.T) {
	p, err := NewDNSProtocol(slog.Default(), DNSConfig{})
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.NotNil(t, p.messageChan)
	assert.NotNil(t, p.peers)
	assert.NotNil(t, p.client)
}

func TestDNSProtocol_DefaultDomain(t *testing.T) {
	p, err := NewDNSProtocol(slog.Default(), DNSConfig{})
	require.NoError(t, err)
	assert.Equal(t, "apa.dns", p.config.Domain)
}

func TestDNSProtocol_CustomDomain(t *testing.T) {
	p, err := NewDNSProtocol(slog.Default(), DNSConfig{Domain: "custom.test"})
	require.NoError(t, err)
	assert.Equal(t, "custom.test", p.config.Domain)
}

func TestDNSProtocol_ReceiveMessages(t *testing.T) {
	p, err := NewDNSProtocol(slog.Default(), DNSConfig{})
	require.NoError(t, err)
	ch := p.ReceiveMessages()
	assert.NotNil(t, ch)
}

func TestDNSProtocol_GetConnectionInfo(t *testing.T) {
	p, err := NewDNSProtocol(slog.Default(), DNSConfig{})
	require.NoError(t, err)
	info := p.GetConnectionInfo()
	assert.NotNil(t, info)
	assert.Equal(t, ConnectionConnected, info.Status)
}

func TestDNSProtocol_GetHealthMetrics(t *testing.T) {
	p, err := NewDNSProtocol(slog.Default(), DNSConfig{})
	require.NoError(t, err)
	metrics := p.GetHealthMetrics()
	assert.NotNil(t, metrics)
}

func TestDNSProtocol_RegisterPeerEndpoint(t *testing.T) {
	p, err := NewDNSProtocol(slog.Default(), DNSConfig{})
	require.NoError(t, err)
	p.RegisterPeerEndpoint("peer1", "10.0.0.1:9053")
	addr, ok := p.peers["peer1"]
	assert.True(t, ok)
	assert.Equal(t, "10.0.0.1:9053", addr)
}

func TestDNSProtocol_Close(t *testing.T) {
	p, err := NewDNSProtocol(slog.Default(), DNSConfig{})
	require.NoError(t, err)
	err = p.Close()
	assert.NoError(t, err)
}

func TestNewQUICProtocol(t *testing.T) {
	p, err := NewQUICProtocol(slog.Default(), QUICConfig{})
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.NotNil(t, p.messageChan)
	assert.NotNil(t, p.connections)
	assert.NotNil(t, p.peers)
	assert.NotNil(t, p.tlsConfig)
}

func TestQUICProtocol_ReceiveMessages(t *testing.T) {
	p, err := NewQUICProtocol(slog.Default(), QUICConfig{})
	require.NoError(t, err)
	ch := p.ReceiveMessages()
	assert.NotNil(t, ch)
}

func TestQUICProtocol_GetConnectionInfo(t *testing.T) {
	p, err := NewQUICProtocol(slog.Default(), QUICConfig{})
	require.NoError(t, err)
	info := p.GetConnectionInfo()
	assert.NotNil(t, info)
	assert.Equal(t, ConnectionConnected, info.Status)
}

func TestQUICProtocol_GetHealthMetrics(t *testing.T) {
	p, err := NewQUICProtocol(slog.Default(), QUICConfig{})
	require.NoError(t, err)
	metrics := p.GetHealthMetrics()
	assert.NotNil(t, metrics)
}

func TestQUICProtocol_RegisterPeerEndpoint(t *testing.T) {
	p, err := NewQUICProtocol(slog.Default(), QUICConfig{})
	require.NoError(t, err)
	p.RegisterPeerEndpoint("peer1", "127.0.0.1:9005")
	addr, ok := p.peers["peer1"]
	assert.True(t, ok)
	assert.Equal(t, "127.0.0.1:9005", addr)
}

func TestQUICProtocol_Close(t *testing.T) {
	p, err := NewQUICProtocol(slog.Default(), QUICConfig{})
	require.NoError(t, err)
	err = p.Close()
	assert.NoError(t, err)
}

func TestQUICProtocol_WithCustomConfig(t *testing.T) {
	p, err := NewQUICProtocol(slog.Default(), QUICConfig{ListenAddr: ":9000"})
	require.NoError(t, err)
	assert.Equal(t, ":9000", p.config.ListenAddr)
}

func TestNewLibP2PProtocol(t *testing.T) {
	p, err := NewLibP2PProtocol(slog.Default(), LibP2PConfig{})
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.NotNil(t, p.messageChan)
	assert.NotNil(t, p.peers)
	assert.NotNil(t, p.healthMetrics)
}

func TestLibP2PProtocol_ReceiveMessages(t *testing.T) {
	p, err := NewLibP2PProtocol(slog.Default(), LibP2PConfig{})
	require.NoError(t, err)
	ch := p.ReceiveMessages()
	assert.NotNil(t, ch)
}

func TestLibP2PProtocol_GetConnectionInfo_PreInit(t *testing.T) {
	p, err := NewLibP2PProtocol(slog.Default(), LibP2PConfig{})
	require.NoError(t, err)
	info := p.GetConnectionInfo()
	assert.Equal(t, ConnectionDisconnected, info.Status)
}

func TestLibP2PProtocol_GetHealthMetrics(t *testing.T) {
	p, err := NewLibP2PProtocol(slog.Default(), LibP2PConfig{})
	require.NoError(t, err)
	metrics := p.GetHealthMetrics()
	assert.NotNil(t, metrics)
}

func TestLibP2PProtocol_RegisterPeerEndpoint(t *testing.T) {
	p, err := NewLibP2PProtocol(slog.Default(), LibP2PConfig{})
	require.NoError(t, err)
	p.RegisterPeerEndpoint("peer1", "/ip4/127.0.0.1/tcp/9006")
	_, ok := p.peers["peer1"]
	assert.True(t, ok)
}

func TestLibP2PProtocol_Close(t *testing.T) {
	p, err := NewLibP2PProtocol(slog.Default(), LibP2PConfig{})
	require.NoError(t, err)
	err = p.Close()
	assert.NoError(t, err)
}

func TestLibP2PProtocol_WithCustomConfig(t *testing.T) {
	p, err := NewLibP2PProtocol(slog.Default(), LibP2PConfig{ProtocolID: "/custom/1.0.0"})
	require.NoError(t, err)
	assert.Equal(t, "/custom/1.0.0", p.config.ProtocolID)
}

func TestProtocolHealthMetrics(t *testing.T) {
	metrics := &ProtocolHealthMetrics{
		ProtocolType:        ProtocolTCP,
		ConnectionStatus:    ConnectionConnected,
		Latency:             10 * time.Millisecond,
		Throughput:          50.0,
		ErrorRate:           0.01,
		Availability:        0.99,
		ConsecutiveFailures: 0,
		TotalMessagesSent:   100,
		TotalMessagesRecv:   90,
		BytesTransmitted:    10000,
		BytesReceived:       9000,
	}
	assert.Equal(t, ProtocolTCP, metrics.ProtocolType)
	assert.Equal(t, ConnectionConnected, metrics.ConnectionStatus)
	assert.Equal(t, 100, int(metrics.TotalMessagesSent))
}

func TestConnectionInfo(t *testing.T) {
	info := &ConnectionInfo{
		LocalAddress:  "127.0.0.1:0",
		RemoteAddress: "10.0.0.1:9001",
		Protocol:      ProtocolTCP,
		Status:        ConnectionConnected,
	}
	assert.Equal(t, "127.0.0.1:0", info.LocalAddress)
	assert.Equal(t, ConnectionConnected, info.Status)
}

func TestNewIntelligentRoutingEngine(t *testing.T) {
	logger := slog.Default()
	ire := NewIntelligentRoutingEngine(logger)
	require.NotNil(t, ire)
	assert.NotNil(t, ire.routingTable)
	assert.NotNil(t, ire.performanceCache)
}

func TestForceProtocolSwitch_UnknownProtocol(t *testing.T) {
	mpm, err := NewMultiProtocolManager(slog.Default(), MultiProtocolConfig{})
	require.NoError(t, err)
	err = mpm.ForceProtocolSwitch("nonexistent")
	assert.Error(t, err)
}
