# ADR-003: Makefile Structure and Targets

**Status:** Accepted
**Date:** 2024 (retrospective)
**Deciders:** Architecture team
**Tags:** build-system, makefile, developer-experience

## Context

The APA project has multiple build variants (default, enhanced, minimal),
cross-compilation targets, testing profiles, and distribution packaging needs.
Developers and CI need a consistent, discoverable interface for all build and
test operations. Without a structured Makefile, contributors must remember
ad-hoc `go build` flag combinations, leading to errors and inconsistent builds.

## Decision Drivers

- Single entry point for all common development tasks
- Must support three build profiles: default, enhanced (`-tags enhanced`), minimal (`-tags minimal`)
- Cross-compilation matrix for release artifacts (Linux, macOS, Windows, multiple architectures)
- Testing variants: unit, race, integration, coverage, enhanced-tagged
- CI uses the same targets as local development (no divergence)
- Distribution packaging (tarballs, zip archives, checksums)

## Considered Options

| Option | Why Not Chosen |
|--------|----------------|
| **No Makefile; document go commands in CONTRIBUTING.md** | Prone to drift. Developers copy-paste outdated commands. No tab-completion or discoverability. CI ends up with its own ad-hoc scripts. |
| **Taskfile (go-task)** | Additional dependency. Not installed by default on most systems. Another config format to learn. |
| **Justfile** | Same problems as Taskfile — not universally available. |
| **Shell scripts in `scripts/`** | Hard to discover. No `.PHONY`-style help output. Scripts grow organically without structure. |
| **Makefile** | Available on every platform. Simple declarative syntax. `.PHONY` targets are self-documenting. No additional dependencies. CI can call the same targets. |

## Decision

Provide a top-level **Makefile** with categorized targets following the
pattern `category-subtask`. Common targets are grouped:

| Category | Targets |
|----------|---------|
| **Build** | `build`, `build-standalone`, `build-enhanced`, `build-linux`, `build-windows`, `build-darwin`, `build-matrix`, `build-matrix-minimal` |
| **Test** | `test`, `test-race`, `test-pkg`, `test-enhanced`, `test-integration` |
| **Quality** | `lint`, `lint-fix`, `coverage`, `check` (lint + test-race) |
| **Package** | `dist`, `package-matrix`, `checksums` |
| **Docker** | `docker-build`, `docker-buildx` |
| **Housekeeping** | `clean`, `ci-local` |

All targets are `.PHONY`; the build output directory is `bin/` and distribution
artifacts go to `dist/`. The default target (`all`) builds both `agentd` and
`standalone-agent`.

## Consequences

### Positive

- Single `make <TAB>` discovers all available operations
- CI workflow files call `make build-enhanced`, `make test-enhanced`, etc. —
  no inline shell duplication
- New contributors can build the entire project with `make all`
- Cross-compilation and packaging are one-command operations
- Clean separation between build, test, quality, and release phases

### Negative

- Makefile is tied to the current toolchain (Go); swapping build tools would
  require a rewrite
- Windows users need a Make-compatible environment (MSYS2, Cygwin, or WSL)
- The matrix targets (`build-matrix`) do not parallelize natively (serial `for`
  loop over platforms)
