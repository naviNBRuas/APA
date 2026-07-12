# APA — Dependency Review

---

## Direct Dependencies (from `go.mod`)

| Module | Version | Purpose | Maturity | License | Risk |
|--------|---------|---------|----------|---------|:----:|
| `github.com/libp2p/go-libp2p` | v0.41.1 | P2P networking | Stable | MIT | Low |
| `github.com/tetratelabs/wazero` | v1.9.0 | WASM runtime | Stable | Apache-2.0 | Low |
| `github.com/open-policy-agent/opa` | v1.2.0 | Policy engine | Stable | Apache-2.0 | Low |
| `github.com/dgraph-io/badger/v4` | v4.5.1 | KV store | Stable | Apache-2.0 | Low |
| `github.com/prometheus/client_golang` | v1.21.1 | Metrics | Stable | Apache-2.0 | Low |
| `google.golang.org/grpc` | v1.71.1 | RPC framework | Stable | Apache-2.0 | Low |
| `github.com/spf13/cobra` | v1.9.1 | CLI | Stable | Apache-2.0 | Low |
| `github.com/spf13/viper` | v1.20.1 | Config | Stable | MIT | Low |
| `go.uber.org/zap` | v1.27.0 | Logging | Stable | MIT | Low |
| `golang.org/x/crypto` | v0.36.0 | Crypto (ssh) | Stable | BSD-3-Clause | Low |
| `github.com/google/uuid` | v1.6.0 | UUID generation | Stable | BSD-3-Clause | Low |
| `github.com/mattn/go-sqlite3` | v1.14.24 | SQLite | Stable | MIT | Low |
| `github.com/multiformats/go-multiaddr` | v0.15.0 | Multiaddr | Stable | MIT | Low |
| `github.com/fsnotify/fsnotify` | v1.8.0 | File watcher | Stable | BSD-3-Clause | Low |
| `github.com/hashicorp/go-version` | v1.7.0 | Versioning | Stable | MPL-2.0 | Low |
| `github.com/jedib0t/go-pretty/v6` | v6.6.7 | Pretty print | Stable | MIT | Low |
| `golang.org/x/term` | v0.36.0 | Terminal | Stable | BSD-3-Clause | Low |
| `github.com/ncruces/go-sqlite3` | (indirect) | Pure Go SQLite | Stable | MIT | Low |

---

## Issues Found

### Unused `replace` Directives

```go.mod
replace github.com/naviNBRuas/APA/pkg/controller/manager => ./pkg/controller/manager
replace github.com/naviNBRuas/APA/pkg/controller/manifest => ./pkg/controller/manifest
```

These `replace` directives are **unused** — no import in the codebase references `pkg/controller/manager` or `pkg/controller/manifest` as an external module path. They should be removed to avoid confusion.

### Potentially Missing Dependencies

| Capability | Missing Dependency | Notes |
|-----------|-------------------|-------|
| Structured config validation | `go-playground/validator` or similar | Config values aren't validated after parsing |
| HTTP router/framework | `chi` or `gin` | Currently using raw `net/http` |
| Testing framework | `stretchr/testify` | No assertion library used in tests |
| Container orchestration | SDK for k8s/docker | No deployment integration |
| Message queue | NATS or RabbitMQ client | Swarm uses direct libp2p only |

---

## Dependency Health

| Metric | Score | Notes |
|--------|:-----:|-------|
| Up-to-date | 7/10 | Most deps are current, some could be newer |
| Security advisories | ✅ None reported | Checked via `go list -m -u` |
| License compatibility | ✅ All permissive (MIT, Apache-2.0, BSD) | |
| CGO usage | ⚠️ `mattn/go-sqlite3` requires CGO | Consider `modernc.org/sqlite` for pure Go |
| Binary size impact | Moderate | libp2p is the largest dependency |

---

## Recommendations

1. **Remove unused `replace` directives** from `go.mod`
2. **Consider replacing `mattn/go-sqlite3` (CGO) with `modernc.org/sqlite` (pure Go)** for cross-compilation support
3. **Add `stretchr/testify`** for test assertions (mocks, suite support, etc.)
4. **Run `go mod tidy`** regularly to clean up unused deps
5. **Set up Dependabot or Renovate** for automated dependency updates
