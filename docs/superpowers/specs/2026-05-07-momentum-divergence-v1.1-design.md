# momentum-divergence v1.1 Design: Market-Cap Weighted Tier Means

**Date:** 2026-05-07  
**Status:** Design — pending implementation  
**Scope:** momentum-divergence metric only (md)

---

## Motivation

Live run on 2026-05-07 produced `neutral` with `mid_vs_large = +1.72pp`. Inspection of
`tier_detail.mid` revealed TON at +27.2% and NEAR at +13.6% — together adding roughly
+1.0pp to the 39-coin mid-tier average. The simple arithmetic mean weights a $7B
mid-cap coin identically to a $30B coin. Under simple means, a single idiosyncratic
catalyst in a 40-coin tier can swing the tier average by ~0.7pp and move the spread
closer to (or further from) classification thresholds without reflecting any genuine
market-wide rotation.

This is the v1 Implementation Compromise explicitly documented in
`docs/metrics/momentum-divergence.md`:

> "BTC simple-mean distortion (most consequential known limitation). BTC and ETH
> together often represent 60–70% of aggregate large-cap market cap but carry only
> 2 of 10 equal weights in the simple mean."

The fix: replace arithmetic means with **market-cap weighted means**, so each coin's
contribution to its tier average reflects its economic weight.

---

## What Changes

### Formula

**v1 (current):**
```
tier_avg = sum(price_change_24h_i) / n_valid
```

**v1.1:**
```
tier_avg = sum(price_change_24h_i * market_cap_i) / sum(market_cap_i)
```
where both sums run over coins with non-null `price_change_percentage_24h` AND
non-zero `market_cap`. A coin with a valid price change but null/zero market_cap is
excluded from the weighted average (same null-exclusion pattern as v1's price-change
exclusion). If all valid-price coins in a tier have null/zero market_cap, the tier
falls back to simple mean and documents this in meta (`weighting_fallback: true`).

### Data layer

`CoinMarketsBreadthEntry` (the shared entry type, `internal/api/coingecko/client.go`)
does not currently capture `market_cap` from the API response. The `/coins/markets`
endpoint already returns it in the wire format — no URL change needed.

**Required changes (additive, no breaking impact on market-breadth):**
1. Add `MarketCap *float64 \`json:"market_cap"\`` to `CoinMarketsBreadthEntry`
2. Add `MarketCap float64` to `CoinMarketsRankedCoin` (the per-coin struct passed to
   momentum-divergence compute)
3. Update `ParseCoinMarketsRankedResponse` to copy `e.MarketCap` (dereference or 0
   if nil) to the ranked coin

### Compute layer

`internal/metrics/momentumdivergence/v1/compute.go`:

Replace `meanTier(coins []tierCoin)` with `weightedMeanTier(coins []tierCoin)`:

```go
func weightedMeanTier(coins []tierCoin) (avg float64, fallback bool) {
    weightedSum := 0.0
    totalWeight  := 0.0
    for _, tc := range coins {
        if tc.marketCap > 0 {
            weightedSum += tc.change24h * tc.marketCap
            totalWeight += tc.marketCap
        }
    }
    if totalWeight == 0 {
        // All coins have null/zero market_cap — fall back to simple mean
        return meanTier(coins), true
    }
    return weightedSum / totalWeight, false
}
```

`tierCoin` struct gains a `marketCap float64` field.

`computedMeta` gains a `WeightingFallback bool` to surface the fallback condition.

### Output schema changes

**`meta` (extended and full detail):**

```json
"weighting_method": "market_cap"   // new field; "simple" if full fallback
```

**`tier_detail` entries (full detail only):**

```json
{
  "id": "bitcoin",
  "return_24h": -0.318,
  "market_cap": 1953000000000,   // new field
  "weight_pct": 52.3             // share of tier market cap, 2dp; aids outlier inspection
}
```

`weight_pct` = `market_cap_i / sum(market_cap_tier) * 100`. Lets agents immediately
see "TON was 4.7% of mid-tier weight" instead of "1 of 39 coins".

**`TierCoinDetail` struct:**

```go
type TierCoinDetail struct {
    ID        string   `json:"id"`
    Return24h *float64 `json:"return_24h"`
    MarketCap float64  `json:"market_cap,omitempty"`   // new; omitted if zero
    WeightPct float64  `json:"weight_pct,omitempty"`   // new; omitted if zero
}
```

### Version bump

`MetricVersion = "v1.1.0"`. Output envelope `version` field changes from `"v1.0.0"`
to `"v1.1.0"`. Numeric values in `tier_averages` and `spreads` will differ from v1.
Classification labels may change for borderline cases.

---

## Design Decisions

### Why market-cap weighted, not volume weighted?

The spec's Future Enhancements section lists both:
- "Volume-weighted tier means" (as the fix for outlier sensitivity)
- "Volume conviction (`tier_volume_intensity`)" (as a separate signal)

These are different things. Market-cap weighting fixes the *averaging* problem (a
coin's contribution to the tier average should reflect its economic weight). Volume
conviction is a *separate boolean flag* indicating whether a price move had trading
volume behind it. Volume conviction requires a prior cached baseline value for
comparison, which has no clean solution in the current cache architecture (noted in
the v1 spec). Market-cap weighting has no such dependency — the market_cap field is
in the same API response, same call, no new endpoints.

**Decision:** v1.1 implements market-cap weighted means. Volume conviction is deferred
to v1.2, pending the historical cache layer.

### Stablecoin impact under market-cap weighting

USDT (~$150B market cap) at rank 3–4 in the large tier will carry significant weight
under market-cap weighting. Its near-zero 24h return will dampen `large_avg` toward
zero more than it does under simple means. This is *correct* — USDT's economic
footprint in the large-cap tier is real. The effect may make `top_heavy` and
`flight_to_safety` harder to trigger (large_avg is pulled toward 0 by stable weight).

The `--exclude-stables` flag (already in the v1 spec Future Enhancements) becomes
more impactful under market-cap weighting than under simple means. It remains out of
scope for v1.1 — document this interaction in the spec update.

### Threshold recalibration

The v1 thresholds (risk_on > +5pp, top_heavy < -3pp, min_positivity_guard = 1.0)
were set against simple-mean values. Under market-cap weighting:
- `large_avg` will be more dominated by BTC/ETH → spreads likely *narrower* in
  absolute value (large and mid averages pulled toward BTC/ETH behavior)
- The +5pp `risk_on` threshold may be too conservative for weighted means

**Decision:** v1.1 ships with the same thresholds. Threshold recalibration is a
post-v1.1 task that requires accumulated real output data to backtest. The spec
already documents this as a planned step. Add a note in `meta.thresholds` commentary
that thresholds were calibrated against v1 simple means and may require adjustment
once v1.1 data accumulates.

### Backward compatibility

- Field names in `data` are unchanged; values differ.
- `version` bumps to `v1.1.0` — agents that branch on version can detect the change.
- Two new fields added to `meta` (`weighting_method`) and `tier_detail` entries
  (`market_cap`, `weight_pct`). Both additive.
- `weighting_method: "market_cap"` is always present at extended/full detail.
- E2E test assertions on specific numeric values (if any exist) must be updated.

---

## What Is NOT in v1.1

| Enhancement | Reason deferred |
|---|---|
| `--sensitivity` flag | Orthogonal to weighting; own v1.1 item, separate PR |
| Volume conviction (`tier_volume_intensity`) | Requires prior-cache baseline — no clean solution yet |
| `delta_24h` on spreads | Same cache dependency |
| `--exclude-stables` | Useful with weighting, but scope increase; v1.2 |
| Threshold recalibration | Needs real v1.1 output data to backtest |
| Pagination above 200 | Infrastructure change, unrelated |

---

## Implementation Plan

**T1 — API data layer** (`internal/api/coingecko/client.go`)
- Add `MarketCap *float64 \`json:"market_cap"\`` to `CoinMarketsBreadthEntry`
- Add `MarketCap float64` to `CoinMarketsRankedCoin`
- Update `ParseCoinMarketsRankedResponse` to populate `MarketCap`
- Update existing `coinmarkets_test.go` parser tests to cover `market_cap` field

**T2 — Compute layer** (`internal/metrics/momentumdivergence/v1/`)
- Add `marketCap float64` to `tierCoin` (internal struct)
- Add `WeightingFallback bool` to `computedMeta`
- Implement `weightedMeanTier` alongside existing `meanTier` (keep `meanTier` as
  the fallback path — don't delete it)
- Update tier assignment loop to copy `MarketCap` from coin to `tierCoin`
- Replace `meanTier(coins)` calls with `weightedMeanTier(coins)`
- Add `MarketCap float64` and `WeightPct float64` to `TierCoinDetail`
- Populate `weight_pct` in `TierDetail` construction
- Update `Thresholds` map to add `"weighting_method_note"` comment (or separate
  meta field — see schema above)

**T3 — Provider layer** (`internal/metrics/momentumdivergence/v1/provider.go`)
- Add `weighting_method` to `metaMap` (value from `compMeta.WeightingFallback`)
- Version constant: `MetricVersion = "v1.1.0"`

**T4 — Spec update** (`docs/metrics/momentum-divergence.md`)
- Bump version to `v1.1.0` in header
- Update Formula section with weighted mean formula and null-market_cap handling
- Update Output Schema to document `weighting_method`, new `tier_detail` fields
- Move "volume-weighted tier means" from Future Enhancements to Implemented
- Update Implementation Compromises: remove "Simple means over volume-weighted
  means" compromise; add new compromise about stablecoin market-cap weight impact
- Add note that thresholds were calibrated against simple means and may need
  recalibration once v1.1 data accumulates

**T5 — Tests**
- `coinmarkets_test.go`: add fixture field `market_cap`, verify it parses into
  `CoinMarketsRankedCoin.MarketCap`
- `compute_test.go`: update all existing fixtures to include market_cap values;
  add table-driven test for `weightedMeanTier` covering: normal weighting, zero
  market_cap fallback, mixed null/non-null, single coin, equal weights
- `provider_test.go`: update `rankedFixture` to include `market_cap` fields;
  verify `weighting_method` present in meta at extended detail
- `momentum_divergence_e2e_test.go`: add assertion that `weighting_method` is
  present at extended detail

---

## Open Questions

1. **`weight_pct` precision:** 2dp (`52.34`) vs 4dp vs integer. Recommendation: 2dp
   — it's a display/inspection aid, not used in computation.

2. **`weighting_method` in basic detail:** Currently only extended/full detail
   exposes meta. Weighting method is a fundamental change to how the metric works —
   should it surface at basic detail too? Recommendation: no. Basic detail is
   signal-only. Agents that care about the weighting method can use extended detail.

3. **Fallback threshold:** If only 1 coin in a tier has a valid market_cap, it
   carries 100% weight — effectively a single-coin average. Should there be a
   minimum number of weighted coins before fallback triggers? The v1 `TierFloorMinCoins = 3`
   floor already ensures at least 3 price-valid coins per tier; applying the same
   floor to market-cap-valid coins is the natural choice.
   Recommendation: if fewer than `TierFloorMinCoins` coins have non-zero market_cap,
   flag `weighting_fallback: true` and use simple mean for that tier.

4. **Impact measurement:** We have today's live output (2026-05-07, `neutral`,
   mid_vs_large +1.72pp). Running v1.1 against the same cached data would show the
   exact numeric impact. Recommendation: after implementation, run both versions
   against the same payload and document the delta in the commit message.
