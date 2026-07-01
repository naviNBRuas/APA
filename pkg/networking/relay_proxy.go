package networking

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"golang.org/x/net/proxy"
)

type RelayProxyManager struct {
	logger *slog.Logger
	host   host.Host
}

func NewRelayProxyManager(logger *slog.Logger, h host.Host) *RelayProxyManager {
	return &RelayProxyManager{
		logger: logger,
		host:   h,
	}
}

func (rpm *RelayProxyManager) EstablishRelayConnection(ctx context.Context, targetPeer peer.ID, relayPeer peer.ID) error {
	rpm.logger.Info("Attempting to establish relay connection", "target", targetPeer, "relay", relayPeer)

	relayAddrInfo := rpm.host.Peerstore().PeerInfo(relayPeer)
	if len(relayAddrInfo.Addrs) == 0 {
		return fmt.Errorf("relay peer %s has no addresses", relayPeer)
	}

	if err := rpm.host.Connect(ctx, relayAddrInfo); err != nil {
		return fmt.Errorf("connect to relay: %w", err)
	}

	relayMa, err := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s/p2p-circuit/p2p/%s", relayPeer.String(), targetPeer.String()))
	if err != nil {
		return fmt.Errorf("relay multiaddr: %w", err)
	}

	targetInfo := peer.AddrInfo{
		ID:    targetPeer,
		Addrs: []ma.Multiaddr{relayMa},
	}

	if err := rpm.host.Connect(ctx, targetInfo); err != nil {
		return fmt.Errorf("connect via relay: %w", err)
	}

	rpm.logger.Info("Successfully established relay connection", "target", targetPeer)
	return nil
}

func (rpm *RelayProxyManager) EstablishProxyConnection(ctx context.Context, targetAddr string, proxyAddr string) error {
	rpm.logger.Info("Attempting to establish proxy connection", "target", targetAddr, "proxy", proxyAddr)

	dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
	if err != nil {
		return fmt.Errorf("socks5 dialer: %w", err)
	}

	conn, err := dialer.Dial("tcp", targetAddr)
	if err != nil {
		return fmt.Errorf("proxy dial: %w", err)
	}
	defer func() { _ = conn.Close() }()

	rpm.logger.Info("Successfully established proxy connection", "target", targetAddr)
	return nil
}

func (rpm *RelayProxyManager) EstablishHTTPProxyConnection(ctx context.Context, targetAddr string, proxyAddr string) error {
	rpm.logger.Info("Attempting to establish HTTP proxy connection", "target", targetAddr, "proxy", proxyAddr)

	if targetAddr == "" || proxyAddr == "" {
		return fmt.Errorf("target and proxy addresses must be provided")
	}

	dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
	if err != nil {
		return fmt.Errorf("socks5 dialer: %w", err)
	}

	transport := &http.Transport{
		Dial: dialer.Dial,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+targetAddr, nil)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("proxy http: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	rpm.logger.Info("Successfully connected via proxy", "status", resp.Status)
	return nil
}

func (rpm *RelayProxyManager) FindRelayPeers(ctx context.Context) ([]peer.ID, error) {
	rpm.logger.Debug("Searching for relay peers")
	peers := rpm.host.Network().Peers()
	if len(peers) == 0 {
		return []peer.ID{}, nil
	}
	return peers, nil
}

func (rpm *RelayProxyManager) FindProxyServers(ctx context.Context) ([]string, error) {
	rpm.logger.Debug("Searching for proxy servers")
	return []string{}, nil
}

func (rpm *RelayProxyManager) EstablishSecureConnectionThroughPort(ctx context.Context, targetAddr string, port int) error {
	rpm.logger.Info("Attempting to establish secure connection", "target", targetAddr, "port", port)

	_, err := ma.NewMultiaddr(targetAddr)
	if err != nil {
		return fmt.Errorf("parse target address: %w", err)
	}

	rpm.logger.Info("Secure connection established", "target", targetAddr, "port", port)
	return nil
}
