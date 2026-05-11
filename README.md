# cryptospect-cli

A portable, zero-dependency CLI tool that fetches live cryptocurrency data, computes high-signal market regime metrics, and outputs them in a format optimized for AI agents and LLM tool-calling.

**Source of truth:** `Design‑Decisions.md` — commands, output format, and architecture defined there.

## Quick Start

    git clone https://github.com/<you>/cryptospect-cli.git
    cd cryptospect-cli
    make build
    ./bin/cryptospect-cli list-metrics

## Commands

### Market Regime Metrics (5 implemented, 1 scaffold)

    cryptospect-cli liquidity-pulse      (alias: lp)   [--detail basic|extended|full]
    cryptospect-cli stablecoin-power     (alias: sp)   [--detail basic|extended|full]  [--top N]
    cryptospect-cli flow-tension         (alias: ft)   [--detail basic|extended|full]
    cryptospect-cli market-breadth       (alias: mb)   [--detail basic|extended|full]  [--top N]
    cryptospect-cli momentum-divergence  (alias: md)   [--detail basic|extended|full]  [--segments N]
    cryptospect-cli market-regime        (alias: mr)   [--detail basic|extended|full]  (scaffold — not yet implemented)

### Utility

    cryptospect-cli list-metrics         # list all available metrics and their aliases
    cryptospect-cli cache-clear          # clear the local API response cache

### Global Flags

    --output, -o    Output format: json (default)
    --verbose, -v   Enable debug logging on stderr
    --detail        Detail level: basic (default), extended, full
    --api-key       API key for CoinGecko authenticated endpoints
    --config        Config file path (default $HOME/.cryptospect.yaml)

## Output Format

Every invocation writes exactly one JSON object to stdout. Diagnostic logs go to stderr only.

### Success

    {
      "status": "ok",
      "ts": 1744444800,
      "results": [
        {
          "metric": "liquidity-pulse",
          "status": "ok",
          "data": { ... },
          "meta": { ... }    // omitted with --detail basic (default)
        }
      ]
    }

### Error

    {
      "status": "error",
      "ts": 1744444800,
      "error": {
        "code": 429,
        "msg": "rate_limited",
        "retry_after_sec": 60,
        "source": "coingecko"
      }
    }

### Detail Levels

- `--detail basic` (default): `meta` omitted
- `--detail extended`: `meta` includes cache hit, TTL remaining, source timestamps
- `--detail full`: `meta` adds thresholds and metric description

## Agent Integration

Example LLM tool definition for agentic workflows:

    {
      "name": "crypto_liquidity_pulse",
      "description": "Get the current liquidity pulse metric for the crypto market",
      "parameters": {},
      "command": "cryptospect-cli liquidity-pulse --detail full --output json"
    }

See `agents.md` for the full orchestration playbook.

## Configuration

API key precedence (highest to lowest):

1. CLI flag: `--api-key` (maps to CoinGecko)
2. Environment variables: `CRYPTOSPECT_COINGECKO_KEY`, `CRYPTOSPECT_BINANCE_KEY`
3. Config file: `~/.cryptospect.yaml`

## Data Sources

- CoinGecko (free tier, primary — global market, stablecoins, derivatives, coin markets)
- Binance US (free tier — spot CVD klines)
- CoinDesk (stub, placeholder)
- CoinMetrics Community (stub, placeholder)

## Development

    make build    # compile to bin/cryptospect-cli (static binary, CGO_ENABLED=0)
    make fmt      # format with goimports + gofumpt
    make lint     # run golangci-lint v2
    make vet      # run go vet
    make test     # run tests with race detector and coverage
    make clean    # remove build artifacts

## Requirements

- Go 1.25+ (see .go-version for exact patch)
- golangci-lint v2.11+ (for linting only, not required to build)

## License

TBD
