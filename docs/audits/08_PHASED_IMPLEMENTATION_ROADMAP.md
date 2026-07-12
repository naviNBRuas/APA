# APA — Phased Implementation Roadmap

**Total estimated effort**: 20-28 weeks (1 full-time developer)

---

## Phase 0: Emergency Fixes (Week 1)

| Task | Effort | Description |
|------|--------|-------------|
| P0-1 | 1 hr | Remove or gate `CodeRegeneration` LLM exec behind `--dangerous` flag |
| P0-2 | 30 min | Fix TLS serial number to use `crypto/rand` |
| P0-3 | 1 hr | Fix `.golangci.yml` — re-enable G404, G306 (fix violations or add `//nolint` locally) |
| P0-4 | 15 min | Remove unused `go.mod` `replace` directives |
| P0-5 | 1 hr | Remove hardcoded API key from `configs/agent-config.yaml`, move to env var |
| P0-6 | 1 day | Audit and fix goroutine cancellation — ensure all Start methods respect cancel func |
| P0-7 | 2 hr | Remove commented-out code blocks across the codebase |

**Phase 0 total**: ~2 days

---

## Phase 1: Foundation (Weeks 2–4)

| Task | Effort | Description |
|------|--------|-------------|
| P1-1 | 1 week | Split `runtime.go` into `bootstrap/`, `lifecycle/`, `health/` packages |
| P1-2 | 1 week | Add `stretchr/testify` and write unit tests for core subsystems: `security`, `persistence`, `health`, `rbac` |
| P1-3 | 3 days | Add CI job: `go test -tags=enhanced ./...` |
| P1-4 | 2 days | Implement `ReturnError()` or remove stubs in `policy/enforcer.go` |
| P1-5 | 3 days | Add configuration validation using `go-playground/validator` |
| P1-6 | 2 days | Write ADR-001 through ADR-005 (architecture decisions to date) |

**Phase 1 total**: ~3 weeks

---

## Phase 2: Core Functionality (Weeks 5–10)

| Task | Effort | Description |
|------|--------|-------------|
| P2-1 | 1 week | Implement basic polymorphic transformations (symbol obfuscation, string encryption) |
| P2-2 | 2 weeks | Fix WASM executor — integrate wazero properly, add module loading tests |
| P2-3 | 1 week | Implement `CommHandshake()` and `ExecuteAndRecover()` in networking |
| P2-4 | 1 week | Add peer authentication to swarm discovery (Ed25519 signatures) |
| P2-5 | 1 week | Implement actual memory optimization in selfhealing |
| P2-6 | 1 week | Add admin API authentication middleware + rate limiting + audit logging |
| P2-7 | 3 days | Add path input validation for controller/WASM module loading |

**Phase 2 total**: ~6 weeks

---

## Phase 3: Testing & Quality (Weeks 11–14)

| Task | Effort | Description |
|------|--------|-------------|
| P3-1 | 2 weeks | Write integration tests: networking (libp2p dial), OPA policy evaluation, persistence I/O |
| P3-2 | 1 week | Add e2e tests: full agent lifecycle (start → connect → process → shutdown) |
| P3-3 | 1 week | Add benchmarks for critical paths: networking throughput, OPA evaluation, storage I/O |
| P3-4 | 2 days | Add code coverage reporting to CI, set minimum 50% threshold |
| P3-5 | 2 days | Set up Dependabot for automated dependency updates |

**Phase 3 total**: ~4 weeks

---

## Phase 4: Production Readiness (Weeks 15–20)

| Task | Effort | Description |
|------|--------|-------------|
| P4-1 | 2 weeks | Implement swarm consensus (Raft-based via libp2p-raft or custom implementation) |
| P4-2 | 1 week | Add graceful degradation: if enhanced build fails, fall back to basic agent |
| P4-3 | 1 week | Add production deployment artifacts: Docker Compose, systemd units, Helm chart |
| P4-4 | 1 week | Implement proper crypto (AES-256-GCM encryption/decryption in `pkg/security/crypto.go`) |
| P4-5 | 2 weeks | Complete OPA policy engine integration — enforce all Rego policies at runtime checkpoints |
| P4-6 | 1 week | Add pprof endpoints for production profiling |
| P4-7 | 1 week | Security audit: penetration testing, dependency scanning, SAST integration |

**Phase 4 total**: ~6 weeks

---

## Phase 5: Advanced Features (Weeks 21–28)

| Task | Effort | Description |
|------|--------|-------------|
| P5-1 | 3 weeks | Full polymorphic engine: AST-level code transformation, control flow flattening, dead code insertion |
| P5-2 | 2 weeks | EDR integration: system call monitoring, process tree tracking, file integrity monitoring |
| P5-3 | 2 weeks | Intelligence layer: integrate LLM agent for autonomous decision-making |
| P5-4 | 2 weeks | Code regeneration redesign: safe, validated, human-in-the-loop code generation |
| P5-5 | 1 week | WASM plugin marketplace: signed modules, versioning, dependency resolution |

**Phase 5 total**: ~8 weeks

---

## Timeline Summary

```
Week  1: ████████████████░░░░░░░░░░░░░░░░░░  Phase 0 (Emergency)
Week  4: ██████████████████████░░░░░░░░░░░░  Phase 1 (Foundation)
Week 10: ████████████████████████████████░░  Phase 2 (Core)
Week 14: ██████████████████████████████████  Phase 3 (Testing)
Week 20: ██████████████████████████████████  Phase 4 (Production)
Week 28: ██████████████████████████████████  Phase 5 (Advanced)
```

**Total**: ~28 weeks with 1 full-time developer (~14 weeks with 2 developers, parallelizing Phase 2 and 3)
