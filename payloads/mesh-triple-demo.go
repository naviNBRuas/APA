//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

// mesh-triple-demo spins three hosts and shows basic connectivity + DHT bootstrap.
func main() {
	ctx := context.Background()

	mk := func() (*dht.IpfsDHT, error) {
		h, err := libp2p.New()
		if err != nil {
			return nil, err
		}
		kdht, err := dht.New(ctx, h)
		if err != nil {
			return nil, err
		}
		return kdht, nil
	}

	d1, err := mk()
	must(err)
	defer d1.Host().Close()
	defer d1.Close()

	d2, err := mk()
	must(err)
	defer d2.Host().Close()
	defer d2.Close()

	d3, err := mk()
	must(err)
	defer d3.Host().Close()
	defer d3.Close()

	// Connect a small line: d1 <-> d2 <-> d3
	must(connect(ctx, d1.Host(), d2.Host()))
	must(connect(ctx, d2.Host(), d3.Host()))

	// Bootstrap all three
	must(d1.Bootstrap(ctx))
	must(d2.Bootstrap(ctx))
	must(d3.Bootstrap(ctx))

	time.Sleep(2 * time.Second)

	fmt.Printf("Peers seen by d1: %d\n", len(d1.Host().Network().Peers()))
	fmt.Printf("Peers seen by d2: %d\n", len(d2.Host().Network().Peers()))
	fmt.Printf("Peers seen by d3: %d\n", len(d3.Host().Network().Peers()))

	// Query for providers (none registered) to exercise DHT lookup path
	ctxQ, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	out := d2.FindProvidersAsync(ctxQ, peer.ID("nonexistent"), 1)
	for p := range out {
		fmt.Printf("Found provider: %s\n", p.ID)
	}

	fmt.Println("mesh-triple-demo complete")
}

func connect(ctx context.Context, a host.Host, b host.Host) error {
	return a.Connect(ctx, peer.AddrInfo{ID: b.ID(), Addrs: b.Addrs()})
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
