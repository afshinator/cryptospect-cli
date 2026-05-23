# cryptospect-cli

CLI tool that fetches live crypto data, computes market regime metrics, and outputs agent-optimized JSON. Rewrite of original cryptospect-cli.

**Source of truth:** `Design‑Decisions.md` — all conventions, schemas, and build order are defined there.

## Project Status (2026‑05‑13)
- **liquidity-pulse complete:** lp-1 through lp-7 all done
- **stablecoin-power complete:** sp-1 through sp-7 all done
- **flow-tension complete:** ft-1 through ft-9 all done
- **market-breadth complete:** mb-1 through mb-9 all done. Multi-timeframe breadth composite (1h/24h/7d/30d) with null-exclusion per timeframe, weight redistribution, Ghost Rally divergence detection, Binance directional consensus validator. 92.6% coverage. 11 compute tests + 6 provider tests + 8 E2E tests.
- **momentum-divergence complete:** md v1.1.0. Market-cap weighted tier averages with configurable `--segments` flag (default 5), cache starvation guard. 92.8% coverage.
- **market-regime complete:** mr v1.0.0. Aggregator metric — calls lp/sp/ft/mb Compute functions internally via pure-Go imports. 10-regime classification matrix (dominance × breadth × conviction, capitulation split), weighted macro confidence scoring, dominance-cold-start detection, Ghost Rally divergence passthrough. 92.1% coverage. 67 compute/provider tests + 6 E2E tests.
- **Original 6 metrics complete** (lp, sp, ft, mb, md, mr) with full E2E test suites.
- **4 additional metrics implemented** — `dominance` (`dom`), `fear-greed-index` (`fgi`), `china-m2` (`cnm2`), `volatility` (`vol`) — golden + provider tests; no E2E tests yet.
- **Binance KlinesData extended:** Close, Open, OpenTime added (additive — FT and LP unaffected)
- **CoinGecko CoinMarketsBreadthData restructured:** per-timeframe GreenCount/TotalCount with BTC reference extraction

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
- `internal/metrics/registry.go` — metric catalog with aliases (`lp`, `sp`, `ft`, `mb`, `md`, `mr`, `dom`, `fgi`, `cnm2`, `vol`)
- `internal/metrics/<name>/` — pure compute functions, types, constants
- `internal/config/` — config loading, source mapping
- `internal/cache/` — file‑based cache (endpoint‑keyed, atomic writes)
- `internal/httpclient/` — retry, backoff, APIError
- `internal/api/` — HTTP clients for CoinGecko, Binance US, CoinDesk, CoinMetrics, AlternativeMe, DBnomics, DefiLlama

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

## Code Style
- Always check returned errors immediately; do not use blank identifiers (`_`) for errors.
- Keep cognitive complexity low. Break nested blocks into independent, testable helper functions.

## Documentation
- **Source‑truth:** `docs/metrics/<metric>.md` (Overview, Formula, Output Schema, Interpretation, Data Source)

## Orchestration (for agents calling this tool)
1. Run `market-regime` first to establish macro context
2. Follow up with individual metrics for drill-down: `liquidity‑pulse`, `stablecoin‑power`, `flow‑tension`, `market‑breadth`, `momentum‑divergence`, `dominance`, `fear-greed-index`, `china-m2`, `volatility`
3. Use `--detail full` for thresholds and descriptions
4. Always pass `--output json` (default)