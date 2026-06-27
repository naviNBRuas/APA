//go:build enhanced

package networking

import (
	"context"
	"encoding/json"
	"net"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

type CommunicationProtocol interface {
	Initialize(ctx context.Context) error
	SendMessage(to peer.ID, message *NetworkMessage) error
	ReceiveMessages() <-chan *NetworkMessage
	GetConnectionInfo() *ConnectionInfo
	GetHealthMetrics() *ProtocolHealthMetrics
	Close() error
}

type ProtocolType string

const (
	ProtocolLibP2P    ProtocolType = "libp2p"
	ProtocolHTTP      ProtocolType = "http"
	ProtocolWebSocket ProtocolType = "websocket"
	ProtocolQUIC      ProtocolType = "quic"
	ProtocolTCP       ProtocolType = "tcp"
	ProtocolUDP       ProtocolType = "udp"
	ProtocolDNS       ProtocolType = "dns"
	ProtocolBluetooth ProtocolType = "bluetooth"
	ProtocolSatellite ProtocolType = "satellite"
)

type ProtocolHealthMetrics struct {
	ProtocolType        ProtocolType
	ConnectionStatus    ConnectionStatus
	Latency             time.Duration
	Throughput          float64
	ErrorRate           float64
	Availability        float64
	LastHealthCheck     time.Time
	ConsecutiveFailures int
	TotalMessagesSent   int64
	TotalMessagesRecv   int64
	BytesTransmitted    int64
	BytesReceived       int64
}

type NetworkMessage struct {
	ID        string                 `json:"id"`
	From      peer.ID                `json:"from"`
	To        peer.ID                `json:"to"`
	Type      MessageType            `json:"type"`
	Payload   json.RawMessage        `json:"payload"`
	Timestamp time.Time              `json:"timestamp"`
	Priority  MessagePriority        `json:"priority"`
	Protocol  ProtocolType           `json:"protocol"`
	Metadata  map[string]interface{} `json:"metadata"`
	Signature []byte                 `json:"signature,omitempty"`
}

type MessageType string

const (
	MessageTypePing        MessageType = "ping"
	MessageTypePong        MessageType = "pong"
	MessageTypeData        MessageType = "data"
	MessageTypeCommand     MessageType = "command"
	MessageTypeEvent       MessageType = "event"
	MessageTypeQuery       MessageType = "query"
	MessageTypeResponse    MessageType = "response"
	MessageTypeBroadcast   MessageType = "broadcast"
	MessageTypeHandshake   MessageType = "handshake"
	MessageTypeDisconnect  MessageType = "disconnect"
	MessageTypeHealthCheck MessageType = "health_check"
)

type FailureEvent struct {
	Timestamp   time.Time       `json:"timestamp"`
	Protocol    ProtocolType    `json:"protocol"`
	FailureType string          `json:"failure_type"`
	Description string          `json:"description"`
	Severity    FailureSeverity `json:"severity"`
}

type FailureSeverity string

const (
	SeverityLow      FailureSeverity = "low"
	SeverityMedium   FailureSeverity = "medium"
	SeverityHigh     FailureSeverity = "high"
	SeverityCritical FailureSeverity = "critical"
)

type ConnectionStatus string

const (
	ConnectionConnected    ConnectionStatus = "connected"
	ConnectionConnecting   ConnectionStatus = "connecting"
	ConnectionDisconnected ConnectionStatus = "disconnected"
	ConnectionFailed       ConnectionStatus = "failed"
)

type MessagePriority int

const (
	PriorityLow MessagePriority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

type SecurityLevel string

const (
	SecurityNone    SecurityLevel = "none"
	SecurityBasic   SecurityLevel = "basic"
	SecurityHigh    SecurityLevel = "high"
	SecurityMaximum SecurityLevel = "maximum"
)

type LoadBalancingStrategy string

const (
	StrategyRoundRobin LoadBalancingStrategy = "round_robin"
	StrategyLeastLoad  LoadBalancingStrategy = "least_load"
	StrategyRandom     LoadBalancingStrategy = "random"
	StrategyAdaptive   LoadBalancingStrategy = "adaptive"
)

type SecurityRequirements struct {
	MinimumEncryption SecurityLevel `yaml:"minimum_encryption"`
	RequireMTLS       bool          `yaml:"require_mtls"`
	RequireSignatures bool          `yaml:"require_signatures"`
}

type QoSRequirements struct {
	MinBandwidth   float64       `yaml:"min_bandwidth"`
	MaxLatency     time.Duration `yaml:"max_latency"`
	MinReliability float64       `yaml:"min_reliability"`
}

type ConnectionInfo struct {
	LocalAddress  string           `json:"local_address"`
	RemoteAddress string           `json:"remote_address"`
	Protocol      ProtocolType     `json:"protocol"`
	Status        ConnectionStatus `json:"status"`
	Established   time.Time        `json:"established"`
	LastActivity  time.Time        `json:"last_activity"`
}

type PeerConnection struct {
	PeerID     peer.ID            `json:"peer_id"`
	Connection net.Conn           `json:"connection"`
	Protocol   ProtocolType       `json:"protocol"`
	Connected  time.Time          `json:"connected"`
	LastSeen   time.Time          `json:"last_seen"`
	Metrics    *ConnectionMetrics `json:"metrics"`
}

type ConnectionMetrics struct {
	BytesSent     int64         `json:"bytes_sent"`
	BytesReceived int64         `json:"bytes_received"`
	MessagesSent  int64         `json:"messages_sent"`
	MessagesRecv  int64         `json:"messages_recv"`
	AverageRTT    time.Duration `json:"average_rtt"`
	PacketLoss    float64       `json:"packet_loss"`
}

type SwitchEvent struct {
	Timestamp    time.Time     `json:"timestamp"`
	FromProtocol ProtocolType  `json:"from_protocol"`
	ToProtocol   ProtocolType  `json:"to_protocol"`
	Reason       string        `json:"reason"`
	Success      bool          `json:"success"`
	Duration     time.Duration `json:"duration,omitempty"`
}

type RoutingStrategy string

const (
	RouteShortestPath RoutingStrategy = "shortest_path"
	RouteLeastCost    RoutingStrategy = "least_cost"
	RouteMostReliable RoutingStrategy = "most_reliable"
	RouteAdaptive     RoutingStrategy = "adaptive"
)

type SyncStrategy string

const (
	SyncPrimaryOnly SyncStrategy = "primary_only"
	SyncAll         SyncStrategy = "all"
	SyncMajority    SyncStrategy = "majority"
)
