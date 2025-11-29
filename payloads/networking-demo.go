package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
)

func main() {
	ctx := context.Background()

	// Create two libp2p hosts
	host1, err := libp2p.New()
	if err != nil {
		log.Fatal(err)
	}
	defer host1.Close()

	host2, err := libp2p.New()
	if err != nil {
		log.Fatal(err)
	}
	defer host2.Close()

	// Create DHTs for both hosts
	dht1, err := dht.New(ctx, host1)
	if err != nil {
		log.Fatal(err)
	}

	dht2, err := dht.New(ctx, host2)
	if err != nil {
		log.Fatal(err)
	}

	// Connect host2 to host1
	addrInfo := peer.AddrInfo{
		ID:    host1.ID(),
		Addrs: host1.Addrs(),
	}

	err = host2.Connect(ctx, addrInfo)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Host1 ID: %s\n", host1.ID())
	fmt.Printf("Host2 ID: %s\n", host2.ID())
	fmt.Println("Hosts connected successfully!")

	// Bootstrap the DHTs
	dht1.Bootstrap(ctx)
	dht2.Bootstrap(ctx)

	// Create discovery services
	discovery1 := routing.NewRoutingDiscovery(dht1)
	discovery2 := routing.NewRoutingDiscovery(dht2)

	fmt.Printf("Discovery services created: %v, %v\n", discovery1, discovery2)

	// Advertise a service
	serviceTag := "apa-networking-demo"
	ttl, err := discovery1.Advertise(ctx, serviceTag)
	if err != nil {
		log.Printf("Failed to advertise service: %v", err)
	} else {
		fmt.Printf("Advertised service '%s' with TTL: %v\n", serviceTag, ttl)
	}

	// Wait a bit for advertisement to propagate
	time.Sleep(2 * time.Second)

	// Try to find providers
	fmt.Printf("Searching for providers of service '%s'...\n", serviceTag)
	providers := make(map[peer.ID]peer.AddrInfo)
	providerCh, err := discovery2.FindPeers(ctx, serviceTag)
	if err != nil {
		log.Printf("Failed to find peers: %v", err)
	} else {
		for provider := range providerCh {
			providers[provider.ID] = provider
			fmt.Printf("Found provider: %s\n", provider.ID)
		}
	}

	fmt.Printf("Found %d providers\n", len(providers))
	fmt.Println("Networking demo completed successfully!")
}