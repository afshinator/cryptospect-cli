# market-breadth

**version:** `v1.0.0`
**Alias:** `mb`
**Endpoints:** `coingecko.coin_markets_breadth`, `binance.spot_cvd_btc_1h`

## Overview

Measures market participation — whether a price move is a "rising tide" lifting all sectors or a "hollow" move driven by a narrow set of assets. Computed as the recency-biased weighted average of the percentage of coins in the top N (default: 250) that are "green" across four timeframes: 1h, 24h, 7d, and 30d. A high score confirms broad health; a low score with rising BTC price triggers the Breadth Divergence Detector, flagging a potential "Ghost Rally" — capital hiding in BTC while the broader market bleeds.

## Formula (or how to compute)

```
per-timeframe breadth (null-exclusion per timeframe):
  green_pct_1h  = GreenCount["1h"]  / TotalCount["1h"]   // TotalCount = coins where field is non-null
  green_pct_24h = GreenCount["24h"] / TotalCount["24h"]
  green_pct_7d  = GreenCount["7d"]  / TotalCount["7d"]
  green_pct_30d = GreenCount["30d"] / TotalCount["30d"]

  Coins with null fields are excluded from BOTH numerator and denominator for that timeframe.
  Denominator is NOT len(all_coins) — it is the per-timeframe valid count only.

zero-weight redistribution and statistical floors (edge cases):
  Per-timeframe floor: if TotalCount["X"] < 50, that timeframe is treated as absent —
    its weight is redistributed proportionally across the remaining valid timeframes.
    Example: if 1h has TotalCount < 50, its 0.10 weight splits 3:4:2 across 24h/7d/30d,
    yielding effective weights of 0.333/0.444/0.222.
  Note: TotalCount == 0 is a special case of TotalCount < 50 — treated identically.

  Global floor: if the total number of valid coins (coins with at least one non-null timeframe field)
    falls below 50, status → "degraded". Computing breadth on fewer than 50 coins produces
    a statistically unreliable signal regardless of whether the math is defined.
  If all four timeframes are absent across the entire coin set: status → "unavailable".

weighted composite (recency-biased):
  market_breadth_score = (0.10 × green_pct_1h)
                       + (0.30 × green_pct_24h)
                       + (0.40 × green_pct_7d)
                       + (0.20 × green_pct_30d)

result is a float in [0.0, 1.0], reported as a decimal (e.g., 0.63 = 63% weighted breadth)

breadth divergence detection (directional consensus):
  sign(x) is defined as:
    +1  if x > 0
    -1  if x < 0
     0  if x == 0  ("neutral" — treated as consensus pass, not a mismatch)

  dir_cg      = sign(btc_price_change_24h)           // from CoinGecko primary
  dir_binance = sign(kline_close - kline_open)       // from most recent binance.spot_cvd_btc_1h candle

  discrepancy_detected = (dir_cg != 0) AND (dir_binance != 0) AND (dir_cg != dir_binance)
  // Only fires when BOTH sides have a non-zero direction AND they disagree.
  // 0 vs +1, 0 vs -1, or 0 vs 0 → discrepancy_detected: false (consensus pass).
  // +1 vs -1 or -1 vs +1 → discrepancy_detected: true.

  divergence_detected = (btc_price_change_24h > 2.0) AND (market_breadth_score < 0.40)
  // CoinGecko price_change_percentage_24h_in_currency returns percentage directly:
  // 5.0 == +5%, so the threshold is 2.0, not 0.02.
```

**Weight rationale:** The 24h and 7d windows carry the most signal weight (30% and 40% respectively) because they capture the "sweet spot" for trend confirmation — long enough to filter noise, short enough to reflect current regime. The 1h window (10%) prevents a single hourly spike from skewing the verdict. The 30d window (20%) provides macro context without anchoring the score to a month-old regime that may no longer apply. Weights are a heuristic derived from common momentum trading strategies and are documented in `meta.weights` at extended/full detail for auditability.

**Threshold source:** The "Broad" / "Narrow" boundaries (60% / 40%) are based on historical crypto market participation studies and standard technical analysis breadth benchmarks. They are calibrated thresholds, not arbitrary round numbers, but should be treated as heuristics rather than statistically proven boundaries.

**Validator logic note:** The validator uses directional consensus, not a quantitative percentage threshold. Comparing a 1h kline delta against a 24h percentage change as a numeric ratio is mathematically invalid — they measure different windows and would produce meaningless discrepancy numbers. Instead, the validator checks whether Binance's short-term BTC direction (`sign(close - open)`) agrees with CoinGecko's 24h direction. Disagreement means the medium-term trend and the current short-term momentum are opposite — itself a microstructure signal (intraday reversal forming), not merely a data quality flag. A quantitative 24h-vs-24h threshold comparison is planned for v2 using `binance.ticker_24h`.

## Interpretation

- **Broad (>0.60):** "Healthy Growth." Most sectors are participating. Historically safe to look for laggard sectors and initiate new positions. When combined with high `stablecoin-power`, this is a structurally strong bull configuration.
- **Mixed (0.40–0.60):** "Selective Participation." The market is uncertain — some sectors lead while others bleed. Be selective; avoid treating broad exposure as equivalent to concentrated conviction plays.
- **Narrow (<0.40):** "The Illusion." Often signals a "flight to quality" where BTC or large-caps rise while the majority of the market is stagnant or falling. New long entries outside top-tier assets carry elevated risk.
- **Divergence Active (`divergence_detected: true`):** "Ghost Rally." BTC is up >2% on the day but breadth is weak (<40%). Capital is concentrating into BTC as a safe harbor while the broader market deteriorates. This condition historically precedes broader market reversals. Treat any rally as mechanically fragile until breadth confirms.

**V-shape lag caveat:** The multi-timeframe proxy can lag during rapid "V-shape" recoveries. A coin that is green on 24h but still red on 7d will be counted differently per timeframe, causing the 7d-weighted composite to understate short-term recovery momentum. This is the most likely scenario where this metric produces a misleading read — a sharp bottom reversal will appear as "Narrow" or "Mixed" for 24–48h until the 7d window catches up. The `1h vs 7d` recency-bias read in Agentic Logic is specifically designed to detect this case early.

## Classification

| Condition | Threshold |
|-----------|-----------|
| `broad` | `>= 0.60` |
| `mixed` | `0.40 – 0.60` |
| `narrow` | `< 0.40` |

**Divergence overlay** (independent of base classification):

| Condition | Logic |
|-----------|-------|
| `divergence_detected: true` | `btc_price_change_24h > 2.0` AND `market_breadth_score < 0.40` |
| `divergence_detected: false` | All other cases |

**Validator discrepancy** (independent signal, not a classification):

| `discrepancy_detected` | Meaning |
|------------------------|---------|
| `true` | Binance 1h BTC direction opposes CoinGecko 24h direction — intraday reversal may be forming |
| `false` | Both sources agree on BTC directional trend |

## Data Source(s)

- **Primary API:** CoinGecko
- **Endpoint key:** `coingecko.coin_markets_breadth` — `/coins/markets` with `per_page=250` (or `--top` value), `order=market_cap_desc`, `price_change_percentage=1h,24h,7d,30d`
- **Fields used:** `price_change_percentage_1h_in_currency`, `price_change_percentage_24h`, `price_change_percentage_7d_in_currency`, `price_change_percentage_30d_in_currency` (per coin); `id`, `price_change_percentage_24h` for the BTC entry (divergence reference)
- **Validator API:** Binance-US — `binance.spot_cvd_btc_1h` — most recent 1h kline `open` and `close` prices for directional consensus check

**Endpoint efficiency note:** A single `coingecko.coin_markets_breadth` call with `per_page=250` retrieves all four timeframe change fields for the full coin universe simultaneously, including the BTC reference needed for divergence detection — 1 API credit for the complete breadth matrix. This is the optimal cost-per-signal ratio for the CoinGecko Free/Demo tier.

**Validator cache alignment:** The `binance.spot_cvd_btc_1h` endpoint key is shared with `flow-tension`. When both metrics are called in the same session, the dispatcher serves the validator data from cache at zero additional API cost. This is a deliberate architectural choice — the validator was aligned to this key specifically to take advantage of the existing cache hit.

**Parser implementation requirements** (for `internal/api/`):

*CoinGecko parser — `ParseCoinMarketsBreadthResponse`:*
- Use a `TimeframeMetric{GreenCount int, TotalCount int}` struct per timeframe, stored in a `map[string]TimeframeMetric` keyed by `"1h"`, `"24h"`, `"7d"`, `"30d"`. Do not pre-divide to ratios in the parser — pass raw counts to `Compute` for accurate null-exclusion division.
- In the iteration loop: for each coin, for each timeframe field, increment `TotalCount` only if the field is non-null; increment `GreenCount` only if the field is non-null AND `> 0`.
- **Loop structure note:** A single per-coin outer loop is correct and efficient. The null check for each timeframe field must be evaluated independently — a null on the 1h field must not affect whether that coin is counted in the 7d denominator. Do not use `len(coins)` as the denominator for any timeframe. Do not skip remaining timeframe updates when one field is null. Each `(coin, timeframe)` pair is an independent inclusion decision.
- BTC reference: check `if entry.ID == "bitcoin"` (not `i == 0`) and store `entry.PriceChangePercentage24h` in `CoinMarketsBreadthData.BTCReference.PriceChange24h`. The ID check is preferred over index assumption — `market_cap_desc` ordering is stable but undocumented; a silent reorder would corrupt the divergence flag without error. **Note:** The actual struct field in the existing code is `Change24h *float64` tagged `price_change_percentage_24h_in_currency` — cross-reference `CoinMarketsBreadthEntry` in `internal/api/coingecko/` when implementing.
- **BTC null guard:** If no `ID == "bitcoin"` entry is found, or if the matched entry's `Change24h` is nil: set `BTCReference.PriceChange24h = 0.0` and set a `BTCReferenceUnavailable bool` flag on the struct. In `Compute`, if `BTCReferenceUnavailable`, set `divergence_detected: false`, `btc_change_24h_pct: 0.0`, `confidence: "medium"`, and `discrepancy_note: "BTC reference unavailable in response — divergence check skipped"`.

*Binance parser — `ParseKlinesResponse`:*
- Add `Close float64` and `OpenTime int64` (candle open timestamp, milliseconds epoch) to the parsed kline struct. Index 4 of the raw kline array is the close price; index 0 is the open time in milliseconds. Both are currently parsed but discarded. Expose them. This is an additive change: `flow-tension`'s `Compute` only reads CVD fields and is unaffected.
- The validator uses the **most recent candle only** (last element in the slice by timestamp). Do not average across candles.
- **Staleness check:** The relevant timestamp is `kline.OpenTime` — the time the candle itself opened — not `fetchedAt` (when the dispatcher retrieved the response). `fetchedAt` reflects cache retrieval time and will be recent even when the underlying candle is hours old; it is the wrong input for a freshness check. In `Compute`, convert `kline.OpenTime` from milliseconds to seconds (`openTimeSec := kline.OpenTime / 1000`) and compare against `input.Now.Unix()`. If `input.Now.Unix() - openTimeSec > 5400` (90 minutes), skip the directional comparison entirely: set `confidence: "low"`, `discrepancy_detected: true`, `discrepancy_note: "Binance candle stale (>90m) — validator skipped, directional consensus unavailable"`. Do not compute a fake direction from a stale candle.
- **Missing or zero Close guard:** Go's default `float64` value is `0.0`. If the `Close` field parses as `0.0`, treat it as a parse failure — a valid BTC/USDT price cannot be zero. In this case, skip the directional comparison and set `confidence: "low"` with a `discrepancy_note` of `"Binance Close price is 0.0 — likely a parse failure, validator skipped"`. Do not use a zero-value Close in the direction calculation. Distinguish "price is genuinely zero" (impossible for BTC/USDT) from "field was absent and Go defaulted to zero."

*`Compute` function — error semantics:*
```go
func Compute(input Input) (Data, error)
```
`Compute` returns a non-nil `error` **only** for unrecoverable internal inconsistency — for example, if `input` is in an impossible state that the dispatcher contract guarantees cannot occur (e.g., both `input.BreadthData` and `input.KlinesData` are nil simultaneously when the dispatcher guarantees at least the primary). All documented edge cases — null BTC reference, stale candle, zero Close, missing timeframe fields, per-timeframe TotalCount below floor, global coins_counted below floor — return `(Data{...}, nil)` with the appropriate sentinel values and status/confidence flags set on the returned `Data` struct. The provider maps `Data.Status` to the output `status` field; it does not treat edge-case `Data` returns as errors.

*`RegisterFlags`:*
```go
func RegisterFlags(cmd *cobra.Command) {
    cmd.Flags().Int("top", 250, "Number of top coins by market cap to include (min 50, max 250 in v1)")
}
```
Follows the same pattern as `stablecoin-power`. The clamp logic (enforce [50, 250]) lives in `Compute`, not in flag parsing.

**Field availability note:** Verify that `price_change_percentage_Xh_in_currency` fields are fully populated for the top 250 coins on your specific CoinGecko API tier. The 1h field in particular is occasionally omitted for lower-liquidity assets. The null-exclusion and weight-redistribution logic handles individual missing fields, but systematic omission of an entire timeframe for many coins will trigger weight redistribution — check `meta.weights_used` at extended/full detail to confirm which timeframes contributed to the composite.

## Cross-Source Verification

This metric uses the **Primary with Anchor-Asset Directional Consensus** pattern.

| Role | Source | Purpose |
|------|--------|---------|
| Primary | CoinGecko (`coingecko.coin_markets_breadth`) | Full-breadth computation across top 250 coins, all four timeframes |
| Validator | Binance-US (`binance.spot_cvd_btc_1h`) | BTC short-term direction check (directional consensus, not quantitative threshold) |

**Anchor asset:** BTC

**Validation method:** Directional consensus — not a quantitative percentage threshold.

```
sign(x): +1 if x > 0, -1 if x < 0, 0 if x == 0 (neutral)

dir_cg      = sign(CoinGecko BTC price_change_percentage_24h)
dir_binance = sign(most recent kline close - most recent kline open)

discrepancy_detected = (dir_cg != 0) AND (dir_binance != 0) AND (dir_cg != dir_binance)
```

**Consensus truth table:**

| `dir_cg` | `dir_binance` | `discrepancy_detected` | Interpretation |
|----------|---------------|------------------------|----------------|
| `+1` | `+1` | `false` | Both bullish — consensus |
| `-1` | `-1` | `false` | Both bearish — consensus |
| `0` | any | `false` | CG flat — neutral, consensus pass |
| any | `0` | `false` | Binance flat candle — neutral, consensus pass |
| `+1` | `-1` | `true` | 24h up, 1h down — intraday reversal signal |
| `-1` | `+1` | `true` | 24h down, 1h up — potential bottom forming |

**Why directional consensus, not a numeric threshold:** Comparing a 1h kline delta to a 24h percentage change as a ratio is mathematically invalid — the two values measure different time windows and have no natural common denominator. A coin can be +0.5% in the last hour but -5% over 24h; no percentage discrepancy threshold applied to those two numbers produces a meaningful result. Directional consensus is the correct comparison: it asks whether Binance's current short-term momentum agrees with CoinGecko's medium-term trend, which is a structurally sound question.

**What `discrepancy_detected: true` means:** Short-term BTC direction (Binance 1h) opposes medium-term BTC direction (CoinGecko 24h) — this indicates a short-term intraday trend reversal against the 24h trend, or stale Binance data. It is **not** a systematic failure of the primary CoinGecko feed; the breadth score and all other data fields remain fully valid. Agents should not discard the metric when this flag is true — they should read it as microstructure context (intraday momentum shifting) while trusting the breadth score itself.

**Behavior on mismatch:**
- `discrepancy_detected: false`, candle fresh: `confidence: "high"` — directions agree
- `discrepancy_detected: true`, candle fresh: `confidence: "medium"` — directional disagreement; `divergence_detected` carries elevated uncertainty
- Candle stale (> 90 minutes) OR `Close == 0.0` (parse failure): validator skipped entirely; `confidence: "low"`

The breadth score is always computed from CoinGecko primary and is unaffected by validator result or confidence level.

**`meta.weights_used` always reflects effective weights, not nominal design weights.** If a timeframe's `TotalCount` fell below the 50-coin floor and its weight was redistributed, `weights_used` will show `0.0` for that timeframe and adjusted values for the others. Agents reading the output must use `weights_used` to understand how the composite was actually built — the nominal weights (10/30/40/20) are the design intent, not a guarantee of what ran.

**Why not a quantitative 24h-vs-24h Binance comparison?** That would require `binance.ticker_24h` — a new endpoint key not currently registered, which would break cache alignment and introduce a net-new API hit. Quantitative 24h validation is queued for v2 once `binance.ticker_24h` is available as a registered endpoint.

**Why not CoinDesk as validator?** CoinDesk's institutional "Top 50" breadth cannot be directly validated against a 250-coin universe — the sample size mismatch makes a discrepancy threshold unreliable. Noted as a future enhancement.

## CLI Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--top` | `int` | `250` | Number of top coins by market cap to include in breadth computation. Min: 50. Max: 250 (v1 hard clamp). |

**Minimum enforcement:** If `--top` < 50, CLI clamps to 50:
```json
"top_clamped": true,
"top_clamped_reason": "Minimum 50 coins required for statistically significant breadth. Value adjusted from N to 50."
```

**Maximum enforcement (v1 hard clamp):** If `--top` > 250, CLI clamps to 250:
```json
"top_clamped": true,
"top_clamped_reason": "Maximum 250 coins enforced in v1 to maintain single-call predictability. Values above 250 require pagination and risk rate limits on the Free/Demo tier."
```
This is a deliberate scope constraint. Pagination support is queued as a future enhancement.

**Tier guidance:**
- `--top 10`: Large-cap dominance check only. Not a meaningful breadth signal.
- `--top 50`: Institutional-tier breadth, aligns with CoinDesk index coverage.
- `--top 250` (default): Full-breadth benchmark. One CoinGecko call, maximum coverage.

## Output Schema

```json
{
    "metric":  "market-breadth",
    "version": "v1.0.0",
    "status":  "string",  // "ok" / "degraded" / "unavailable"
                          // "unavailable": primary CoinGecko parse failed entirely,
                          //   OR all four timeframes absent across the full coin set
                          // "degraded": primary succeeded but structurally incomplete —
                          //   total valid coins fell below 50 after null exclusion;
                          //   OR >20% of coins missing a given timeframe field
                          //   (per-timeframe TotalCount < 50 triggers weight redistribution)
                          // "ok": primary fully populated; validator results in confidence, not status
                          // Note: validator failure, stale candle, and discrepancy_detected
                          //   do NOT affect status — they are reflected in confidence only

    "data": {
        "market_breadth_score": "float64",  // weighted composite, [0.0, 1.0]
        "coins_counted":        "int",      // coins with ≥1 non-null field across ANY timeframe.
                                            // This is the GLOBAL count, not a per-timeframe count.
                                            // coins_counted >= any individual TotalCount["X"].
                                            // A coin null in 7d/30d but valid in 1h/24h contributes
                                            // to coins_counted and to TotalCount["1h"/"24h"],
                                            // but NOT to TotalCount["7d"/"30d"].
                                            // Use meta.timeframe_counts for per-timeframe validity.
        "timeframe_breadth": {
            "1h":  "float64",   // GreenCount["1h"] / TotalCount["1h"]
            "24h": "float64",   // GreenCount["24h"] / TotalCount["24h"]
            "7d":  "float64",   // GreenCount["7d"]  / TotalCount["7d"]
            "30d": "float64"    // GreenCount["30d"] / TotalCount["30d"]
        },
        "divergence_detected": "bool",      // Ghost Rally flag; always present
        "btc_change_24h_pct":  "float64",   // BTC 24h price change (CoinGecko); always present
        "classification": {
            "label":       "string",  // "broad" / "mixed" / "narrow"
            "description": "string"   // "Healthy Growth" / "Selective Participation" / "The Illusion" / "Ghost Rally"
        },
        "summary": "string"
    },

    "meta": {
        // Omitted when --detail basic.
        // Present when --detail extended or full:
        "cache_hit":            "bool",
        "ttl_remaining_sec":    "int",
        "primary_source":       "coingecko",
        "validator_source":     "binance_us",
        "discrepancy_detected": "bool",     // true if BTC 1h direction (Binance) != 24h direction (CoinGecko)
        "discrepancy_note":     "string",   // only if discrepancy_detected == true;
                                            // e.g. "BTC 24h trend positive (CoinGecko) but 1h candle negative (Binance) — intraday reversal signal"
        "confidence":           "string",   // "high": directions agree, candle fresh
                                            // "medium": directional disagreement (discrepancy_detected: true), candle fresh
                                            // "low": candle stale (>90m) OR Close == 0.0 (parse failure) — validator skipped
        "top_clamped":          "bool",     // present if --top was outside enforced range
        "top_clamped_reason":   "string",   // present if top_clamped == true
        "weights_used": {                   // actual weights after zero-weight redistribution (if any)
            "1h":  "float64",              // may differ from nominal 0.10 if timeframe had TotalCount == 0
            "24h": "float64",
            "7d":  "float64",
            "30d": "float64"
        },
        "timeframe_counts": {              // raw GreenCount / TotalCount per timeframe
            "1h":  { "green": "int", "total": "int" },
            "24h": { "green": "int", "total": "int" },
            "7d":  { "green": "int", "total": "int" },
            "30d": { "green": "int", "total": "int" }
        }
        // Additionally when --detail full:
        // "thresholds": {
        //     "broad": 0.60,
        //     "narrow": 0.40,
        //     "divergence_btc_change_min": 2.0,   // percentage units: 2.0 == 2%
        //     "divergence_breadth_max": 0.40
        // }
        // "description": "string"
    }
}
```

**Status determination logic:**

| Condition | Status | Confidence |
|-----------|--------|------------|
| Primary CoinGecko parse fails entirely | `"unavailable"` | n/a |
| All four timeframes absent across full coin set | `"unavailable"` | n/a |
| Total valid coins < 50 after null exclusion | `"degraded"` | per validator |
| Per-timeframe `TotalCount` < 50 (weight redistributed) | `"degraded"` | per validator |
| Primary succeeds; validator (Binance) fetch fails | `"ok"` | `"low"` |
| Primary succeeds; candle stale (> 90 minutes) | `"ok"` | `"low"` |
| Primary succeeds; `Close == 0.0` (parse failure) | `"ok"` | `"low"` |
| Primary succeeds; BTC entry absent or `Change24h == nil` | `"ok"` | `"medium"` — divergence check skipped |
| Primary fully populated; directions agree, candle fresh | `"ok"` | `"high"` |

`status` reflects data availability and structural completeness of the primary source. `confidence` reflects validator result and data quality. These are independent axes — a `"degraded"` metric can still have `confidence: "high"` if the primary data is thin but the validator agrees.

**Enhancements** (conditional — present when specific conditions are met):

| Field | Condition | Description |
|-------|-----------|-------------|
| `divergence_detected` | Always | Ghost Rally boolean; always present — agents must never check for field absence |
| `btc_change_24h_pct` | Always | BTC 24h price change (CoinGecko); always present alongside `divergence_detected` |
| `timeframe_breadth` | Always | Per-timeframe green fractions; always present for recency-bias reads |
| `discrepancy_note` | `discrepancy_detected == true` | Describes which directions disagreed and the microstructure interpretation |
| `top_clamped` / `top_clamped_reason` | `--top` outside enforced range | Indicates clamp was applied and why |
| `weights_used` | Always (extended/full) | Actual weights after redistribution; documents which timeframes contributed to the composite |
| `timeframe_counts` | Always (extended/full) | Raw GreenCount / TotalCount per timeframe for audit and debugging |
| `delta_24h` | Prior cache data exists | Percentage change in `market_breadth_score` from 24h ago. Not yet implemented in v1.0 |

## Usage

```bash
# Basic
cryptospect-cli market-breadth

# With alias
cryptospect-cli mb

# Extended detail (weights_used, timeframe_counts, discrepancy info)
cryptospect-cli market-breadth --detail extended

# Full detail (thresholds, description)
cryptospect-cli market-breadth --detail full

# Custom universe size (min 50, max 250 in v1)
cryptospect-cli market-breadth --top 50

# Combined
cryptospect-cli market-breadth --detail full --top 100
```

## Long Description

### High-level meaning and value

Market Breadth answers the question: *"Is the whole market moving, or just a few large-cap names carrying the index?"*

A rising total market cap with strong breadth confirms a genuine bull move — buyers are rotating across sectors and market-cap tiers, signaling broad conviction. A rising total market cap with collapsing breadth is the signature of a "Ghost Rally": BTC or a handful of mega-caps are dragging the index upward while 70–80% of assets stagnate or decline. This pattern historically precedes corrections because the rally lacks the mechanical depth needed to sustain upward pressure once the lead assets pause.

Breadth is also a leading indicator of rotation exhaustion. When breadth peaks and begins contracting while prices hold or rise, it signals that the "easy" part of the rally — when everything rises together — has ended and the market is entering a phase where alpha comes from selection, not exposure.

### Exact definition and data needs, logic

**Why a multi-timeframe proxy instead of moving averages:**
True MA-based breadth (e.g., "% of coins above their 50-day MA") is the institutional standard but requires historical price data for every coin in the universe. The CoinGecko public API exposes pre-computed percentage changes over fixed windows (1h, 24h, 7d, 30d) rather than raw OHLCV history for all coins. The multi-timeframe proxy uses these fields as a structurally equivalent substitute: a coin that has a positive 7d price change is effectively "above its prior-week reference price," which captures the same regime signal as a 7-day MA crossover without requiring historical data storage.

**Formula:**
```
market_breadth_score = 0.10 × green_pct_1h
                     + 0.30 × green_pct_24h
                     + 0.40 × green_pct_7d
                     + 0.20 × green_pct_30d
```

Where each `green_pct` is `GreenCount / TotalCount` for that timeframe, with both counts derived from per-field null exclusion. The denominator is never `len(all_coins)` — it is the count of coins with a valid (non-null) value for that specific field.

**Weight design rationale:**
The 7d window receives the highest weight (40%) because it is the "sweet spot" for trend confirmation — long enough to filter the noise of hourly volatility, short enough to reflect the current regime rather than a month-old one. The 24h window (30%) captures today's active market sentiment. The 1h window (10%) is included to detect very short-term momentum but weighted low enough that a single-candle spike cannot push the score into "Broad" territory on its own. The 30d window (20%) anchors the score to the macro trend without letting a stale month-old rally inflate current breadth readings during a nascent correction. These weights are heuristics derived from common momentum trading strategies.

**Zero-weight redistribution:**
If all coins in the universe are missing a given timeframe field (e.g., the CoinGecko API tier does not populate 1h for this query), `TotalCount` for that timeframe is 0, making division undefined. In this case the timeframe's nominal weight is redistributed proportionally across the remaining valid timeframes. The actual weights used are always reported in `meta.weights_used` at extended/full detail so the computation is auditable. If all four timeframes are absent, the metric returns `status: "unavailable"`.

**Coin universe:**
The default universe is the top 250 coins by market cap (`--top 250`). Hard-clamped to 250 in v1 to guarantee single-call predictability and avoid rate limit exposure. Values above 250 require pagination, which is queued as a future enhancement.

**BTC reference extraction:**
BTC's `price_change_percentage_24h` is extracted from the `/coins/markets` response during the same single-pass iteration used for breadth counting. The parser checks `entry.ID == "bitcoin"` (not `i == 0`) — the ID check is preferred over index assumption because `market_cap_desc` ordering is a stable but undocumented CoinGecko contract; a silent reorder would corrupt the divergence flag without triggering any error.

**Divergence detection:**
```
divergence_detected = (btc_change_24h_pct > 2.0) AND (market_breadth_score < 0.40)
// CoinGecko returns percentage directly: 5.0 == +5%. Threshold is 2.0, not 0.02.
```
The 2% BTC threshold filters trivial intraday noise. The <40% breadth threshold ensures divergence fires only when breadth is genuinely weak.

**BTC reference null guard:** If no entry with `ID == "bitcoin"` is found in the response (practically impossible for the top 250, but must be handled), or if the BTC entry's change field is nil: set `divergence_detected: false`, set `btc_change_24h_pct: 0.0`, set `confidence: "medium"`, and add `discrepancy_note: "BTC reference unavailable in response — divergence check skipped"`. Do not leave `divergence_detected` undefined.

**Validator — directional consensus:**
```
sign(x): +1 if x > 0, -1 if x < 0, 0 if x == 0 (neutral — consensus pass)

dir_cg      = sign(btc_change_24h_pct)
dir_binance = sign(kline_close - kline_open)

discrepancy_detected = (dir_cg != 0) AND (dir_binance != 0) AND (dir_cg != dir_binance)
```
The `sign()` function is explicitly defined to handle the zero case. A flat CoinGecko 24h reading (e.g., a stablecoin or an asset that hasn't moved) returns `dir_cg = 0`, which is treated as neutral and never triggers a discrepancy — preventing false positives on low-volatility ticks. Only genuine opposing non-zero directions fire.

**Staleness check:** Before computing directional consensus, the validator checks `candle.OpenTime`. If the most recent candle is more than 90 minutes old, the comparison is skipped entirely — a stale candle's direction is not meaningful as a current short-term signal. In this case: `confidence: "low"`, `discrepancy_detected: true`, with a `discrepancy_note` explaining the candle age. This is the backstop that prevents stale cached Binance data from producing misleading directional signals.

**Zero-Close guard:** Go's default `float64` is `0.0`. A BTC/USDT price of exactly zero is impossible in practice; if `kline.Close == 0.0`, it indicates the field was absent and Go defaulted. In this case the validator is skipped: `confidence: "low"`, with a `discrepancy_note` of "Binance Close price is 0.0 — parse failure, validator skipped." The implementation must distinguish a genuine zero-price (impossible here) from a missing-field default. Verify via Binance API docs or test-net that the Close field for an active BTC/USDT pair cannot legitimately return 0.0; if it somehow can, the sanity check `Close > 0` still applies.

A quantitative percentage threshold (e.g., "20% discrepancy") is not applicable here — comparing a 1h candle delta to a 24h percentage change as a ratio is mathematically invalid because the two values measure different time windows. Directional consensus is the correct framing. The validator uses the most recent kline candle only (last element by timestamp); close price and open time are exposed by the existing `binance.spot_cvd_btc_1h` parser via additive `Close float64` and `OpenTime int64` fields (kline array indices 4 and 0, previously parsed but discarded).

### Possible values and associated verdicts

**Broad (>0.60) — "Healthy Growth"**
More than 60% of the top-N universe is positive across the weighted timeframe composite. Sector rotation is active; capital is broadly deployed. Conditions are favorable for looking at laggard sectors as potential catch-up plays. When combined with `stablecoin-power` showing High dry powder, this is the strongest bull configuration.

**Mixed (0.40–0.60) — "Selective Participation"**
The market is divided. Some sectors are trending while others decay. Position selection matters far more than in a Broad regime. A declining breadth score within this band (via `delta_24h` when implemented) is more bearish than a stable one at the same level.

**Narrow (<0.40) — "The Illusion"**
Fewer than 40% of the top-N universe is net positive on the composite. Often signals "flight to quality" — capital concentrating into BTC and top-5 assets while rotating out of mid and small caps. Apparent rallies in total market cap are being driven by BTC's index weight, not real sector-wide appreciation.

**Divergence Active — "Ghost Rally"**
BTC >+2% on the day but breadth <40%. Capital is using BTC as a safe haven within crypto — not rotating into broader risk. The rally is mechanically fragile; a BTC-only move without breadth participation tends to reverse or stall at the next resistance level.

### Other details

**CLI Flags:**
`--top N` controls the coin universe size. Hard-clamped to the range [50, 250] in v1. The 250 upper bound guarantees single-call predictability on the CoinGecko Free/Demo tier. Institutional users focused on large-cap should use `--top 50`.

**Enhancements:**
- `divergence_detected` and `btc_change_24h_pct` are always present — agents must never need to handle a missing field before acting on the Ghost Rally signal.
- `timeframe_breadth` is always present — the 1h-vs-7d spread is a first-class output for recency-bias reads.
- `weights_used` in meta exposes the actual weights after zero-weight redistribution, allowing agents to detect when a timeframe was dropped and adjust their reading of the composite accordingly.
- `timeframe_counts` in meta exposes raw `GreenCount / TotalCount` per timeframe for debugging and audit.
- `delta_24h` (not yet implemented): percentage change in `market_breadth_score` from prior cached value. To be implemented when historical cache is available.

**Cross-Source Verification:**
The validator uses directional consensus rather than a quantitative threshold because the 1h kline and 24h percentage change are incommensurable time windows. `discrepancy_detected: true` should be interpreted as a microstructure signal (short-term BTC momentum opposing medium-term trend) rather than a data error. It reduces `confidence` to `"medium"` and adds uncertainty to `divergence_detected`, but does not affect `status` or the breadth score. Full quantitative validation (24h-vs-24h comparison using `binance.ticker_24h`) is queued for v2.

**Implementation Compromises:**
- **Single-point-in-time comparisons, not true MA breadth.** `price_change_percentage_Xd > 0` approximates "above the X-day MA." A coin that fell for 6 days then recovered above its window open on day 7 registers as green. True MA breadth would not count it. Across 250 coins this noise averages out, but the approximation is a known limitation.
- **V-shape recovery lag.** The 7d-dominant weighting means a sharp bottom reversal appears as "Narrow" or "Mixed" for 24–48h until the 7d window catches up. The 1h-vs-7d spread is the recommended early-detection mechanism.
- **Per-timeframe statistical floor (50 coins).** If `TotalCount` for a given timeframe falls below 50, that timeframe is dropped and its weight redistributed. This floor prevents the breadth score from being computed on a statistically thin sample (e.g., 20 coins having a valid 1h field) that would produce a misleading sub-score. The `weights_used` field documents when redistribution occurred.
- **Global statistical floor (50 coins).** If total valid coins across all timeframes falls below 50, status becomes `"degraded"`. Breadth computed on fewer than 50 coins is not a reliable signal regardless of mathematical validity.
- **Candle staleness threshold (90 minutes).** The validator uses a 90-minute freshness window. A stale cached candle from 3+ hours ago would compare a stale short-term direction against a live 24h trend, producing a meaningless result. When stale, the validator is skipped entirely and `confidence` set to `"low"`.
- **Zero-Close parse failure guard.** Go's `float64` defaults to `0.0` for missing fields. Since BTC/USDT cannot have a genuine price of zero, `Close == 0.0` is treated as a parse failure and the validator is skipped. The parser must expose `Close` and `OpenTime` as explicit fields rather than relying on implicit zero-values.
- **Directional consensus limitation.** The validator can only detect direction disagreement, not magnitude disagreement. A CoinGecko 24h of +0.1% opposing a Binance 1h of -0.1% triggers `discrepancy_detected: true` even though both represent near-zero moves. This is an accepted v1 limitation; quantitative validation via `binance.ticker_24h` in v2 will address it.
- **Neutral zero treated as consensus pass.** A flat CoinGecko 24h (`sign = 0`) or a flat Binance candle (`close == open`, `sign = 0`) does not trigger discrepancy. This avoids false positives during dead-flat low-volatility periods. The tradeoff: a genuinely flat 24h BTC reading provides no direction anchor for the validator.
- **BTC ID check, not index.** Parser uses `entry.ID == "bitcoin"` to extract the BTC reference. Robust against API ordering changes; negligible cost for 250 coins.
- **`--top` hard-clamped to 250.** Pagination for values above 250 is a future enhancement.
- **No stablecoin filtering.** Stablecoins register ~0% change and create a slight conservative bias. A future `--usd-stable-filter` flag will address this.
- **BTC dominance effect on divergence.** Divergence fires only on BTC-vs-altcoin patterns. ETH-specific divergence detection is a future enhancement.

**Future Enhancements:**
- `delta_24h`: Percentage change in `market_breadth_score` from prior cached value. Requires historical cache.
- `--usd-stable-filter` flag: Excludes stablecoins from the coin universe.
- **`binance.ticker_24h` validator (v2):** Adds a registered 24h Binance ticker endpoint enabling true quantitative 24h-vs-24h BTC price comparison with a numeric discrepancy threshold, replacing the directional consensus proxy.
- **`--top` pagination above 250:** Automatic paginated fetch for values above 250, with rate-limit-aware throttle. Prerequisite: Pro API key or confirmed rate limit headroom.
- **ETH divergence detector:** Flag ETH-specific divergence (ETH >+2% with breadth <40%).
- **Tier-segmented breadth:** Split the score into large-cap (top 10), mid-cap (11–50), long-tail (51–250) tiers independently.
- **CoinDesk institutional breadth:** Alignment/divergence note when `--top 50` is passed.
- **True MA-based breadth:** Requires historical price storage per coin. Not feasible in v1 given public API constraints.

**Agentic Logic (Strategic Notes)**

When an LLM or agent calls this tool, it should apply the following heuristics:

- **Ghost Rally priority — `divergence_detected` overrides the classification label.** If `divergence_detected: true`, disregard whether the base label reads `"mixed"` or `"narrow"`. The operative strategy is **Defensive** regardless. The boolean is surfaced as a high-priority flag precisely so agents do not need to re-derive the threshold logic. Do not enter new broad-market longs. `btc_change_24h_pct` is provided alongside it to assess severity — a BTC move of 2.1% is borderline caution, while 6.5% is a strong defensive signal.

- **Recency bias read — compare `timeframe_breadth.1h` against `timeframe_breadth.7d` to determine direction of travel within the composite label:**
  - `1h` significantly *lower* than `7d`: the market is **cooling**. A strong medium-term trend is losing short-term participation. Watch for the composite to drift toward the next lower classification band.
  - `1h` significantly *higher* than `7d`: a **bottom reversal** may be forming — short-term participation recovering while medium-term trend is still weak. Most likely a V-shape recovery scenario where the 7d window has not yet caught up. Do not act on this alone; cross-reference `flow-tension` CVD for taker confirmation.
  - `1h` ≈ `7d`: stable regime, not transitioning.

- **Check `meta.weights_used` if the composite score seems anomalous.** If a timeframe was dropped due to zero `TotalCount` and its weight redistributed, the composite is a different formula than the nominal one. A score of 0.55 computed on three timeframes is not directly comparable to a 0.55 computed on all four.

- **`discrepancy_detected: true` is a microstructure signal, not just a confidence flag.** It means Binance's short-term BTC direction opposes CoinGecko's medium-term trend. This can indicate an intraday reversal forming. Read it alongside `divergence_detected`: if BTC is up 24h (no Ghost Rally) but 1h is turning negative (discrepancy), the rally may be stalling intraday.

- **`confidence` has three tiers with distinct meanings for agents:**
  - `"high"`: validator confirmed directional agreement, candle fresh — full signal integrity.
  - `"medium"`: directional disagreement (`discrepancy_detected: true`), candle fresh — this is a microstructure signal (intraday reversal forming), not a data failure. The breadth score is unaffected. If `divergence_detected: true` simultaneously, treat the Ghost Rally signal as a caution flag rather than a hard block.
  - `"low"`: validator was skipped — either the Binance candle was stale (> 90 minutes) or the Close price failed to parse (`Close == 0.0`). In this case `discrepancy_detected` is set to `true` as a conservative default but carries no directional meaning — it is an availability signal, not a microstructure signal. Do not interpret a `"low"` confidence `discrepancy_detected: true` as an intraday reversal indicator. The breadth score remains valid; only the validator output is unreliable.

- **Treat `narrow` as a regime signal, not a sell signal.** BTC and ETH may be fine entry candidates during Narrow breadth — the risk is concentrated in mid and small cap tiers.

- **Cross-metric use — named high-signal combinations:**
  - `broad` + `stablecoin-power` High → **Max Conviction Bull.** Broad participation confirmed, fuel available. Strongest configuration for initiating or holding broad long exposure.
  - `narrow` + `flow-tension` negative CVD → **Structural Decay / Distribution.** Market thinning, sellers aggressive. Active defensive positioning warranted.
  - `narrow` + `divergence_detected: true` + `stablecoin-power` High → **Ghost Rally with dry powder.** BTC holding while altcoins bleed, stablecoin capital accumulating. Wait for CVD confirmation before acting on reversal potential.
  - `mixed` + stable OI + neutral funding → accumulation or distribution, indeterminate. Cross-reference `flow-tension` for which side is doing the accumulating.