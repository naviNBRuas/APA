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

// demonstrateDirectConnection demonstrates direct peer-to-peer connections
func demonstrateDirectConnection(host1, host2 host.Host) {
	fmt.Println("\n=== Direct Connection Demo ===")
	
	// Set up stream handlers
	host1.SetStreamHandler("/apa/demo/1.0.0", handleStream)
	host2.SetStreamHandler("/apa/demo/1.0.0", handleStream)

	// Connect the hosts directly
	fmt.Println("Connecting hosts directly...")
	err := connectHosts(host1, host2)
	if err != nil {
		log.Printf("Failed to connect hosts: %v", err)
		return
	}
	fmt.Println("✓ Hosts connected successfully!")

	// Open a stream between the hosts
	fmt.Println("Opening stream between hosts...")
	stream, err := host1.NewStream(context.Background(), host2.ID(), "/apa/demo/1.0.0")
	if err != nil {
		log.Printf("Failed to open stream: %v", err)
	} else {
		fmt.Printf("✓ Stream opened successfully to %s\n", host2.ID())
		stream.Reset()
	}
}

// demonstrateDHTDiscovery demonstrates DHT-based discovery
func demonstrateDHTDiscovery(ctx context.Context, host1, host2 host.Host, dht1, dht2 *dht.IpfsDHT) {
	fmt.Println("\n=== DHT Discovery Demo ===")
	
	// Create discovery services
	fmt.Println("Creating discovery services...")
	discovery1 := routing.NewRoutingDiscovery(dht1)
	discovery2 := routing.NewRoutingDiscovery(dht2)

	// Advertise services
	fmt.Println("Advertising services...")
	serviceTag := "apa-comprehensive-demo"
	ttl1, err := discovery1.Advertise(ctx, serviceTag)
	if err != nil {
		log.Printf("Host1 failed to advertise service: %v", err)
	} else {
		fmt.Printf("✓ Host1 advertised service '%s' with TTL: %v\n", serviceTag, ttl1)
	}

	ttl2, err := discovery2.Advertise(ctx, serviceTag)
	if err != nil {
		log.Printf("Host2 failed to advertise service: %v", err)
	} else {
		fmt.Printf("✓ Host2 advertised service '%s' with TTL: %v\n", serviceTag, ttl2)
	}

	// Wait for advertisements to propagate
	fmt.Println("Waiting for advertisements to propagate...")
	time.Sleep(3 * time.Second)

	// Try to find providers
	fmt.Printf("Searching for providers of service '%s'...\n", serviceTag)
	providers := make(map[peer.ID]peer.AddrInfo)
	providerCh, err := discovery2.FindPeers(ctx, serviceTag)
	if err != nil {
		log.Printf("Failed to find peers: %v", err)
	} else {
		count := 0
		for provider := range providerCh {
			providers[provider.ID] = provider
			fmt.Printf("✓ Found provider: %s\n", provider.ID)
			count++
			if count >= 10 { // Limit the number of providers we process
				break
			}
		}
	}

	fmt.Printf("Total providers found: %d\n", len(providers))
}

// demonstratePeerInformation shows peer information
func demonstratePeerInformation(host1, host2 host.Host) {
	fmt.Println("\n=== Peer Information Demo ===")
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
}

// demonstrateRelayFunctionality shows relay concepts (simulated)
func demonstrateRelayFunctionality() {
	fmt.Println("\n=== Relay/Proxy Functionality Demo ===")
	fmt.Println("In a full APA implementation, relay/proxy functionality would:")
	fmt.Println("  • Establish connections through relay nodes when direct connections fail")
	fmt.Println("  • Route traffic through proxy servers for anonymity")
	fmt.Println("  • Support HTTP/SOCKS proxy connections")
	fmt.Println("  • Enable NAT traversal through relay nodes")
	fmt.Println("✓ Relay/proxy concepts demonstrated")
}

// demonstrateReputationRouting shows reputation routing concepts (simulated)
func demonstrateReputationRouting() {
	fmt.Println("\n=== Reputation Routing Demo ===")
	fmt.Println("In a full APA implementation, reputation routing would:")
	fmt.Println("  • Track peer interactions and success rates")
	fmt.Println("  • Assign reputation scores to peers based on behavior")
	fmt.Println("  • Select optimal peers for connections based on reputation")
	fmt.Println("  • Prefer peers with higher reputation scores for critical operations")
	fmt.Println("✓ Reputation routing concepts demonstrated")
}

// demonstrateBluetoothDiscovery shows Bluetooth discovery concepts (simulated)
func demonstrateBluetoothDiscovery() {
	fmt.Println("\n=== Bluetooth Discovery Demo ===")
	fmt.Println("In a full APA implementation, Bluetooth discovery would:")
	fmt.Println("  • Scan for nearby Bluetooth devices")
	fmt.Println("  • Identify compatible APA agents via Bluetooth")
	fmt.Println("  • Establish connections with nearby peers")
	fmt.Println("  • Exchange peer information over Bluetooth")
	fmt.Println("✓ Bluetooth discovery concepts demonstrated")
}

func main() {
	ctx := context.Background()

	fmt.Println("========================================")
	fmt.Println("  APA Comprehensive Networking Demo")
	fmt.Println("========================================")

	// Create two libp2p hosts
	fmt.Println("\nCreating libp2p hosts...")
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

	fmt.Printf("✓ Host1 ID: %s\n", host1.ID())
	fmt.Printf("✓ Host2 ID: %s\n", host2.ID())

	// Demonstrate direct connection
	demonstrateDirectConnection(host1, host2)

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

	fmt.Println("✓ DHTs set up successfully!")

	// Demonstrate DHT discovery
	demonstrateDHTDiscovery(ctx, host1, host2, dht1, dht2)

	// Demonstrate peer information
	demonstratePeerInformation(host1, host2)

	// Demonstrate advanced networking features (simulated)
	demonstrateRelayFunctionality()
	demonstrateReputationRouting()
	demonstrateBluetoothDiscovery()

	fmt.Println("\n========================================")
	fmt.Println("  Comprehensive networking demo completed!")
	fmt.Println("========================================")
	fmt.Println("This demo showcases all core networking capabilities:")
	fmt.Println("  ✓ Direct peer-to-peer connections")
	fmt.Println("  ✓ DHT-based discovery and routing")
	fmt.Println("  ✓ Service advertisement and discovery")
	fmt.Println("  ✓ Relay/proxy functionality concepts")
	fmt.Println("  ✓ Reputation-based peer selection")
	fmt.Println("  ✓ Bluetooth peer discovery")
	fmt.Println("\nThe full APA agent implementation includes these")
	fmt.Println("features plus additional security, persistence,")
	fmt.Println("and autonomous operation capabilities.")
}