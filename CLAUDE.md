# cryptospect-cli

CLI tool that fetches live crypto data, computes market regime metrics, and outputs agent-optimized JSON. Rewrite of original cryptospect-cli.

**Source of truth:** `Design‑Decisions.md` — all conventions, schemas, and build order are defined there.

## Project Status (2026‑04‑17)
- **Infrastructure through CLI complete:** Steps 1‑18 of the build order are implemented and tested.
- **API clients:** CoinGecko (global, stables markets, derivatives, coin markets) and Binance US (spot CVD) fully implemented with tests.
- **Placeholder clients:** CoinDesk and CoinMetrics (stubs) satisfy build order.
- **Cache‑first fetcher:** `internal/api/fetcher.go` implements memory → file cache → HTTP API → stale fallback; comprehensive test suite passes with `‑race`.
- **Go 1.25 caching patterns:** Sharded maps (16 shards) eliminate lock contention for concurrent endpoint fetches; `unique.Handle` provides zero‑allocation keying. Concurrent test verifies parallelism.
- **Code review completed:** Agency‑agents engineering‑code‑reviewer scored infrastructure **9/10**. Critical blockers (security, context propagation, race condition) fixed.
- **Endpoint parameterization complete:** Distinct constants for breadth/momentum endpoints, Binance CVD with explicit parameters.
- **CLI command infrastructure complete:** Cobra root with viper config precedence, cache‑clear and list‑metrics subcommands, command‑level integration tests.
- **Linting & CI fixes:** 71 golangci‑lint errors resolved, CI pipeline passes with zero warnings.
- **Ready for:** First metric template implementation (liquidity‑pulse).

## Stack
- Go 1.25, single static binary (CGO_ENABLED=0)
- cobra + viper for CLI, log/slog for logging
- No external runtime dependencies

## Commands
- `make build` — compile to bin/cryptospect-cli
- `make test` — run tests with race detector
- `make lint` — golangci-lint v2
- `make fmt` — goimports + gofumpt

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
- **Classification:** Per‑metric typed struct with package‑level constants for categorical values (e.g., `TradeValidationNormal`)
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
- **LLM‑focused:** `docs/llm/<metric>.md` (command, example JSON, classification table)

## Orchestration (for agents calling this tool)
1. Run `regime` first to establish macro context (future)
2. For v1, use standalone commands: `liquidity‑pulse`, `stablecoin‑power`, `flow‑tension`, `market‑breadth`, `momentum‑divergence`, `market‑regime`
3. Use `--detail full` for thresholds and descriptions
4. Always pass `--output json` (default)