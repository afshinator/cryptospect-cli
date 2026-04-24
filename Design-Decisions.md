# cryptospect-cli: Design Decisions & Step Log

This document captures every design decision, convention, and schema
defined during project setup (steps 1-11). Anything here is subject
to change as the project evolves. Update this file when decisions change.

Last updated: 2026-04-21


## Step 1: Environment & Project Init

### Go Version
- Target: Go 1.25 (previous stable, fully supported)
- Exact patch: 1.25.9 (pinned in .go-version and go.mod toolchain directive)
- Rationale: 1.26 is too new (2 months old), 1.24 and below are end-of-life.
  Go 1.25 is settled in CI images and Docker base images. Nothing in 1.26
  changes this project's capabilities.
- Upgrade plan: bump to 1.26 when 1.27 ships (~August 2026), at which point
  1.25 drops out of support.

### Module Path
- github.com/afshinator/cryptospect-cli

### Files Created
- go.mod (go 1.25.9 — in Go 1.25+, the go line serves as toolchain spec)
- .go-version (contains "1.25.9")


## Step 2: Guardrails

### .golangci.yml (v2 format)
- version: "2" is required — v2 binary does not parse v1 configs
- linters.default: standard (errcheck, govet, staticcheck, unused, gosimple,
  ineffassign, typecheck)
- Additional linters enabled: gocritic, revive, misspell, prealloc, noctx,
  exhaustive, errorlint, bodyclose, nilerr
- Formatters section (new in v2): goimports, gofumpt
- Key linter notes:
  - bodyclose: critical for this project — every API client opens HTTP response
    bodies that must be closed
  - noctx: all HTTP requests must carry a context for timeout/cancellation
  - errorlint: enforces proper error wrapping with %w

### Makefile Targets
- build: CGO_ENABLED=0 GOOS=linux GOARCH=amd64, outputs to bin/cryptospect-cli
- fmt: runs goimports -w . then gofumpt -w .
- lint: golangci-lint run ./...
- vet: go vet ./...
- test: GOMODCACHE=$(PWD)/.gomodcache GOCACHE=$(PWD)/.cache/go go test -race -cover ./...
- clean: rm -rf bin/ dist/
- release: goreleaser release --clean
- Version injection via ldflags: -X main.version=$(VERSION)
- VERSION derived from git describe --tags --always --dirty, falls back to "dev"

### Agent Discovery Files
- CLAUDE.md: for Claude Code, concise project onboarding (<300 lines)
- agents.md: for Cursor, Copilot, and other agents — CLI signatures and
  orchestration playbook
- .cursorrules: symlink to agents.md

### Orchestration Playbook (documented in agents.md)
1. Run regime first to establish macro context
2. Then zscore and rvol for confirmation signals
3. Run correlation only when comparing cross-asset behavior
4. Always pass --output json


## Step 3: CLI Framework & Flag Parsing

### Dependencies
- github.com/spf13/cobra — subcommands and flag parsing
- github.com/spf13/viper — config file + env var binding

### Command Tree (v1 — implemented)
    cryptospect-cli liquidity-pulse      (alias: lp)   [--detail basic|extended|full]
    cryptospect-cli stablecoin-power     (alias: sp)   [--detail basic|extended|full]
    cryptospect-cli flow-tension         (alias: ft)   [--detail basic|extended|full]
    cryptospect-cli market-breadth       (alias: mb)   [--detail basic|extended|full]
    cryptospect-cli momentum-divergence  (alias: md)   [--detail basic|extended|full]
    cryptospect-cli market-regime        (alias: mr)   [--detail basic|extended|full]
    cryptospect-cli list-metrics
    cryptospect-cli cache-clear

### Future Per-Asset Commands (deferred, not yet implemented)
    cryptospect-cli regime        --asset <SYM> --window <DURATION>
    cryptospect-cli zscore        --asset <SYM> --period <DURATION>
    cryptospect-cli rvol          --asset <SYM>
    cryptospect-cli correlation   --pair <SYM,SYM> --window <DURATION>
    cryptospect-cli summary       --assets <SYM,SYM,...>

### Global Flags
- --output, -o: "json" (default)
- --verbose, -v: enable debug logging
- --detail: "basic" (default), "extended", "full" — controls metadata in response
- --api-key: API key for CoinGecko authenticated endpoints
- --config: config file path (default $HOME/.cryptospect.yaml)

### Config Precedence (viper handles this)
1. CLI flag (--api-key → maps to CoinGecko)
2. Environment variables (CRYPTOSPECT_COINGECKO_KEY, CRYPTOSPECT_BINANCE_KEY)
3. Config file (~/.cryptospect.yaml)


## Step 4: Linting & Static Analysis

### Tool
- golangci-lint v2.11.4 (installed via curl script, not go install)
- Rationale for curl: go install can use unexpected dependency versions,
  producing untested binaries

### Verification
- golangci-lint run ./... must pass with zero warnings before any commit
- CI runs the same check via golangci/golangci-lint-action@v7


## Step 5: Code Formatting

### Tools (run in this order)
1. goimports — formats + organizes imports (superset of gofmt)
2. gofumpt — stricter superset of gofmt (no empty lines at start of
   functions, etc.)

### Rule
- make fmt then make lint, in that order, before every commit
- VS Code should run goimports on save (go.formatTool setting)


## Step 6: Makefile Verification

### Build Output
- Static binary: CGO_ENABLED=0 produces a statically linked ELF binary
- No libc dependency — runs on any Linux without shared libraries
- Binary location: bin/cryptospect-cli
- Cross-compilation handled by goreleaser at release time (linux/darwin,
  amd64/arm64)


## Step 7: Testing Strategy

### Framework
- stdlib testing package only — no testify or third-party assertions

### Patterns
- Table-driven tests for all metrics (pure functions)
- httptest.NewServer for mocking API responses
- Fixture JSON files in testdata/ directories (go build ignores testdata/)

### Directories
- internal/api/testdata/ — canned API responses
- internal/metrics/testdata/ — (if needed for complex test data)

### Flags
- -race: always on (catches concurrency bugs in API clients)
- -cover: always on (target 60-70% once real code exists)

### File Naming
- Tests live alongside code: zscore.go -> zscore_test.go
- No separate tests/ directory


## Step 8: Error Handling & Structured Logging

### Error Wrapping
- Always: fmt.Errorf("context: %w", err)
- Every function that receives and returns an error adds context

### Sentinel Errors (per package)
- internal/api/:
  - ErrRateLimited
  - ErrAssetNotFound
  - ErrAPIUnavailable
- internal/metrics/:
  - ErrInsufficientData
- internal/config/:
  - ErrMissingAPIKey
- Callers check with errors.Is(err, api.ErrRateLimited)

### stdout/stderr Boundary (HARD RULE)
- stdout: ONLY the JSON envelope (CLIResponse). One object per invocation.
- stderr: ONLY slog diagnostic output. Gated by --verbose.
- No fmt.Println anywhere. No log.Printf. Ever.
- Only slog.Debug/Info/Warn/Error for diagnostics.
- Only output.WriteSuccess / output.WriteError for results.

### slog Configuration
- Handler: slog.NewTextHandler pointed at os.Stderr
- Default level: Warn
- With --verbose: Debug
- Set in cobra root command's PersistentPreRun

### Exit Codes
- 0: success AND handled errors (rate limits, missing assets, etc.)
- 1: unrecoverable failures only (can't parse flags, can't encode JSON)
- Agents parse the JSON status field, not the exit code


## Step 9: JSON Output Schema

NOTE: These schemas are likely to change as development progresses.

### Envelope (all commands)

    CLIResponse {
        status    string        // "ok" or "error"
        ts        int64         // Unix seconds when response was created
        results   []MetricResult // zero or more metric results
        error     CLIError      // omitted on success
    }

    MetricResult {
        metric     string        // canonical metric name, e.g., "liquidity-pulse"
        namespace  string        // provider namespace, e.g., "cryptospect" — always present
        version    string        // SemVer of the provider, e.g., "v1.0.0" — always present
        status     string        // "ok", "degraded", or "unavailable"
        data       <varies>      // metric-specific payload
        meta       <varies>      // metadata, omitted when --detail basic
    }

    NOTE: version is a mandatory identity field — non-pointer plain string,
    never omitted regardless of --detail level. A provider registering with empty
    version is rejected at registration time.
    NOTE: namespace was removed in the fix for issue 4 — it was redundant
    (all core metrics use namespace "cryptospect").

    CLIError {
        code             int       // HTTP status or custom code
        msg              string    // short machine-readable label
        retry_after_sec  int       // hint for rate-limited retries (omitempty)
        source           string    // which API failed, e.g. "coingecko" (omitempty)
    }

### Per-Metric `data` Payload (v1 global metrics)
Each metric command returns a `MetricResult.data` field whose shape is metric-specific.
The `data` struct is defined in `internal/metrics/<name>/types.go` for each metric.
See Step 12 for the metric conventions and `docs/metrics/<metric>.md` once implemented.

NOTE: The per-asset schemas below (RegimeOutput, ZScoreOutput, etc.) were designed for
future per-asset commands (regime, zscore, rvol, correlation) that are not yet implemented.
They are preserved here for reference; do not treat them as current v1 output shapes.

### NL Summary Format
- Under 40 tokens
- Label:value pairs separated by pipes
- No prose words like "currently" or "looking"
- Examples:
  - "BTC 30d: High-Vol Bear Expansion | Z:-2.1 | RVOL:1.8x | Conviction:High"
  - "ETH 90d: Low-Vol Sideways | Z:-0.3 | RVOL:1.1x | Conviction:Moderate"

### Field Name Conventions
- snake_case in JSON tags
- Short but unambiguous (ts not timestamp, stddev not standard_deviation)
- Timestamps: unix seconds as int64, not RFC3339 strings


## Step 10: Build & Release Pipeline

### GoReleaser (.goreleaser.yml)
- version: 2
- Targets: linux/darwin, amd64/arm64
- CGO_ENABLED=0 for all builds
- Archives as tar.gz with checksums

### GitHub Actions CI (.github/workflows/ci.yml)
- Triggers: push to main, pull requests to main
- Steps in order:
  1. Checkout
  2. Setup Go 1.25
  3. Format check (goimports -l, gofumpt -l — fails if any output)
  4. Lint (golangci/golangci-lint-action@v7, v2.11.4)
  5. Vet (go vet ./...)
  6. Test (GOMODCACHE=$(PWD)/.gomodcache GOCACHE=$(PWD)/.cache/go go test -race -cover ./...)
  7. Build (go build to /dev/null — proves it compiles)
- Format check technique: Excludes cache directories (.cache/, .gomodcache/, .go/, vendor/) using find command
  - Command: find . -name "*.go" -not -path "./.cache/*" -not -path "./.gomodcache/*" -not -path "./.go/*" -not -path "./vendor/*" -exec goimports -l {} \;
  - Same pattern for gofumpt

### Files in Repo
- cmd/cryptospect-cli/main.go — throwaway stub committed so CI build step
  passes (will be replaced with real code in step 12)


## Step 11: Documentation

### README.md
- Targets humans finding the repo on GitHub
- Sections: quick start, commands, output format, agent integration,
  configuration, data sources, development, requirements

### Agent Docs (created in step 2)
- agents.md — CLI signatures, orchestration playbook, output envelope,
  error handling, API key injection
- CLAUDE.md — concise project onboarding for Claude Code

### JSON Tool Definition
- README includes example LLM tool definition showing how an agent
  framework would call cryptospect-cli
- This bridges "CLI tool" to "agent tool" for anyone reading the repo


## Project Directory Structure (after step 11)

    cryptospect-cli/
    ├── .cursorrules -> agents.md
    ├── .github/
    │   └── workflows/
    │       └── ci.yml
    ├── .go-version
    ├── .golangci.yml
    ├── .goreleaser.yml
    ├── agents.md
    ├── CLAUDE.md
    ├── cmd/
    │   └── cryptospect-cli/
    │       └── main.go          (throwaway stub)
    ├── internal/
    │   ├── api/
    │   │   └── testdata/
    │   │       └── .gitkeep
    │   └── metrics/
    │       └── testdata/
    │           └── .gitkeep
    ├── go.mod
    ├── go.sum
    ├── Makefile
    └── README.md


## Step 12: Metric Conventions & Implementation Plan (2026‑04‑15)

### 1. Registry with Aliases
- **Location:** `internal/metrics/registry.go`
- **MetricDef fields:** `Name`, `Aliases` (lowercase), `Endpoints`, `Sources` (datapoint → endpoint‑key map), `Description`
- **Aliases:** `lp`, `sp`, `ft`, `mb`, `md`, `mr` (unique, lowercase)
- **Purpose:** CLI `list‑metrics`, validation, foundation for future `get` command
- **Compute wiring:** `cmd/root.go` uses generic dispatcher (`buildMetricRunE`) that iterates `BestProviders()` — this IS compute wiring. May evolve as metrics are implemented.

### 2. Output Envelope (JSON)
- **Top‑level:** `CLIResponse { status, ts, results[], error? }`
- **Per‑metric:** `MetricResult { metric, status, data, meta? }`
- **Metadata:** `MetaBasic` (cache_hit, ttl_remaining), `MetaExtended` (+ sources), `MetaFull` (+ thresholds, description)
- **Envelope behavior:** Single‑metric commands return `Results` with one element. `--detail basic` → `Meta` omitted; `extended` → `MetaExtended`; `full` → `MetaFull`

### 3. Metric‑Specific Conventions
- **Types (`internal/metrics/<name>/types.go`):** `Data` struct with metric‑specific fields + `Classification` struct (per‑metric, typed fields) + `summary` string
- **Compute function:** `func Compute(in Input) (Data, error)` (pure, no I/O)
- **Classification:** Each metric defines a typed `Classification` struct (e.g., `TradeValidation`, `MarketCondition` fields). Classification values are package‑level constants; complete mapping table documented in LLM‑focused docs
- **Constants:** Define classification values as package‑level constants

### 4. CLI Command Wiring
- Each metric command imports aliases from registry, uses Cobra’s `Aliases` field
- `--detail` flag inherited from root (global flag)
- Builds `MetricResult` with appropriate `Meta` based on `--detail`
- Calls `output.WriteSuccess([]output.MetricResult{…})`

### 5. Documentation
- **Source‑truth (`docs/metrics/<metric>.md`):** Overview, Formula, Output Schema, Interpretation, Data Source, Usage, Calibration
- **Directory structure:** `docs/metrics/`

### 6. Testing Conventions
- Table‑driven tests for each `Compute` function
- Fixture checklist: happy path, empty/nil, invalid JSON, semantically invalid, extreme values, thin‑data guard
- Mock API with `httptest`
- All tests pass `‑race ‑cover`

### 7. Error Handling
- **Statuses:** `"ok"`, `"degraded"`, `"unavailable"` per metric; top‑level `"error"` only for unrecoverable failures
- **Sentinel errors** per package; wrap with `%w`; structured CLI errors via envelope
- **Exit codes:** 0 for success AND handled errors; non‑zero only for unrecoverable failures

### 8. Status Detection Helper (`DetectStatus`)
- **Location:** `internal/metrics/helpers.go`
- **Signature:** `func DetectStatus(confidence float64, thinData bool) string`
- **Mapping:** confidence ≥ 0.8 → `"ok"`; confidence ≥ 0.5 → `"degraded"`; else `"unavailable"`. If `thinData` is true, downgrade by one level (e.g., `"ok"` → `"degraded"`, `"degraded"` → `"unavailable"`, `"unavailable"` unchanged).
- **Usage:** Each metric’s `Compute` function calls `DetectStatus` to set the `MetricResult.Status` field.

### 9. Command‑Level Integration Tests
- **Pattern:** Each metric command gets an `_e2e_test.go` file (e.g., `liquidity‑pulse_e2e_test.go`) that tests the full CLI flow.
- **Scope:** Uses `httptest.NewServer` to mock API responses, invokes the Cobra command via `cmd.Execute()`, validates the JSON envelope.
- **Goal:** Verify that flag parsing, config loading, cache, fetcher, compute, and output work together.
- **Placement:** Same directory as the command file (`cmd/`).
- **Naming:** `TestLiquidityPulse_Integration` (table‑driven with mock scenarios).

---
## Step 13: Document Sync & Single Source of Truth

**Primary source:** `Design‑Decisions.md` (this file). All other documents derive from it.

### Dependent Documents (update when this file changes)

| Document | Purpose | Last Sync | Notes |
|----------|---------|-----------|-------|
| `CLAUDE.md` | Claude‑Code onboarding, stack, conventions | 2026‑04‑17 (session 2) | Keep concise; reference this file for details. |
| `agents.md` | Agent‑focused CLI signatures, envelope, error handling | 2026‑04‑17 | CLI commands, JSON envelope, error‑handling rules. |
| `README.md` | Human‑facing GitHub docs, quick start, examples | 2026‑04‑17 | Keep friendly; link to `agents.md` for agent integration. |
| `/vault/Knowledge/CryptoSpect‑CLI‑new.md` | Architectural summary, metric tiers, build order | 2026‑04‑16 | Snapshot of this file + original‑project context. |

**Sync checklist** (run after any change to this file):
1. Update `Last Sync` date above.
2. Propagate changed conventions to `CLAUDE.md` (metric conventions, testing, etc.).
3. Update `agents.md` if CLI signatures, envelope schema, or error handling changed.
4. Update `README.md` if commands, output format, or examples changed.
5. Update vault knowledge file if architecture or build order changed.

**Rationale:** Manual sync is error‑prone but manageable with this table. A future script could diff sections and auto‑update, but for v1, this table + checklist suffices.

## Code Review & Critical Fixes (2026‑04‑16)

**Review summary:** Agency‑agents engineering‑code‑reviewer performed thorough review of infrastructure (Steps 1‑14). Score: **9/10** – exceptional foundation with minor but important fixes needed.

### ✅ **Critical Blockers Fixed**
1. **Cache file permissions** – `0600` (owner‑only) instead of `0644` (`internal/cache/cache.go:99`)
2. **Context propagation** – HTTP client now uses `http.NewRequestWithContext` (`internal/httpclient/client.go`)
3. **Fetcher race condition** – Coarse‑grained locking eliminates duplicate API calls (`internal/api/fetcher.go:60‑156`)

### 🟡 **Suggestions Addressed (first pass)**
- **TTL validation** – Bounds checking added: negative → default 300, zero → 60, >86400 → cap at 1 day (`internal/api/fetcher.go`)
- **Unused variable** – Named `suffix` in endpoint parsing (`internal/api/fetcher.go`)
- **Permission check** – Group permissions now also enforced (`&0077` vs `&0177`) (`internal/config/config.go`)
- Note: `rand.Seed` was briefly added for jitter then removed by linting (deprecated since Go 1.20 — auto‑seeded)


### ✅ **Linting & CI Fixes (2026‑04‑16)**

- **71 lint errors resolved** – All `golangci‑lint` issues fixed, CI pipeline passes with zero warnings
- **Error handling** – Added `_ = ` for unchecked error returns in tests (`cache.Close()`, `resp.Body.Close()`, `w.Write()`, `conn.Close()`)
- **Error type assertions** – Replaced `err.(*Type)` with `errors.As()` for wrapped‑error compatibility
- **Missing comments** – Added GoDoc for all exported types, constants, and functions (`revive` compliance)
- **Performance** – Changed `string(entry.Data) != string(data)` to `!bytes.Equal(entry.Data, data)` (`gocritic`)
- **Deprecated API** – Removed `rand.Seed` (Go 1.20+ auto‑seeds)
- **Unused code** – Deleted unused test helper (`mockDoer`), unused constant (`klinesWrongTypeFixture`)
- **Type safety** – Fixed `APIError.Body` type (`[]byte` vs `string`) in tests
- **Struct naming** – Renamed `CacheEntry` → `Entry` (eliminates stutter)
- **Switch style** – Converted `if‑else` chain to tagged `switch` in `resolveConfigPath`


### ✅ **Endpoint Parameterization Completed** (2026‑04‑16)
- Added distinct endpoint constants: `CoinGeckoCoinMarketsBreadth`, `CoinGeckoCoinMarketsMomentum`, `BinanceSpotCVD_BTC_1h`
- Updated registry sources mapping with new datapoint names: `coin_markets_breadth`, `coin_markets_momentum`
- Updated fetcher `resolveURL` to map each constant to appropriate URL builder with explicit parameters
- All tests updated and passing

### ✅ **Go 1.25 Caching Patterns Implemented** (2026‑04‑16)
- **Sharded maps (16 shards)** – Eliminate lock contention for concurrent endpoint fetches; each shard has its own `sync.Mutex` and `map[unique.Handle[string]]…`.
- **Zero‑allocation keying with `unique.Handle`** – Global `sync.Map` canonicalizes endpoint strings to `unique.Handle[string]`; memory cache lookups compare pointer‑sized integers instead of allocating strings.
- Concurrent test (`TestFetchConcurrentDifferentEndpoints`) verifies parallelism (elapsed time ≤ `delay * (N‑1)`).
- Backward compatibility maintained; all existing tests pass with `‑race`.

### ✅ **CLI Command Infrastructure Completed** (2026‑04‑17)
- **Steps 15‑18 implemented:** Cobra root command (`root.go`), cache‑clear subcommand (`cache.go`), list‑metrics subcommand (`list.go`)
- **Configuration:** `LoadWithViper()` integrated with Cobra flag binding, environment variable precedence
- **Output envelope:** All commands produce valid JSON envelopes via `output.WriteSuccess`/`WriteError`
- **Command‑level integration tests:** Pending (to be added with metric templates)
- **Build status:** `make test` passes, `make build` produces functional binary

### ✅ **Suggestions Addressed (second pass, 2026‑04‑17)**
- **Dead code removal** – Deleted `GetWithKey` from `internal/httpclient/client.go` (CoinGecko‑specific method on a generic client, never called from production code) and its test
- **Double stat eliminated** – `config.LoadWithViper` now calls `os.Stat` once and reuses the result (`internal/config/config.go`)
- **Test reliability** – `TestStaleEntry` in `internal/cache/cache_test.go` uses TTL=0 instead of `time.Sleep(2s)`
- **Write coverage** – Added `TestWrite` to `internal/config/config_test.go` (happy path, file‑exists error, nested dir creation)
- **TTL bounds coverage** – Added `TestResolveTTLBounds` to `internal/api/resolve_test.go` (negative → 300, zero → 60, >86400 → 86400)

### 📋 **Remaining Suggestions & Technical Debt** (post‑CLI‑infrastructure)
- **Addressed in config enhancements (commit 058de9f):**
  - **Config file extensions** – Support for both `.yaml` and `.yml` via `resolveConfigPath()` (nit from review)
  - **Output package thread‑safety** – `SetWriter()`/`Writer()` with `sync.RWMutex` for test stdout redirection (pre‑existing)
  - **Critical error handling** – Fixed unchecked `os.Remove()` and `cache.Set()` errors (errcheck) (pre‑existing)
- **Still deferred (non‑blocking for v1):**
  - **Error‑type preservation** – Keep `httpclient` typed errors accessible
  - **Registry config loading** – Consider YAML‑based metric definitions (YAGNI for v1)
  - **JSON RawMessage validation** – Add `Validate()` method (low priority)
  - **Test coverage gaps** – `Clear()` error paths, `Validate` edge cases (`Write()` now covered)
  - **Go 1.25 structured caching patterns** – ~~Sharded maps, `unique.Handle`~~ **implemented**; `testing/synctest`, JSON v2 experiment deferred

### 🚀 **Next Steps**
1. Implement Step 15: Plugin Architecture (MetricProvider interface, SemVer registry, scaffolds, generic dispatcher — see Step 15 section below)
2. Proceed with first metric compute implementation (`liquidity‑pulse`) inside its scaffolded package
3. Repeat for remaining metrics

---

## Build Order (Step 18 – Completed)

✅ **Steps 1‑18 and Step 15 (Plugin Architecture) implemented and tested.**

1. `internal/output/envelope.go` — `CLIResponse`, `MetricResult`, `CLIError`
2. `internal/output/meta.go` — `MetaBasic`, `MetaExtended`, `MetaFull`, `SourceMeta`
3. `internal/output/writer.go` — `WriteSuccess`, `WriteError`
4. `internal/metrics/registry.go` — Registry with alias support, `MetricDef`, `RegisterDefaultMetrics`
5. `internal/metrics/helpers.go` — `DetectStatus()` helper (confidence/thin‑data → "ok"/"degraded"/"unavailable")
6. `internal/api/constants.go` — Endpoint‑key constants (`coingecko.global`, `coingecko.coins_markets`, etc.)
7. `internal/config/config.go` — `Config`, `Load`, `SourceFor` (uses endpoint‑key constants)
8. `internal/cache/cache.go` — `Get`, `Set`, `Clear`, atomic writes, endpoint‑keyed file names
9. `internal/httpclient/client.go` — retry, backoff, `APIError` (generic HTTP client)
10. `internal/api/coingecko/client.go` — CoinGecko API client (global, coins/markets, market_chart)
11. `internal/api/binance/client.go` — Binance US API client (klines, futures OI/funding)
12. `internal/api/coindesk/client.go` — CoinDesk API client (asset top list)
13. `internal/api/coinmetrics/client.go` — CoinMetrics Community API client
14. `internal/api/fetcher.go` — `Fetch(ctx, endpointKey)` helper (cache‑first, provider dispatch, atomic writes)
15. `cmd/cryptospect‑cli/root.go` — Cobra root, global flags (`--output`, `--verbose`, `--detail`)
16. Replace `cmd/cryptospect‑cli/main.go`
17. `cmd/cache.go` — cache‑clear subcommand
18. `cmd/list.go` — list metrics from registry (`list‑metrics`)
15a. **Plugin Architecture Refactor** — MetricProvider interface, SemVer registry, 6 scaffolded metric packages, `catalog.go`, generic dispatcher in `root.go`
19. **First metric compute: liquidity‑pulse** — implement `Compute()` inside `internal/metrics/liquiditypulse/v1/provider.go`; add command‑level integration test
20. Repeat for remaining Tier 2+3 metrics (`stablecoin‑power`, `flow‑tension`, `market‑breadth`, `momentum‑divergence`, `market‑regime`) — each includes command‑level integration test

---

## Step 15: Plugin Architecture (2026-04-21)

### Goal
Replace the hardcoded `RegisterDefaultMetrics` function with a compile-time plugin system where each metric self-registers via `init()`. Enables decentralized metric authorship, versioned output schemas, and agent-predictable CLI surface.

### MetricProvider Interface (`internal/metrics/provider.go`)

```go
type MetricDef struct {
    Name        string   `json:"name"`
    Namespace   string   `json:"namespace"` // e.g. "cryptospect"; enforced non-empty at registration
    Version     string   `json:"version"`   // SemVer "v1.0.0"; v-prefix required
    Aliases     []string `json:"aliases"`
    Endpoints   []string `json:"endpoints"` // EndpointKey constants
    Description string   `json:"description,omitempty"`
}

type MetricProvider interface {
    Def() MetricDef
    Compute(ctx context.Context, data map[string]json.RawMessage) (output.MetricResult, error)
}
```

`Sources map[string]string` removed from `MetricDef` — redundant with `Endpoints`.

### SemVer (`internal/metrics/semver.go`)
- `ParseSemVer(s string) ([3]int, error)` — rejects strings without leading `v`
- `CompareSemVer(a, b [3]int) int` — returns -1, 0, 1
- No external dependencies (stdlib `strings` + `strconv` only)

### Registry (`internal/metrics/registry.go`)

```go
type Registry struct {
    providers  map[string]MetricProvider  // "namespace/name@v1.0.0" → provider
    aliasIndex map[string][]string        // alias → []full keys
    nameIndex  map[string][]string        // name  → []full keys
    coreNS     string                     // "cryptospect"
}
```

Global registry initialized as a package-level var (not lazy `sync.Once`) so `init()` calls from metric packages can register before `main()` runs:

```go
var globalRegistry = NewRegistry()
func GlobalRegistry() *Registry { return globalRegistry }
func MustRegister(p MetricProvider) { /* panics on error */ }
```

`Register()` rejects providers with empty `Name`, `Namespace`, or `Version`, or a `Version` that fails `ParseSemVer`. Returns `ErrDuplicateMetric` on duplicate full key.

### Resolution (`bestProvider`)

1. Gather candidates from `nameIndex[name]` or `aliasIndex[alias]`
2. If any candidate has `namespace == "cryptospect"`, discard all non-core candidates
3. Sort remaining with `slices.SortFunc`: SemVer descending, then namespace ascending (lexicographic) as tiebreaker
4. Return index 0

Fully deterministic regardless of registration order or map iteration. Forks never hijack core aliases by bumping version numbers.

### Scaffold Pattern

Each metric lives in `internal/metrics/<name>/v1/provider.go`:

```go
func init() { metrics.MustRegister(&Provider{}) }

func (p *Provider) Compute(_ context.Context, _ map[string]json.RawMessage) (output.MetricResult, error) {
    msg, _ := json.Marshal(map[string]string{"error": "metric not yet implemented: " + MetricName})
    return output.MetricResult{
        Metric:    MetricName,
        Namespace: CoreNamespace,
        Version:   MetricVersion,
        Status:    "unavailable",
        Data:      json.RawMessage(msg),
    }, nil  // nil error — unavailability is expressed in the result, not as a Go error
}
```

Non-nil error from `Compute` is reserved for catastrophic/unrecoverable failures only.

### Generic Dispatcher (`cmd/cryptospect-cli/root.go`)

```go
for _, p := range reg.BestProviders() { // one provider per unique name
    p := p
    def := p.Def()
    cmd := &cobra.Command{
        Use:     def.Name,
        Aliases: def.Aliases,
        Short:   def.Description,
        Long:    fmt.Sprintf("%s\n\nVersion: %s | Namespace: %s", def.Description, def.Version, def.Namespace),
        RunE:    buildMetricRunE(p),
    }
    root.AddCommand(cmd)
}
```

Full metric name is `Use`; shorthands (e.g. `lp`) are cobra `Aliases`. The existing `PersistentPreRunE` on rootCmd is sufficient — cobra propagates it to all dynamic subcommands.

### Catalog (`cmd/cryptospect-cli/catalog.go`)

Blank imports that trigger `init()` registration for all 6 core metrics:

```go
import (
    _ "github.com/afshinator/cryptospect-cli/internal/metrics/liquiditypulse/v1"
    _ "github.com/afshinator/cryptospect-cli/internal/metrics/stablecoinpower/v1"
    // ...
)
```

### Config Version Pinning

`~/.cryptospect.yaml` can pin a specific metric version:

```yaml
metrics:
  lp: "v1.2.0"               # exact version
  mb: "fork-user/mb@v1.1.0"  # explicit namespace + version
```

This is a future-layer concern; Step 15 does not implement source substitution.

### Acceptance Criteria
- `cryptospect-cli list-metrics` shows all 6 metrics with `version` and `namespace`
- `cryptospect-cli lp` returns valid JSON: outer `status: "ok"`, metric `status: "unavailable"`, `version` and `namespace` present
- All existing tests pass with `-race`
- `golangci-lint` passes
