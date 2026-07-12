# APA — Prioritized Backlog

**Format**: `P<priority>.<sub-rank> — <task> (effort)`

---

## P0: Must Fix (Immediate)

| ID | Task | Effort | Risk |
|:--:|------|:------:|:----:|
| P0.01 | Remove/gate CodeRegeneration LLM strategy | 1 hr | Security |
| P0.02 | Fix TLS serial number with crypto/rand | 30 min | Security |
| P0.03 | Remove unused go.mod replace directives | 15 min | Cleanliness |
| P0.04 | Move admin API key from config to env var | 1 hr | Security |
| P0.05 | Re-enable G404 linter, fix math/rand usages | 4 hr | Security |
| P0.06 | Remove hardcoded API key placeholder from sample config | 15 min | Security |
| P0.07 | Remove commented-out code blocks across repo | 2 hr | Cleanliness |
| P0.08 | Fix panic() calls in production paths | 4 hr | Reliability |

## P1: High Priority (Sprint 1-2)

| ID | Task | Effort |
|:--:|------|:------:|
| P1.01 | Split runtime.go into manageable modules | 1 week |
| P1.02 | Add testify + unit tests for security, persistence, health, rbac | 1 week |
| P1.03 | Add CI job for `go test -tags=enhanced ./...` | 3 days |
| P1.04 | Fix goroutine cancellation in all Start methods | 2 days |
| P1.05 | Fix policy/enforcer.go no-op methods | 2 days |
| P1.06 | Add config validation (go-playground/validator) | 3 days |
| P1.07 | Re-enable G304/G306/G107 linters, fix violations | 2 days |
| P1.08 | Write ADR-001 through ADR-005 | 2 days |
| P1.09 | Add input validation for WASM/controller paths | 2 days |
| P1.10 | Set up Dependabot for dependency updates | 1 day |

## P2: Medium Priority (Sprint 3-4)

| ID | Task | Effort |
|:--:|------|:------:|
| P2.01 | Implement basic polymorphic transformations | 1 week |
| P2.02 | Fix WASM executor integration | 2 weeks |
| P2.03 | Implement networking CommHandshake() | 1 week |
| P2.04 | Add peer auth to swarm discovery | 1 week |
| P2.05 | Implement memory optimization strategy | 3 days |
| P2.06 | Add admin API rate limiting | 2 days |
| P2.07 | Add admin API audit logging | 2 days |
| P2.08 | Run go mod tidy, clean up unused dependencies | 1 day |
| P2.09 | Update README/docs to accurately reflect codebase state | 2 days |
| P2.10 | Add code coverage reporting to CI | 1 day |

## P3: Standard Priority (Sprint 5-6)

| ID | Task | Effort |
|:--:|------|:------:|
| P3.01 | Write networking integration tests | 1 week |
| P3.02 | Write OPA policy evaluation tests | 3 days |
| P3.03 | Write persistence I/O tests | 2 days |
| P3.04 | Add e2e tests for agent lifecycle | 1 week |
| P3.05 | Add benchmarks for critical paths | 1 week |
| P3.06 | Implement graceful degradation (enhanced → basic fallback) | 1 week |
| P3.07 | Add production Docker Compose | 2 days |
| P3.08 | Add systemd unit files | 1 day |
| P3.09 | Add troubleshooting guide to docs | 2 days |

## P4: Lower Priority (Sprint 7-8)

| ID | Task | Effort |
|:--:|------|:------:|
| P4.01 | Implement swarm consensus (Raft) | 2 weeks |
| P4.02 | Complete crypto (AES-256-GCM Encrypt/Decrypt) | 1 week |
| P4.03 | Add pprof endpoints | 1 day |
| P4.04 | Replace CGO sqlite3 with pure Go | 2 days |
| P4.05 | Add deployment guide | 2 days |
| P4.06 | Run penetration testing / security audit | 1 week |
| P4.07 | Write security model documentation | 2 days |

## P5: Future (Backlog)

| ID | Task | Effort |
|:--:|------|:------:|
| P5.01 | Full AST-level polymorphic engine | 3 weeks |
| P5.02 | EDR system call monitoring | 2 weeks |
| P5.03 | LLM-based autonomous decision-making | 2 weeks |
| P5.04 | Safe code regeneration with human-in-loop | 2 weeks |
| P5.05 | WASM plugin marketplace | 1 week |
| P5.06 | Helm chart for Kubernetes deployment | 1 week |
| P5.07 | Performance optimization pass | 2 weeks |
