# APA — Quick-Wins List

**Tasks that can be completed in < 4 hours with high impact**

---

## Critical Security (Do Today)

### 1. Gate CodeRegeneration (1 hour)
**File**: `pkg/regeneration/strategies.go`

Add `--dangerous-enable-code-regeneration` CLI flag defaulting to `false`. Wrap the LLM exec call in a conditional check.

```go
if !cfg.EnableCodeRegeneration {
    return fmt.Errorf("code regeneration is disabled; enable with --dangerous-enable-code-regeneration")
}
```

### 2. Fix TLS Serial Number (30 minutes)
**File**: `pkg/security/tls.go`

Replace:
```go
serialNumber := big.NewInt(1)
```
With:
```go
serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
if err != nil {
    return nil, fmt.Errorf("failed to generate serial number: %w", err)
}
```

### 3. Remove Unused go.mod replace Directives (15 minutes)
```bash
go mod edit -dropreplace github.com/naviNBRuas/APA/pkg/controller/manager
go mod edit -dropreplace github.com/naviNBRuas/APA/pkg/controller/manifest
go mod tidy
```

### 4. Move API Key from Config to Env Var (1 hour)
**Files**: `configs/agent-config.yaml`, `pkg/agent/config.go`

- Remove `admin_api_key` from config file
- Add `APA_ADMIN_API_KEY` env var in Viper config
- Rename `agent-config.yaml` to `agent-config.yaml.example`
- Update README with environment variable documentation

### 5. Remove Commented-Out Code (2 hours)
Files with dead code:
- `pkg/swarm/discovery.go` (~15 lines)
- `pkg/networking/advanced_protocol_manager.go` (~30 lines)
- `pkg/selfhealing/strategies.go` (~10 lines)
- Various others found during grep

---

## Linter Configuration Fixes (3 hours)

### 6. Re-enable G404 Incrementally
**File**: `.golangci.yml`

```yaml
# Remove from exclude:
# - G404
# Then fix specific files with //nolint:gosec where crypto/rand isn't needed
```

### 7. Re-enable G306 and Fix Permissions
Find files opened with `0644` that should be `0600` (configs, keys, certs).

### 8. Add `go test` to CI
**File**: `.github/workflows/ci.yml`

Add a job:
```yaml
test:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
    - run: go test ./...
    - run: go test -tags=enhanced ./...
```

---

## Small Quality Improvements (1-2 hours each)

### 9. Remove panic() Calls
Search for `panic(` across codebase and replace with error returns.

### 10. Add `go.sum` to `.gitignore` Review
Ensure `go.sum` is tracked (it should be for Go modules). Verify `.gitignore` doesn't exclude it.

### 11. Add Basic Makefile Targets
```makefile
test:
    go test ./...
test-enhanced:
    go test -tags=enhanced ./...
lint:
    golangci-lint run
```

### 12. Fix Sample Config Comments
Add inline comments to `agent-config.yaml.example` explaining each field.

---

## Documentation Quick Wins (2-3 hours)

### 13. Update README "Current Status" Section
Add an honest status table showing which features work, which are stubs.

### 14. Add ADR-001
Document the decision to use libp2p over alternatives (gRPC, NATS, MQTT).

### 15. Add ADR-002
Document the build-tag gating decision (`//go:build enhanced`).

---

## Estimated Effort Summary

| Category | Tasks | Total Time |
|----------|:-----:|:----------:|
| Security (P0) | 5 | ~5 hours |
| Code quality | 4 | ~4 hours |
| Testing/CI | 2 | ~1 hour |
| Documentation | 3 | ~3 hours |
| **Total** | **14** | **~13 hours** |

All quick wins can be completed in **2 working days** by a single developer.
