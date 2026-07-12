# APA — Technical Debt Report

## Overview

Estimated effort to resolve all identified technical debt: **8-12 weeks (1-2 developers)**

---

## 1. Empty Method Bodies (Stub Debt)

These are methods with empty bodies or `return nil` that are documented as complete features.

| Location | Method | Lines of Stub | Impact | Est. Fix |
|----------|--------|:------------:|--------|----------|
| `pkg/polymorphic/engine.go` | `Obfuscate()` | ~5 | Core stealth feature non-functional | 2 weeks |
| `pkg/polymorphic/engine.go` | `Mutate()` | ~5 | Core stealth feature non-functional | 2 weeks |
| `pkg/polymorphic/engine.go` | `Reobfuscate()` | ~5 | Core stealth feature non-functional | 1 week |
| `pkg/selfhealing/memory_optimization.go` | `OptimizeMemory()` | ~10 | Auto-memory optimization missing | 3 days |
| `pkg/networking/advanced_protocol_manager.go` | `CommHandshake()` | ~20 | Protocol negotiation broken | 1 week |
| `pkg/networking/advanced_protocol_manager.go` | `ExecuteAndRecover()` | ~15 | Fault-tolerant execution missing | 1 week |
| `pkg/security/crypto.go` | `Encrypt()` | ~5 | Data-at-rest encryption missing | 2 days |
| `pkg/security/crypto.go` | `Decrypt()` | ~5 | Data-at-rest encryption missing | 2 days |
| `pkg/policy/enforcer.go` | Multiple no-op methods | ~30 | Policy enforcement non-functional | 1 week |
| `pkg/obfuscation/engine.go` | All methods | ~20 | Obfuscation non-functional | 1 week |

**Estimated fix time**: ~10 weeks

---

## 2. Dangerous Code

| Location | Issue | Risk | Fix |
|----------|-------|------|-----|
| `pkg/regeneration/strategies.go:50-120` | `exec.Command("ollama", "run", ...)` with hardcoded system prompt to rewrite Go files | Remote code execution | Remove or gate behind `--dangerous` flag and add validation |
| `pkg/security/tls.go:45` | `big.NewInt(1)` static serial number | Invalid X.509 (all certs same serial) | Use `crypto/rand` |
| `configs/agent-config.yaml:15` | Hardcoded admin API key | Credential leak | Use env vars, remove from repo |

---

## 3. Commented-Out Code

| File | Lines | Description |
|------|-------|-------------|
| `pkg/swarm/discovery.go` | ~15 | `AnnounceResource()` expiration logic commented out |
| `pkg/networking/advanced_protocol_manager.go` | ~30 | Legacy handshake logic commented out |
| `pkg/selfhealing/strategies.go` | ~10 | Old strategy execution flow |

---

## 4. Dead Code / Unused Assets

| Asset | Reason |
|-------|--------|
| `go.mod` `replace pkg/controller/manager => ./pkg/controller/manager` | Not used in any import |
| `go.mod` `replace pkg/controller/manifest => ./pkg/controller/manifest` | Not used in any import |
| `pkg/swarm/discovery.go:owners` | Append-only, never read |
| 1 of 9 networking protocol types | Defined but never instantiated |

---

## 5. Architectural Debt

| Issue | Details |
|-------|---------|
| `runtime.go` (969 lines) | Monolithic orchestrator — should be split into bootstrap/lifecycle/health packages |
| Build-tag gating | All "real" features behind `//go:build enhanced` — never compiled/tested by default |
| No dependency injection framework | Manual wiring in runtime.go, hard to test subsystems in isolation |
| No configuration validation | `agent-config.yaml` is parsed but values aren't validated (no zero-value checks, no required-field enforcement) |
| Multiple middleware layers | Networking has 3+ abstraction layers for 9 protocols where most are no-ops |

---

## 6. Testing Debt

| Gap | Impact |
|-----|--------|
| Only 2 test files in entire repo | No regression protection |
| No `enhanced`-tagged tests | Full agent path untested |
| No CI test execution | Tests (if they existed) would not run in CI |
| No benchmarks | No performance baseline |
| No integration tests | Cross-subsystem interactions untested |

---

## 7. Debt Prioritization

| Priority | Item | Effort | Impact |
|:--------:|------|--------|--------|
| P0 | Remove dangerous CodeRegeneration LLM exec | 1 hour | Critical security |
| P0 | Fix TLS serial number | 30 mins | Security compliance |
| P0 | Remove unused go.mod replace directives | 5 mins | Cleanliness |
| P1 | Implement or remove polymorphic stubs | 3 weeks | Feature credibility |
| P1 | Split runtime.go | 3 days | Maintainability |
| P1 | Add test coverage for core subsystems | 2 weeks | Reliability |
| P2 | Remove commented-out code | 1 hour | Cleanliness |
| P2 | Fix goroutine cancellation | 2 days | Reliability |
| P2 | Add input validation to WASM/controller paths | 2 days | Security |
| P3 | Implement memory optimization | 3 days | Performance |
| P3 | Add proper error handling (remove panics) | 1 week | Robustness |
