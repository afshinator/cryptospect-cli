# cryptospect-cli: Design Decisions & Step Log

This document captures every design decision, convention, and schema
defined during project setup (steps 1-11). Anything here is subject
to change as the project evolves. Update this file when decisions change.

Last updated: 2026-04-12


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
- github.com/<you>/cryptospect-cli

### Files Created
- go.mod (with toolchain go1.25.9)
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
- test: go test -race -cover ./...
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

### Command Tree
    cryptospect-cli regime        --asset <SYM> --window <DURATION>
    cryptospect-cli zscore        --asset <SYM> --period <DURATION>
    cryptospect-cli rvol          --asset <SYM>
    cryptospect-cli correlation   --pair <SYM,SYM> --window <DURATION>
    cryptospect-cli summary       --assets <SYM,SYM,...>

### Global Flags
- --output, -o: "json" (default) or "nl"
- --verbose, -v: enable debug logging
- --api-key: API key for authenticated endpoints

### Config Precedence (viper handles this)
1. CLI flag (--api-key)
2. Environment variable (CRYPTOSPECT_API_KEY)
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
        data      <varies>      // payload on success, omitted on error
        error     CLIError      // omitted on success
    }

    CLIError {
        code             int       // HTTP status or custom code
        msg              string    // short machine-readable label
        retry_after_sec  int       // hint for rate-limited retries (omitempty)
        source           string    // which API failed, e.g. "coingecko" (omitempty)
    }

### RegimeOutput (regime command)

    {
        "asset":     "BTC",           // string
        "window":    "30d",           // string
        "regime":    "high_vol_bear", // string, see Regime Values below
        "vol_score": 0.82,           // float64
        "z_score":   -2.1,           // float64
        "rvol":      1.8,            // float64
        "ts":        1744444800,     // int64, unix seconds
        "summary":   "..."           // string, omitempty, only with --output nl
    }

    Regime Values:
    - high_vol_bear
    - high_vol_bull
    - low_vol_bear
    - low_vol_bull
    - low_vol_sideways

### ZScoreOutput (zscore command)

    {
        "asset":         "ETH",
        "period":        "90d",
        "z_score":       -1.5,        // float64
        "mean":          2450.00,     // float64
        "stddev":        320.50,      // float64
        "current_price": 1970.00,    // float64
        "ts":            1744444800
    }

### RVOLOutput (rvol command)

    {
        "asset":       "BTC",
        "rvol":        1.8,          // float64, ratio of current to average
        "current_vol": 45000000,     // float64
        "avg_vol":     25000000,     // float64
        "ts":          1744444800
    }

### CorrelationOutput (correlation command)

    {
        "pair":      "BTC,ETH",
        "window":    "60d",
        "pearson_r": 0.87,          // float64, -1 to 1
        "ts":        1744444800
    }

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
  6. Test (go test -race -cover ./...)
  7. Build (go build to /dev/null — proves it compiles)
- Format check technique: goimports -l . | tee /dev/stderr | (! read)
  — lists unformatted files and fails if any exist

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


## Next: Step 12 (Build Order)

With all design and setup complete, the build order is:

1. internal/output/envelope.go — CLIResponse and CLIError structs,
   WriteSuccess and WriteError functions
2. cmd/cryptospect-cli/root.go — cobra root command, global flags,
   slog-to-stderr setup in PersistentPreRun
3. Replace cmd/cryptospect-cli/main.go — real entrypoint calling rootCmd
4. Regime stub subcommand — hardcoded JSON through the envelope
5. Error path stub — force a fake 429, confirm structured error JSON
6. make lint, make test, make build all passing
7. Wire real API clients in internal/api/
8. Wire real metrics in internal/metrics/
9. Connect everything to the subcommands