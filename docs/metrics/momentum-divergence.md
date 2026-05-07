# momentum-divergence

**version:** `v1.1.0`
**Alias:** `md`
**Endpoints:** `coingecko.coin_markets_breadth`

## Overview

Identifies shifts in risk appetite by segmenting the top 200 coins into three tiers — **Large** (Top 10), **Mid** (11–50), and **Small** (51–200) — and comparing average 24h price performance across tiers. A positive mid-vs-large spread signals capital rotating down the risk curve into higher-beta assets ("Risk-On"); a negative spread signals capital concentrating into mega-caps while the broader market lags or bleeds ("Top-Heavy"). A `tail_extension` boolean flags when the long-tail is also outperforming, confirming full alt-season rotation or, in late-cycle conditions, a speculative overshoot.

No cross-source validator is used. The tier structure is self-validating by construction: Tier 1 price data is already present in the primary call and directly anchors the rotation verdict.

## Formula (or how to compute)

```
Tier construction (null-exclusion per coin):
  Assign each coin to a tier by market_cap_rank field (not response index):
    rank 1–10:   tier_large
    rank 11–50:  tier_mid
    rank 51–200: tier_small
  Include coin in tier stats only if price_change_percentage_24h is non-null
    AND market_cap is non-zero.

  Statistical floor: if any tier has fewer than 3 valid coins after double-null exclusion,
    that tier is marked absent. Absent tiers affect confidence (see Classification).
  If all three tiers are absent: status → "unavailable".

Tier averages (market-cap weighted, double-null-excluded):
  large_avg = sum(price_change_24h_i * market_cap_i) / sum(market_cap_i) for tier_large valid coins
  mid_avg   = sum(price_change_24h_i * market_cap_i) / sum(market_cap_i) for tier_mid   valid coins
  small_avg = sum(price_change_24h_i * market_cap_i) / sum(market_cap_i) for tier_small valid coins

  // v1.1: market-cap weighted means. Coins with null/zero market_cap are excluded from
  // both numerator and denominator and are invalid for tier construction — they do not
  // count toward the statistical floor and do not enter tier averages.
  // CoinGecko returns percentage directly: 5.0 == +5%, not 0.05.

Spread matrix (nil-safe -- only computed when both constituent tiers are valid):
  mid_vs_large   = mid_avg   - large_avg  if tier_large != nil AND tier_mid   != nil, else null
  small_vs_large = small_avg - large_avg  if tier_large != nil AND tier_small != nil, else null
  small_vs_mid   = small_avg - mid_avg    if tier_mid   != nil AND tier_small != nil, else null

  // IMPORTANT: absent tiers produce null spreads, never 0.0.
  // 0.0 is a valid spread value (tiers performing identically); returning 0.0 for a
  //   missing spread is a dangerous false signal that suppresses a classification
  //   that should instead be unavailable.
  // In Go: use *float64 (pointer) for each spread field; nil = absent, 0.0 = real zero.
  // Classification defaults to 'neutral' if mid_vs_large is null.

  // All spreads are in percentage-point units.
  // Positive = smaller tier outperforming larger tier (risk-on direction).
  // Negative = larger tier outperforming smaller tier (risk-off / concentration).

Primary classification:
  label =
    "risk_on"          if mid_vs_large > +5.0 AND mid_avg > 1.0
    "top_heavy"        if mid_vs_large < -3.0 AND large_avg > +0.5
    "flight_to_safety" if mid_vs_large < -3.0 AND large_avg <= -0.5
    "neutral"          otherwise  // ← includes the dead band: large_avg in (-0.5, +0.5]
                                   //   when mid_vs_large < -3.0 but large_avg is within
                                   //   ±0.5%, label is "neutral" (Concentration / ambiguous).
                                   //   This prevents rapid oscillation between top_heavy and
                                   //   flight_to_safety when the market is effectively flat.
                                   //   The ±0.5% band is a v1 heuristic; document in
                                   //   meta.thresholds as concentration_dead_band_pct: 0.5
                                   //   for v1.1 recalibration.

  // Asymmetric thresholds are deliberate:
  //   +5.0pp for risk_on: clears routine daily divergence noise across a 40-coin average.
  //   -3.0pp for top_heavy / flight_to_safety: tighter trigger — concentration patterns are
  //     structurally specific even at -3pp average spread across 40 coins.
  //   mid_avg > 1.0 guard: prevents "barely positive mid-caps bleeding slower than mega-caps"
  //     from triggering a risk_on verdict. 0.1% average is noise in crypto; 1.0% is conviction.
  //   top_heavy vs flight_to_safety: split on sign of large_avg. Large-cap positive = narrow
  //     rally with mega-cap dominance. Large-cap zero or negative = broad selloff with capital
  //     retreating into mega-caps as least-bad position. These require different agent responses
  //     and must not share a label.

Tail extension (standalone signal, decoupled from label):
  tail_extension = small_vs_large > +5.0

  // tail_extension is always present in data regardless of label.
  // Decoupled from risk_on to surface the 'Barbell / Speculative Extension' pattern:
  //   long-tail outperforming mega-caps without mid-cap confirmation. This occurs during
  //   meme/micro-cap season and BTC-halo retail flows where mid-caps (L1s, utility tokens)
  //   lag. Under the old logic (gated on risk_on), this pattern was invisible.
  // CAUTION: standalone tail_extension: true (in neutral or top_heavy/flight_to_safety)
  //   is susceptible to single-cluster outlier distortion — a correlated group of meme coins
  //   can move the 150-coin tail average without genuine broad rotation. Treat independently
  //   of label with appropriate skepticism (see Agentic Logic).
```

**Threshold rationale:** The ±5pp / –3pp boundaries are calibrated to clear routine daily divergence at the tier-average level. Individual crypto assets routinely diverge by 1–3pp intraday; tier averages over 40 mid-cap coins dampen single-asset noise, but a 2pp average spread can still emerge from random variance without genuine rotation. The 5pp risk-on threshold requires a consistent directional bias across the majority of the 40-coin mid-cap tier. The –3pp top-heavy threshold is tighter because concentration patterns are structurally more specific — even a –3pp average spread across 40 coins implies meaningful sector-wide underperformance relative to mega-caps. These thresholds are v1 heuristics and are explicitly documented in `meta.thresholds` for v1.1 recalibration once real output data accumulates.

## Interpretation

- **`risk_on` (`mid_vs_large > +5.0pp` AND `mid_avg > 1.0`):** Mid-caps are meaningfully outpacing mega-caps with genuine positive momentum. Capital is rotating down the risk curve. If `tail_extension: true`, rotation has extended into the long-tail: maximum risk appetite. See Agentic Logic for late-cycle caution on full tail extension.

- **`top_heavy` (`mid_vs_large < -3.0pp` AND `large_avg > 0`):** Mega-caps rising while mid and small caps lag. BTC/ETH dominance expanding. The total market cap may look healthy but is being carried by a narrow set of assets. Altcoin exposure underperforms even in a rising index. Historically precedes broader corrections when lead assets stall — no second rotation wave is ready. Watch `tier_averages.large` trending lower as a leading indicator of transition to `flight_to_safety`.

- **`flight_to_safety` (`mid_vs_large < -3.0pp` AND `large_avg <= 0`):** Entire market selling off, but mid and small caps bleeding much harder. Capital retreating to mega-caps as the "least bad" position. Mega-caps are not a safe haven in absolute terms — they are simply where survival capital concentrates when risk appetite collapses. Higher urgency than `top_heavy`; warrants active defensive positioning across the altcoin tier. If `stablecoin-power` `supply_trend_7d` is also contracting, treat as macro risk-off confirmation.

- **`neutral`:** Tiers moving in rough concert, or spread below threshold in either direction. No high-conviction rotation signal detected. Note: `neutral` does not mean rotation is absent — in low-volatility regimes, genuine rotation can occur below the 5pp detection threshold. See Agentic Logic for the sub-threshold cross-check.

- **`tail_extension: true`:** Long-tail outperforming mega-caps by >5pp. Now a standalone signal — fires regardless of primary label. Interpretation depends on label context:
  - With `risk_on`: Highest-conviction long signal for long-tail assets. Also historically correlated with late-cycle speculative peaks. Cross-reference `stablecoin-power` and `flow-tension` funding rate.
  - With `neutral`: **Barbell / Speculative Extension.** Long-tail outperforming without mid-cap confirmation. Historically indicates retail-driven speculative flows (meme/micro-cap season) or BTC-halo rallies where mid-caps lag. High volatility risk; susceptible to single-cluster outlier distortion in the 150-coin tail. Do not treat with the same weight as a `risk_on` + `tail_extension` combination.
  - With `top_heavy` or `flight_to_safety`: Unusual configuration. Long-tail moving against the concentration trend. Likely noise or a specific micro-cap catalyst — not a structural rotation signal.

- **`spreads.small_vs_mid` microstructure read:** Positive = rotation heat propagating fully from mid-caps into long-tail. Negative in a `risk_on` regime = rotation stalling at mid-caps, long-tail not yet ignited. Nuance lost with a single spread value.

## Classification

**Primary label:**

| Label | Condition |
|-------|-----------|
| `risk_on` | `mid_vs_large > +5.0` AND `mid_avg > 1.0` |
| `top_heavy` | `mid_vs_large < -3.0` AND `large_avg > +0.5` |
| `flight_to_safety` | `mid_vs_large < -3.0` AND `large_avg < -0.5` |
| `neutral` (concentration dead band) | `mid_vs_large < -3.0` AND `large_avg` within (–0.5, +0.5] |
| `neutral` | All other cases |

**Tail extension** (standalone boolean, always present, decoupled from label):

| `tail_extension` | Condition |
|------------------|-----------|
| `true` | `small_vs_large > +5.0` (regardless of primary label) |
| `false` | All other cases |

**Confidence** (reflects tier completeness, not cross-source agreement):

| `confidence` | Condition |
|--------------|-----------|
| `"high"` | All three tiers have ≥ 3 valid coins |
| `"low"` | Any tier has < 3 valid coins |

## Data Source(s)

- **Primary API:** CoinGecko
- **Endpoint key:** `coingecko.coin_markets_breadth` — `/coins/markets` with `per_page=200`, `order=market_cap_desc`, `price_change_percentage=24h`
- **Fields used:** `id`, `market_cap_rank`, `price_change_percentage_24h`, `market_cap` (per coin)

**Endpoint cache sharing:** `coingecko.coin_markets_breadth` is the same endpoint key used by `market-breadth`. When both metrics are called in the same session, the dispatcher serves data from cache at zero additional API cost. `market-breadth` defaults to `per_page=250`; if both metrics are active, the dispatcher should use `per_page=250` as the canonical fetch. `momentum-divergence` uses only ranks 1–200; ranks 201–250 are silently ignored.

**Parser implementation note:** Tier assignment must use the `market_cap_rank` field, not positional index. `market_cap_desc` ordering is stable but undocumented; rank-field assignment is the defensive implementation.

## Cross-Source Verification

No cross-source verification in v1.

The rotation verdict is internally self-consistent: Tier 1 price data is present in the primary CoinGecko call and anchors the spread computation. This is **self-consistent, not self-validating** — the math flows correctly from the inputs, but there is no independent check that the inputs themselves are trustworthy. If CoinGecko returns a stale or corrupted payload where `price_change_percentage_24h` is zero or near-zero for all assets, the metric will compute `neutral` with `confidence: "high"` and emit no warning. See Implementation Compromises for the stale-data guard.

A BTC CVD validator (Binance-US `binance.spot_cvd_btc_1h`) was evaluated and explicitly rejected. In a divergence metric, a validator checking for BTC momentum "agreement" with the rotation verdict is adversarial by design — a successful rotation (mid-caps flying while BTC dumps) produces a strongly negative BTC CVD that would incorrectly reduce confidence on a valid `risk_on` verdict.

CoinMarketCap free tier is the natural future validator for per-coin price data cross-check but is rate-limited to 30 calls/month — unsuitable for a real-time CLI. Noted as a future enhancement.

## CLI Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--segments` | `string` | `"10,50,200"` | Comma-separated tier boundaries: `large_ceiling,mid_ceiling,small_ceiling`. |

**Segment clamping rules:**
- `large_ceiling` minimum: **5**. Values below 5 produce a tier average over fewer than 5 coins, making a single outlier asset capable of distorting the spread and firing a spurious classification. CLI clamps silently and documents in `meta`.
- `small_ceiling` maximum: **200** (v1 hard clamp). Values above 200 require pagination, violating the single-pass architecture. CLI clamps silently and documents in `meta`.
- Values must be strictly ascending (`large_ceiling < mid_ceiling < small_ceiling`). Violated ordering returns an error: `"invalid --segments: boundaries must be strictly ascending"`.

```json
// When large_ceiling was clamped from user input:
"segments_clamped": true,
"segments_clamped_reason": "Large-cap ceiling below minimum of 5. Adjusted from N to 5 to ensure tier averages are statistically representative."
```

**Tier guidance:**
- `--segments 5,20,100`: Tighter large-cap tier (top 5 only). More sensitive to BTC/ETH idiosyncratic moves; use when focusing on BTC vs. ETH divergence.
- `--segments 10,50,200` (default): Standard institutional segmentation. 10-coin large tier dampens single-asset outlier distortion.

## Output Schema

```json
{
    "metric":  "momentum-divergence",
    "version": "v1.1.0",
    "status":  "string",  // "ok" / "degraded" / "unavailable"
                          // "unavailable": CoinGecko parse failed OR all three tiers absent
                          // "degraded": one or more tiers has < 3 valid coins
                          // "ok": all three tiers valid

    "data": {
        "tier_averages": {
            "large": "float64",  // mean 24h return for tier_large; percentage units (5.0 == +5%)
            "mid":   "float64",  // mean 24h return for tier_mid
            "small": "float64"   // mean 24h return for tier_small
        },
        "spreads": {
            "mid_vs_large":   "float64?",  // primary classification signal; pp units
                                           // null if tier_large or tier_mid is absent
            "small_vs_large": "float64?",  // tail extension input
                                           // null if tier_large or tier_small is absent
            "small_vs_mid":   "float64?"   // rotation depth: positive = heat propagating to long-tail
                                           // null if tier_mid or tier_small is absent
            // NEVER 0.0 for a missing spread -- see Formula section for nil-safety contract
        },
        "tail_extension": "bool",  // always present; true if small_vs_large > +5.0pp, regardless of label
        "classification": {
            "label":       "string",  // "risk_on" / "top_heavy" / "flight_to_safety" / "neutral"
                                      // label alone is sufficient for agent switch/case logic;
                                      // top_heavy and flight_to_safety are mutually exclusive
            "description": "string"   // "Risk-On Rotation"
                                      // "Top-Heavy — Narrow Rally"
                                      // "Flight to Safety — Defensive Capital Concentration"
                                      // "Neutral"
                                      // "Barbell — Speculative Tail Extension" (neutral + tail_extension)
        },
        "summary": "string"  // NL verdict, e.g.:
                             // "Mid-caps outpacing mega-caps by +7.2pp (risk_on, mid avg +3.4%);
                             //  long-tail not yet extending (small_vs_large +2.1pp)."
                             // "Mega-caps down -1.2%, mid-caps down -8.4% — defensive capital
                             //  flight into large-caps (top_heavy, spread -7.2pp)."
    },

    "meta": {
        // Omitted when --detail basic.
        // Present when --detail extended or full:
        "cache_hit":         "bool",
        "ttl_remaining_sec": "int",
        "primary_source":    "coingecko",
        "confidence":        "string",   // "high" (all tiers ≥ 3 coins) / "low" (any tier < 3 coins)
        "weighting_method":  "market_cap_weighted"   // v1.1; extended detail only
        "tier_counts": {
            "large": "int",  // valid coin count after null exclusion
            "mid":   "int",
            "small": "int"
        },
        "segments_used": {               // actual boundaries after clamping
            "large_ceiling": "int",
            "mid_ceiling":   "int",
            "small_ceiling": "int"
        },
        "segments_clamped":        "bool",    // present if --segments triggered a clamp
        "segments_clamped_reason": "string"   // present if segments_clamped == true
        // Additionally when --detail full:
        // "thresholds": {
        //     "risk_on_spread":        5.0,   // pp; mid_vs_large must exceed this
        //     "top_heavy_spread":     -3.0,   // pp; mid_vs_large must be below this
        //     "min_positivity_guard":  1.0,   // mid_avg must exceed this for risk_on to fire
        //     "tail_extension_spread": 5.0,   // pp; small_vs_large must exceed this
        //     "tier_floor_min_coins":  3,     // minimum valid coins per tier
        //     "segments_large_min":    5,     // minimum large_ceiling
        //     "segments_small_max":  200,     // maximum small_ceiling in v1
        //     "concentration_dead_band_pct": 0.5  // large_avg ±band where top_heavy/
        //                                          //   flight_to_safety collapse to neutral
        // }
        // "description": "string"
        // "tier_detail": {
        //     "large": [ { "id": "bitcoin",   "return_24h": 2.10, "market_cap": 1953000000000, "weight_pct": 52.34 }, ... ],
        //     "mid":   [ { "id": "chainlink", "return_24h": -1.40, "market_cap": 15000000000, "weight_pct": 8.12 }, ... ],
        //     "small": [ { "id": "gmx",       "return_24h": -4.20, "market_cap": 1000000000,  "weight_pct": 2.50 }, ... ]
        // }
    }
}
```

**Enhancements** (conditional — present when specific conditions are met):

| Field | Condition | Description |
|-------|-----------|-------------|
| `tail_extension` | Always | Boolean overlay; always present — agents must never check for field absence |
| `spreads.small_vs_mid` | Always | Rotation depth signal; positive = heat propagating to long-tail, negative = stalling at mid-caps |
| `weighting_method` | `--detail extended` or `full` | v1.1: always `"market_cap_weighted"`; tier averages use market-cap weighting |
| `market_cap` on tier_detail entries | `--detail full` | v1.1: per-coin market cap (USD); aids outlier weight inspection |
| `weight_pct` on tier_detail entries | `--detail full` | v1.1: per-coin share of tier market cap (2dp); e.g. 52.34 means BTC is 52.34% of large-tier weight |
| `tier_detail` | `--detail full` | Per-coin breakdown (id, return_24h, market_cap, weight_pct) for each tier; primary tool for detecting outlier distortion |
| `segments_clamped` / `segments_clamped_reason` | `--segments` outside enforced range | Documents that clamping occurred and why |
| `delta_24h` on spreads | Prior cache data exists | Not yet implemented — percentage change in `mid_vs_large` spread from prior cached value |

## Usage

```bash
# Basic
cryptospect-cli momentum-divergence

# With alias
cryptospect-cli md

# Extended detail (tier counts, segments used, confidence)
cryptospect-cli momentum-divergence --detail extended

# Full detail (thresholds, description, per-coin tier breakdown)
cryptospect-cli momentum-divergence --detail full

# Custom tier boundaries
cryptospect-cli momentum-divergence --segments 5,20,100

# Combined
cryptospect-cli momentum-divergence --detail full --segments 5,20,100
```

## Long Description

### High-level meaning and value

Momentum Divergence is the "Risk Appetite Gauge" of the suite. It answers: *"Where is the smart money going — into safety, or down the risk curve?"*

In crypto, capital rotation is highly visible in the market-cap tier structure. Mega-caps (top 10) function as the blue-chip tier — lower beta, higher liquidity, first to attract inflows during uncertainty. Mid-caps (11–50) are the speculative intermediate: they amplify bull moves and bleed harder in bear markets. Long-tail assets (51–200) are the highest-beta cohort, rocketing in genuine bull markets and collapsing first when risk appetite fades.

When mid-caps start outpacing mega-caps, investors are deliberately accepting more risk for more return — a structurally healthy signal that the rally has conviction beyond BTC dominance. When mega-caps pull ahead of everything else, the market is either concentrating into a narrow rally that lacks participation depth, or actively retreating into the "least bad" assets during a broad selloff. These two scenarios share the same label (`top_heavy`) but carry different implications, distinguished in the output by `classification.description` and directly readable from `tier_averages.large`.

### Exact definition and data needs, logic

**Data source:** A single `coingecko.coin_markets_breadth` call (`/coins/markets`, `per_page=200`, `order=market_cap_desc`, `price_change_percentage=24h`) returns all required fields — `id`, `market_cap_rank`, `price_change_percentage_24h`, and `market_cap` — per coin. No additional endpoints are needed. When called in the same session as `market-breadth`, the dispatcher serves this data from cache at zero additional API cost.

**Tier assignment:** Coins are assigned by `market_cap_rank` field, not positional index. This is defensive against silent API ordering changes.

**Null exclusion:** Coins with null `price_change_percentage_24h` are excluded from their tier's count and sum. The denominator for each tier average is the valid coin count for that tier, not the nominal tier size.

**Statistical floor (3 coins):** Each tier requires at least 3 valid coins to produce a reliable average. The top-10 tier is unlikely to fall below this floor under normal conditions; the floor exists as a defensive guard against severe API failures or partial response bodies.

**Market-cap weighted means:** Tier averages are weighted by market cap. BTC and ETH together carry ~60-70% of large-tier weight, reducing single-coin outlier distortion compared to simple means. `tier_detail` at full detail exposes per-coin data including `market_cap` and `weight_pct` for weight inspection.

**No `relative_performance` ratio:** The ratio `mid_avg / large_avg` was evaluated and rejected. When `large_avg` approaches zero, the ratio explodes. When `large_avg` is negative, the ratio's sign flips non-intuitively. The spread (percentage-point difference) is always defined, always sign-correct, and directly interpretable. Division-by-zero and "ghost ratio" protection logic is eliminated entirely.

**Spread matrix:** Three spreads are always emitted. `mid_vs_large` drives classification. `small_vs_large` powers `tail_extension`. `small_vs_mid` is the rotation depth signal: a positive `mid_vs_large` combined with a negative `small_vs_mid` reveals that rotation is stalling at mid-caps and not yet reaching the long-tail — a nuance lost with a single spread value.

**`--segments` flag:** Allows custom tier boundaries. The `small_ceiling` is silently clamped to 200 — the maximum single-call result from CoinGecko's `/coins/markets`. Values above 200 require pagination, which violates the single-pass architecture. The `large_ceiling` is clamped to a minimum of 5: with fewer than 5 coins in the large tier, a single idiosyncratic move can dominate the tier average and fire a spurious classification.

**TTL:** 4 hours. Tier rotation operates on a 4–24h rhythm. A 1h TTL creates unnecessary cache churn; a 24h TTL makes the signal too stale to detect intraday rotation shifts.

### Possible values and associated verdicts

**`risk_on` — "Risk-On Rotation"**
Mid-caps generating meaningfully higher returns than mega-caps, with mid-cap average above 1% (genuine positive momentum, not just "bleeding slower"). Capital is flowing down the risk curve. In a rising total market, this confirms structural depth beyond BTC dominance. When combined with `market-breadth` showing Broad, this is a high-conviction bull configuration.

If `tail_extension: true`, capital is extending into the highest-beta tier. Strongest long signal for long-tail assets — but also historically the condition most correlated with late-cycle speculative peaks. Before acting: check `stablecoin-power` (dry powder remaining?) and `flow-tension` funding rate (leverage already crowded?).

**`top_heavy` — Two sub-scenarios:**

*Narrow Rally (`large_avg > 0`):* Mega-caps rising while mid and small caps lag. BTC/ETH dominance expanding. Total market cap may look healthy but is being carried by a narrow set of assets. Altcoin exposure underperforms even as the index rises. Historically precedes broader corrections when lead assets stall, as there is no second wave of rotation ready to sustain upward pressure.

*Defensive Capital Flight (`large_avg ≤ 0`):* Entire market selling off, but mid and small caps bleeding much harder. Capital retreating to mega-caps as the "least bad" position. Risk-off signal warranting defensive positioning across the altcoin tier. Mega-caps are not a safe haven in absolute terms — they are simply where capital concentrates when risk appetite collapses.

**`neutral` — "No Rotation Signal"**
Tiers moving in rough concert, or spread below threshold in either direction. Cross-reference `market-breadth` and `stablecoin-power` for context. `neutral` with `stablecoin-power` High and `flow-tension` OI building is a pre-rotation coil condition.

### Other details

**CLI Flags:**
`--segments large,mid,small` controls tier boundaries. Large minimum of 5 enforced; small maximum of 200 enforced in v1. Strictly ascending order required. Both clamping events are documented in `meta`.

**Enhancements:**
- `tail_extension` is always present in `data` — agents must not check for field absence.
- `spreads.small_vs_mid` is the rotation depth signal. Positive `mid_vs_large` with negative `small_vs_mid` = early/incomplete rotation; long-tail has not yet ignited.
- `tier_detail` at full detail exposes per-coin data for each tier. Primary tool for detecting outlier distortion in the top-10 tier.

**The "Dud Detector" agentic pattern:**
When `classification.label` is `risk_on` and `tier_detail` is available (`--detail full`), an agent with portfolio context can compare each portfolio asset's `return_24h` against its tier's value in `tier_averages`. Any portfolio asset significantly below its tier average — or negative while its tier is positive — in a `risk_on` environment is a **Heavy Anchor**: failing to capture sector momentum in a favorable environment. This pattern often precedes further relative underperformance as capital rotates toward the responding assets within the tier.

**Cross-Source Verification:**
No cross-source verification in v1. The BTC CVD validator was explicitly rejected: in a divergence metric, a validator checking for BTC momentum "agreement" is adversarial by design — a successful mid-cap rotation (mid-caps +10%, BTC -5%) would produce a negative CVD that incorrectly lowers confidence on a valid `risk_on` verdict. The tier structure is self-validating.

**Implementation Compromises:**
- **Market-cap weighted means (v1.1).** v1 used simple means; v1.1 replaces with market-cap weighted means. BTC and ETH now contribute to `large_avg` in proportion to their economic weight (~60-70% of large-tier market cap) rather than 2 of (up to) 10 equal weights. The `weighting_method` meta field (always `"market_cap_weighted"`) identifies this as a v1.1+ run. Coins without market cap data are excluded from tier computation entirely — they do not count toward the statistical floor and do not enter tier averages.
- **Stablecoin market-cap weight impact (v1.1).** USDT (~$150B market cap) at rank 3-4 in the large tier now carries significant weight under market-cap weighting. Its near-zero 24h return dampens `large_avg` toward zero, making `top_heavy` and `flight_to_safety` slightly harder to trigger than under simple means. This is correct — USDT's economic footprint in the large-cap tier is real — but agents should be aware of the systematic dampening effect. A future `--exclude-stables` flag is more impactful under market-cap weighting than under simple means.
- **Top-10 tier outlier sensitivity.** With only 10 coins, a single idiosyncratic event can still shift the large-cap average, but market-cap weighting (v1.1) significantly reduces single-coin distortion compared to simple means. Borderline `top_heavy` or `risk_on` signals (near ±3–5pp) should be cross-checked against `tier_detail.large` before acting.
- **Stablecoins included without filtering.** USDT and USDC appear in the top 200 and register near-zero 24h returns. See stablecoin market-cap weight impact above.
- **24h price change, not intraday.** The metric uses CoinGecko's pre-computed 24h percentage change. Tier averages are slightly lagged relative to live intraday prices. A future 1h variant addresses this.
- **`--segments` small_ceiling hard-clamped to 200.** Pagination for values above 200 is a future enhancement.
- **No volume conviction.** Volume intensity (`total_volume / market_cap` per tier) was designed and deferred. The baseline comparison problem has no clean solution given the current cache infrastructure. Deferred to v1.2.
- **Thresholds calibrated against v1 simple means.** The ±5pp / –3pp / 1pp guard thresholds were calibrated against v1 simple-mean values and are carried forward unchanged in v1.1. Under market-cap weighting, large-tier averages are more dominated by BTC/ETH → spreads may be narrower in absolute value. Threshold recalibration is a post-v1.1 task requiring accumulated v1.1 output data. A v1.1 Calibration Note is documented below.
- **No stale-data guard in v1.** If CoinGecko returns a payload where `price_change_percentage_24h` is zero or near-zero for all coins (stale cache, API degradation), the metric will compute `neutral` with `confidence: "high"` and emit no warning. A future sanity check should flag `status: "degraded"` if all tier averages are simultaneously within ±0.1% of zero — statistically near-impossible for live BTC/ETH data and indicative of a frozen or default-valued payload. Not implemented in v1; the `tier_detail` breakdown at full detail is the manual inspection path.
- **The `large_avg` zero-crossing boundary is noise-level in practice.** The `top_heavy` / `flight_to_safety` split fires at exactly `large_avg == 0`, but a difference of +0.01% vs -0.01% is indistinguishable from rounding and API precision noise. The label distinction is only reliable when `large_avg` is meaningfully above or below zero. The practical dead band is ±0.5%: within this range, the two labels should be treated as equivalent by downstream consumers. This is a known limitation of binary threshold classification at a zero boundary; a hysteresis buffer is the correct long-term fix but is not implemented in v1.
- **The 5pp threshold is a high-conviction filter — slow rotations are invisible.** In a low-volatility "crab" market, mid-caps can outperform mega-caps by 3pp consistently for weeks — a genuine, healthy rotation — and this metric remains `neutral` throughout. The threshold is intentionally conservative for v1 ("fire only when the signal is unambiguous") but this means the metric has a structural blind spot for gradual regime shifts. An agent seeing persistent `neutral` readings should not conclude rotation is absent; it may be occurring below the detection threshold. The correct cross-check is `market-breadth`'s `timeframe_breadth` spread over multiple calls (see Agentic Logic).

**Future Enhancements:**
- **Volume conviction (`tier_volume_intensity`):** `total_volume / market_cap` per tier compared against a prior cached snapshot. Tiers with rising price but declining intensity surface a per-tier `conviction_hook: "weak"`. Deferred to v1.2.
- **`delta_24h` on spreads:** Percentage change in `mid_vs_large` spread from prior cached value. A widening spread (rotation accelerating) is more signal-rich than the absolute level.
- **`--exclude-stables` flag:** Removes stablecoins from tier computation for a pure-equity rotation signal. More impactful under market-cap weighting (v1.1). Deferred to v1.2.
- **`--segments` pagination above 200:** Automatic paginated fetch for `small_ceiling` values above 200.
- **1h return variant (`--window 1h`):** Uses `price_change_percentage_1h_in_currency` for intraday rotation detection.
- **ETH sub-classification within `top_heavy`:** Distinguish "BTC-only rally" from "BTC+ETH rally." These imply different rotation timelines.
- **Threshold recalibration:** The ±5pp / –3pp / 1pp guard thresholds are v1 heuristics carried forward into v1.1. Once real v1.1 output data accumulates, these should be backtested to validate or adjust the boundaries for market-cap weighted tier averages.
- **`--sensitivity` flag:** A user-selectable sensitivity level — e.g., `--sensitivity high` lowers thresholds to `risk_on > +3.0pp`, `top_heavy < -2.0pp` — to detect slower rotations in low-volatility regimes. This is the correct solution because it has no external dependencies: the caller supplies the regime context rather than requiring the metric to infer it. The default (`--sensitivity normal`) preserves the current ±5pp / –3pp behavior. A `--sensitivity low` option (thresholds ±7pp / –5pp) targets high-volatility environments where the standard threshold fires too frequently.
- **Regime-aware dynamic thresholds (v1.2+):** Automatically adjust thresholds based on detected market volatility — e.g., lower the `risk_on` threshold when BTC realized volatility is low. Requires either a new volatility endpoint (new API dependency) or a rolling self-history of spread values (cache dependency). The `--sensitivity` flag is the prerequisite stepping stone; dynamic thresholds build on top of it once the threshold-adjustment logic is proven.
- **CoinMarketCap cross-validation:** Per-coin 24h return cross-check once a suitable rate-limit-free tier is available.

## v1.1 Calibration Note

v1.1 replaces simple arithmetic means with market-cap weighted tier averages. The following metrics are designed for comparing v1.0 and v1.1 output to characterize the impact:

- **`label_drift`:** How many classifications change between v1 and v1.1 when run against the same payload. A high drift rate indicates the simple mean was systematically over- or under-weighting certain coins within tiers. This is the primary signal for whether the v1 thresholds remain appropriate under weighted means.
- **`avg_magnitude_shift`:** Percentage-point delta on `large_avg` between v1 and v1.1. BTC and ETH now carry ~60-70% of large-tier weight (vs. 20% under simple means), so `large_avg` will systematically shift toward BTC/ETH behavior. A large shift confirms the BTC simple-mean distortion documented in v1.
- **Threshold conservatism note:** Under market-cap weighting, spreads may be narrower in absolute value (BTC/ETH dominance pulls both large and mid tiers toward similar behavior). The +5pp `risk_on` threshold may need to be lowered once sufficient v1.1 data is available for backtesting. The thresholds ship unchanged in v1.1 to avoid calibrating against a single snapshot; recalibration is a data-accumulation task.

**Agentic Logic (Strategic Notes)**

When an LLM or agent calls this tool, it should apply the following heuristics:

- **BTC-skew suppression detection (false negative pattern).** When `mid_vs_large` is between +2pp and +5pp — above noise but below the `risk_on` threshold — the metric stays `neutral`. Before accepting that verdict, agents should check `tier_detail.large` at full detail for the following pattern: BTC and/or ETH are positive outliers (e.g., +8% and +5%) while the remaining 8 large-cap coins are flat or negative. If present, `large_avg` is being pulled upward by BTC/ETH weight, suppressing the spread and masking what is functionally a `risk_on` environment for the non-BTC large-cap cohort. This is the most likely source of false `neutral` readings during BTC-led markets. In this case, treat the `mid_vs_large` spread as understated and cross-reference `flow-tension` CVD for directional confirmation.

- **`top_heavy` → `flight_to_safety` transition warning — and the ±0.5% dead band.** The label flip is abrupt: it changes the moment `large_avg` crosses zero, but a difference of +0.01% vs -0.01% is market noise, not a regime change. Two rules for agents: (1) If `tier_averages.large` is within ±0.5% of zero, treat `top_heavy` and `flight_to_safety` as a single **Concentration** regime — do not act on the label difference, do not treat a flip within this band as a meaningful signal. The distinction only becomes reliable and actionable when `large_avg` is clearly above +0.5% (`top_heavy`) or clearly below -0.5% (`flight_to_safety`). (2) Outside the dead band, the leading indicator of an approaching flip is `tier_averages.large` trending toward zero across successive calls while `mid_vs_large` remains below –3pp. Begin shifting defensive posture before the label flips, not after.

- **`tail_extension` is a confirmation signal, not a trigger signal.** It should only elevate conviction when a corroborating signal is already present — do not initiate positions based on `tail_extension` alone. Specifically: (a) `risk_on` + `tail_extension: true` = elevated conviction, act with appropriate late-cycle caution; (b) `neutral` + `tail_extension: true` + rising `market-breadth` timeframe spread = tentative Barbell pattern, watch for `risk_on` confirmation before acting; (c) `neutral` + `tail_extension: true` with no corroborating signal = **non-action state** — the long-tail is moving but nothing else confirms it. The 150-coin tail is susceptible to single-cluster distortion (a correlated meme-coin group pumping together can move the average 5pp without genuine rotation). In this state, inspect `tier_detail.small` for outlier concentration and wait for volume confirmation (v1.1) before treating as actionable.

- **Default sensitivity is conservative by design — set `--sensitivity high` in low-volatility regimes.** The 5pp `risk_on` threshold will not fire during a slow crab-market rotation. This is intentional for v1 (avoid false positives) but means agents running default settings will miss accumulation-phase rotation signals. If the market has been in a low-volatility sideways regime for more than a week, switch to `--sensitivity high` (2.5pp threshold, available in v1.1) or cross-check `market-breadth` timeframe spreads as a proxy.

- **`neutral` does not mean rotation is absent — it means no high-conviction rotation is detectable.** In a sustained low-volatility environment, mid-caps can outperform mega-caps by 3pp for days or weeks without triggering `risk_on`. The 5pp threshold is a high-conviction filter, not a rotation presence/absence detector. If an agent sees persistent `neutral` readings and wants to check for sub-threshold rotation, the correct cross-check is `market-breadth`'s `timeframe_breadth`: if `breadth.7d` is consistently above `breadth.24h` across multiple `market-breadth` calls, a slow rotation is likely in progress below this metric's detection floor. Do not act on it as aggressively as a confirmed `risk_on` signal, but do not treat the market as fully rotation-absent either.

- **Read `classification.label` first, then `classification.description` for `top_heavy` sub-scenario.** Narrow rally warrants caution on new altcoin longs. Defensive capital flight warrants active positioning out of altcoin exposure. These are different responses to the same label.

- **`tail_extension: true` is not unconditionally bullish.** Highest-conviction long signal for long-tail assets AND most historically correlated with late-cycle peaks. Before acting: if `stablecoin-power` is High, likely early-cycle. If `stablecoin-power` is Low AND `flow-tension` funding is `"overheated"`, likely blow-off. Same condition, opposite responses.

- **`spreads.small_vs_mid` reveals rotation depth.** In a `risk_on` regime: positive = full rotation, long-tail igniting. Negative = rotation stalling at mid-caps, long-tail not yet following. Not a failure of the `risk_on` signal — a caution flag against overweighting long-tail exposure until the long-tail confirms.

- **`top_heavy` on a green day = Exhaustion signal for alts.** Rally losing breadth. BTC-led moves with declining altcoin participation tend to stall at the next BTC resistance as there is no second rotation wave to sustain upward pressure. Cross-check `market-breadth`.

- **`top_heavy` on a red day = Hedge signal.** Capital actively fleeing mid and small caps into the "least bad" position. Higher urgency than the green-day variant. Not about BTC dominance expanding during a bull phase — about survival capital concentrating during risk-off. Treat as a `stablecoin-power` capital flight confirmation if `supply_trend_7d` is also contracting.

- **The "Alt-Dump" Trap.** `top_heavy` reflects tier-average concentration, not unanimous mega-cap strength. A single large-cap asset surging on an idiosyncratic event (e.g., ETH on a specific catalyst) can drag `large_avg` high enough to trigger `top_heavy` while the rest of the market is neutral. Before acting on a borderline `top_heavy` signal (spread near –3pp to –5pp), check `tier_detail.large` at full detail for single-asset outlier distortion.

- **The "Dud Detector" pattern.** In a `risk_on` regime with `--detail full`, an agent with portfolio context should: (1) find each user portfolio asset in `tier_detail`; (2) compare its `return_24h` against its tier's value in `tier_averages`; (3) flag any asset significantly below its tier average, or negative while its tier is positive, as a **Heavy Anchor** — failing to capture sector momentum in a favorable environment.

- **Cross-metric use — named high-signal combinations:**
  - `risk_on` + `tail_extension: true` + `market-breadth broad` + `stablecoin-power high` → **Maximum Bull Confirmation.** Full rotation, fuel available, broad participation. Strongest configuration for initiating broad altcoin exposure.
  - `top_heavy` (green day) + `market-breadth narrow` → **Ghost Rally Amplified.** Both metrics confirm concentration. Altcoin longs carry maximum relative underperformance risk.
  - `risk_on` + `small_vs_mid < 0` + `flow-tension funding: "overheated"` → **Stalled Rotation / Blow-Off Warning.** Mid-caps rotating but long-tail not following, leverage crowded. Rotation may be ending rather than beginning.
  - `neutral` + `stablecoin-power high` + `flow-tension OI: "building"` → **Pre-Rotation Coil.** Dry powder present, leverage loading, no rotation yet. Watch for `mid_vs_large` breaking above +5pp as ignition confirmation.
  - `top_heavy` (red day) + `flow-tension OI: "unwinding"` + `stablecoin-power contracting` → **Macro Risk-Off.** Capital fleeing both altcoins and crypto entirely. Most defensive configuration; treat as `stablecoin-power` capital flight confirmation.