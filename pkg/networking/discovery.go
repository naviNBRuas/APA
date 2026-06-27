package networking

import (
	"context"
	"fmt"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	ma "github.com/multiformats/go-multiaddr"
)

// StartDiscovery starts the peer discovery process.
func (p *P2P) StartDiscovery(ctx context.Context) {
	p.logger.Info("Starting peer discovery")

	// Bootstrap the DHT
	if err := p.dht.Bootstrap(ctx); err != nil {
		p.logger.Error("Failed to bootstrap DHT", "error", err)
	}

	// Start advanced discovery
	if p.advancedDiscovery != nil {
		p.advancedDiscovery.Start(ctx)
	}

	// Start resilience loops to keep discovery alive and reconnect when isolated
	p.startResilienceLoops(ctx)
}

// connectToBootstrapPeers connects to the configured bootstrap peers.
func (p *P2P) connectToBootstrapPeers(ctx context.Context, bootstrapPeers []string) error {
	var peers []ma.Multiaddr

	if len(bootstrapPeers) == 0 {
		p.logger.Info("No bootstrap peers configured, using default IPFS bootstrap peers")
		peers = dht.DefaultBootstrapPeers
	} else {
		for _, addrStr := range bootstrapPeers {
			addr, err := ma.NewMultiaddr(addrStr)
			if err != nil {
				p.logger.Error("Failed to parse bootstrap peer address", "address", addrStr, "error", err)
				continue
			}
			peers = append(peers, addr)
		}
	}

	for _, addr := range peers {
		peerInfo, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			p.logger.Error("Failed to extract peer info from address", "address", addr.String(), "error", err)
			continue
		}

		p.host.Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, peerstore.PermanentAddrTTL)

		if err := p.host.Connect(ctx, *peerInfo); err != nil {
			p.logger.Debug("Failed to connect to bootstrap peer", "peer", peerInfo.ID, "error", err)
			ctxRetry, cancel := context.WithTimeout(ctx, 5*time.Second)
			retryErr := p.host.Connect(ctxRetry, *peerInfo)
			cancel()
			if retryErr != nil {
				p.logger.Debug("Retry connect to bootstrap peer failed", "peer", peerInfo.ID, "error", retryErr)
				continue
			}
		}
		p.logger.Info("Connected to bootstrap peer", "peer", peerInfo.ID)
	}

	return nil
}

// ConnectToAddrInfo dials a peer using the provided address info.
func (p *P2P) ConnectToAddrInfo(ctx context.Context, info peer.AddrInfo) error {
	if p == nil || p.host == nil {
		return fmt.Errorf("host not initialized")
	}
	return p.host.Connect(ctx, info)
}
