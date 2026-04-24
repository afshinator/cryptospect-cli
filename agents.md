# Agent Guide: cryptospect-cli

**Source of truth:** `Design‑Decisions.md` — CLI signatures, envelope schema, and error handling derived from there.

## What This Tool Does
Computes crypto market regime metrics for agentic consumption. Outputs machine-readable JSON optimized for low token count.

## CLI Signatures (v1)

### Global Metrics (Tier 2+3) — Planned for v1, not yet implemented
    cryptospect-cli liquidity-pulse      (alias: lp)   [--detail basic|extended|full]
    cryptospect-cli stablecoin-power     (alias: sp)   [--detail basic|extended|full]
    cryptospect-cli flow-tension         (alias: ft)   [--detail basic|extended|full]
    cryptospect-cli market-breadth       (alias: mb)   [--detail basic|extended|full]
    cryptospect-cli momentum-divergence  (alias: md)   [--detail basic|extended|full]
    cryptospect-cli market-regime        (alias: mr)   [--detail basic|extended|full]

### Utility Commands
    cryptospect-cli list-metrics         # list all available metrics + aliases
    cryptospect-cli cache-clear          # clear the local API response cache

### Future Per‑Asset Commands (planned)
    cryptospect-cli regime               --asset <SYM> --window <DURATION>
    cryptospect-cli zscore               --asset <SYM> --period <DURATION>
    cryptospect-cli rvol                 --asset <SYM>
    cryptospect-cli correlation          --pair <SYM,SYM> --window <DURATION>
    cryptospect-cli summary              --assets <SYM,SYM,...> --output json

## Output Envelope (all commands)
Every invocation returns this structure on stdout:

    {
      "status": "ok|error",
      "ts": 1744444800,
      "results": [
        {
          "metric": "liquidity-pulse",
          "status": "ok|degraded|unavailable",
          "data": { ... },           # metric‑specific fields
          "meta": { ... }            # omitted if --detail basic
        }
      ],
      "error": {
        "code": 429,
        "msg": "rate_limited",
        "retry_after_sec": 60,
        "source": "coingecko"
      }
    }

- Single‑metric commands return a `results` array with one element (forward‑compatible).
- `--detail basic` (default): `meta` omitted.
- `--detail extended`: `meta` includes cache hit, TTL, source timestamps.
- `--detail full`: `meta` includes thresholds, metric description.

## Error Handling
- On rate limit (429): parse retry_after_sec, sleep, retry
- On error: check error.source to determine which API failed
- Never parse stderr — it contains debug logs only
- Exit 0 for success AND handled errors; non‑zero only for unrecoverable failures

## API Key Injection
Precedence: --api-key flag (CoinGecko) > CRYPTOSPECT_COINGECKO_KEY / CRYPTOSPECT_BINANCE_KEY env vars > ~/.cryptospect.yaml