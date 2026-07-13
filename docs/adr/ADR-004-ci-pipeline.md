# ADR-004: CI Pipeline Structure

**Status:** Accepted
**Date:** 2024 (retrospective)
**Deciders:** Architecture team
**Tags:** ci, github-actions, automation

## Context

APA ships as a standalone binary, Docker image, and multi-platform release
artifact. The CI pipeline must catch regressions across all build profiles,
enforce code quality, verify security, and produce signed release artifacts —
all without manual intervention. Multiple workflow files had accumulated
organically, with overlapping concerns and inconsistent patterns.

## Decision Drivers

- **Fast feedback**: the basic build-test-lint cycle should complete in
  under 15 minutes on push/PR
- **Profile coverage**: default, enhanced, and minimal build profiles must all
  compile; the enhanced profile must also pass its tests
- **Cross-platform**: releases target Linux, macOS, and Windows; CI must build
  for all before tagging
- **Security scanning**: dependency vulnerabilities, SAST (CodeQL), and
  container scanning should run on a schedule, not block every PR
- **Release automation**: tagged commits (`v*`) should produce GitHub releases
  with binaries, checksums, and Docker images
- **CI-local reproducibility**: `make ci-local` should validate workflow files
  without pushing to GitHub

## Considered Options

| Option | Why Not Chosen |
|--------|----------------|
| **Single monolithic workflow** | Long wall-clock time. Security scans delay the basic CI signal. Cannot selectively re-run jobs. Hard to read. |
| **No CI; rely on local `make check`** | No regression protection for PRs. No release automation. No cross-platform verification. |
| **Self-hosted runner** | Maintenance burden. Less reliable than GitHub-hosted. Not necessary for current scale. |
| **Multiple focused workflows** | Each workflow triggers independently on the events it cares about. Security scans can run on a schedule. Release workflow runs only on tags. Fast feedback from the core build-test job. |

## Decision

Use **four focused GitHub Actions workflows**, each with its own trigger scope
and concurrency group:

| Workflow | File | Triggers | Timeout | Purpose |
|----------|------|----------|---------|---------|
| **CI** | `ci.yml` | Push/PR to `main` + weekly schedule | 15-30m | Build, test (all OSes), vet, enhanced build+test, cross-compile matrix |
| **Code Quality** | `code-quality.yml` | Push/PR to `main` + weekly schedule + manual | 10-45m | CodeQL, dependency review, Trivy container scan |
| **Documentation** | `documentation.yml` | Push/PR touching `.go`/`docs`/`README.md` | 30m | Generate API docs, coverage HTML, deploy to GitHub Pages |
| **Release** | `release.yml` | Tag push `v*` + manual dispatch | 45m | Test, build matrix, package, checksum, GitHub Release, Docker image |

Workflows share no inline shell duplication — they delegate to `make` targets
or `go` commands directly. Concurrency groups use `cancel-in-progress: true`
(except release) to avoid wasting CI minutes on stale runs.

## Consequences

### Positive

- **Basic CI completes in ~10 minutes** on Ubuntu (build + test + vet + lint)
- **Enhanced profile verified** in a dedicated job (not gating the basic signal)
- **Security scans are non-blocking** — they run on a schedule and post results
  to the Security tab without failing PRs
- **Release workflow** builds for 5 platform/arch combinations, packages them,
  generates checksums, creates a GitHub Release, and pushes a Docker image
- **Cross-platform test matrix** (Linux, macOS, Windows) catches OS-specific
  issues before merge

### Negative

- **Four workflows to maintain** — configuration drift is possible if
  dependencies or tool versions are updated inconsistently
- **Windows CI is build-only** (no `go vet`, no test runner) due to runner
  constraints; gaps in Windows coverage are accepted for now
- **Docker push is `continue-on-error`** — a registry failure does not fail
  the release, which could produce a release without a corresponding image
- **CodeQL analysis takes ~45 minutes** and runs on every push to `main`, not
  just on a schedule; this could be optimized to run only on demand
