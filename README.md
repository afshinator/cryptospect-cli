# cryptospect-cli

A portable, zero-dependency CLI tool that fetches live cryptocurrency data, computes high-signal market regime metrics, and outputs them in a format optimized for AI agents and LLM tool-calling.

## Metrics

Six signals, each answering a different question about current market conditions:

| Metric | Alias | What it tells you |
|--------|-------|-------------------|
| `liquidity-pulse` | `lp` | Is money actively moving, or is the market thin and idle? Measures 24h volume relative to total market cap. |
| `stablecoin-power` | `sp` | How much dry powder is sitting on the sidelines? High stablecoin dominance signals latent buying capacity; a sharp drop signals rotation into risk assets. |
| `flow-tension` | `ft` | Are buyers or sellers winning right now? CVD (cumulative volume delta) reveals who is absorbing pressure; OI trend and funding rate show whether the futures market agrees. |
| `market-breadth` | `mb` | Is a move broad-based or driven by a handful of large caps? Measures how many of the top coins are participating across 1h / 24h / 7d / 30d windows. |
| `momentum-divergence` | `md` | Where is capital rotating? Compares momentum across large-, mid-, and small-cap tiers — rotation into small caps signals risk-on appetite. |
| `market-regime` | `mr` | What is the overall market state? Aggregates all five signals into a single regime label (e.g. Bull Trend, Stagnation, Capitulation) with a confidence score. |

Run `market-regime` first for macro context, then drill into individual metrics for detail.

## Quick Start

    git clone https://github.com/afshinator/cryptospect-cli
    cd cryptospect-cli
    make build
    ./bin/cryptospect-cli list-metrics

## Commands

### Market Regime Metrics — all 6 implemented

    cryptospect-cli liquidity-pulse      (alias: lp)   [--detail basic|extended|full]
    cryptospect-cli stablecoin-power     (alias: sp)   [--detail basic|extended|full]  [--top N]
    cryptospect-cli flow-tension         (alias: ft)   [--detail basic|extended|full]
    cryptospect-cli market-breadth       (alias: mb)   [--detail basic|extended|full]  [--top N]
    cryptospect-cli momentum-divergence  (alias: md)   [--detail basic|extended|full]  [--segments N]
    cryptospect-cli market-regime        (alias: mr)   [--detail basic|extended|full]

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

Every invocation writes exactly one JSON object to stdout. Diagnostic logs go to stderr only, gated by `--verbose`. Exit code is `0` for both success and handled errors (e.g. degraded data); non-zero only for unrecoverable failures.

### Success

    {
      "status": "ok",
      "ts": 1744444800,
      "results": [
        {
          "metric": "liquidity-pulse",
          "version": "v1.0.0",
          "namespace": "cryptospect",
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

- `--detail basic` (default): `meta` omitted — minimal payload for agents
- `--detail extended`: `meta` includes `cache_hit`, `ttl_remaining_sec`, source timestamps
- `--detail full`: `meta` adds thresholds and metric description

### Pretty-print

By default output is compact (single line). To enable indented JSON, set `output.pretty: true` in your config file (see [Configuration](#configuration)). There is no CLI flag — this is a persistent preference, not a per-call option.

    # compact (default)
    {"status":"ok","ts":1744444800,"results":[...]}

    # pretty (output.pretty: true in config)
    {
      "status": "ok",
      "ts": 1744444800,
      "results": [...]
    }

The `-o` / `--output` flag currently only accepts `json` and exists as a forward-compatibility placeholder for future formats (e.g. `csv`, `text`).

## API Keys

All 6 metrics work without any API key on the free public tiers. Keys are optional and unlock higher rate limits or additional data sources.

| Key | How to set | Status | What it unlocks |
|-----|-----------|--------|-----------------|
| `CRYPTOSPECT_COINGECKO_KEY` | Env var, config, or `--api-key` | Implemented | CoinGecko Demo tier — appended as `x_cg_demo_api_key` on the `/derivatives` endpoint used by `flow-tension`; relaxes rate limits |
| `CRYPTOSPECT_BINANCE_KEY` | Env var or config | Not yet wired | Reserved for Binance Futures (OI, funding rate); the key is read from config but not passed to any request today (only Binance US spot is implemented) |

Key precedence (highest to lowest):

1. CLI flag: `--api-key` (maps to CoinGecko)
2. Environment variables: `CRYPTOSPECT_COINGECKO_KEY`, `CRYPTOSPECT_BINANCE_KEY`
3. Config file: `~/.cryptospect.yaml` (see [Configuration](#configuration))

Without a CoinGecko key, `flow-tension` fetches the `/derivatives` endpoint without authentication and may hit public rate limits; it falls back to spot-CVD only and reports `status: degraded` if the endpoint is unavailable.

## Cache

Each metric fetches data from one or more free public APIs (CoinGecko, Binance US, DefiLlama). These APIs are rate-limited, and some endpoints are slow. Without caching, running several metrics in quick succession would exhaust free-tier rate limits and make every call wait on the network.

The cache stores each API response on disk, keyed by endpoint. Subsequent calls within the TTL window read from disk instead of hitting the network — this makes repeated or back-to-back metric calls fast and rate-limit safe.

**What happens when you run a metric:**

1. **First run (cold cache):** the tool fetches from the API, writes the response to disk, and returns the result. This is the slowest path — expect a brief network delay.
2. **Within TTL (warm cache):** the tool reads from disk. Fast, no network call. `--detail extended` will show `cache_hit: true` and the seconds remaining.
3. **TTL expired:** the tool fetches fresh data, updates the cache, and returns the result.
4. **API unreachable, stale cache exists:** the tool uses the expired cached response and reports `status: degraded` — you still get a result, but it may be minutes or hours old.
5. **API unreachable, no cache:** the metric reports `status: unavailable` with an error message.

**Cache location:** `~/.cryptospect-cli/cache/` by default. Override with `cache.dir` in your config file.

**TTLs** (approximate defaults):

| Source | Endpoint | TTL |
|--------|----------|-----|
| CoinGecko | Global market, stablecoins | 300 s |
| CoinGecko | Coin markets (breadth, momentum) | 300 s |
| Binance US | Spot klines (CVD) | 60 s |
| DefiLlama | Stablecoins | 300 s |

The cache is **shared across all metrics** — a `market-regime` call will reuse the same CoinGecko response that `liquidity-pulse` fetched moments earlier. Running all six metrics back-to-back typically results in only 3–4 actual API calls.

**Clearing the cache:**

    cryptospect-cli cache-clear

Use this to force fresh data after a market event, after changing API keys, or to troubleshoot unexpectedly stale output. The cache repopulates automatically on the next metric call.

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

The config file lives at `~/.cryptospect.yaml` by default. Override with `--config /path/to/file`. The file must have permissions `0600` or stricter (the tool refuses to read world-readable configs).

Full example with all supported fields:

```yaml
# ~/.cryptospect.yaml

apis:
  coingecko:
    api_key: ""          # or set CRYPTOSPECT_COINGECKO_KEY env var
  binance:
    api_key: ""          # or set CRYPTOSPECT_BINANCE_KEY env var

cache:
  enabled: true
  dir: ""                # default: ~/.cryptospect-cli/cache/
  ttl:                   # per-endpoint TTL overrides (seconds)
    coingecko.global_market: 300
    binance.spot_cvd_btc_1h: 60

output:
  format: json           # only "json" supported today
  pretty: false          # set true for indented JSON output
```

Key precedence (highest to lowest): CLI flag `--api-key` → env var → config file.

## Data Sources

- CoinGecko (free tier, primary — global market, stablecoins, derivatives, coin markets)
- Binance US (free tier — spot CVD klines)
- CoinDesk (stub, placeholder)
- CoinMetrics Community (stub, placeholder)

## Build

    make build    # compile to bin/cryptospect-cli (static binary, CGO_ENABLED=0)
    make fmt      # format with goimports + gofumpt
    make lint     # run golangci-lint v2
    make vet      # run go vet
    make test     # run tests with race detector and coverage
    make clean    # remove build artifacts

**`make build` vs `go build`**

`make build` produces a stripped static binary with `CGO_ENABLED=0 GOOS=linux GOARCH=amd64`. It also injects a version string from the nearest git tag (if one exists).

`go build -o ./cryptospect-cli ./cmd/cryptospect-cli/` is faster for development — it skips cross-compilation and produces a debug binary. The version will show `v1.0.0 (commit-dirty)` using the commit hash from build metadata.

**Version strings**

| Build method | Example version |
|---|---|
| `make build` with a git tag | `v1.0.0` |
| `make build` without a tag (current) | `v1.0.0 (c2331fe-dirty)` |
| `go build` (dev) | `v1.0.0 (c2331fe-dirty)` |

The source default is always `v1.0.0`. `make build` overrides it via ldflags when a tag exists.

*(No git tags exist yet — both `make build` and `go build` currently produce the dev format.)*

## Requirements

- Go 1.25+ (see .go-version for exact patch)
- golangci-lint v2.11+ (for linting only, not required to build)

## License

MIT — see [LICENSE](LICENSE).
