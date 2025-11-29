package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
)

// handleStream handles incoming streams
func handleStream(s network.Stream) {
	fmt.Printf("Received stream from %s\n", s.Conn().RemotePeer())
	s.Reset()
}

// createHost creates a libp2p host with basic configuration
func createHost() (host.Host, error) {
	h, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
		libp2p.Ping(false),
	)
	return h, err
}

// connectHosts connects two hosts directly
func connectHosts(h1, h2 host.Host) error {
	addrInfo := peer.AddrInfo{
		ID:    h1.ID(),
		Addrs: h1.Addrs(),
	}
	return h2.Connect(context.Background(), addrInfo)
}

// setupDHT sets up a DHT for a host
func setupDHT(ctx context.Context, h host.Host, bootstrappers []peer.AddrInfo) (*dht.IpfsDHT, error) {
	kademliaDHT, err := dht.New(ctx, h)
	if err != nil {
		return nil, err
	}

	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		return nil, err
	}

	// Connect to bootstrap nodes if provided
	for _, addrInfo := range bootstrappers {
		if addrInfo.ID == h.ID() {
			continue // Skip connecting to ourselves
		}
		if err := h.Connect(ctx, addrInfo); err != nil {
			log.Printf("Failed to connect to bootstrap node: %v", err)
		} else {
			fmt.Printf("Connected to bootstrap node: %s\n", addrInfo.ID)
		}
	}

	return kademliaDHT, nil
}

func main() {
	ctx := context.Background()

	fmt.Println("=== APA Full Networking Demo ===")

	// Create two libp2p hosts
	fmt.Println("Creating libp2p hosts...")
	host1, err := createHost()
	if err != nil {
		log.Fatal(err)
	}
	defer host1.Close()

	host2, err := createHost()
	if err != nil {
		log.Fatal(err)
	}
	defer host2.Close()

	fmt.Printf("Host1 ID: %s\n", host1.ID())
	fmt.Printf("Host2 ID: %s\n", host2.ID())

	// Set up stream handlers
	host1.SetStreamHandler("/apa/demo/1.0.0", handleStream)
	host2.SetStreamHandler("/apa/demo/1.0.0", handleStream)

	// Connect the hosts directly
	fmt.Println("\nConnecting hosts directly...")
	err = connectHosts(host1, host2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Hosts connected successfully!")

	// Open a stream between the hosts
	fmt.Println("\nOpening stream between hosts...")
	stream, err := host1.NewStream(ctx, host2.ID(), "/apa/demo/1.0.0")
	if err != nil {
		log.Printf("Failed to open stream: %v", err)
	} else {
		fmt.Printf("Stream opened successfully to %s\n", host2.ID())
		stream.Reset()
	}

	// Set up DHTs for both hosts
	fmt.Println("\nSetting up DHTs...")
	dht1, err := setupDHT(ctx, host1, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer dht1.Close()

	dht2, err := setupDHT(ctx, host2, []peer.AddrInfo{{ID: host1.ID(), Addrs: host1.Addrs()}})
	if err != nil {
		log.Fatal(err)
	}
	defer dht2.Close()

	// Create discovery services
	fmt.Println("Creating discovery services...")
	discovery1 := routing.NewRoutingDiscovery(dht1)
	discovery2 := routing.NewRoutingDiscovery(dht2)

	// Advertise services
	fmt.Println("\nAdvertising services...")
	serviceTag := "apa-full-demo"
	ttl1, err := discovery1.Advertise(ctx, serviceTag)
	if err != nil {
		log.Printf("Host1 failed to advertise service: %v", err)
	} else {
		fmt.Printf("Host1 advertised service '%s' with TTL: %v\n", serviceTag, ttl1)
	}

	ttl2, err := discovery2.Advertise(ctx, serviceTag)
	if err != nil {
		log.Printf("Host2 failed to advertise service: %v", err)
	} else {
		fmt.Printf("Host2 advertised service '%s' with TTL: %v\n", serviceTag, ttl2)
	}

	// Wait for advertisements to propagate
	fmt.Println("\nWaiting for advertisements to propagate...")
	time.Sleep(3 * time.Second)

	// Try to find providers
	fmt.Printf("\nSearching for providers of service '%s'...\n", serviceTag)
	providers := make(map[peer.ID]peer.AddrInfo)
	providerCh, err := discovery2.FindPeers(ctx, serviceTag)
	if err != nil {
		log.Printf("Failed to find peers: %v", err)
	} else {
		count := 0
		for provider := range providerCh {
			providers[provider.ID] = provider
			fmt.Printf("Found provider: %s\n", provider.ID)
			count++
			if count >= 10 { // Limit the number of providers we process
				break
			}
		}
	}

	fmt.Printf("Found %d providers\n", len(providers))

	// Demonstrate peer information
	fmt.Println("\n=== Peer Information ===")
	fmt.Printf("Host1 has %d connections\n", len(host1.Network().Conns()))
	fmt.Printf("Host2 has %d connections\n", len(host2.Network().Conns()))

	// Show host addresses
	fmt.Printf("\nHost1 addresses:\n")
	for _, addr := range host1.Addrs() {
		fmt.Printf("  %s/p2p/%s\n", addr, host1.ID())
	}

	fmt.Printf("\nHost2 addresses:\n")
	for _, addr := range host2.Addrs() {
		fmt.Printf("  %s/p2p/%s\n", addr, host2.ID())
	}

	fmt.Println("\nFull networking demo completed successfully!")
	fmt.Println("This demonstrates the core networking capabilities of the APA agent:")
	fmt.Println("  - libp2p host creation and management")
	fmt.Println("  - Direct peer-to-peer connections")
	fmt.Println("  - Stream handling for communication")
	fmt.Println("  - DHT-based discovery and routing")
	fmt.Println("  - Service advertisement and discovery")
}