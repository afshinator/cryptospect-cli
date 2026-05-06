# momentum-divergence — Implementation Design

**date:** 2026-05-06
**status:** approved
**metric:** `momentum-divergence` (`md`)
**source spec:** `docs/metrics/momentum-divergence.md`

## Decisions

| # | Topic | Decision |
|---|-------|----------|
| D1 | Endpoint | Share `coingecko.coin_markets_breadth` with market-breadth (cache-sharing). Not a separate momentum endpoint. |
| D2 | Parser | New `ParseCoinMarketsRankedResponse` that reads per-coin `market_cap_rank` and `price_change_percentage_24h_in_currency`. Nil-rank coins excluded entirely (no positional fallback). Existing breadth parser untouched. |
| D3 | API type | Add `MarketCapRank *int` to `CoinMarketsBreadthEntry`. New `CoinMarketsRankedData` / `CoinMarketsRankedCoin` output types. |
| D4 | Flags | `--segments` (comma-separated string, default `"10,50,200"`). No `--top` flag. |
| D5 | Fetch count | Hardcoded `CoinMarketsBreadthURL(250)`. Both md and mb use 250; each filters logically. Cache starvation guard: if CoinCount < 250, confidence "low". |
| D6 | Small ceiling | Hard-clamped to 250 (up from 200 in spec). Large ceiling minimum 5. |
| D7 | Spreads | `*float64` — nil = absent tier, 0.0 = valid real value. Never serialize 0.0 for missing spreads. |
| D8 | tail_extension | Always-present `bool` in Data. Decoupled from `risk_on` label. |
| D9 | Barbell modifier | Fires on `neutral` + `tail_extension: true` + `midVsLarge > 1.0pp`. Description override only; label stays `neutral`. |
| D10 | Summary prefix | `[RISK_ON]`, `[TOP_HEAVY]`, `[FLIGHT_TO_SAFETY]`, `[NEUTRAL]` prefix. Long-tail weakness warning when `smallVsLarge < -5pp` during `risk_on`. |
| D11 | Dead band | `large_avg` within ±0.5% → `top_heavy`/`flight_to_safety` collapse to `neutral`. |
| D12 | Confidence | "high" if all 3 tiers have ≥ 3 valid coins; "low" if any tier < 3; "low" if CoinCount < 250 (starvation). |
| D13 | DataTimestamp | From cache `FetchedAt` metadata, not `time.Now()`. |
| D14 | SegmentsClamped | `*bool` with omitempty; set before Compute call. |
| D15 | TierDetail | Per-coin `*float64` returns; null-change coins appear (nil) for transparency. Full detail only. |
| D16 | Validator | None. BTC CVD rejected as adversarial by design. Self-consistent, not self-validating. |

## Files

| File | Action | Purpose |
|------|--------|---------|
| `internal/api/coingecko/client.go` | Modify | Add `MarketCapRank` to entry; add `CoinMarketsRankedData`/`CoinMarketsRankedCoin`; add `ParseCoinMarketsRankedResponse` |
| `internal/metrics/registry.go` | Modify | Change md endpoint from `CoinGeckoCoinMarketsMomentum` to `CoinGeckoCoinMarketsBreadth` |
| `internal/metrics/momentumdivergence/v1/types.go` | Create | Input, Data, TierAverages, Spreads, Classification, Meta, constants |
| `internal/metrics/momentumdivergence/v1/compute.go` | Create | `Compute(Input) (Data, Meta, error)` — 6-stage pipeline |
| `internal/metrics/momentumdivergence/v1/provider.go` | Rewrite | Full Provider with `--segments`, parsing, compute call, meta |
| `internal/metrics/momentumdivergence/v1/compute_test.go` | Create | Table-driven compute tests |
| `internal/metrics/momentumdivergence/v1/provider_test.go` | Rewrite | Provider + flag tests |
| `cmd/cryptospect-cli/momentum_divergence_e2e_test.go` | Create | E2E CLI tests |

## Architecture

```
cmd/cryptospect-cli/root.go  ←  generic dispatcher (existing, no changes)
         │
         ▼
momentumdivergence/v1/provider.go  ←  I/O boundary
    │ parses --segments from ctx
    │ fetches data[coingecko.coin_markets_breadth]
    │ calls coingecko.ParseCoinMarketsRankedResponse()
    │ builds Input
    │ calls Compute()
    │ marshals Data + Meta
    │ returns output.MetricResult
    ▼
momentumdivergence/v1/compute.go  ←  pure function
    Stage 1: Tier construction (rank-based, null-exclusion)
    Stage 2: Statistical floor check
    Stage 3: Tier averages (simple mean)
    Stage 4: Spread matrix (*float64, nil-safe)
    Stage 5: Classification (label + tree)
    Stage 6: Summary string
```

## Compute Pipeline

### Stage 1 — Tier Construction
- Iterate `input.Coins`. Skip if `Change24h == nil`.
- Assign by `coin.MarketCapRank` to tier_large (1..largeCeiling), tier_mid (largeCeiling+1..midCeiling), tier_small (midCeiling+1..smallCeiling).

### Stage 2 — Statistical Floor
- Each tier needs ≥ 3 valid coins (TierFloorMinCoins).
- Absent tier → produce nil spreads for any constituent pair.
- All three absent → status "unavailable".

### Stage 3 — Tier Averages
- Simple mean of non-null Change24h values per tier.

### Stage 4 — Spread Matrix
- `midVsLarge`, `smallVsLarge`, `smallVsMid` = `*float64`.
- Computed only when both constituent tiers are present. Else nil.

### Stage 5 — Classification
```
if midVsLarge == nil:
    label = "neutral"
else if *midVsLarge > +5.0 AND mid_avg > 1.0:
    label = "risk_on"
else if *midVsLarge < -3.0:
    if large_avg > +0.5:   label = "top_heavy"
    else if large_avg < -0.5: label = "flight_to_safety"
    else: label = "neutral"   // dead band
else:
    label = "neutral"

tail_extension = (smallVsLarge != nil && *smallVsLarge > +5.0)
```
Barbell modifier: `neutral` + `tail_extension` + `midVsLarge > +1.0` → desc = "Barbell — Speculative Tail Extension".

### Stage 6 — Summary String
`[LABEL]` prefix. Key spreads and tier averages. Long-tail weakness warning when `smallVsLarge < -5pp` during `risk_on`.

## Constraints

- Go stdlib only. No testify, no external test libs.
- Pure Compute function — no I/O, no side effects.
- Stdout always `CLIResponse` JSON. Stderr for slog only.
- TDD: red-green-refactor for every behavior.
- After changes: `go build`, `make test -race -cover`, `make lint`, `make fmt`.
