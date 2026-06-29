package networking

import (
	"log/slog"
	"sync"
	"time"
)

type RedundancyManager struct {
	logger          *slog.Logger
	level           int
	activeChannels  map[ProtocolType]CommunicationProtocol
	backupChannels  map[ProtocolType]CommunicationProtocol
	synchronization *ChannelSynchronizer

	mu sync.RWMutex
}

type ChannelSynchronizer struct {
	logger       *slog.Logger
	primaryChan  CommunicationProtocol
	backupChans  []CommunicationProtocol
	syncStrategy SyncStrategy

	mu           sync.RWMutex
	lastSyncTime time.Time
}
