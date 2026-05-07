# momentum-divergence v1.1 Design: Market-Cap Weighted Tier Means

**Date:** 2026-05-07  
**Status:** Finalized — ready for implementation  
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
non-zero `market_cap`. A coin must have valid price change AND non-zero market cap
to enter tier averages. Coins with null price change or zero market_cap are excluded.
If all price-valid coins in a tier have null/zero market_cap, the tier falls back to
simple mean and documents this in meta (`weighting_fallback: true`).

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

`internal/metrics/momentumdivergence/v1/{compute.go, types.go}`:

**Tier assignment (validity condition change):** In v1, a coin enters tier averages if
`Change24h != nil`. In v1.1, a coin must have `Change24h != nil` **AND** `MarketCap > 0`.
The statistical floor check remains at `TierFloorMinCoins = 3`. If fewer than
`TierFloorMinCoins` coins have *both* valid price change and non-zero market cap,
the tier is marked absent (same floor as v1).

**Weighting fallback semantics:**
`weighting_fallback: true` fires only in a specific edge case: a tier has
≥ `TierFloorMinCoins` price-valid coins but zero market-cap-valid coins — the
degenerate API state where the endpoint returns ranks and prices but no market_cap
values. This is not the tier-absent condition; it means the tier is present (enough
price data) but weighting degrades to simple mean because market_cap data is absent.

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

### T1 — API data layer
**File:** `internal/api/coingecko/client.go`

1. Add `MarketCap *float64 \`json:"market_cap"\`` to `CoinMarketsBreadthEntry` (shared entry type, additive — mb unaffected)
2. Add `MarketCap float64` to `CoinMarketsRankedCoin`
3. In `ParseCoinMarketsRankedResponse`: copy `e.MarketCap` (dereference to 0.0 if nil) into the ranked coin
4. Add `market_cap` fixture to `coinmarkets_test.go`; verify it parses into `CoinMarketsRankedCoin.MarketCap`

---

### T2 — Compute layer
**Files:** `internal/metrics/momentumdivergence/v1/{compute.go, types.go}`

**types.go:**
- Add `MarketCap float64` field to `tierCoin` (internal)
- Add `WeightingFallback bool` to `computedMeta`

**compute.go — tier assignment loop:**
- Change validity condition from `Change24h == nil` exclusion only → **both** `Change24h == nil` AND `MarketCap == 0` exclusion. A coin must have valid price change AND non-zero market cap to enter tier averages.
- Copy `coin.MarketCap` into `tierCoin.marketCap`

**New function:**
```go
func weightedMeanTier(coins []tierCoin) (avg float64, fallback bool) {
    // sum(change24h_i * marketCap_i) / sum(marketCap_i)
    // coins with marketCap == 0 are skipped
    // if totalWeight == 0 → return meanTier(coins), true (weighting_fallback)
}
```

Replace `meanTier(coins)` calls with `weightedMeanTier(coins)`, capture the fallback bool.

**TierDetail construction:** Add `MarketCap` and `WeightPct` (2dp) to `TierCoinDetail`. Populate both from `tierCoin`:
- `WeightPct = tc.marketCap / sumTierMarketCap * 100`

If fewer than `TierFloorMinCoins` coins have non-zero market cap: tier is absent (same as v1 floor — no special fallback needed at tier floor level).

`weighting_fallback: true` is set only when a tier has ≥ `TierFloorMinCoins` price-valid coins but zero market-cap-valid coins — the degenerate API state.

---

### T3 — Provider layer
**File:** `internal/metrics/momentumdivergence/v1/provider.go`

1. Add `weighting_method: "market_cap"` (or `"simple"` if fallback) to `metaMap` — extended detail only
2. Bump `MetricVersion = "v1.1.0"`
3. Pass `compMeta.WeightingFallback` from compute result into `metaMap`

---

### T4 — Spec update
**File:** `docs/metrics/momentum-divergence.md`

1. Bump version header: `v1.0.0` → `v1.1.0`
2. Update Formula: weighted mean formula + double-null exclusion rule
3. Update Output Schema: add `weighting_method` (extended), `market_cap` (tier_detail, full), `weight_pct` (tier_detail, full)
4. **New section:** `## v1.1 Calibration Note` — document `label_drift` and `avg_magnitude_shift` as the baseline comparison metrics (not just in the commit message)
5. Move "volume-weighted tier means" from Future Enhancements → Implemented
6. Remove "Simple means over volume-weighted means" from Implementation Compromises
7. Add new Implementation Compromise: stablecoin market-cap weight impact under weighting
8. Add note: thresholds calibrated against v1 simple means — may need recalibration once v1.1 data accumulates

---

### T5 — Tests

- `coinmarkets_test.go` — add `market_cap` field to fixture; assert `MarketCap` parses correctly.
- `compute_test.go`:
  - Update all existing fixtures to include `market_cap` values (realistic distribution — BTC dominant, mid/small distributed)
  - New table-driven test `TestWeightedMeanTier` covering:
    - Normal weighting (BTC 52%, ETH 25%, rest 23%)
    - Zero market_cap fallback → `fallback: true`
    - Mixed null/non-null market_cap → only non-null count toward tier
    - Single coin with 100% weight
    - Equal weights (all same market cap)
  - Expected behavior baseline: assert specific `large_avg` values using calibration data — this becomes the regression anchor
- `provider_test.go`:
  - Add `MarketCap` to `rankedFixture`; assert `weighting_method` present in meta at extended detail
- `momentum_divergence_e2e_test.go`:
  - Assert `weighting_method` present in meta at extended detail (no assertions on specific numeric values that would break on real API data)

---

## Post-Implementation

After code is green, run both v1 and v1.1 against the same cached payload:
- Document `label_drift` (how many classifications changed)
- Document `avg_magnitude_shift` (pp delta on `large_avg`)
- Add these as comments or a test fixture so future refactors can detect regressions

---

## Order to Execute

```
T1 → T2 → T3 → T5 (compute/provider tests) → T4 → smoke test → make lint/fmt
```

T4 (spec) is last because it's documentation — don't finalize the doc until the code is stable.

---

## Post-Review (2026-05-07)

The fallback path and `WeightingFallback` field were removed during code review. The original
design specified a simple-mean fallback when a tier had ≥ `TierFloorMinCoins` price-valid
coins but zero market-cap-valid coins. This path was unreachable through production code
because `Compute()` filters out coins with `MarketCap <= 0` before tier construction.

Instead of restructuring the double-validity logic to make the fallback reachable, the
decision was to remove it entirely. `weighting_method` is always `"market_cap_weighted"`.
Coins without market cap are invalid at tier construction — they do not count toward the
statistical floor and do not enter tier averages. This makes the contract simpler (no
degenerate API edge case to surface) and removes the `WeightingFallback` field from
`computedMeta`.

Additional cleanups from the review: `omitempty` removed from `WeightPct` and `MarketCap`
in `TierCoinDetail` (stable JSON output), `TierAverages` comment fixed from "simple mean"
to "market-cap weighted", and `weighting_method` enum value renamed from `"market_cap"` to
`"market_cap_weighted"` for forward clarity.
