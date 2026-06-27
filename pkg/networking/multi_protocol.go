//go:build enhanced

package networking

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

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

	mpm.routingEngine = NewIntelligentRoutingEngine(logger)
	mpm.failureDetector = NewProtocolFailureDetector(logger, config.FailoverTimeout)
	mpm.switchingEngine = NewProtocolSwitchingEngine(logger, config.AdaptiveSwitching)

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

	mpm.selectInitialProtocol()

	return nil
}

func (mpm *MultiProtocolManager) selectInitialProtocol() {
	bestProtocol := ProtocolLibP2P

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

	route := mpm.routingEngine.SelectBestRoute(to, message)
	if route != nil {
		if altProto, exists := mpm.protocols[route.Protocol]; exists {
			protocol = altProto
		}
	}

	if mpm.config.RedundancyLevel > 0 {
		return mpm.sendWithRedundancy(to, message, protocol)
	}

	return protocol.SendMessage(to, message)
}

func (mpm *MultiProtocolManager) sendWithRedundancy(to peer.ID, message *NetworkMessage, primary CommunicationProtocol) error {
	var wg sync.WaitGroup
	errors := make(chan error, mpm.config.RedundancyLevel+1)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := primary.SendMessage(to, message); err != nil {
			errors <- fmt.Errorf("primary protocol failed: %w", err)
		}
	}()

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

	var errorList []error
	for err := range errors {
		errorList = append(errorList, err)
	}

	if len(errorList) > mpm.config.RedundancyLevel {
		return fmt.Errorf("message delivery failed: %v", errorList)
	}

	return nil
}

func (mpm *MultiProtocolManager) ReceiveMessages() <-chan *NetworkMessage {
	messageChan := make(chan *NetworkMessage, 1000)

	mpm.mu.RLock()
	defer mpm.mu.RUnlock()

	for protoType, protocol := range mpm.protocols {
		protoMsgChan := protocol.ReceiveMessages()

		mpm.wg.Add(1)
		go func(pt ProtocolType, ch <-chan *NetworkMessage) {
			defer mpm.wg.Done()

			for {
				select {
				case msg := <-ch:
					if msg != nil {
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

func (mpm *MultiProtocolManager) Start() error {
	mpm.mu.Lock()
	if mpm.isRunning {
		mpm.mu.Unlock()
		return fmt.Errorf("multi-protocol manager is already running")
	}
	mpm.isRunning = true
	mpm.mu.Unlock()

	mpm.logger.Info("Starting multi-protocol networking")

	mpm.wg.Add(1)
	go mpm.healthMonitoringLoop()

	mpm.wg.Add(1)
	go mpm.protocolSwitchingLoop()

	mpm.wg.Add(1)
	go mpm.failureDetectionLoop()

	return nil
}

func (mpm *MultiProtocolManager) Stop() {
	mpm.mu.Lock()
	if !mpm.isRunning {
		mpm.mu.Unlock()
		return
	}
	mpm.isRunning = false
	mpm.mu.Unlock()

	mpm.logger.Info("Stopping multi-protocol networking")

	mpm.cancel()

	mpm.mu.RLock()
	for protoType, protocol := range mpm.protocols {
		if err := protocol.Close(); err != nil {
			mpm.logger.Error("Failed to close protocol", "protocol", protoType, "error", err)
		}
	}
	mpm.mu.RUnlock()

	mpm.wg.Wait()

	mpm.logger.Info("Multi-protocol networking stopped")
}

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

func (mpm *MultiProtocolManager) performHealthChecks() {
	mpm.mu.RLock()
	defer mpm.mu.RUnlock()

	for protoType, protocol := range mpm.protocols {
		health := protocol.GetHealthMetrics()
		if health != nil {
			mpm.protocolHealth[protoType] = health

			if protoType == mpm.activeProtocol && !mpm.isProtocolHealthy(health) {
				mpm.logger.Warn("Active protocol is unhealthy, considering switch",
					"protocol", protoType,
					"availability", health.Availability)

				go mpm.evaluateProtocolSwitch()
			}
		}
	}
}

func (mpm *MultiProtocolManager) isProtocolHealthy(health *ProtocolHealthMetrics) bool {
	if health.ConnectionStatus != ConnectionConnected {
		return false
	}

	if health.Availability < 0.95 {
		return false
	}

	if health.ErrorRate > 0.05 {
		return false
	}

	if health.ConsecutiveFailures > 3 {
		return false
	}

	return true
}

func (mpm *MultiProtocolManager) evaluateProtocolSwitch() {
	mpm.mu.Lock()
	defer mpm.mu.Unlock()

	bestProtocol := mpm.findBestProtocol()
	if bestProtocol != mpm.activeProtocol {
		mpm.switchProtocol(bestProtocol)
	}
}

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

	if bestProtocol == "" {
		bestProtocol = mpm.activeProtocol
	}

	return bestProtocol
}

func (mpm *MultiProtocolManager) calculateProtocolScore(protoType ProtocolType, health *ProtocolHealthMetrics) float64 {
	latencyWeight := 0.3
	throughputWeight := 0.25
	availabilityWeight := 0.25
	priorityWeight := 0.2

	normalizedLatency := 1.0 - clamp(health.Latency.Seconds()/2.0, 0, 1)
	normalizedThroughput := clamp(health.Throughput/100.0, 0, 1)
	normalizedAvailability := health.Availability

	priority := 1.0
	if prio, exists := mpm.config.ProtocolPriorities[protoType]; exists {
		priority = float64(prio) / 10.0
	}

	score := (normalizedLatency * latencyWeight) +
		(normalizedThroughput * throughputWeight) +
		(normalizedAvailability * availabilityWeight) +
		(priority * priorityWeight)

	return score
}

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

func (mpm *MultiProtocolManager) protocolSwitchingLoop() {
	defer mpm.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
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

			if failure.Severity == SeverityCritical && protoType == mpm.activeProtocol {
				mpm.logger.Error("Critical protocol failure detected, triggering immediate failover",
					"protocol", protoType,
					"failures", health.ConsecutiveFailures)
				go mpm.evaluateProtocolSwitch()
			}
		}
	}
}

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

func (mpm *MultiProtocolManager) GetActiveProtocol() ProtocolType {
	mpm.mu.RLock()
	defer mpm.mu.RUnlock()
	return mpm.activeProtocol
}

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

func (mpm *MultiProtocolManager) ForceProtocolSwitch(protocol ProtocolType) error {
	mpm.mu.Lock()
	defer mpm.mu.Unlock()

	if _, exists := mpm.protocols[protocol]; !exists {
		return fmt.Errorf("protocol %s is not available", protocol)
	}

	mpm.switchProtocol(protocol)
	return nil
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
