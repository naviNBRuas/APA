# ADR-002: Build-Tag Gating over Runtime Feature Flags

**Status:** Accepted  
**Date:** 2024 (retrospective)  
**Deciders:** Architecture team  
**Tags:** build-system, feature-gating, binary-size

## Context

APA targets two distinct deployment profiles:

1. **Full-featured** (servers, cloud, development) — all networking protocols,
   intelligence engine, multi-protocol failover, comprehensive testing
2. **Constrained** (edge, IoT, embedded) — minimal binary size, reduced attack
   surface, only essential functionality

The codebase needed a mechanism to gate which capabilities are compiled into
each binary without maintaining separate forks or branches.

## Decision Drivers

- Binary size matters for edge/IoT targets (targeting `<15 MB` minified)
- Attack surface must be compilable away entirely (not just disabled)
- Runtime branches for disabled features waste CPU and memory
- Single codebase, no forking
- Compile-time guarantees that disabled features cannot be called

## Considered Options

| Option | Why Not Chosen |
|--------|----------------|
| **Runtime feature flags** (bool checks in code) | Dead code still compiles, increasing binary size and attack surface. Conditional branches in hot paths. Feature flags can be tampered with or misconfigured at runtime. Bugs in disabled features still break the build. |
| **Config toggles** (YAML/env-var gated code paths) | Same problems as feature flags, plus config files can be edited on the target. No compile-time verification that disabled code paths are safe. |
| **Build tags** (`//go:build enhanced` / `//go:build minimal`) | Dead code is literally never compiled. Zero binary overhead. Compiler proves untaken paths contain no bugs — they do not exist in the binary. Cannot be tampered with at runtime. |

## Decision

Use **Go build tags** to selectively compile features:

- **No tag** — default `agentd` binary with core features
- **`enhanced`** — full agent with intelligence engine, multi-protocol, test suite
- **`minimal`** — stripped-down binary for constrained environments (via
  `pkg/platform/profile_minimal.go`)

## Consequences

### Positive

- Binary size scales with included features (no dead code)
- Compile-time elimination: enhanced-only code literally absent from basic builds
- No runtime configuration surface area for disabled features
- Clear compiler errors if a tagged file references untagged symbols
- Follows existing Go idioms (`//go:build windows`, `//go:build tinygo`)

### Negative

- Inflated perceived technical debt (lint errors in tagged files look like
  defects to tools that don't see the build constraint)

### Mitigations

- `make build-enhanced` and `make test-enhanced` targets compile and test the
  `enhanced` profile locally (see ADR-003).
- An `enhanced-build-test` CI job builds and tests under `-tags enhanced` on
  every push and pull request (see ADR-004).
- `make check` runs on the default (untagged) profile, which covers the
  majority of code paths and is the most commonly deployed variant.
