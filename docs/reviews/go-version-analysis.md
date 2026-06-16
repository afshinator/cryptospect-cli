# Go Version Analysis — Why 1.25.9 and What to Do Now

## 1. Why 1.25.9 Was Pinned

The project was initialized on **2026-04-12** with `go 1.25.9` in `go.mod` and `.go-version`. The rationale appears only in `Design-Decisions.md:12-19`:

> **Target:** Go 1.25 (previous stable, fully supported)  
> **Exact patch:** 1.25.9 (pinned in .go-version and go.mod)  
> **Rationale:** 1.26 is too new (2 months old), 1.24 and below are end-of-life.  
> Go 1.25 is settled in CI images and Docker base images. Nothing in 1.26 changes this project's capabilities.  
> **Upgrade plan:** bump to 1.26 when 1.27 ships (~August 2026).

**1.25.9 is not special** — it was simply the latest patch of Go 1.25 available at project creation. No project feature, dependency, or tool specifically requires a patch ≥1.25.9. The `go` directive in `go.mod` acts as the toolchain spec in Go 1.25+; pinning the exact patch ensures every build uses the same toolchain.

## 2. What Actually Requires Go 1.25+

The codebase uses exactly **three** language/stdlib features from Go 1.21+:

| Feature | Min Go | File | Notes |
|---------|--------|------|-------|
| `unique.Handle` | 1.23 (optimized in 1.25) | `internal/api/fetcher.go:10,22,45-46,78-79` | Zero-allocation cache keying — the most recent feature used |
| `slices.SortFunc` / `slices.Sort` | 1.21 | `internal/metrics/registry.go:103,148` | SemVer sort in `bestProvider()` |
| `min()` builtin | 1.21 | `internal/version/version.go:38` | Commit hash truncation |

**Bottom line:** Any Go ≥1.23 would compile and pass tests. The 1.25 choice was conservative stability policy, not technical necessity.

## 3. Dependency Constraints

All direct and indirect dependencies are satisfied by Go 1.23+ (the highest `go` directive among deps is `go 1.23.0`):

| Dependency | Min Go |
|------------|--------|
| viper v1.21.0 | 1.23 |
| afero v1.15.0 | 1.23 |
| locafero v0.11.0 | 1.23 |
| cobra v1.10.2 | 1.15 |
| All others | ≤1.21 |

No dependency forces ≥1.25.9.

## 4. Current Situation (2026-06-15)

- **Container Go:** `1.25.9` (matches pin)
- **CI setup:** `actions/setup-go@v5` with `go-version: "1.25"` (resolves to latest 1.25.x — currently 1.25.9)
- **6 reachable CVEs** from project code via `internal/httpclient/client.go` and `internal/api/coingecko/client.go` (govulncheck audit, 2026-06-15)
- **20 stdlib-only CVEs** affecting stdlib packages the binary links against
- **gofumpt v0.10.0** built against `go1.25.9` (referenced in CI stability plan)

## 5. Options

### Option A: Stay at 1.25.9 (status quo)

- The project builds, tests pass, lint is clean
- 26 known CVEs in the Go stdlib go unpatched
- 6 are reachable from project code (HTTP client TLS handshake, crypto/x509 hostname verification)
- Risk is low (the app connects only to trusted APIs), but the CVEs are real
- No feature or compatibility gain from staying

### Option B: Upgrade to 1.25.11 (latest 1.25.x)

- Fixes all 26 CVEs in a single change
- `go 1.25.11` in go.mod, `1.25.11` in .go-version
- CI `"1.25"` resolves to 1.25.11 automatically
- Go patch releases are backward-compatible — zero code changes needed
- No risk of breaking anything (same minor version, no API changes)
- Side benefit: potential minor GC/codegen improvements backported to the 1.25 branch

### Option C: Jump to 1.26.x (latest stable)

- Design-Decisions.md says "bump to 1.26 when 1.27 ships (~August 2026)"
- 1.26 has been out for ~4 months now (since ~Feb 2026)
- Stated reason for deferring was "nothing in 1.26 changes this project's capabilities"
- Minimal risk but breaks the stated upgrade plan
- No benefit over Option B for this project

## 6. Risk Assessment of the 6 Reachable CVEs

These are the only findings that matter for a CLI tool talking to trusted APIs:

| CVE | Component | Impact | Realistic risk |
|-----|-----------|--------|---------------|
| CVE-2026-39836 | `net.DialContext` panic on NUL byte | **Windows-only** | **Zero on Linux** |
| CVE-2026-33814 | HTTP/2 SETTINGS infinite loop | DoS if server sends malicious frame | **Very low** — trusted APIs with fixed infrastructure |
| CVE-2026-42507 | Unescaped error in textproto | Information leak via error strings | **Low** — errors go to stderr/stderr log |
| CVE-2026-27145 | Slow hostname parsing (x509) | CPU exhaustion on crafted cert | **Low** — trusted APIs with valid certs |
| CVE-2026-33811 | Crash on long CNAME (net) | DoS | **Very low** — requires malicious DNS response |
| CVE-2026-39824 | Integer overflow in x/sys/windows | **Windows-only** | **Zero on Linux** |

**Net assessment:** Real but low risk for this application. Would be higher for a server accepting untrusted input.

## 7. Recommendation

**Upgrade to Go 1.25.11.** This is a pure patch bump — same minor version, fully backward-compatible, zero code changes required. It:

1. Fixes all 26 CVEs immediately
2. Requires changing exactly two files: `go.mod` (the `go` directive) and `.go-version`
3. CI is already versioned as `"1.25"` (resolves to latest patch automatically)
4. `gofumpt` remains compatible (it builds against whatever Go it's compiled with)

The original plan ("bump to 1.26 when 1.27 ships") remains intact — this is just a patch-level maintenance update, same as pulling a security update for any other OS package.

Files to change:
- `go.mod:3` — `go 1.25.9` → `go 1.25.11`
- `.go-version:1` — `1.25.9` → `1.25.11`

Verify with:
```bash
go test -race -cover ./...     # must pass
golangci-lint run ./...        # must pass
make build                     # must produce working binary
```

The container Go installation should also be upgraded to 1.25.11 to keep the dev environment in sync.
