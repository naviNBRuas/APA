# ADR-001: Choose libp2p as the Primary Networking Layer

**Status:** Accepted  
**Date:** 2024 (retrospective)  
**Deciders:** Architecture team  
**Tags:** networking, p2p, transport

## Context

APA is a decentralized agent that must discover peers, exchange messages,
distribute WASM modules, propagate updates, and coordinate consensus —
all without a central server. The networking layer must support:

- **Peer discovery** (LAN, WAN, DHT-based)
- **NAT traversal** (relay, hole-punching, port mapping)
- **Stream multiplexing** across multiple transports
- **Pub/sub messaging** for heartbeats, module announcements, leader election
- **Encrypted, authenticated communications** by default
- **Cross-platform support** (Linux, macOS, Windows, ARM)

## Decision Drivers

- Must operate in fully decentralized (serverless) mode
- Must traverse NATs/firewalls without manual configuration
- Must support resource-constrained edge devices
- Must be embeddable as a Go library (no external daemons)
- Single codebase, single API surface for all transport types

## Considered Options

| Option | Why Not Chosen |
|--------|----------------|
| **gRPC** | Requires centralized service definitions and protobuf schemas. No P2P discovery, DHT, or NAT traversal. Client-server model at its core. Not suited for mesh topologies. |
| **Raw TCP** | No built-in stream multiplexing, encryption, peer identity, or hole-punching. Would require implementing all transport security, peer addressing, and NAT traversal from scratch — a full protocol stack reimplementation. |
| **WebRTC** | Excellent browser-to-browser, but requires complex signaling, has no DHT or pub/sub, and has poor server-side performance. Used indirectly via pion/webrtc as a fallback transport. |
| **HTTP/2** | Client-server model. No P2P primitives, discovery, or NAT traversal. |
| **libp2p** | Provides all required primitives in a single, embeddable Go library. |

## Decision

Use **libp2p (go-libp2p)** as the primary networking layer, with a
MultiProtocolManager that can fall back to QUIC, WebSocket, HTTP, TCP, or
UDP when libp2p is unavailable.

## Consequences

### Positive

- Single library provides DHT, GossipSub, circuit relay, hole-punching, mDNS,
  stream multiplexing, and cryptographic peer identity
- Multiple transports (TCP, QUIC, WebSocket, WebRTC, WebTransport) under one
  API via multiaddr
- NAT traversal built in (relay, hole-punch, UPnP)
- Peer identity is cryptographic by default (Ed25519 or RSA keys)
- Battle-tested by IPFS, Filecoin, and Ethereum ecosystems

### Negative

- go-libp2p is a large dependency (100+ transitive Go modules)
- API surface is complex; steep learning curve for new contributors
- Some advanced features (e.g., relay manager, selective forwarding) needed
  custom wrappers beyond what libp2p provides out of the box
- DHT bootstrap latency on cold start

### Mitigations

- `minimal` build tag strips all enhanced networking to produce smaller
  binaries for edge/IoT targets
- MultiProtocolManager provides automatic failover to simpler protocols
- Testing harness uses docker-compose with dedicated p2pnode containers
