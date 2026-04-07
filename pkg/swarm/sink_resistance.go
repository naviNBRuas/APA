package swarm

import (
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// RekeyFunc triggers network re-keying when suspicion is confirmed locally.
type RekeyFunc func()

// SuspicionSignal describes abnormal behavior detected locally.
type SuspicionSignal struct {
	Peer   peer.ID
	Source string
	Reason string
	At     time.Time
}

// SinkResistance coordinates local consensus around suspicious peers, evicting
// and triggering re-keying without any central validation.
type SinkResistance struct {
	tm      *TopologyManager
	window  time.Duration
	quorum  int
	rekeyFn RekeyFunc

	// peer -> source -> timestamp
	signals map[peer.ID]map[string]time.Time
}

// NewSinkResistance creates a new resistance module.
func NewSinkResistance(tm *TopologyManager, window time.Duration, quorum int, rekey RekeyFunc) *SinkResistance {
	if window <= 0 {
		window = 10 * time.Minute
	}
	if quorum <= 0 {
		quorum = 2
	}
	return &SinkResistance{
		tm:      tm,
		window:  window,
		quorum:  quorum,
		rekeyFn: rekey,
		signals: make(map[peer.ID]map[string]time.Time),
	}
}

// Observe registers a suspicion signal. It returns true if the quorum is met and action taken.
func (sr *SinkResistance) Observe(sig SuspicionSignal) bool {
	if sig.Peer == "" || sig.Source == "" {
		return false
	}
	if sig.At.IsZero() {
		sig.At = time.Now()
	}

	// prune old signals for this peer
	sr.prune(sig.Peer, sig.At)

	srcs, ok := sr.signals[sig.Peer]
	if !ok {
		srcs = make(map[string]time.Time)
		sr.signals[sig.Peer] = srcs
	}
	srcs[sig.Source] = sig.At

	if len(srcs) >= sr.quorum {
		sr.tm.RemovePeer(sig.Peer)
		if sr.rekeyFn != nil {
			sr.rekeyFn()
		}
		return true
	}
	return false
}

func (sr *SinkResistance) prune(pid peer.ID, now time.Time) {
	srcs, ok := sr.signals[pid]
	if !ok {
		return
	}
	for src, ts := range srcs {
		if now.Sub(ts) > sr.window {
			delete(srcs, src)
		}
	}
	if len(srcs) == 0 {
		delete(sr.signals, pid)
	}
}
