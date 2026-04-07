// Package networking provides advanced multi-protocol networking with redundancy and failover capabilities.
package networking

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/miekg/dns"
	"github.com/quic-go/quic-go"
)

// MultiProtocolManager handles multiple communication protocols with intelligent switching.
type MultiProtocolManager struct {
	logger          *slog.Logger
	config          MultiProtocolConfig
	protocols       map[ProtocolType]CommunicationProtocol
	activeProtocol  ProtocolType
	protocolHealth  map[ProtocolType]*ProtocolHealthMetrics
	routingEngine   *IntelligentRoutingEngine
	failureDetector *ProtocolFailureDetector
	switchingEngine *ProtocolSwitchingEngine

	mu        sync.RWMutex
	isRunning bool
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// ProtocolType represents different communication protocols.
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

// MultiProtocolConfig holds configuration for multi-protocol networking.
type MultiProtocolConfig struct {
	EnabledProtocols      []ProtocolType        `yaml:"enabled_protocols"`
	ProtocolPriorities    map[ProtocolType]int  `yaml:"protocol_priorities"`
	HealthCheckInterval   time.Duration         `yaml:"health_check_interval"`
	FailoverTimeout       time.Duration         `yaml:"failover_timeout"`
	LoadBalancingStrategy LoadBalancingStrategy `yaml:"load_balancing_strategy"`
	SecurityRequirements  SecurityRequirements  `yaml:"security_requirements"`
	QoSRequirements       QoSRequirements       `yaml:"qos_requirements"`
	AdaptiveSwitching     bool                  `yaml:"adaptive_switching"`
	RedundancyLevel       int                   `yaml:"redundancy_level"`
}

// CommunicationProtocol interface for different protocol implementations.
type CommunicationProtocol interface {
	Initialize(ctx context.Context) error
	SendMessage(to peer.ID, message *NetworkMessage) error
	ReceiveMessages() <-chan *NetworkMessage
	GetConnectionInfo() *ConnectionInfo
	GetHealthMetrics() *ProtocolHealthMetrics
	Close() error
}

// ProtocolHealthMetrics tracks protocol performance and reliability.
type ProtocolHealthMetrics struct {
	ProtocolType        ProtocolType
	ConnectionStatus    ConnectionStatus
	Latency             time.Duration
	Throughput          float64 // Mbps
	ErrorRate           float64
	Availability        float64 // 0-1 scale
	LastHealthCheck     time.Time
	ConsecutiveFailures int
	TotalMessagesSent   int64
	TotalMessagesRecv   int64
	BytesTransmitted    int64
	BytesReceived       int64
}

// NetworkMessage represents a standardized message format.
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

// MessageType categorizes different types of network messages.
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

// IntelligentRoutingEngine makes routing decisions based on network conditions.
type IntelligentRoutingEngine struct {
	logger           *slog.Logger
	routingTable     map[peer.ID][]RouteOption
	performanceCache map[string]*RoutePerformance
	strategy         RoutingStrategy

	mu sync.RWMutex
}

// RouteOption represents a potential routing path.
type RouteOption struct {
	Protocol     ProtocolType
	Address      string
	Cost         float64
	Reliability  float64
	Latency      time.Duration
	Bandwidth    float64
	Security     SecurityLevel
	LastUsed     time.Time
	SuccessCount int64
	FailureCount int64
}

// ProtocolFailureDetector monitors protocol failures and triggers failover.
type ProtocolFailureDetector struct {
	logger         *slog.Logger
	failureHistory map[ProtocolType][]*FailureEvent
	thresholds     FailureThresholds

	mu sync.RWMutex
}

// ProtocolSwitchingEngine handles intelligent protocol switching.
type ProtocolSwitchingEngine struct {
	logger            *slog.Logger
	switchingHistory  []*SwitchEvent
	switchingCriteria SwitchingCriteria
	cooldownPeriod    time.Duration

	mu              sync.RWMutex
	lastSwitchTime  time.Time
	currentProtocol ProtocolType
}

// Advanced protocol implementations

// LibP2PProtocol implements libp2p communication with enhancements.
type LibP2PProtocol struct {
	logger        *slog.Logger
	host          interface{} // Actual libp2p host implementation
	config        LibP2PConfig
	messageChan   chan *NetworkMessage
	healthMetrics *ProtocolHealthMetrics

	mu          sync.RWMutex
	isConnected bool
	peers       map[peer.ID]*PeerConnection
}

// HTTPProtocol implements HTTP-based communication.
type HTTPProtocol struct {
	logger        *slog.Logger
	client        *http.Client
	server        *http.Server
	config        HTTPConfig
	messageChan   chan *NetworkMessage
	healthMetrics *ProtocolHealthMetrics

	mu          sync.RWMutex
	endpoints   map[peer.ID]string
	activeConns map[string]*http.Response
}

// WebSocketProtocol implements WebSocket communication.
type WebSocketProtocol struct {
	logger        *slog.Logger
	upgrader      websocket.Upgrader
	connections   map[peer.ID]*websocket.Conn
	config        WebSocketConfig
	messageChan   chan *NetworkMessage
	healthMetrics *ProtocolHealthMetrics

	mu          sync.RWMutex
	isListening bool
}

// QUICProtocol implements QUIC-based communication.
type QUICProtocol struct {
	logger        *slog.Logger
	listener      quic.Listener
	connections   map[peer.ID]interface{}
	config        QUICConfig
	messageChan   chan *NetworkMessage
	healthMetrics *ProtocolHealthMetrics

	mu        sync.RWMutex
	tlsConfig *tls.Config
}

// TCPProtocol implements raw TCP communication.
type TCPProtocol struct {
	logger        *slog.Logger
	listener      net.Listener
	connections   map[peer.ID]net.Conn
	config        TCPConfig
	messageChan   chan *NetworkMessage
	healthMetrics *ProtocolHealthMetrics

	mu          sync.RWMutex
	isListening bool
}

// UDPProtocol implements UDP communication with reliability enhancements.
type UDPProtocol struct {
	logger        *slog.Logger
	conn          *net.UDPConn
	config        UDPConfig
	messageChan   chan *NetworkMessage
	healthMetrics *ProtocolHealthMetrics
	packetBuffer  map[string]*UDPPacket

	mu          sync.RWMutex
	isListening bool
}

// DNSProtocol implements DNS-based communication for covert channels.
type DNSProtocol struct {
	logger        *slog.Logger
	client        *dns.Client
	config        DNSConfig
	messageChan   chan *NetworkMessage
	healthMetrics *ProtocolHealthMetrics
	cache         map[string]*DNSCacheEntry

	mu sync.RWMutex
}

// RedundancyManager handles multiple simultaneous connections for reliability.
type RedundancyManager struct {
	logger          *slog.Logger
	level           int
	activeChannels  map[ProtocolType]CommunicationProtocol
	backupChannels  map[ProtocolType]CommunicationProtocol
	synchronization *ChannelSynchronizer

	mu sync.RWMutex
}

// ChannelSynchronizer ensures consistency across redundant channels.
type ChannelSynchronizer struct {
	logger       *slog.Logger
	primaryChan  CommunicationProtocol
	backupChans  []CommunicationProtocol
	syncStrategy SyncStrategy

	mu           sync.RWMutex
	lastSyncTime time.Time
}

// NewMultiProtocolManager creates a new multi-protocol networking manager.
func NewMultiProtocolManager(logger *slog.Logger, config MultiProtocolConfig) (*MultiProtocolManager, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	ctx, cancel := context.WithCancel(context.Background())

	mpm := &MultiProtocolManager{
		logger:         logger,
		config:         config,
		protocols:      make(map[ProtocolType]CommunicationProtocol),
		protocolHealth: make(map[ProtocolType]*ProtocolHealthMetrics),
		ctx:            ctx,
		cancel:         cancel,
	}

	// Initialize routing engine
	mpm.routingEngine = NewIntelligentRoutingEngine(logger)

	// Initialize failure detector
	mpm.failureDetector = NewProtocolFailureDetector(logger, config.FailoverTimeout)

	// Initialize switching engine
	mpm.switchingEngine = NewProtocolSwitchingEngine(logger, config.AdaptiveSwitching)

	// Initialize protocols based on configuration
	if err := mpm.initializeProtocols(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize protocols: %w", err)
	}

	logger.Info("Multi-protocol manager initialized",
		"protocols", len(config.EnabledProtocols),
		"adaptive_switching", config.AdaptiveSwitching,
		"redundancy_level", config.RedundancyLevel)

	return mpm, nil
}

// initializeProtocols sets up all configured communication protocols.
func (mpm *MultiProtocolManager) initializeProtocols() error {
	var errs []error

	for _, protoType := range mpm.config.EnabledProtocols {
		var protocol CommunicationProtocol
		var err error

		switch protoType {
		case ProtocolLibP2P:
			protocol, err = NewLibP2PProtocol(mpm.logger)
		case ProtocolHTTP:
			protocol, err = NewHTTPProtocol(mpm.logger)
		case ProtocolWebSocket:
			protocol, err = NewWebSocketProtocol(mpm.logger)
		case ProtocolQUIC:
			protocol, err = NewQUICProtocol(mpm.logger)
		case ProtocolTCP:
			protocol, err = NewTCPProtocol(mpm.logger)
		case ProtocolUDP:
			protocol, err = NewUDPProtocol(mpm.logger)
		case ProtocolDNS:
			protocol, err = NewDNSProtocol(mpm.logger)
		default:
			err = fmt.Errorf("unsupported protocol type: %s", protoType)
		}

		if err != nil {
			errs = append(errs, fmt.Errorf("failed to initialize %s protocol: %w", protoType, err))
			continue
		}

		if err := protocol.Initialize(mpm.ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to initialize %s protocol: %w", protoType, err))
			continue
		}

		mpm.protocols[protoType] = protocol
		mpm.protocolHealth[protoType] = &ProtocolHealthMetrics{
			ProtocolType:     protoType,
			ConnectionStatus: ConnectionDisconnected,
		}

		mpm.logger.Info("Protocol initialized", "protocol", protoType)
	}

	if len(errs) > 0 {
		return fmt.Errorf("protocol initialization errors: %v", errs)
	}

	// Set initial active protocol
	mpm.selectInitialProtocol()

	return nil
}

// selectInitialProtocol chooses the best protocol to start with.
func (mpm *MultiProtocolManager) selectInitialProtocol() {
	bestProtocol := ProtocolLibP2P // Default fallback

	// Find highest priority protocol that's available
	maxPriority := -1
	for protoType, protocol := range mpm.protocols {
		priority, exists := mpm.config.ProtocolPriorities[protoType]
		if exists && priority > maxPriority && protocol != nil {
			maxPriority = priority
			bestProtocol = protoType
		}
	}

	mpm.activeProtocol = bestProtocol
	mpm.logger.Info("Initial protocol selected", "protocol", bestProtocol)
}

// SendMessage sends a message using the optimal protocol.
func (mpm *MultiProtocolManager) SendMessage(to peer.ID, message *NetworkMessage) error {
	mpm.mu.RLock()
	if !mpm.isRunning {
		mpm.mu.RUnlock()
		return fmt.Errorf("multi-protocol manager is not running")
	}

	protocol := mpm.protocols[mpm.activeProtocol]
	mpm.mu.RUnlock()

	if protocol == nil {
		return fmt.Errorf("active protocol %s is not available", mpm.activeProtocol)
	}

	// Route through intelligent routing engine
	route := mpm.routingEngine.SelectBestRoute(to, message)
	if route != nil {
		// Use the recommended protocol from routing engine
		if altProto, exists := mpm.protocols[route.Protocol]; exists {
			protocol = altProto
		}
	}

	// Send message with redundancy if configured
	if mpm.config.RedundancyLevel > 0 {
		return mpm.sendWithRedundancy(to, message, protocol)
	}

	return protocol.SendMessage(to, message)
}

// sendWithRedundancy sends message through multiple protocols for reliability.
func (mpm *MultiProtocolManager) sendWithRedundancy(to peer.ID, message *NetworkMessage, primary CommunicationProtocol) error {
	var wg sync.WaitGroup
	errors := make(chan error, mpm.config.RedundancyLevel+1)

	// Send via primary protocol
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := primary.SendMessage(to, message); err != nil {
			errors <- fmt.Errorf("primary protocol failed: %w", err)
		}
	}()

	// Send via backup protocols
	backupCount := 0
	for protoType, protocol := range mpm.protocols {
		if protoType == mpm.activeProtocol || backupCount >= mpm.config.RedundancyLevel {
			continue
		}

		wg.Add(1)
		go func(pt ProtocolType, p CommunicationProtocol) {
			defer wg.Done()
			if err := p.SendMessage(to, message); err != nil {
				errors <- fmt.Errorf("backup protocol %s failed: %w", pt, err)
			}
		}(protoType, protocol)

		backupCount++
		if backupCount >= mpm.config.RedundancyLevel {
			break
		}
	}

	wg.Wait()
	close(errors)

	// Collect errors
	var errorList []error
	for err := range errors {
		errorList = append(errorList, err)
	}

	if len(errorList) > mpm.config.RedundancyLevel {
		return fmt.Errorf("message delivery failed: %v", errorList)
	}

	return nil
}

// ReceiveMessages returns a channel for receiving messages from all protocols.
func (mpm *MultiProtocolManager) ReceiveMessages() <-chan *NetworkMessage {
	messageChan := make(chan *NetworkMessage, 1000)

	mpm.mu.RLock()
	defer mpm.mu.RUnlock()

	// Start goroutines to receive from each protocol
	for protoType, protocol := range mpm.protocols {
		protoMsgChan := protocol.ReceiveMessages()

		mpm.wg.Add(1)
		go func(pt ProtocolType, ch <-chan *NetworkMessage) {
			defer mpm.wg.Done()

			for {
				select {
				case msg := <-ch:
					if msg != nil {
						// Add protocol information to message
						msg.Protocol = pt
						select {
						case messageChan <- msg:
						case <-mpm.ctx.Done():
							return
						}
					}
				case <-mpm.ctx.Done():
					return
				}
			}
		}(protoType, protoMsgChan)
	}

	return messageChan
}

// Start begins multi-protocol networking operations.
func (mpm *MultiProtocolManager) Start() error {
	mpm.mu.Lock()
	if mpm.isRunning {
		mpm.mu.Unlock()
		return fmt.Errorf("multi-protocol manager is already running")
	}
	mpm.isRunning = true
	mpm.mu.Unlock()

	mpm.logger.Info("Starting multi-protocol networking")

	// Start health monitoring
	mpm.wg.Add(1)
	go mpm.healthMonitoringLoop()

	// Start protocol switching logic
	mpm.wg.Add(1)
	go mpm.protocolSwitchingLoop()

	// Start failure detection
	mpm.wg.Add(1)
	go mpm.failureDetectionLoop()

	return nil
}

// Stop gracefully shuts down all protocols and networking operations.
func (mpm *MultiProtocolManager) Stop() {
	mpm.mu.Lock()
	if !mpm.isRunning {
		mpm.mu.Unlock()
		return
	}
	mpm.isRunning = false
	mpm.mu.Unlock()

	mpm.logger.Info("Stopping multi-protocol networking")

	// Cancel context to stop all goroutines
	mpm.cancel()

	// Close all protocols
	mpm.mu.RLock()
	for protoType, protocol := range mpm.protocols {
		if err := protocol.Close(); err != nil {
			mpm.logger.Error("Failed to close protocol", "protocol", protoType, "error", err)
		}
	}
	mpm.mu.RUnlock()

	// Wait for all goroutines to finish
	mpm.wg.Wait()

	mpm.logger.Info("Multi-protocol networking stopped")
}

// healthMonitoringLoop continuously monitors protocol health.
func (mpm *MultiProtocolManager) healthMonitoringLoop() {
	defer mpm.wg.Done()

	ticker := time.NewTicker(mpm.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-mpm.ctx.Done():
			return
		case <-ticker.C:
			mpm.performHealthChecks()
		}
	}
}

// performHealthChecks evaluates all protocol health metrics.
func (mpm *MultiProtocolManager) performHealthChecks() {
	mpm.mu.RLock()
	defer mpm.mu.RUnlock()

	for protoType, protocol := range mpm.protocols {
		health := protocol.GetHealthMetrics()
		if health != nil {
			mpm.protocolHealth[protoType] = health

			// Check if current active protocol is unhealthy
			if protoType == mpm.activeProtocol && !mpm.isProtocolHealthy(health) {
				mpm.logger.Warn("Active protocol is unhealthy, considering switch",
					"protocol", protoType,
					"availability", health.Availability)

				// Trigger protocol switching evaluation
				go mpm.evaluateProtocolSwitch()
			}
		}
	}
}

// isProtocolHealthy determines if a protocol meets health thresholds.
func (mpm *MultiProtocolManager) isProtocolHealthy(health *ProtocolHealthMetrics) bool {
	if health.ConnectionStatus != ConnectionConnected {
		return false
	}

	// Check availability threshold (95% minimum)
	if health.Availability < 0.95 {
		return false
	}

	// Check error rate threshold (less than 5%)
	if health.ErrorRate > 0.05 {
		return false
	}

	// Check consecutive failures
	if health.ConsecutiveFailures > 3 {
		return false
	}

	return true
}

// evaluateProtocolSwitch determines if protocol switching is needed.
func (mpm *MultiProtocolManager) evaluateProtocolSwitch() {
	mpm.mu.Lock()
	defer mpm.mu.Unlock()

	bestProtocol := mpm.findBestProtocol()
	if bestProtocol != mpm.activeProtocol {
		mpm.switchProtocol(bestProtocol)
	}
}

// findBestProtocol selects the optimal protocol based on current conditions.
func (mpm *MultiProtocolManager) findBestProtocol() ProtocolType {
	var bestProtocol ProtocolType
	bestScore := -1.0

	for protoType, health := range mpm.protocolHealth {
		if !mpm.isProtocolHealthy(health) {
			continue
		}

		score := mpm.calculateProtocolScore(protoType, health)
		if score > bestScore {
			bestScore = score
			bestProtocol = protoType
		}
	}

	// Fallback to current if no better option found
	if bestProtocol == "" {
		bestProtocol = mpm.activeProtocol
	}

	return bestProtocol
}

// calculateProtocolScore computes a weighted score for protocol selection.
func (mpm *MultiProtocolManager) calculateProtocolScore(protoType ProtocolType, health *ProtocolHealthMetrics) float64 {
	// Base weights
	latencyWeight := 0.3
	throughputWeight := 0.25
	availabilityWeight := 0.25
	priorityWeight := 0.2

	// Normalize metrics (0-1 scale)
	normalizedLatency := 1.0 - clamp(health.Latency.Seconds()/2.0, 0, 1) // Assume 2s max acceptable
	normalizedThroughput := clamp(health.Throughput/100.0, 0, 1)         // Assume 100Mbps max
	normalizedAvailability := health.Availability

	// Get priority from config
	priority := 1.0
	if prio, exists := mpm.config.ProtocolPriorities[protoType]; exists {
		priority = float64(prio) / 10.0 // Normalize to 0-1 scale
	}

	score := (normalizedLatency * latencyWeight) +
		(normalizedThroughput * throughputWeight) +
		(normalizedAvailability * availabilityWeight) +
		(priority * priorityWeight)

	return score
}

// switchProtocol changes the active communication protocol.
func (mpm *MultiProtocolManager) switchProtocol(newProtocol ProtocolType) {
	oldProtocol := mpm.activeProtocol
	mpm.activeProtocol = newProtocol

	event := &SwitchEvent{
		Timestamp:    time.Now(),
		FromProtocol: oldProtocol,
		ToProtocol:   newProtocol,
		Reason:       "Performance degradation",
		Success:      true,
	}

	mpm.switchingEngine.RecordSwitch(event)

	mpm.logger.Info("Protocol switched",
		"from", oldProtocol,
		"to", newProtocol,
		"reason", event.Reason)
}

// protocolSwitchingLoop handles automatic protocol switching.
func (mpm *MultiProtocolManager) protocolSwitchingLoop() {
	defer mpm.wg.Done()

	ticker := time.NewTicker(30 * time.Second) // Evaluate switching every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-mpm.ctx.Done():
			return
		case <-ticker.C:
			if mpm.config.AdaptiveSwitching {
				mpm.evaluateProtocolSwitch()
			}
		}
	}
}

// failureDetectionLoop monitors for protocol failures.
func (mpm *MultiProtocolManager) failureDetectionLoop() {
	defer mpm.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-mpm.ctx.Done():
			return
		case <-ticker.C:
			mpm.detectProtocolFailures()
		}
	}
}

// detectProtocolFailures identifies and handles protocol failures.
func (mpm *MultiProtocolManager) detectProtocolFailures() {
	mpm.mu.RLock()
	defer mpm.mu.RUnlock()

	for protoType, health := range mpm.protocolHealth {
		if health.ConsecutiveFailures > 0 {
			failure := &FailureEvent{
				Timestamp:   time.Now(),
				Protocol:    protoType,
				FailureType: "ConnectionLoss",
				Description: fmt.Sprintf("Protocol has %d consecutive failures", health.ConsecutiveFailures),
				Severity:    mpm.determineFailureSeverity(health),
			}

			mpm.failureDetector.RecordFailure(failure)

			// Trigger immediate failover if severity is high
			if failure.Severity == SeverityCritical && protoType == mpm.activeProtocol {
				mpm.logger.Error("Critical protocol failure detected, triggering immediate failover",
					"protocol", protoType,
					"failures", health.ConsecutiveFailures)
				go mpm.evaluateProtocolSwitch()
			}
		}
	}
}

// determineFailureSeverity assesses the severity of a protocol failure.
func (mpm *MultiProtocolManager) determineFailureSeverity(health *ProtocolHealthMetrics) FailureSeverity {
	if health.ConsecutiveFailures >= 5 || health.Availability < 0.5 {
		return SeverityCritical
	}
	if health.ConsecutiveFailures >= 3 || health.Availability < 0.8 {
		return SeverityHigh
	}
	if health.ConsecutiveFailures >= 1 {
		return SeverityMedium
	}
	return SeverityLow
}

// GetActiveProtocol returns the currently active protocol.
func (mpm *MultiProtocolManager) GetActiveProtocol() ProtocolType {
	mpm.mu.RLock()
	defer mpm.mu.RUnlock()
	return mpm.activeProtocol
}

// GetProtocolHealth returns health metrics for all protocols.
func (mpm *MultiProtocolManager) GetProtocolHealth() map[ProtocolType]*ProtocolHealthMetrics {
	mpm.mu.RLock()
	defer mpm.mu.RUnlock()

	healthCopy := make(map[ProtocolType]*ProtocolHealthMetrics)
	for protoType, health := range mpm.protocolHealth {
		healthCopy[protoType] = &ProtocolHealthMetrics{
			ProtocolType:        health.ProtocolType,
			ConnectionStatus:    health.ConnectionStatus,
			Latency:             health.Latency,
			Throughput:          health.Throughput,
			ErrorRate:           health.ErrorRate,
			Availability:        health.Availability,
			LastHealthCheck:     health.LastHealthCheck,
			ConsecutiveFailures: health.ConsecutiveFailures,
			TotalMessagesSent:   health.TotalMessagesSent,
			TotalMessagesRecv:   health.TotalMessagesRecv,
			BytesTransmitted:    health.BytesTransmitted,
			BytesReceived:       health.BytesReceived,
		}
	}

	return healthCopy
}

// ForceProtocolSwitch forces switching to a specific protocol.
func (mpm *MultiProtocolManager) ForceProtocolSwitch(protocol ProtocolType) error {
	mpm.mu.Lock()
	defer mpm.mu.Unlock()

	if _, exists := mpm.protocols[protocol]; !exists {
		return fmt.Errorf("protocol %s is not available", protocol)
	}

	mpm.switchProtocol(protocol)
	return nil
}

// Utility functions

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// Supporting types and constants

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
	MinBandwidth   float64       `yaml:"min_bandwidth"` // Mbps
	MaxLatency     time.Duration `yaml:"max_latency"`
	MinReliability float64       `yaml:"min_reliability"` // 0-1 scale
}

type FailureThresholds struct {
	ConsecutiveFailures int           `yaml:"consecutive_failures"`
	TimeWindow          time.Duration `yaml:"time_window"`
	ErrorRateThreshold  float64       `yaml:"error_rate_threshold"`
}

type SwitchingCriteria struct {
	LatencyImprovementThreshold    float64       `yaml:"latency_improvement_threshold"`
	ThroughputImprovementThreshold float64       `yaml:"throughput_improvement_threshold"`
	AvailabilityThreshold          float64       `yaml:"availability_threshold"`
	MinTimeBetweenSwitches         time.Duration `yaml:"min_time_between_switches"`
}

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

type UDPPacket struct {
	ID        string    `json:"id"`
	Data      []byte    `json:"data"`
	Timestamp time.Time `json:"timestamp"`
	Sequence  int       `json:"sequence"`
	Ack       bool      `json:"ack"`
}

type DNSCacheEntry struct {
	Data      []byte    `json:"data"`
	Timestamp time.Time `json:"timestamp"`
	Expires   time.Time `json:"expires"`
	Hits      int       `json:"hits"`
}

type SyncStrategy string

const (
	SyncPrimaryOnly SyncStrategy = "primary_only"
	SyncAll         SyncStrategy = "all"
	SyncMajority    SyncStrategy = "majority"
)

// Factory functions for protocol implementations (will be implemented in separate files)
func NewLibP2PProtocol(logger *slog.Logger) (*LibP2PProtocol, error) {
	return &LibP2PProtocol{
		logger:        logger,
		messageChan:   make(chan *NetworkMessage, 100),
		peers:         make(map[peer.ID]*PeerConnection),
		healthMetrics: &ProtocolHealthMetrics{ProtocolType: ProtocolLibP2P},
	}, nil
}

func NewHTTPProtocol(logger *slog.Logger) (*HTTPProtocol, error) {
	return &HTTPProtocol{
		logger:        logger,
		client:        &http.Client{Timeout: 30 * time.Second},
		messageChan:   make(chan *NetworkMessage, 100),
		endpoints:     make(map[peer.ID]string),
		activeConns:   make(map[string]*http.Response),
		healthMetrics: &ProtocolHealthMetrics{ProtocolType: ProtocolHTTP},
	}, nil
}

func NewWebSocketProtocol(logger *slog.Logger) (*WebSocketProtocol, error) {
	return &WebSocketProtocol{
		logger: logger,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		connections:   make(map[peer.ID]*websocket.Conn),
		messageChan:   make(chan *NetworkMessage, 100),
		healthMetrics: &ProtocolHealthMetrics{ProtocolType: ProtocolWebSocket},
	}, nil
}

func NewQUICProtocol(logger *slog.Logger) (*QUICProtocol, error) {
	return &QUICProtocol{
		logger:        logger,
		messageChan:   make(chan *NetworkMessage, 100),
		connections:   make(map[peer.ID]interface{}),
		healthMetrics: &ProtocolHealthMetrics{ProtocolType: ProtocolQUIC},
	}, nil
}

func NewTCPProtocol(logger *slog.Logger) (*TCPProtocol, error) {
	return &TCPProtocol{
		logger:        logger,
		messageChan:   make(chan *NetworkMessage, 100),
		connections:   make(map[peer.ID]net.Conn),
		healthMetrics: &ProtocolHealthMetrics{ProtocolType: ProtocolTCP},
	}, nil
}

func NewUDPProtocol(logger *slog.Logger) (*UDPProtocol, error) {
	return &UDPProtocol{
		logger:        logger,
		messageChan:   make(chan *NetworkMessage, 1000),
		packetBuffer:  make(map[string]*UDPPacket),
		healthMetrics: &ProtocolHealthMetrics{ProtocolType: ProtocolUDP},
	}, nil
}

func NewDNSProtocol(logger *slog.Logger) (*DNSProtocol, error) {
	return &DNSProtocol{
		logger:        logger,
		client:        &dns.Client{Net: "udp"},
		messageChan:   make(chan *NetworkMessage, 100),
		cache:         make(map[string]*DNSCacheEntry),
		healthMetrics: &ProtocolHealthMetrics{ProtocolType: ProtocolDNS},
	}, nil
}

// Method implementations for protocol interfaces (will be expanded in separate files)
func (lp *LibP2PProtocol) Initialize(ctx context.Context) error                  { return nil }
func (lp *LibP2PProtocol) SendMessage(to peer.ID, message *NetworkMessage) error { return nil }
func (lp *LibP2PProtocol) ReceiveMessages() <-chan *NetworkMessage               { return lp.messageChan }
func (lp *LibP2PProtocol) GetConnectionInfo() *ConnectionInfo                    { return &ConnectionInfo{} }
func (lp *LibP2PProtocol) GetHealthMetrics() *ProtocolHealthMetrics              { return lp.healthMetrics }
func (lp *LibP2PProtocol) Close() error                                          { return nil }

func (hp *HTTPProtocol) Initialize(ctx context.Context) error                  { return nil }
func (hp *HTTPProtocol) SendMessage(to peer.ID, message *NetworkMessage) error { return nil }
func (hp *HTTPProtocol) ReceiveMessages() <-chan *NetworkMessage               { return hp.messageChan }
func (hp *HTTPProtocol) GetConnectionInfo() *ConnectionInfo                    { return &ConnectionInfo{} }
func (hp *HTTPProtocol) GetHealthMetrics() *ProtocolHealthMetrics              { return hp.healthMetrics }
func (hp *HTTPProtocol) Close() error                                          { return nil }

func (wp *WebSocketProtocol) Initialize(ctx context.Context) error                  { return nil }
func (wp *WebSocketProtocol) SendMessage(to peer.ID, message *NetworkMessage) error { return nil }
func (wp *WebSocketProtocol) ReceiveMessages() <-chan *NetworkMessage               { return wp.messageChan }
func (wp *WebSocketProtocol) GetConnectionInfo() *ConnectionInfo                    { return &ConnectionInfo{} }
func (wp *WebSocketProtocol) GetHealthMetrics() *ProtocolHealthMetrics              { return wp.healthMetrics }
func (wp *WebSocketProtocol) Close() error                                          { return nil }

func (qp *QUICProtocol) Initialize(ctx context.Context) error                  { return nil }
func (qp *QUICProtocol) SendMessage(to peer.ID, message *NetworkMessage) error { return nil }
func (qp *QUICProtocol) ReceiveMessages() <-chan *NetworkMessage               { return qp.messageChan }
func (qp *QUICProtocol) GetConnectionInfo() *ConnectionInfo                    { return &ConnectionInfo{} }
func (qp *QUICProtocol) GetHealthMetrics() *ProtocolHealthMetrics              { return qp.healthMetrics }
func (qp *QUICProtocol) Close() error                                          { return nil }

func (tp *TCPProtocol) Initialize(ctx context.Context) error                  { return nil }
func (tp *TCPProtocol) SendMessage(to peer.ID, message *NetworkMessage) error { return nil }
func (tp *TCPProtocol) ReceiveMessages() <-chan *NetworkMessage               { return tp.messageChan }
func (tp *TCPProtocol) GetConnectionInfo() *ConnectionInfo                    { return &ConnectionInfo{} }
func (tp *TCPProtocol) GetHealthMetrics() *ProtocolHealthMetrics              { return tp.healthMetrics }
func (tp *TCPProtocol) Close() error                                          { return nil }

func (up *UDPProtocol) Initialize(ctx context.Context) error                  { return nil }
func (up *UDPProtocol) SendMessage(to peer.ID, message *NetworkMessage) error { return nil }
func (up *UDPProtocol) ReceiveMessages() <-chan *NetworkMessage               { return up.messageChan }
func (up *UDPProtocol) GetConnectionInfo() *ConnectionInfo                    { return &ConnectionInfo{} }
func (up *UDPProtocol) GetHealthMetrics() *ProtocolHealthMetrics              { return up.healthMetrics }
func (up *UDPProtocol) Close() error                                          { return nil }

func (dp *DNSProtocol) Initialize(ctx context.Context) error                  { return nil }
func (dp *DNSProtocol) SendMessage(to peer.ID, message *NetworkMessage) error { return nil }
func (dp *DNSProtocol) ReceiveMessages() <-chan *NetworkMessage               { return dp.messageChan }
func (dp *DNSProtocol) GetConnectionInfo() *ConnectionInfo                    { return &ConnectionInfo{} }
func (dp *DNSProtocol) GetHealthMetrics() *ProtocolHealthMetrics              { return dp.healthMetrics }
func (dp *DNSProtocol) Close() error                                          { return nil }

// Supporting component factory functions
func NewIntelligentRoutingEngine(logger *slog.Logger) *IntelligentRoutingEngine {
	return &IntelligentRoutingEngine{
		logger:           logger,
		routingTable:     make(map[peer.ID][]RouteOption),
		performanceCache: make(map[string]*RoutePerformance),
	}
}

func NewProtocolFailureDetector(logger *slog.Logger, timeout time.Duration) *ProtocolFailureDetector {
	return &ProtocolFailureDetector{
		logger:         logger,
		failureHistory: make(map[ProtocolType][]*FailureEvent),
		thresholds:     FailureThresholds{ConsecutiveFailures: 3, TimeWindow: timeout, ErrorRateThreshold: 0.1},
	}
}

func NewProtocolSwitchingEngine(logger *slog.Logger, adaptive bool) *ProtocolSwitchingEngine {
	return &ProtocolSwitchingEngine{
		logger:           logger,
		switchingHistory: make([]*SwitchEvent, 0),
		switchingCriteria: SwitchingCriteria{
			LatencyImprovementThreshold:    0.2,
			ThroughputImprovementThreshold: 0.15,
			AvailabilityThreshold:          0.95,
			MinTimeBetweenSwitches:         5 * time.Minute,
		},
		cooldownPeriod: 30 * time.Second,
	}
}

func (ire *IntelligentRoutingEngine) SelectBestRoute(to peer.ID, message *NetworkMessage) *RouteOption {
	return nil
}

func (pse *ProtocolSwitchingEngine) RecordSwitch(event *SwitchEvent) {
	pse.mu.Lock()
	defer pse.mu.Unlock()
	pse.switchingHistory = append(pse.switchingHistory, event)
	pse.lastSwitchTime = event.Timestamp
	pse.currentProtocol = event.ToProtocol
}

func (pfd *ProtocolFailureDetector) RecordFailure(event *FailureEvent) {
	pfd.mu.Lock()
	defer pfd.mu.Unlock()
	pfd.failureHistory[event.Protocol] = append(pfd.failureHistory[event.Protocol], event)
}

// Placeholder types for compilation
type LibP2PConfig struct{}
type HTTPConfig struct{}
type WebSocketConfig struct{}
type QUICConfig struct{}
type TCPConfig struct{}
type UDPConfig struct{}
type DNSConfig struct{}
type RoutePerformance struct{}
