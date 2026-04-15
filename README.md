# cryptospect-cli

A portable, zero-dependency CLI tool that fetches live and historical cryptocurrency data, computes high-signal market regime metrics, and outputs them in a format optimized for AI agents and LLM tool-calling.

**Source of truth:** `Design‑Decisions.md` — commands, output format, and architecture defined there.

## Quick Start

    git clone https://github.com/<you>/cryptospect-cli.git
    cd cryptospect-cli
    make build
    ./bin/cryptospect-cli regime --asset BTC --window 30d

## Commands

    cryptospect-cli regime        --asset <SYM> --window <DURATION> --output json
    cryptospect-cli zscore        --asset <SYM> --period <DURATION> --output json
    cryptospect-cli rvol          --asset <SYM> --output json
    cryptospect-cli correlation   --pair <SYM,SYM> --window <DURATION> --output json
    cryptospect-cli summary       --assets <SYM,SYM,...> --output json

### Global Flags

    --output, -o    Output format: json (default; natural‑language summaries embedded in JSON)
    --verbose, -v   Enable debug logging on stderr
    --api-key       API key for authenticated endpoints

## Output Format

Every invocation writes exactly one JSON object to stdout. Diagnostic logs go to stderr only.

### Success

    {
      "status": "ok",
      "ts": 1744444800,
      "results": [
        {
          "metric": "regime",
          "status": "ok",
          "data": {
            "asset": "BTC",
            "window": "30d",
            "regime": "high_vol_bear",
            "vol_score": 0.82,
            "z_score": -2.1,
            "rvol": 1.8,
            "summary": "BTC 30d: High-Vol Bear Expansion | Z:-2.1 | RVOL:1.8x | Conviction:High"
          }
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

## Agent Integration

Example LLM tool definition for agentic workflows:

    {
      "name": "crypto_regime",
      "description": "Get current market regime for a cryptocurrency",
      "parameters": {
        "asset": {"type": "string", "enum": ["BTC", "ETH", "SOL"]},
        "window": {"type": "string", "default": "30d"}
      },
      "command": "cryptospect-cli regime --asset {asset} --window {window} --output json"
    }

See agents.md for the full orchestration playbook.

## Configuration

API key precedence (highest to lowest):

1. CLI flag: --api-key
2. Environment variable: CRYPTOSPECT_API_KEY
3. Config file: ~/.cryptospect.yaml

## Data Sources

- CoinGecko (free tier, primary)
- Binance-US (free tier)
- CoinDesk (free tier)
- CoinMarketCap (requires API key)

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