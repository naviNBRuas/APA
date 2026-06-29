# Autonomous Polymorphic Agent (APA)

[![CI](https://github.com/naviNBRuas/APA/workflows/CI/badge.svg)](https://github.com/naviNBRuas/APA/actions?query=workflow%3ACI)
[![Code Quality and Security](https://github.com/naviNBRuas/APA/workflows/Code%20Quality%20and%20Security/badge.svg)](https://github.com/naviNBRuas/APA/actions?query=workflow%3A%22Code+Quality+and+Security%22)
[![Release](https://github.com/naviNBRuas/APA/workflows/Release/badge.svg)](https://github.com/naviNBRuas/APA/actions?query=workflow%3ARelease)
[![Go Report Card](https://goreportcard.com/badge/github.com/naviNBRuas/APA)](https://goreportcard.com/report/github.com/naviNBRuas/APA)
[![License](https://img.shields.io/github/license/naviNBRuas/APA)](LICENSE)

## Overview

The **Autonomous Polymorphic Agent (APA)** is a self-healing, decentralized software agent platform designed for robust, secure, and autonomous operation across diverse computing environments. Built with Go, APA combines advanced networking, self-healing mechanisms, and modular controller architecture.

## Key Features

### Core Capabilities
- **Multi-Protocol Networking**: TCP, UDP, HTTP, WebSocket, and libp2p support with intelligent switching
- **Cross-Platform Compatibility**: Runs on Linux, macOS, and Windows across AMD64 and ARM64
- **Self-Healing**: Automatic fault detection, process restart, and recovery orchestration
- **Decentralized Design**: Peer-to-peer networking with DHT discovery and pubsub messaging
- **Modular Architecture**: Extensible through WASM modules and controller plugins
- **Security Framework**: End-to-end encryption, Ed25519 signature verification, OPA policy engine
- **Self-Updating**: Secure over-the-air updates with Ed25519-verified artifacts
- **Health Monitoring**: Continuous system health assessment with binary integrity checks
- **Ephemeral Identity**: Automatic identity rotation with HMAC-derived session keys
- **Admin API**: REST API for health, metrics, audit, and module management

### Architecture Highlights
- **Decentralized Design**: Peer-to-peer networking with DHT discovery
- **Microservices Pattern**: Modular components with clear separation of concerns
- **Event-Driven**: Reactive architecture for optimal performance

## Installation

### From Source

```bash
git clone https://github.com/naviNBRuas/APA.git
cd APA
go build -o apa cmd/standalone-agent/main.go
./apa --help
```

### Docker

```bash
docker build -t apa .
docker run -it apa --help
```

## Usage

### Command Line Options

```
Usage of ./apa:
  -config string
        Path to agent configuration YAML file (default "configs/agent.yaml")
  -version
        Show version information
```

### Configuration

Create a configuration file:

```yaml
log_level: info
peer_port: 9000
admin_port: 9001
bootstrap_peers:
  - /ip4/1.2.3.4/tcp/9000/p2p/QmPeerID...
```

Then run:

```bash
./apa --config=config.yaml
```

## Testing

### Run All Tests

```bash
go test ./...
```

### Run Specific Test Suites

```bash
# Unit tests
go test ./pkg/...

# Race condition detection
go test -race ./pkg/...

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### CI/CD Pipeline

The project includes comprehensive GitHub Actions workflows:

- **CI**: Build, test, and lint on every push/PR across Linux, macOS, and Windows
- **Code Quality**: golangci-lint, CodeQL static analysis, container vulnerability scan
- **Release**: Manual workflow_dispatch release with cross-platform artifact builds
- **Documentation**: Auto-generated API docs and coverage reports

## Documentation

In the [`docs/`](docs/) directory:

- [Project Overview](docs/PROJECT_DESCRIPTION.md)
- [Development Plan](docs/PROJECT_DEVELOPMENT_PLAN.md)
- [Networking Demo Summary](docs/NETWORKING_DEMO_SUMMARY.md)
- [Release Readiness Guide](docs/RELEASE_READINESS.md)

### Examples

Example modules and usage patterns are available in the [`examples/`](examples/) directory:

- WASM modules
- Network drivers
- Controller implementations

## Contributing

We welcome contributions! Please see [Contributing Guidelines](CONTRIBUTING.md).

### Development Setup

```bash
git clone https://github.com/naviNBRuas/APA.git
cd APA
go mod tidy
go test ./...
go build -o apa cmd/standalone-agent/main.go
```

### Code Quality Standards

- All code must pass `gofmt`, `govet`, and `golangci-lint`
- Tests should cover new functionality
- Security scans must pass

## Security

### Reporting Vulnerabilities

Please report security vulnerabilities to [founder@nbr.company](mailto:founder@nbr.company).

### Security Features

- End-to-end encryption via AES-GCM encrypted messenger
- Ed25519 signature verification for updates and modules
- OPA policy engine for authorization
- CodeQL static analysis in CI
- Binary integrity monitoring (SHA-256, 5-minute interval)
- Anti-analysis: debugger and sandbox detection
- Ephemeral identity rotation

## Project Status

[![CI](https://github.com/naviNBRuas/APA/workflows/CI/badge.svg)](https://github.com/naviNBRuas/APA/actions?query=workflow%3ACI)
[![CodeQL](https://github.com/naviNBRuas/APA/workflows/CodeQL/badge.svg)](https://github.com/naviNBRuas/APA/actions?query=workflow%3ACodeQL)
[![Go Report Card](https://goreportcard.com/badge/github.com/naviNBRuas/APA)](https://goreportcard.com/report/github.com/naviNBRuas/APA)

## License

MIT License — see [LICENSE](LICENSE).

## Acknowledgments

- [libp2p](https://libp2p.io/) — Decentralized networking
- [Go](https://golang.org/) — Programming language
- [Open Policy Agent](https://www.openpolicyagent.org/) — Policy engine
- [WebAssembly](https://webassembly.org/) — WASM module runtime

See [ACKNOWLEDGMENTS.md](ACKNOWLEDGMENTS.md) for a complete list.
