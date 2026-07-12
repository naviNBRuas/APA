# APA — Long-Term Architecture Improvement Proposal

---

## Vision

Transform APA from a solo-developer proof-of-concept into a production-grade, extensible, autonomous agent platform suitable for team development, third-party contributions, and real-world deployment.

---

## Proposal 1: Modular Bootstrap Architecture

### Current State
`pkg/agent/runtime.go` (969 lines) handles: config loading, dependency wiring, subsystem startup, health monitoring, signal handling, error aggregation, and shutdown orchestration.

### Proposed Architecture

```
pkg/agent/
├── bootstrap/
│   ├── loader.go       // Config loading, validation
│   ├── injector.go     // Dependency injection
│   └── bootstrap.go    // Entry point orchestration
├── lifecycle/
│   ├── manager.go      // Subsystem lifecycle (start/stop/restart)
│   ├── health.go       // Health monitoring per subsystem
│   └── shutdown.go     // Graceful shutdown coordinator
└── runtime.go          // Thin wrapper (~100 lines)
```

**Benefits**: Testable in isolation, replaceable bootstrap implementations, clear responsibility boundaries.

---

## Proposal 2: Plugin Architecture via WASM

### Current State
WASM module loading is broken. Controller loading works but is tightly coupled to the filesystem.

### Proposed Architecture
```
pkg/plugin/
├── registry.go        // Plugin registry (wasm + native)
├── wasm/
│   ├── runtime.go     // wazero integration
│   ├── sdk.go         // Host function SDK for WASM plugins
│   └── sandbox.go     // Resource limits, syscall filtering
├── native/
│   └── loader.go      // Shared library loading (go plugin)
└── manifest.go        // Plugin metadata + signature verification
```

**Benefits**: First-class extensibility, sandboxed execution, signed module distribution, plugin marketplace potential.

---

## Proposal 3: Event-Driven Architecture

### Current State
Subsystems communicate through direct method calls orchestrated by `runtime.go`. No event bus, no pub/sub.

### Proposed Architecture
```
pkg/events/
├── bus.go             // In-memory event bus
├── types.go           // Event type definitions
├── subscriptions.go   // Subscription management
└── middleware.go       // Logging, metrics, retry middleware
```

Each subsystem publishes events (`.OnPeerConnected`, `.OnPolicyViolation`, `.OnHealthDegraded`) and subscribes to relevant events. This decouples subsystems and enables reactive behavior.

**Benefits**: Looser coupling, easier to add new subsystems (just subscribe to events), built-in audit trail.

---

## Proposal 4: Service Mesh Pattern for Networking

### Current State
Direct libp2p connections with 3+ abstraction layers. No service discovery, load balancing, or circuit breaking.

### Proposed Architecture
```
pkg/networking/
├── mesh/
│   ├── discovery.go   // Service discovery (DHT, mDNS)
│   ├── routing.go     // Message routing
│   ├── lb.go          // Load balancing
│   └── circuit.go     // Circuit breaker
├── protocols/         // Keep existing protocol abstraction
└── transport/         // Keep existing transport layer
```

**Benefits**: Resilient multi-hop communication, automatic failover, connection pooling.

---

## Proposal 5: Structured Configuration with Validation

### Current State
Viper-based config loaded from YAML with no schema validation.

### Proposed Architecture
```go
// pkg/agent/config.go
type Config struct {
    Agent     AgentConfig     `yaml:"agent" validate:"required"`
    Networking NetworkingConfig `yaml:"networking" validate:"required"`
    Security  SecurityConfig  `yaml:"security" validate:"required"`
    Features  FeatureFlags    `yaml:"features"`
}

type SecurityConfig struct {
    TLSEnabled  bool   `yaml:"tls_enabled"`
    CertPath    string `yaml:"cert_path" validate:"required_if=TLSEnabled true"`
    AdminAPIKey string `yaml:"-"` // Never serialized to YAML
}
```

**Benefits**: Type-safe configuration, validation at startup, no runtime surprises, clear documentation via struct tags.

---

## Proposal 6: Observability Stack

### Current State
Zap logging, Prometheus metrics endpoint scaffolded.

### Add
```
pkg/observability/
├── logging/      // Structured logging middleware
├── metrics/      // Prometheus + custom metrics
├── tracing/      // OpenTelemetry distributed tracing
└── profiling/    // pprof endpoints
```

**Benefits**: Distributed tracing across swarm, performance bottleneck identification, production debugging capability.

---

## Proposal 7: Formal State Machine for Agent Lifecycle

### Current State
Agent states are implicit — running, degraded, recovering — but not formally modeled.

### Proposed
```go
type AgentState int
const (
    StateBootstrap  AgentState = iota
    StateRunning
    StateDegraded
    StateRecovering
    StateShuttingDown
)
```

A formal state machine ensures:
- Valid state transitions are enforced
- Health checks know what "healthy" means per state
- Recovery procedures are triggered automatically
- Monitoring/alerting can react to state changes

---

## Proposal 8: Multi-Stage Build Pipeline

### Current State
Single `Dockerfile`, single `Makefile`, build-tag gating for features.

### Proposed
```
Containerfile           → CI → Docker Hub / GHCR
├── base                → Alpine + Go runtime
├── agent              → Basic agent image
├── enhanced-agent     → Full-featured image
├── controller-manager → Controller management image
└── swarm-node         → Swarm peer image
```

Multi-architecture builds (linux/amd64, linux/arm64, linux/riscv64) for IoT/edge deployment.

---

## Migration Strategy

| Phase | Proposal | Timeline | Risk |
|:-----:|----------|:--------:|:----:|
| 1 | Modular bootstrap | Month 1 | Low |
| 2 | Event bus | Month 2 | Low |
| 3 | Config validation | Month 2 | Low |
| 4 | Observability | Month 3 | Medium |
| 5 | State machine | Month 3 | Medium |
| 6 | Service mesh | Month 4-5 | High |
| 7 | WASM plugin system | Month 5-6 | High |
| 8 | Multi-stage build | Month 6 | Low |

---

## Architectural Principles (Future)

1. **Every subsystem must work without the build tag system** — build tags are for optimization, not feature gating
2. **Every public API must have tests** — no untested interfaces
3. **Every configuration field must have a default and a validation rule**
4. **Subsystems communicate via events, not direct calls** — runtime.go is a bootstrap coordinator, not an orchestrator
5. **Stubs belong behind `EXPECTED` build tags or feature flags** — don't compile dead code into production binaries
6. **Documentation must match implementation** — if a feature is documented as "coming" not "completed", the docs must say so
