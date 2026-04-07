# Release Readiness Guide

This document provides a practical, repeatable checklist for maintainers preparing an APA release.

## Scope

Use this guide before creating a GitHub Release from `.github/workflows/release.yml`.

## 1) Pre-flight validation

- Confirm branch is up to date with `main`
- Confirm release tag exists locally and remotely
- Confirm `CHANGELOG.md` has a complete `[Unreleased]` section and move notable items into the target version section at release time
- Confirm docs are updated for user-facing changes (`README.md`, `docs/`)

## 2) Local quality gates

Run locally before triggering release:

- Workflow lint / simulation helper:
  - `bash scripts/validate-workflows-local.sh`
  - or `make ci-local`
- Core checks:
  - `go mod verify`
  - `gofmt` clean check
  - `go vet ./...`
  - `go test -v -race -count=1 ./pkg/...`
  - `go test -v ./tests/...`
  - `go build -o standalone-agent cmd/standalone-agent/main.go`

### Optional integration suites

- Local P2P integration tests are opt-in:
  - set `APA_RUN_P2P_INTEGRATION=1` to enable

## 3) CI policy recommendations

Configure repository protections to require these checks on `main`:

- `CI / Build and Test`
- `CI / Cross-Platform Build Matrix`
- `CI / Security Audit`
- `CI / Code Quality`
- `CI / Documentation Check`
- `Code Quality and Security / CodeQL Analysis`
- `Code Quality and Security / Container Security Scan`

Recommended branch protection settings:

- Require pull request before merging
- Require status checks to pass before merging
- Require branches to be up to date before merging
- Restrict force pushes and branch deletions

## 4) Release workflow execution

The release pipeline is manually triggered and expects an existing tag input.

- Trigger workflow: **Release** (`workflow_dispatch`)
- Input:
  - `tag`: existing semantic tag (example: `v1.0.0`)

Expected outputs:

- Cross-platform binaries for Linux/macOS/Windows (amd64 + arm64 where configured)
- Checksums file
- Packaged archives uploaded as release artifacts
- GitHub Release populated with generated notes and assets

## 5) Post-release verification

- Confirm all expected assets are present on the release page
- Validate checksums against downloaded artifacts
- Smoke test at least:
  - Linux amd64 binary startup
  - `--help` and `--version` output
- Confirm changelog and README references are coherent with released behavior

## 6) Rollback and hotfix strategy

If a bad release occurs:

- Mark release as pre-release or draft while triaging
- Create hotfix branch from latest good tag
- Apply minimal patch and re-run full checklist
- Publish next patch tag (avoid mutating existing release artifacts)
