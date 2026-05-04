# market-breadth: Implementation Design

**Date:** 2026-05-04
**Metric doc:** `docs/metrics/market-breadth.md`
**Approach:** Pure compute + provider (matches flow-tension pattern)

## Architecture

```
internal/metrics/marketbreadth/v1/
├── types.go            -- Input, Data, Meta, Classification structs + constants
├── compute.go          -- Pure func Compute(input Input) (Data, error)
├── compute_test.go     -- 25+ table-driven tests
├── provider.go         -- MetricProvider: parse → compute → status/confidence → meta
├── provider_test.go    -- Mock-injection tests

cmd/cryptospect-cli/
└── market_breadth_e2e_test.go  -- 4 E2E scenarios

internal/api/coingecko/
├── client.go           -- Restructure CoinMarketsBreadthData (counts + BTCReference)
└── coinmarkets_test.go -- Update tests for per-timeframe denominator

internal/api/binance/
├── client.go           -- Add Close, OpenTime to KlinesData
└── client_test.go      -- Add assertions for new fields
```

No changes to: catalog.go, root.go, constants.go, fetcher.go, config.go, registry.go, helpers.go, envelope.go, writer.go, meta.go, float.go.

## Parser Changes

### Binance: `KlinesData` extension (additive)

Add `Close float64` (index 4, parseStringFloat) and `OpenTime int64` (index 0, json.Unmarshal to int64). Existing fields unchanged. Flow-tension and liquidity-pulse read only CVD fields — unaffected.

### CoinGecko: `CoinMarketsBreadthData` restructure

Replace fraction-based struct with per-timeframe counts:

```go
type TimeframeMetric struct { GreenCount, TotalCount int }
type BTCReference struct { PriceChange24h float64; Available bool }

type CoinMarketsBreadthData struct {
    TimeframeCounts map[string]TimeframeMetric
    CoinCount       int
    BTCReference    BTCReference
}
```

Parser loops coins once: for each coin, for each timeframe field, increment TotalCount if non-nil, increment GreenCount if non-nil AND > 0. BTC matched by `entry.ID == "bitcoin"`. If absent or Change24h nil: `BTCReference.Available = false`. No callers depend on this parser yet (market-breadth is scaffolded).

## Compute Layer (pure, no I/O)

```go
func Compute(input Input) (Data, error)
```

**Input** carries: per-timeframe GreenCount/TotalCount, BTC 24h change + available flag, Binance close + openTime + available flag, fetchedAt time.Time (for staleness check), topN int, now time.Time.

**Logic order:**
1. Per-timeframe floor: drop timeframes with TotalCount < 50, redistribute weight proportionally across remaining
2. Compute `green_pct = GreenCount / TotalCount` per valid timeframe
3. Weighted composite: `0.10*1h + 0.30*24h + 0.40*7d + 0.20*30d` (using effective weights)
4. Classify: >= 0.60 → broad, >= 0.40 → mixed, else narrow
5. Divergence: BTC available && `btc_change > 2.0 && score < 0.40`
6. Discrepancy: skip if close==0 or stale (>90min). Otherwise `sign(cg) != 0 && sign(bn) != 0 && sign(cg) != sign(bn)`
7. Summary: human-readable string combining score, classification, divergence state

**Returns** Data with: MarketBreadthScore, CoinsCounted, TimeframeBreadth, DivergenceDetected, BTCChange24h, Classification, Summary.

## Provider Layer

`Compute(ctx, data)` flow:
1. Parse `data[api.CoinGeckoCoinMarketsBreadth]` → `CoinMarketsBreadthData`
2. Parse `data[api.BinanceSpotCVD_BTC_1h]` → `KlinesData` (nil if fetch failed → validator unavailable)
3. Read `--top` from context (`config.TopNFromContext`), clamp [50, 250], fallback 250
4. Build `Input` struct, pass `time.Now()`
5. Call pure `Compute(input)`
6. Determine status per spec table: unavailable (CG fails/all absent), degraded (<50 valid coins / per-timeframe floor triggered), ok otherwise
7. Determine confidence: high (directions agree, fresh), medium (discrepancy or BTC null), low (stale, close==0, or Binance unavailable)
8. Build Meta with `weights_used`, `timeframe_counts`; add `thresholds` + `description` (full detail only)
9. Return `output.MetricResult`

Also implements `RegisterFlags(cmd)` for `--top` with default 250 (flagRegistrar interface).

## CLI Flag: --top

Default 250, minimum 50, hard-clamped max 250 in v1. Clamping enforced in provider, not in flag definition. Clamp produces `top_clamped: true` + `top_clamped_reason` in meta. Metric reads from context via existing `config.TopNFromContext` — root.go already propagates `--top` flag.

## Testing Plan

### Phase 0: Parser regression (before any parser change)
- `TestParseKlinesResponse_CloseAndOpenTime` — verify new fields from existing fixture
- `TestParseCoinMarketsBreadthResponse_NullExclusionDenominator` — per-timeframe denominator behavior
- `TestParseCoinMarketsBreadthResponse_BTCReference` — BTC extraction by ID

### Phase 1: Pure compute (compute_test.go, ~25 cases)
Happy path (broad score), narrow + Ghost Rally divergence, null fields across timeframes, weight redistribution (<50 coins), global floor (<50 valid), all absent (unavailable), BTC null, stale candle, zero close, boundary values (0.40, 0.60, 2.0), sign edge cases, summary format.

### Phase 2: Provider mock-injection (provider_test.go, ~10 cases)
Both endpoints healthy, CG fails, Binance fails, confidence tiers, --top clamping.

### Phase 3: E2E (4 scenarios)
Full success with mock servers, CG unavailable, Binance stale, --top flag propagation.

## Status & Confidence Truth Table

| Condition | Status | Confidence |
|-----------|--------|------------|
| CG parse fails | unavailable | n/a |
| All timeframes absent | unavailable | n/a |
| Total valid < 50 | degraded | per validator |
| Per-timeframe TotalCount < 50 | degraded | per validator |
| CG ok, Binance fetch fails | ok | low |
| Candle stale >90min | ok | low |
| Close == 0.0 | ok | low |
| BTC absent or Change24h nil | ok | medium |
| Directions agree, fresh | ok | high |

Status and confidence are independent axes. Status = data availability; confidence = validator quality.
