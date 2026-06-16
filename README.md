# cryptospect-cli

A portable CLI tool that fetches live cryptocurrency data, computes high-signal market regime metrics, and outputs clean JSON — optimized for AI agents, LLM tool-calling, and MCP workflows.


![cryptospect-cli](./project-image-01.png)

## Getting Started

### Download a pre-built binary

Grab the latest release for your OS from the [Releases page](https://github.com/afshinator/cryptospect-cli/releases), extract it, and run:

```bash
# Linux / macOS
tar xzf cryptospect-cli_*.tar.gz
./cryptospect-cli list-metrics

# Windows (PowerShell)
tar xzf cryptospect-cli_*.zip
./cryptospect-cli list-metrics
```

### Build from source

```bash
git clone https://github.com/afshinator/cryptospect-cli
cd cryptospect-cli
go build -o cryptospect-cli ./cmd/cryptospect-cli/
./cryptospect-cli list-metrics
```

No API keys required — all metrics work on free public tiers.  But if you provide a (free) Coingecko api key, it'll work better!

### Your first command

```bash
cryptospect-cli market-regime    # macro snapshot
cryptospect-cli liquidity-pulse  # drill into liquidity
cryptospect-cli --help           # all commands
```

## CoinGecko API Key (recommended)

All 10 metrics work without any API key on free public tiers. However, **CoinGecko is the primary data source** for most metrics, and the free tier is rate-limited. A free CoinGecko Demo API key is strongly recommended for regular use — it relaxes rate limits and unlocks more reliable data on especially the `flow-tension` metric.

Get a free key at [coingecko.com/en/developers](https://www.coingecko.com/en/developers), then set it one of three ways:

```bash
# Environment variable (recommended for agent/MCP use)
export CRYPTOSPECT_COINGECKO_KEY=your_key_here

# CLI flag (per-call)
cryptospect-cli liquidity-pulse --api-key your_key_here

# Config file (~/.cryptospect.yaml)
apis:
  coingecko:
    api_key: your_key_here
```

## What It Measures

Ten signals, each answering a different question about current market conditions:

| Metric | Alias | What it tells you |
|--------|-------|-------------------|
| `liquidity-pulse` | `lp` | Is money actively moving, or is the market thin and idle? |
| `stablecoin-power` | `sp` | How much dry powder is sitting on the sidelines? |
| `flow-tension` | `ft` | Are buyers or sellers winning right now? |
| `market-breadth` | `mb` | Is a move broad-based or driven by a handful of large caps? |
| `momentum-divergence` | `md` | Where is capital rotating across large-, mid-, and small-cap tiers? |
| `market-regime` | `mr` | What is the overall market state? Aggregates all signals into a single regime label with a confidence score. |
| `dominance` | `dom` | Is capital rotating into or out of BTC and ETH? |
| `volatility` | `vol` | Are markets calm or turbulent? |
| `fear-greed-index` | `fgi` | What is the current crowd sentiment — fear or greed? |
| `china-m2` | `cnm2` | Is China loosening or tightening monetary conditions? |

**Good starting point:** run `market-regime`, `fgi`, and `cnm2` together for a macro picture, then drill into individual metrics.

## Install

```bash
git clone https://github.com/afshinator/cryptospect-cli
cd cryptospect-cli
make build
./bin/cryptospect-cli list-metrics
```

> For build details, Go version requirements, and development setup, see [DEVELOPMENT.md](docs/DEVELOPMENT.md).

## Agent & MCP Integration

Every invocation writes exactly one JSON object to stdout. Diagnostic logs go to stderr only. This makes `cryptospect-cli` easy to wrap as an LLM tool or MCP resource.

Example tool definition for an agentic workflow:

```json
{
  "name": "crypto_market_regime",
  "description": "Get the current overall crypto market state — regime label, confidence score, and contributing signals",
  "parameters": {},
  "command": "cryptospect-cli market-regime --detail full --output json"
}
```

Use `--detail full` when feeding output to an LLM — it includes metric descriptions and thresholds that help the model interpret the data. Use `--detail basic` (the default) for lightweight agent loops where token economy matters.

See [agents.md](agents.md) for the reasoning guide: which metrics to run for a given question, 16 named cross-metric signal patterns to recognize, and rules for turning metric output into grounded answers.

## Output Format

### Success

```json
{
  "status": "ok",
  "ts": 1744444800,
  "results": [
    {
      "metric": "liquidity-pulse",
      "version": "v1.0.0",
      "namespace": "cryptospect",
      "status": "ok",
      "data": { "..." : "..." },
      "meta": { "..." : "..." }
    }
  ]
}
```

### Error

```json
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
```

### Detail Levels

| Flag | `meta` contents | Best for |
|------|----------------|----------|
| `--detail basic` (default) | omitted | lightweight agent loops |
| `--detail extended` | cache hit, TTL remaining, source timestamps | debugging / monitoring |
| `--detail full` | + thresholds and metric description | LLM tool input |

Exit code is `0` for both success and handled errors (e.g. degraded data); non-zero only for unrecoverable failures.

## Commands

```bash
cryptospect-cli liquidity-pulse      # alias: lp    [--detail basic|extended|full]
cryptospect-cli stablecoin-power     # alias: sp    [--detail basic|extended|full]  [--top N]
cryptospect-cli flow-tension         # alias: ft    [--detail basic|extended|full]
cryptospect-cli market-breadth       # alias: mb    [--detail basic|extended|full]  [--top N]
cryptospect-cli momentum-divergence  # alias: md    [--detail basic|extended|full]  [--segments N]
cryptospect-cli market-regime        # alias: mr    [--detail basic|extended|full]
cryptospect-cli dominance            # alias: dom   [--detail basic|extended|full]
cryptospect-cli volatility           # alias: vol   [--detail basic|extended|full]
cryptospect-cli fear-greed-index     # alias: fgi   [--detail basic|extended|full]
cryptospect-cli china-m2             # alias: cnm2  [--detail basic|extended|full]

cryptospect-cli list-metrics         # list all metrics and aliases
cryptospect-cli cache-clear          # clear the local API response cache
```

### Global Flags

```
--output, -o    Output format: json (default)
--verbose, -v   Enable debug logging on stderr
--detail        Detail level: basic (default), extended, full
--api-key       CoinGecko API key (per-call override)
--config        Config file path (default: ~/.cryptospect.yaml)
```

## Caching

The tool caches API responses to disk so that running multiple metrics back-to-back stays fast and rate-limit safe. The cache is shared — a `market-regime` call reuses the same CoinGecko response that `liquidity-pulse` fetched moments earlier. Running all ten metrics typically triggers only 3–4 actual API calls.

Cache location: `~/.cryptospect-cli/cache/` (override with `cache.dir` in config).

```bash
cryptospect-cli cache-clear   # force fresh data after a market event or key change
```

If an API is unreachable and a cached response exists, the tool uses it and reports `status: degraded`. If there's no cache at all, it reports `status: unavailable`.

## Configuration

Config file lives at `~/.cryptospect.yaml`. Must have permissions `0600` or stricter.

```yaml
apis:
  coingecko:
    api_key: ""        # or set CRYPTOSPECT_COINGECKO_KEY env var

cache:
  enabled: true
  dir: ""              # default: ~/.cryptospect-cli/cache/
  ttl:
    coingecko.global_market: 300
    binance.spot_cvd_btc_1h: 60

output:
  format: json
  pretty: false        # set true for indented JSON output
```

Key precedence (highest → lowest): CLI flag `--api-key` → environment variable → config file.

## Data Sources

| Source | Used for |
|--------|----------|
| CoinGecko (free tier) | Global market, stablecoins, derivatives, coin markets — primary source |
| Binance US (free tier) | Spot CVD klines, hourly volatility candles |
| alternative.me (free tier) | Fear & Greed Index |
| DBnomics (free tier) | China M2 money supply |
| DefiLlama (free tier) | Stablecoin data |

## License

MIT — see [LICENSE](LICENSE).