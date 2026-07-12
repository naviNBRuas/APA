# APA — Architecture Report

## 1. System Overview

APA is structured as a **modular, decentralized agent platform** with two build profiles:

```
┌────────────────────────────────────────────────────────┐
│                    APA Agent                           │
│                                                        │
│  ┌──────────────────────────────────────────────────┐  │
│  │            runtime.go (Orchestrator)              │  │
│  │  - Lifecycle management                          │  │
│  │  - Dependency injection                          │  │
│  │  - Health monitoring                             │  │
│  │  - Signal handling                               │  │
│  └──────┬──────┬──────┬──────┬──────┬──────┬───────┘  │
│         │      │      │      │      │      │          │
│  ┌──────┴┐ ┌───┴───┐ ┌┴─────┐ ┌┴────┐ ┌┴────┐ ┌────┐ │
│  │Netw. │ │Self-  │ │Poly- │ │Regen│ │Security│ │OPA │ │
│  │      │ │heal   │ │morph │ │     │ │        │ │    │ │
│  └──────┘ └───────┘ └──────┘ └─────┘ └────────┘ └────┘ │
│  ┌──────┐ ┌───────┐ ┌──────┐ ┌─────┐ ┌────────┐ ┌────┐ │
│  │Swarm │ │Persistence│ │EDR │ │Intel│ │Controller│ │... │
│  └──────┘ └───────┘ └──────┘ └─────┘ └────────┘ └────┘ │
└────────────────────────────────────────────────────────┘
```

## 2. Entry Points

| Command | Build Tag | Purpose |
|---------|-----------|---------|
| `cmd/agent/main.go` | none | Basic agent (minimal subsystems) |
| `cmd/enhanced-agent/main.go` | `enhanced` | Full-featured agent with all subsystems |
| `cmd/controller-manager/main.go` | none | External controller management |
| `cmd/health-check/main.go` | none | HTTP health endpoint |
| `cmd/swarm-node/main.go` | none | Standalone swarm peer |
| `cmd/seed-swarm/main.go` | none | Bootstrap seed node for swarm |

## 3. Core Subsystems (29 packages)

### 3.1 Working Subsystems

| Package | Status | Notes |
|---------|--------|-------|
| `agent` | ✅ Basic working | `runtime.go` orchestrates but is bloated |
| `networking` | ⚠️ Partial | libp2p protocols defined, handshake is no-op |
| `security` | ⚠️ Partial | TLS generation works, crypto has stubs |
| `persistence` | ✅ Working | SQLite + Badger backends functional |
| `health` | ✅ Working | HTTP health endpoint |
| `swarm` | ⚠️ Partial | Discovery works, consensus framework exists |
| `opa` | ⚠️ Partial | Engine loads Rego files, limited query support |
| `rbac` | ✅ Working | Basic role-based access control |
| `controller` | ⚠️ Partial | Controller loading works, WASM executor broken |
| `patch` | ✅ Working | Binary patching system |

### 3.2 Broken/Stub Subsystems

| Package | Status | Notes |
|---------|--------|-------|
| `polymorphic` | ❌ Stub | All 3 core methods are empty |
| `regeneration` | ⚠️ Partial | `CodeRegeneration` is dangerously incomplete |
| `selfhealing` | ❌ Partial | 2 of 5 strategies are no-ops |
| `injection` | ❌ Stub | Framework only, no actual injection logic |
| `module` | ⚠️ Partial | Module interface defined, WASM broken |
| `intelligence` | ⚠️ Partial | LLM client scaffolded, no real integration |
| `edr` | ⚠️ Partial | Basic monitoring only |
| `backup` | ⚠️ Partial | Framework exists, strategy implementation pending |
| `recovery` | ⚠️ Partial | Recovery procedures defined but mostly stubs |
| `robustness` | ⚠️ Partial | Fault injection framework exists |
| `update` | ❌ Stub | Update mechanism scaffolded but incomplete |
| `consensus` | ⚠️ Partial | Consensus interface defined, no implementation |
| `driver` | ❌ Stub | Driver abstraction framework |
| `platform` | ⚠️ Partial | OS detection and platform utilities |
| `testing` | ⚠️ Partial | Test helpers and comprehensive test suite (build-tag gated) |
| `policy` | ⚠️ Partial | Policy enforcer wraps OPA, most methods no-op |
| `controlplane` | ⚠️ Partial | Control plane framework scaffolded |
| `obfuscation` | ❌ Stub | Obfuscation framework, empty body |

## 4. Build Tag Architecture

The `//go:build enhanced` tag gates the "enhanced agent" profile:

```
//go:build enhanced

package agent

type EnhancedRuntime struct { ... }
func NewEnhancedRuntime() *EnhancedRuntime { ... }
```

**Impact**: `go build ./...` and `go test ./...` skip all `enhanced`-tagged files. This means:
- The "full" agent is never compiled or tested by default
- CI builds only test the basic agent
- Linting misses the enhanced code paths
- Any regression in enhanced code goes undetected

## 5. Data Flow

```
Config (Viper) → Runtime config → Subsystem initialization
    ↓
Runtime.Start() → goroutine per subsystem
    ↓
Networking ←→ Peers (libp2p)
Swarm ←→ Peer discovery
Persistence ←→ SQLite/Badger
OPA ←→ Rego policies
Controller ←→ Filesystem WASM files
```

## 6. Control Flow

```
Signal (SIGINT/SIGTERM)
    ↓
runtime.go signal handler
    ↓
Cancel context → propagate to all goroutines
    ↓
Shutdown subsystems in reverse order
    ↓
Exit
```

**Problem**: Multiple `Start*` methods ignore the cancel func, making the shutdown flow unreliable.

## 7. Dependency Graph

```
agent/runtime.go
  ├── networking (libp2p)
  ├── security (TLS, crypto)
  ├── persistence (SQLite, Badger)
  ├── selfhealing (5 strategies)
  ├── polymorphic (no-op)
  ├── regeneration (partial)
  ├── swarm (discovery, resources)
  ├── opa (policy engine)
  ├── rbac (authorization)
  ├── controller/manager
  ├── module/wasm (broken)
  ├── intelligence (LLM scaffold)
  ├── edr (monitoring)
  ├── update (stub)
  ├── health (HTTP)
  └── patch (binary patching)
```

The dependency graph is acyclic (except `testing/` which references `agent/`).

## 8. Architectural Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| runtime.go single point of failure | Agent crashes entirely | Modular bootstrap with recovery |
| No graceful degradation | Enhanced build failures = no agent | Fallback to basic mode |
| No service mesh/API gateway | Direct peer exposure | Add envoy/sidecar support |
| No circuit breakers | Cascading failures | Add resilience patterns |
| Build-tag code isolation | Untested enhanced code | Dual compilation in CI |
