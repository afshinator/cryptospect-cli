# cryptospect-cli

CLI tool that fetches live/historical crypto data, computes market regime metrics, and outputs agent-optimized JSON.

## Stack
- Go 1.25, single static binary (CGO_ENABLED=0)
- cobra + viper for CLI, log/slog for logging
- No external runtime dependencies

## Commands
- `make build` — compile to bin/cryptospect-cli
- `make test` — run tests with race detector
- `make lint` — golangci-lint v2
- `make fmt` — goimports + gofumpt

## Architecture
- `cmd/cryptospect-cli/` — main entrypoint, cobra root command
- `internal/api/` — HTTP clients for CoinGecko, Binance, CoinDesk
- `internal/metrics/` — Z-score, RVOL, correlation, regime detection (pure functions)
- `internal/output/` — JSON envelope, token-dense NL summaries
- `internal/config/` — API key loading, flag binding

## Output Contract
- stdout is ALWAYS valid JSON (CLIResponse envelope), never raw text
- stderr is for diagnostic logs only (gated by --verbose)
- Exit 0 for success AND handled errors; non-zero only for unrecoverable failures

## Conventions
- Errors: wrap with fmt.Errorf("context: %w", err), define sentinel errors per package
- Tests: table-driven, use testdata/ for API fixture JSON, httptest for mocking
- NL summaries: under 40 tokens, label:value pairs separated by pipes

## Orchestration (for agents calling this tool)
1. Run `regime` first to establish macro context
2. Then `zscore` and `rvol` for confirmation signals
3. Run `correlation` only when comparing cross-asset behavior
4. Always pass --output json