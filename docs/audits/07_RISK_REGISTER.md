# APA — Risk Register

---

## Risk Assessment Methodology

- **Likelihood**: 1 (rare) → 5 (almost certain)
- **Impact**: 1 (negligible) → 5 (catastrophic)
- **Score**: Likelihood × Impact (1–25)
- **Rating**: ≤5 Low, 6–10 Medium, 11–15 High, 16–20 Very High, 21–25 Critical

---

## Risk Register

| ID | Risk | Description | L | I | Score | Rating | Mitigation |
|:--:|------|-------------|:-:|:-:|:----:|:------:|------------|
| R01 | CodeRegeneration LLM exec produces malicious code | LLM output is written to disk and compiled without validation | 4 | 5 | **20** | Critical | Remove strategy or gate behind `--dangerous` flag + add syntax/compilation validation |
| R02 | No test coverage hides regressions | Only 2 test files for 260+ Go source files — any change can break anything | 5 | 4 | **20** | Critical | Add unit tests for core subsystems, run in CI |
| R03 | Key architectural decisions undocumented | Single developer knowledge — if unavailable, no one can understand *why* | 4 | 4 | **16** | Very High | Write ADRs for all major decisions |
| R04 | Build-tag gating creates untested code path | `enhanced`-tagged code never compiled/tested by default | 5 | 3 | **15** | High | Add CI job that compiles and tests with `-tags=enhanced` |
| R05 | Runtime.go single point of failure | 969-line orchestrator crash = entire agent down | 3 | 5 | **15** | High | Modularize bootstrap, add recovery per subsystem |
| R06 | Stub code misrepresents capability maturity | README/docs describe stubs as complete features | 4 | 4 | **16** | Very High | Update docs to accurately reflect state, remove oversells |
| R07 | Security linter exclusions hide vulnerabilities | G404, G304, G306, G107 excluded from CI | 4 | 4 | **16** | Very High | Re-enable exclusions incrementally, fix violations |
| R08 | No input validation on WASM/controller paths | Path traversal or arbitrary file load | 3 | 4 | **12** | High | Add path sanitization, restrict to configured directory |
| R09 | Goroutine leaks on shutdown | Multiple Start methods ignore cancel func | 4 | 3 | **12** | High | Audit all goroutine lifecycle, fix cancel propagation |
| R10 | Single developer bus factor | 100% of commits from one person, single branch | 3 | 5 | **15** | High | Onboard additional contributors, use feature branches |
| R11 | OPA policy bypass | Policy enforcement is partially stubbed | 3 | 4 | **12** | High | Complete enforcer implementation, add integration tests |
| R12 | Swarm protocol manipulation | No peer authentication in discovery | 3 | 3 | **9** | Medium | Add Ed25519 signature verification on announcements |
| R13 | Configuration poisoning | Config values not validated after parsing | 3 | 3 | **9** | Medium | Add schema validation using go-playground/validator |
| R14 | No graceful degradation | Enhanced build failure = complete failure | 2 | 4 | **8** | Medium | Add fallback to basic agent mode |
| R15 | Admin API brute force | No rate limiting on admin endpoints | 3 | 2 | **6** | Medium | Add rate limiting and IP allowlisting |
| R16 | License compliance slip | Dependencies not tracked | 2 | 3 | **6** | Medium | Add license check to CI (w/ go-license-detector) |

---

## Risk Heat Map

```
Impact →
  5 │ R01 R02
  4 │ R06 R07   R03       R05 R10
  3 │       R11    R08 R09    R14
  2 │    R15
  1 │
    └────────────────────────→ Likelihood
      1   2   3   4   5
```

**Top 5 Risks to Address Immediately**:
1. R01 — CodeRegeneration LLM exec (Critical, 20)
2. R02 — No test coverage (Critical, 20)
3. R06 — Documentation misrepresentation (Very High, 16)
4. R07 — Security linter exclusions (Very High, 16)
5. R03 — Undocumented architecture/R03 — Undocumented architecture (Very High, 16)
