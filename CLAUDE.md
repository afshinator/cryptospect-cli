# cryptospect-cli

CLI tool that fetches live crypto data, computes market regime metrics, and outputs agent-optimized JSON. Rewrite of original cryptospect-cli.

**Source of truth:** `Design‑Decisions.md` — all conventions, schemas, and build order are defined there.

## Project Status (2026‑04‑25)
- **liquidity-pulse complete:** lp-1 through lp-7 all done
- **Detail filtering working:** basic=null, extended=validation, full=all fields
- **Float formatting:** internal/metrics/float.go with MetricFloat type
- **--version flag added**
- **Post-mortem fixes applied** to float precision, version, Binance guard

## Stack
- Go 1.25, single static binary (CGO_ENABLED=0)
- cobra + viper for CLI, log/slog for logging
- No external runtime dependencies

## Commands
- `make build` — compile to bin/cryptospect-cli
- `make test` — run tests with race detector
- `make lint` — golangci-lint v2
- `make fmt` — goimports + gofumpt

**Remember:** After code changes, must REBUILD: `go build -o ./cryptospect-cli ./cmd/cryptospect-cli/`

## Architecture (v1)
- `cmd/cryptospect-cli/` — main entrypoint, cobra root command
- `internal/output/` — JSON envelope (`CLIResponse`, `MetricResult`), metadata structs, writer
- `internal/metrics/registry.go` — metric catalog with aliases (`lp`, `sp`, `ft`, `mb`, `md`, `mr`)
- `internal/metrics/<name>/` — pure compute functions, types, constants
- `internal/config/` — config loading, source mapping
- `internal/cache/` — file‑based cache (endpoint‑keyed, atomic writes)
- `internal/httpclient/` — retry, backoff, APIError
- `internal/api/` — HTTP clients for CoinGecko, Binance US, CoinDesk, CoinMetrics

## Output Contract
- stdout is ALWAYS valid JSON (`CLIResponse` envelope), never raw text
- stderr is ONLY slog diagnostics, gated by `--verbose`
- Exit 0 for success AND handled errors; non‑zero only for unrecoverable failures
- Single‑metric commands return `results` array (one element) for forward compatibility
- Detail levels: `--detail basic|extended|full` adds metadata, thresholds, description

## Metric Conventions
- **Types:** `Data` struct with metric‑specific fields + typed `Classification` struct (per‑metric fields) + `summary` string
- **Compute:** `func Compute(in Input) (Data, error)` (pure, no I/O)
- **Classification:** Per‑metric typed struct; serializes to `{"label": "string", "description": "string"}` in JSON. Package‑level string constants for label values (e.g., `ClassificationHigh = "high"`).
- **Aliases:** Each metric has a lowercase 2‑letter alias (e.g., `lp` for `liquidity‑pulse`)
- **Registry:** Single source of truth for metric names, aliases, endpoints, source mapping, descriptions
- **Status detection:** `DetectStatus(confidence, thinData)` helper sets metric status (`"ok"`, `"degraded"`, `"unavailable"`)

## Testing Conventions
- Table‑driven tests, stdlib only (no testify)
- Fixture checklist: happy path, empty/nil, invalid JSON, semantically invalid, extreme values, thin‑data guard
- Mock API with httptest
- Command‑level integration tests (`_e2e_test.go`) for each metric (full CLI flow)
- Always test with `-race -cover`

## Documentation
- **Source‑truth:** `docs/metrics/<metric>.md` (Overview, Formula, Output Schema, Interpretation, Data Source)

## Orchestration (for agents calling this tool)
1. Run `regime` first to establish macro context (future)
2. For v1, use standalone commands: `liquidity‑pulse`, `stablecoin‑power`, `flow‑tension`, `market‑breadth`, `momentum‑divergence`, `market‑regime`
3. Use `--detail full` for thresholds and descriptions
4. Always pass `--output json` (default)