package networking

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

// ConnectViaRelay dials a target peer using a relay multiaddr of the form
// /ip4/host/tcp/port/p2p/<relayID>. It will first connect to the relay, then
// to the target via /p2p-circuit.
func (p *P2P) ConnectViaRelay(ctx context.Context, relayAddr string, target peer.ID) error {
	if p == nil || p.host == nil {
		return fmt.Errorf("host not initialized")
	}

	relayMa, err := ma.NewMultiaddr(relayAddr)
	if err != nil {
		return fmt.Errorf("invalid relay multiaddr: %w", err)
	}

	relayInfo, err := peer.AddrInfoFromP2pAddr(relayMa)
	if err != nil {
		return fmt.Errorf("failed to parse relay addr info: %w", err)
	}

	if err := p.host.Connect(ctx, *relayInfo); err != nil {
		return fmt.Errorf("failed to connect to relay: %w", err)
	}

	circuitAddr, err := ma.NewMultiaddr(fmt.Sprintf("%s/p2p-circuit/p2p/%s", relayAddr, target))
	if err != nil {
		return fmt.Errorf("failed to build circuit addr: %w", err)
	}

	targetInfo := peer.AddrInfo{ID: target, Addrs: []ma.Multiaddr{circuitAddr}}
	if err := p.host.Connect(ctx, targetInfo); err != nil {
		return fmt.Errorf("failed to connect to target via relay: %w", err)
	}

	return nil
}
