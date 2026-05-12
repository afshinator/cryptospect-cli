# market-regime

**version:** `v1.0.0`
**Alias:** `mr`
**Endpoints:** main: `coingecko.global_market`, `coingecko.coin_markets_breadth`

## Overview

The macro "Master State" indicator. Classifies the market's structural cycle by cross-referencing **Bitcoin Dominance (BTC.D)** against **Market Breadth Score**, gated by **BTC price direction** and **liquidity conviction**. Where `momentum-divergence` measures how fast capital is rotating and `market-breadth` measures how widely it is participating, `market-regime` answers the higher-order question: *"What structural phase is the market in, and is capital flowing toward safety or risk?"*

The metric uses a 2×3 matrix (Dominance Trend × Breadth Score) to produce up to ten named regime labels — nine cells, with the Falling Dom + Narrow Breadth cell producing either Capitulation or Structural Decay depending on conviction level. A `modifier` field captures BTC price direction independently of the regime label, preserving label vocabulary stability while surfacing directional nuance. A `conviction` field derived from the liquidity pulse ratio disambiguates the Capitulation / Structural Decay boundary in the Falling Dom + Narrow Breadth cell.

This metric has a recommended call frequency of 4 hours (`cache_hint_sec: 14400`). It tracks structural shifts, not intraday noise — polling more frequently than the cache refresh interval produces identical output at additional token cost.

## Formula (or how to compute)

This is a matrix-classification metric. There is no single composite score — three input signals are computed independently and combined via the matrix lookup below.

**Signal 1 — Dominance Trend**
```
btc_dominance_pct        = market_cap_percentage["btc"]  // from /global; percentage units e.g. 52.41
dominance_delta_pp       = btc_dominance_pct_current - btc_dominance_pct_prior_cached
                           // percentage-point units: 0.62 means BTC.D moved +0.62pp since last fetch
                           // omitted on cold start (no prior cached snapshot)

dominance_trend =
    "rising"   if dominance_delta_pp >= +0.5
    "falling"  if dominance_delta_pp <= -0.5
    "neutral"  if dominance_delta_pp >  -0.5 AND < +0.5   // exclusive dead-band; also cold-start default
```

The ±0.5pp dead-band prevents flip-flopping in low-volatility environments where dominance oscillates
by fractions of a percentage point with no structural change. On cold start (no prior snapshot in
file cache), dominance_trend defaults to "neutral" and the cold_start note is appended.

**Signal 2 — Market Breadth Score**
```
market_breadth_score = (0.10 × green_pct_1h)
                     + (0.30 × green_pct_24h)
                     + (0.40 × green_pct_7d)
                     + (0.20 × green_pct_30d)
```
Computed using the identical parser and null-exclusion logic as `market-breadth` (mb). Breadth
thresholds are explicitly aligned with mb to guarantee consistency when both metrics are called
in the same session:
```
breadth_band =
    "broad"  if market_breadth_score >= 0.60
    "mixed"  if market_breadth_score >= 0.40 AND < 0.60
    "narrow" if market_breadth_score <  0.40
// Boundary behaviour: 0.60 → "broad"; 0.40 → "mixed". No value is unclassified.
```

**Signal 3 — Conviction (Liquidity Pulse)**
```
lp_ratio   = total_volume_usd / total_market_cap_usd   // from /global
conviction =
    "high"   if lp_ratio >  0.15
    "normal" if lp_ratio >= 0.07 AND <= 0.15
    "low"    if lp_ratio <  0.07
// Boundary behaviour: 0.15 → "normal" (>0.15 required for high); 0.07 → "normal" (>=0.07).
// No value is unclassified. Operators are identical in the Classification table.
```
Conviction thresholds are aligned with `liquidity-pulse` (lp). Used only for the Capitulation
disambiguation rule; does not affect any other matrix cell.

**The 2×3 Regime Matrix**
```
                    | Broad (>=0.60) | Mixed (0.40–0.59) | Narrow (<0.40)   |
--------------------|----------------|-------------------|------------------|
Rising  (>= +0.5pp) | BTC-Led        | Institutional     | Flight to        |
                    | Expansion      | Build             | Safety           |
--------------------|----------------|-------------------|------------------|
Neutral (>-0.5,     | Steady         | Consolidation     | Stagnation       |
        < +0.5pp)   | Appreciation   |                   |                  |
--------------------|----------------|-------------------|------------------|
Falling (<= -0.5pp) | Alt-Season /   | Capital           | Structural       |
                    | Mania          | Rotation          | Decay*           |
--------------------|----------------|-------------------|------------------|
```

*Capitulation Disambiguation Rule (Falling + Narrow cell only):
```
if dominance_trend == "falling" AND breadth_band == "narrow":
    if conviction == "high"  → regime: "Capitulation"
    else                     → regime: "Structural Decay"
```
Capitulation requires high selling volume (lp_ratio > 0.15) to distinguish a panic event from a
low-interest grind. When Capitulation fires, the modifier determines the sub-state note:
```
if regime == "Capitulation":
    if modifier == "negative_pressure" → no note; confidence: "high"
                                          (panic confirmed; price following volume — expected state)
    if modifier == "neutral"           → notes: ["capitulation_price_stabilizing"]; confidence: "medium"
                                          (high volume, alts bleeding, but BTC price has stopped falling —
                                           stabilization or passive absorption of selling)
    if modifier == "positive_momentum" → notes: ["abnormal_capitulation"]; confidence: "medium"
                                          (BTC price actively reversing upward against panic volume —
                                           V-bottom or short squeeze; act with urgency but lower confidence)
```

**The Modifier (Direction Gate)**
```
modifier =
    "positive_momentum" if btc_change_24h_pct >= +0.5
    "negative_pressure" if btc_change_24h_pct <= -0.5
    "neutral"           if btc_change_24h_pct >  -0.5 AND < +0.5   // exclusive dead-band
```
The modifier is independent of the regime label. It captures BTC's absolute price direction
without mutating the label vocabulary. Agents must always read regime + modifier together:
"Institutional Build" + "negative_pressure" means BTC is outperforming alts on a down day
(relative strength / defensive), not genuine accumulation.

## Interpretation

**BTC-Led Expansion** (Rising Dom + Broad Breadth): BTC leads with broad participation. The rally
has both a dominant asset and wide sector participation — historically the most structurally sound
bull configuration. Combine with `stablecoin-power` High and `flow-tension` neutral/positive
funding for the strongest entry signal.

**Institutional Build** (Rising Dom + Mixed Breadth): BTC outperforming while alts show mixed
participation. Early bull cycle or smart-money accumulation phase. Capital is flowing into BTC
before rotating to alts. With `negative_pressure` modifier: BTC showing relative strength on a
down day — not accumulation, closer to Flight to Safety.

**Flight to Safety** (Rising Dom + Narrow Breadth): Market-wide fear. Capital concentrating in
BTC as the crypto safe harbor while alts bleed. Not bullish for BTC in absolute terms — it is
the least-bad asset. Warrants defensive positioning across the alt tier.

**Steady Appreciation** (Neutral Dom + Broad Breadth): Balanced market rising with broad
participation and no dominance shift. Healthy, low-drama bull phase. No rotation extremes in
either direction.

**Consolidation** (Neutral Dom + Mixed Breadth): Market seeking direction. Dominance flat,
participation mixed. Do not increase exposure. With `conviction: "high"`: a Pressure Cooker
condition — high volume within a tightening range often precedes a violent regime break in
either direction. Cross-reference `flow-tension` OI for leverage loading.

**Stagnation** (Neutral Dom + Narrow Breadth): Sideways dominance with poor participation. The
highest-risk environment for alt-coin entry — any BTC volatility will disproportionately affect
alts with no breadth support to absorb it. Distinct from Structural Decay: dominance is not
falling, so alts are not being actively sold relative to BTC — they are simply being ignored.

**Alt-Season / Mania** (Falling Dom + Broad Breadth): Capital rotating down the risk curve with
broad participation. High gains, high blow-off risk. With `negative_pressure` modifier: Speculative
Exhaustion — alts leading the crash, a late-cycle distribution signal. Cross-reference
`flow-tension` funding rate for overheated conditions.

**Capital Rotation** (Falling Dom + Mixed Breadth): Dominance falling with selective participation.
Some sectors capturing rotation while others lag. Early alt-season or incomplete rotation — not yet
confirmed as broad. Monitor `momentum-divergence` mid_vs_large spread for rotation depth.

**Structural Decay** (Neutral Dom + Narrow Breadth OR Falling Dom + Narrow Breadth + low/normal vol):
Broad-based lack of interest. Alts bleeding harder than BTC or the entire market grinding lower on
thin volume. Not a panic event — a slow bleed. No forced exits, no violent bottom. Structurally
different from Capitulation; requires a different agent response.

**Capitulation** (Falling Dom + Narrow Breadth + high vol): Panic selling with high conviction
volume. BTC relatively outperforming a collapsing alt market. Historically provides the best
long-term accumulation opportunities — but requires confirmation that selling is exhausted.
With `abnormal_capitulation` note: V-bottom or short-squeeze in progress; price reversing while
panic volume persists.

## Classification

**Primary Regime Label:**

| Label | Dominance Trend | Breadth Band | Additional Condition |
|-------|----------------|--------------|----------------------|
| `BTC-Led Expansion` | `rising` | `broad` | — |
| `Institutional Build` | `rising` | `mixed` | — |
| `Flight to Safety` | `rising` | `narrow` | — |
| `Steady Appreciation` | `neutral` | `broad` | — |
| `Consolidation` | `neutral` | `mixed` | — |
| `Stagnation` | `neutral` | `narrow` | — |
| `Alt-Season / Mania` | `falling` | `broad` | — |
| `Capital Rotation` | `falling` | `mixed` | — |
| `Capitulation` | `falling` | `narrow` | `conviction == "high"` |
| `Structural Decay` | `falling` | `narrow` | `conviction != "high"` |

**Modifier (independent of regime label):**

| Modifier | Condition |
|----------|-----------|
| `positive_momentum` | `btc_change_24h_pct >= +0.5` |
| `neutral` | `btc_change_24h_pct > -0.5 AND < +0.5` |
| `negative_pressure` | `btc_change_24h_pct <= -0.5` |

**Conviction:**

| Conviction | Threshold |
|------------|-----------|
| `high` | `lp_ratio > 0.15` |
| `normal` | `lp_ratio >= 0.07 AND <= 0.15` |
| `low` | `lp_ratio < 0.07` |

**Dominance Trend:**

| Trend | Threshold |
|-------|-----------|
| `rising` | `dominance_delta_pp >= +0.5` |
| `neutral` | `dominance_delta_pp > -0.5 AND < +0.5`; also cold-start default |
| `falling` | `dominance_delta_pp <= -0.5` |

## Data Source(s)

- **Primary API:** CoinGecko
- **Endpoints:** `coingecko.global_market` — `/global`; `coingecko.coin_markets_breadth` — `/coins/markets`
- **Fields used from `/global`:** `market_cap_percentage["btc"]` (dominance), `total_volume["usd"]` (lp_ratio numerator), `total_market_cap["usd"]` (lp_ratio denominator)

**Parser verification required:** The field nesting of `total_volume` and `total_market_cap` in the CoinGecko `/global` response has varied across API versions — some versions return these as top-level keys, others nest them under a `data` wrapper. Before finalising the Go parser struct in `internal/api/coingecko/`, verify the exact JSON structure against a live `/global` response on your specific API tier. The `market_cap_percentage` map and the volume/mcap fields must be confirmed to exist at the same nesting level as the parser expects. A mismatch will silently produce `lp_ratio = 0` and `btc_dominance_pct = 0`, triggering cold-start-like behaviour with no error.
- **Fields used from `/coins/markets`:** `price_change_percentage_1h_in_currency`, `price_change_percentage_24h`, `price_change_percentage_7d_in_currency`, `price_change_percentage_30d_in_currency` (breadth); `price_change_percentage_24h` for BTC entry (`id == "bitcoin"`) used as `btc_change_24h_pct` for the modifier gate

**Endpoint cache sharing:** Both endpoint keys are shared with existing metrics. `coingecko.global_market` is used by `liquidity-pulse` and `stablecoin-power`; `coingecko.coin_markets_breadth` is used by `market-breadth` and `momentum-divergence`. When any of those metrics are called in the same session, `mr` receives cached data at zero additional API cost.

**Dominance delta computation:** `mr.Compute()` reads the prior `coingecko.global_market` snapshot from the local file cache to compute `dominance_delta_pp`. The cache layer must expose a `Stat(key)` or `GetMetadata(key)` method returning the file modification timestamp so that `prior_snapshot_age_sec` can be computed as `current_unix_timestamp - file_mod_timestamp` without deserializing the cached JSON twice. If `Stat(key)` returns an error or the file does not exist, treat as cold start — do not crash. If `Stat(key)` succeeds but the cached JSON is malformed or `market_cap_percentage["btc"]` is missing from the parsed prior snapshot, also treat as cold start rather than propagating a parse error.

**First-run behavior:** The first invocation of `mr` always functions as a baseline run — it writes the current `/global` snapshot to file cache and produces a Neutral-row regime because no prior snapshot exists to compute a delta. **The second invocation is the first meaningful one.** Implementers and agents must not treat the first call's regime label as a computed signal; `dominance_cold_start: true` in meta is the machine-readable indicator of this state.

**`btc_dominance_pct` verification asymmetry:** `btc_dominance_pct` is present in basic `meta` because it is the primary raw signal underlying `dominance_trend`. However, `btc_dominance_pct` alone (the current snapshot) is insufficient to verify `dominance_trend` — verification requires both the current and prior values to confirm the delta direction. Full verification of `dominance_trend` requires `dominance_delta_since_last_fetch` at `--detail extended`. Agents on basic output should treat `btc_dominance_pct` as context, not as a `dominance_trend` validator.

## Cross-Source Verification

No cross-source verification in v1.

`market-regime` is a synthesis metric — it derives all three input signals from the same two CoinGecko endpoint keys already used by the suite. The breadth score is computed from the identical parser as `market-breadth`, guaranteeing internal consistency via the cache layer rather than cross-source validation. The dominance delta is self-referential (current snapshot vs. prior cached snapshot of the same endpoint).

A BTC CVD validator (Binance-US `binance.spot_cvd_btc_1h`) was evaluated and explicitly rejected. In a regime metric anchored to BTC dominance, a validator checking for BTC momentum "agreement" is adversarial by design — a successful alt-season (alts rising while BTC falls) produces a negative BTC CVD that would incorrectly reduce confidence on a valid `Alt-Season / Mania` verdict.

Cross-source validation against a second dominance data provider (e.g., CoinMarketCap) is noted as a future enhancement.

## CLI Flags

This metric has no CLI flags in v1.

The breadth computation uses the same `per_page=250` default as `market-breadth`. Configurable universe size is deferred to v1.1 pending alignment with `mb`'s `--top` flag behavior.

## Output Schema

```json
{
    "metric":  "market-regime",
    "version": "v1.0.0",
    "status":  "string",   // "ok" / "degraded" / "unavailable"
                           // "unavailable": coingecko.global_market fetch failed entirely
                           // "degraded": breadth parser TotalCount < 50 (insufficient coin sample)
                           // "ok": all signals computed; cold start and weight redistribution
                           //       do not affect status — they are reflected in confidence and notes

    "data": {
        "regime":               "string",   // one of the 10 named labels (see Classification)
        "modifier":             "string",   // "positive_momentum" | "negative_pressure" | "neutral"
        "dominance_trend":      "string",   // "rising" | "falling" | "neutral"
        "conviction":           "string",   // "high" | "normal" | "low"
        "market_breadth_score": "float64",  // weighted composite [0.0, 1.0]; same value as mb
        "classification": {
            "label":       "string",  // regime label (same as data.regime; present for suite consistency)
            "description": "string"   // fixed one-line mapping per label — not dynamic NL.
                                      // Each of the 10 labels maps to exactly one description string;
                                      // this is a static lookup, not generated text. e.g.:
                                      // "BTC-Led Expansion — broad participation with rising BTC dominance"
                                      // "Capitulation — panic selling, high volume, alts collapsing"
                                      // "Stagnation — flat dominance, narrow breadth, market ignored (Pressure Cooker if conviction is high)"
        },
        "summary": "string"   // NL synthesis using only fields available at basic detail level.
                              // Basic summary uses string signals, not numeric deltas (delta is extended-only).
                              // Summary always incorporates conviction level for all regime labels —
                              // e.g. Stagnation with high conviction is a different agent posture than
                              // Stagnation with low conviction, and the summary must reflect that.
                              // e.g.: "BTC dominance rising, breadth broad — BTC-Led Expansion with
                              //        positive momentum. Capital flowing into BTC with broad alt participation."
                              // e.g.: "Dominance neutral, breadth narrow, low conviction — Stagnation.
                              //        Market ignored; alt exposure is high-risk with no breadth support."
                              // e.g.: "Dominance neutral, breadth narrow, high conviction — Stagnation
                              //        (Pressure Cooker). High volume in a directionless market; breakout likely."
                              // Extended/full summary may include numeric values from extended meta fields.
    },

    "meta": {
        // Always present (basic and above):
        "btc_dominance_pct":   "float64",  // current BTC dominance percentage, e.g. 52.41
        "btc_24h_change":      "float64",  // BTC 24h price change percentage; input to modifier gate
        "confidence":          "string",   // "high" | "medium" | "low" — see confidence table below
        "dominance_cold_start": "bool",    // true if no prior cached snapshot exists
        "notes":               "[]string", // enumerated array; empty [] when no conditions active
                                           // values: "cold_start" | "weight_redistribution" |
                                           //         "divergent_momentum" | "abnormal_capitulation" |
                                           //         "capitulation_price_stabilizing" | "missing_reference_data"
        "cache_hint_sec":      14400,      // call frequency recommendation for agents (not dispatcher TTL)
                                           // regime shifts are structural; re-polling within 4h
                                           // produces identical output on unchanged cached data

        // Present when --detail extended or full:
        "cache_hit":                       "bool",
        "ttl_remaining_sec":               "int",
        "primary_source":                  "coingecko",
        "lp_ratio":                        "float64",   // raw volume/mcap ratio; input to conviction
        "dominance_delta_since_last_fetch": "float64?", // percentage points; omitted on cold start
        "prior_snapshot_age_sec":          "int?",      // seconds since prior snapshot was cached;
                                                        // omitted on cold start (not zero — zero is valid).
                                                        // Go struct must use *int with omitempty tag,
                                                        // not plain int, to correctly omit on cold start
                                                        // rather than emitting 0 (a valid age value).
                                                        // Implementation: cache Stat() should return
                                                        // time.Time or int64 unix timestamp; wrapper
                                                        // computes age as int64(now.Unix() - modTime.Unix())
                                                        // and assigns to *int only when non-nil/non-error.
                                                        // A nil pointer marshals to omitted with omitempty;
                                                        // never assign 0 as a sentinel — 0 means age=0sec.
                                                        // agents: contextualize dominance_trend
                                                        // magnitude against this value before acting

        // Additionally when --detail full:
        // "thresholds": {
        //     "dom_dead_band_pp":       0.5,    // dominance delta dead-band in percentage points
        //     "breadth_broad":          0.60,   // aligned with market-breadth
        //     "breadth_narrow":         0.40,   // aligned with market-breadth
        //     "btc_dir_dead_band_pct":  0.5,    // modifier gate dead-band
        //     "conviction_high":        0.15,   // lp_ratio threshold; aligned with liquidity-pulse
        //     "conviction_low":         0.07,   // lp_ratio threshold; aligned with liquidity-pulse
        //     "capitulation_vol_min":   0.15    // lp_ratio floor for Capitulation to fire
        // }
        // "description": "string"  // full Long Description text
    }
}
```

**Confidence determination:**

| Condition | `confidence` |
|-----------|-------------|
| All signals present, dominance delta available, breadth fully weighted | `"high"` |
| Capitulation with `modifier == "negative_pressure"` — panic confirmed, price following volume | `"high"` |
| Cold start (`dominance_cold_start: true`) — dominance trend defaulted to neutral | `"medium"` |
| Weight redistribution in breadth parser (timeframe TotalCount < 50, weight redistributed) | `"medium"` |
| Capitulation with `modifier == "neutral"` (`"capitulation_price_stabilizing"` in notes) | `"medium"` |
| Capitulation with `modifier == "positive_momentum"` (`"abnormal_capitulation"` in notes) | `"medium"` |
| BTC reference absent (`"missing_reference_data"` in notes) — modifier is a fallback, not a signal | `"low"` |
| `status: "degraded"` (breadth TotalCount < 50 globally) | `"low"` |

Multiple conditions can lower confidence simultaneously; the minimum applicable level applies. `confidence: "high"` requires all conditions above to be absent.

**Notes field — enumerated values and trigger conditions:**

| Value | Trigger Condition |
|-------|------------------|
| `"cold_start"` | No prior `coingecko.global_market` snapshot in file cache; `dominance_delta_since_last_fetch` omitted; `dominance_trend` defaulted to neutral |
| `"weight_redistribution"` | One or more breadth timeframe TotalCounts fell below 50; nominal weights redistributed; `meta.weights_used` (at extended detail) reflects actual weights |
| `"divergent_momentum"` | `mr.regime` is in a safety/defensive cell (`Flight to Safety`, `Stagnation`, `Structural Decay`, `Capitulation`) while the most recent cached `momentum-divergence` output (read from `~/.cache/cryptospect/state_md.json`) shows `risk_on`; or vice versa. Cache file must be ≤ 4 hours old — if absent or stale, check is skipped and note is not appended. Agent should seek a third validator (`stablecoin-power` or `flow-tension`) before acting |
| `"abnormal_capitulation"` | `regime: "Capitulation"` fired AND `modifier == "positive_momentum"` — BTC price actively reversing upward against panic volume; V-bottom or short squeeze. `confidence: "medium"`. Agent posture: act with urgency but lower confidence due to volatility |
| `"capitulation_price_stabilizing"` | `regime: "Capitulation"` fired AND `modifier == "neutral"` — high volume and alt bleed continuing, but BTC price has stopped falling. Stabilization phase or passive absorption of selling. `confidence: "medium"`. Agent posture: shift from "observe for exhaustion" toward "begin scaling/DCA" |
| `"missing_reference_data"` | BTC entry absent from `/coins/markets` response or its `price_change_percentage_24h` field is nil; `modifier` defaulted to `"neutral"` as fallback, not as a market signal. `confidence` forced to `"low"` to distinguish this from a genuine flat-price neutral |

Multiple notes values are valid simultaneously. Example: `["cold_start", "weight_redistribution"]` on a first run with partial API data.

**Enhancements** (conditional — present when specific conditions are met):

| Field | Condition | Description |
|-------|-----------|-------------|
| `notes` non-empty | Any trigger condition active | Array of enumerated diagnostic strings; always present as `[]` when empty |
| `dominance_delta_since_last_fetch` | `--detail extended`; omitted on cold start | Percentage-point change in BTC.D since prior cached snapshot |
| `prior_snapshot_age_sec` | `--detail extended`; omitted on cold start | Age of prior snapshot in seconds; use to contextualize delta magnitude |
| `lp_ratio` | `--detail extended` | Raw volume/mcap ratio; input to conviction classification |
| `thresholds` | `--detail full` | All classification thresholds with units |
| `dominance_cold_start: true` | First run with no prior cache | Signals that dominance_trend is a default, not computed |

## Usage

```bash
# Basic
cryptospect-cli market-regime

# With alias
cryptospect-cli mr

# Extended detail (delta, snapshot age, lp_ratio)
cryptospect-cli market-regime --detail extended

# Full detail (thresholds, description)
cryptospect-cli market-regime --detail full
```

## Long Description

### High-level meaning and value

Market Regime is the structural context layer of the suite. While the other five metrics each answer a specific tactical question — how much dry powder exists, how actively the market is trading, whether leverage is accumulating, how broadly assets are participating, and which tier capital is rotating into — `market-regime` synthesizes the macro picture into a single named state: *"What kind of market are we in right now?"*

The two axes of the matrix are chosen deliberately. Bitcoin Dominance is the best single proxy for the direction of capital flow between safety and risk within crypto: when BTC captures a rising share of total market cap, capital is either building a BTC position before rotating to alts (Institutional Build) or retreating from risk entirely (Flight to Safety). When BTC's share falls, capital is rotating down the risk curve into alts (Alt-Season) or leaving a BTC-led rally that is losing its engine (Capital Rotation). Market Breadth Score determines which of those interpretations applies: broad participation confirms a healthy rotation; narrow participation signals concentration or fear.

The modifier and conviction signals add the two dimensions that a 2×3 matrix cannot express cleanly. The modifier answers "is BTC itself going up or down?" without collapsing the regime label into an explosion of sub-labels. Conviction answers "is this a volume-confirmed event or a quiet drift?" — the distinction that separates Capitulation (a real panic floor) from Structural Decay (a slow bleed nobody is watching).

### Exact definition and data needs, logic

**Dominance delta:** `mr.Compute()` reads `market_cap_percentage["btc"]` from the current `/global` response and compares it against the prior cached value of the same field. The delta is in percentage points — the same units as BTC dominance itself (e.g., 52.41% → 53.03% is a +0.62pp delta). The ±0.5pp dead-band prevents the neutral row from emptying out during low-volatility periods when dominance drifts by fractions of a point.

On cold start, no prior snapshot exists. `dominance_trend` defaults to `"neutral"`, `dominance_delta_since_last_fetch` is omitted from output (not set to 0.0 — zero is a valid delta and cannot be distinguished from "no data"), and `dominance_cold_start: true` is set in meta. The `"cold_start"` note is appended. Status remains `"ok"` and confidence is set to `"medium"`. This is the same pattern as `flow-tension`'s OI 24h change on first run.

**Cache ModTime requirement:** Computing `prior_snapshot_age_sec` requires the file modification timestamp of the prior cached snapshot, not a re-deserialization of its content. The `internal/cache` package must expose a `Stat(key)` or `GetMetadata(key)` method returning the file's `ModTime`. `prior_snapshot_age_sec = current_unix_timestamp - file_mod_timestamp`. If `Stat(key)` fails (file absent, permission error, or cache cleared between write and read), treat as cold start — do not propagate the error to `status`.

**Breadth computation:** The breadth score is computed using the identical parser and null-exclusion logic as `market-breadth`. The same `coingecko.coin_markets_breadth` cached response is used. When `mb` and `mr` are called in the same session, the breadth score will be identical because it is derived from the same cache hit. The thresholds (0.40/0.60) are explicitly aligned with `mb`'s `"narrow"` / `"mixed"` / `"broad"` bands. If weight redistribution occurred (a timeframe's TotalCount fell below 50 and its weight was redistributed to other timeframes), `"weight_redistribution"` is appended to notes and confidence is capped at `"medium"`. The breadth score remains valid and `status` remains `"ok"` — this mirrors `mb`'s own behavior under the same condition.

**BTC reference extraction:** `btc_change_24h_pct` is extracted from the `/coins/markets` response by checking `entry.ID == "bitcoin"` (not positional index), using the same guard logic as `market-breadth`. If the BTC entry is absent or its `price_change_percentage_24h` is nil, `modifier` defaults to `"neutral"` as a fallback — not as a market signal. In this case: append `"missing_reference_data"` to notes and force `confidence: "low"`. An agent reading `modifier: "neutral"` without checking notes cannot distinguish genuine price stability from a data-fetch failure; the notes flag and low confidence together provide the machine-readable signal that the neutral modifier is not a measurement.

**Conviction / lp_ratio:** `lp_ratio = total_volume["usd"] / total_market_cap["usd"]` from the `/global` response. Thresholds (0.07 / 0.15) are aligned with `liquidity-pulse`. Conviction is used only as the Capitulation disambiguation input — it does not affect any other matrix cell or the modifier.

**Zero-guard (lp_ratio):** If `total_volume["usd"]` or `total_market_cap["usd"]` parses as zero or is missing from the response, `lp_ratio` must not be computed — set `status: "unavailable"` and halt. A parsed-zero produces `lp_ratio = 0` → `conviction: "low"`, which silently misrepresents a critical parse failure as a quiet low-interest market. Do not allow this. Note: CoinGecko may return HTTP 200 with empty or null data fields under rate-limiting on some tiers (rather than a `429`). Do not rely solely on HTTP status code to detect unavailability — apply the zero-guard as a content-level check regardless of HTTP status.

**TTL:** The `coingecko.global_market` and `coingecko.coin_markets_breadth` endpoint keys have TTLs set at the dispatcher level, shared with other metrics. `cache_hint_sec: 14400` in `mr`'s output is a *call frequency recommendation* for the calling agent — it signals that re-invoking `mr` within 4 hours will return cached data. It does not modify the dispatcher's endpoint TTL.

### Possible values and associated verdicts

**BTC-Led Expansion**
The strongest structurally healthy bull configuration: BTC dominance rising while the majority of the market participates. Capital is flowing into BTC and dragging alts upward. When combined with `stablecoin-power` High and `flow-tension` neutral-to-positive funding, this is the highest-conviction long environment in the suite. With `negative_pressure` modifier: an unusual combination suggesting BTC is rising in dominance on an absolute down day — monitor whether this resolves as Institutional Build or Flight to Safety on the next call.

**Institutional Build**
BTC outperforming a mixed alt market. Typical of early bull cycles or accumulation phases where sophisticated capital positions in BTC before rotating to alts. Expect this to transition to BTC-Led Expansion if alts begin to follow, or to Flight to Safety if the overall market deteriorates. With `negative_pressure` modifier: BTC showing relative strength in a declining market — defensive concentration, not accumulation. Do not enter alt positions in this sub-state.

**Flight to Safety**
Capital retreating into BTC as the crypto safe harbor. BTC not necessarily rising in absolute terms — it is simply losing value more slowly than everything else. The alt tier is being actively sold or abandoned. Warrants defensive positioning. If `stablecoin-power` `supply_trend_7d` is also `"contracting"`, treat as macro risk-off confirmation (capital leaving crypto entirely, not just rotating to BTC).

**Steady Appreciation**
The quietest bull signal: balanced market rising with broad participation and no dominance shift. No rotation extremes, no concentration. Healthy and sustainable but lower-alpha than BTC-Led Expansion or Alt-Season.

**Consolidation**
Neutral dominance, mixed participation. The market is seeking direction. Do not increase exposure. With `conviction: "high"` (the Pressure Cooker condition): a large volume of trades is occurring within a tightening range — historically precedes a violent regime break in either direction. Cross-reference `flow-tension` OI hook: if OI is `"building"` simultaneously, a forced resolution is likely.

**Stagnation**
The highest-risk environment for alt entry in the suite. Dominance flat, breadth narrow. Alts are not being actively sold relative to BTC — they are being ignored. Any sudden BTC volatility amplifies into alts with no breadth support to absorb it. Distinct from Structural Decay: no directional dominance signal exists, so this could resolve in either direction.

**Alt-Season / Mania**
Capital rotating down the risk curve with broad participation. Maximum risk appetite. With `positive_momentum` modifier: the classic alt-season confirmation — alts leading a rising market. Cross-reference `stablecoin-power` for dry powder remaining and `flow-tension` funding rate for overheated conditions: a `"high"` conviction alt-season with overheated funding is historically correlated with late-cycle blow-off peaks. With `negative_pressure` modifier: two distinct scenarios share this state and require different responses. The more common case is **Speculative Exhaustion** — alts leading the market lower, a distribution signal where capital that rotated into alts is now being liquidated faster than BTC. The less common case is a **BTC-specific dislocation** — BTC is being uniquely liquidated (e.g., ETF outflows, exchange-specific BTC selling) while alts hold relatively firm, causing dominance to fall on a down day. Check `flow-tension` CVD: aggressive BTC sell CVD with neutral alt CVD suggests the BTC-dislocation case; broad negative CVD across the market confirms Speculative Exhaustion. Do not assume the common case without checking.

**Capital Rotation**
Falling dominance with selective participation. Some sectors capturing rotation while others lag. An early or incomplete alt-season. Cross-reference `momentum-divergence` `mid_vs_large` spread: if `risk_on` is also firing there, the rotation is likely to broaden. If `momentum-divergence` is `"neutral"`, rotation may be stalling before it reaches full breadth.

**Structural Decay**
Falling dominance + narrow breadth + low/normal conviction. Alts bleeding harder than BTC on thin volume. Not a panic event — a slow, directional bleed with no forced exits and no violent bottom signal. The directional lean (dominance falling) distinguishes it from Stagnation, where dominance is flat and the market is simply being ignored. Agents should not treat Structural Decay as a capitulation entry point — the absence of high volume means selling may continue without a flush. The `dominance_trend: "falling"` field always confirms this label vs. Stagnation for programmatic agent logic.

**Capitulation**
Falling dominance + narrow breadth + high conviction volume. Panic selling confirmed by volume. BTC relatively outperforming a collapsing alt market. Historically the condition that precedes the best long-term accumulation opportunities — but not a timing signal on its own. The modifier determines the sub-state and the appropriate agent posture:

- `modifier: "negative_pressure"` (standard): Panic is efficient — price is following volume down. Expected state. `confidence: "high"`. Posture: observe for exhaustion signals (`flow-tension` OI unwinding, funding turning negative) before acting.
- `modifier: "neutral"` (`"capitulation_price_stabilizing"`): High volume and alt bleed continuing, but BTC price has stopped falling. Stabilization phase or passive absorption of selling. `confidence: "medium"`. Posture: shift from observation toward cautious scaling — this is the transition zone between panic and floor.
- `modifier: "positive_momentum"` (`"abnormal_capitulation"`): BTC price actively reversing upward against ongoing panic volume. V-bottom or short squeeze in progress. `confidence: "medium"`. Posture: act with urgency but lower confidence — the reversal may be genuine or a dead-cat bounce. Cross-reference `flow-tension` OI for squeeze confirmation.

**Cache window caveat for fast-moving panics:** `cache_hint_sec: 14400` is calibrated for structural regime detection. During high-volatility capitulation events, the market can transition between these three sub-states within 15–30 minutes — faster than the 4-hour cache window. If an agent calls `mr` during a brief neutral window mid-crash, it will receive `capitulation_price_stabilizing` when the market may still be in active decline. Cross-reference `flow-tension` (1-hour TTL) for faster-cycle confirmation of whether the stabilization is genuine.

### Other details

**CLI Flags:** No CLI flags in v1. Breadth universe size is fixed at `per_page=250`, consistent with `market-breadth` default. Configurable universe size is deferred to v1.1.

**Enhancements:**
- `notes` is always present as an array (empty `[]` when no conditions active). Agents must never check for field absence before reading notes.
- `dominance_delta_since_last_fetch` and `prior_snapshot_age_sec` are both omitted (not null, not zero) on cold start. Agents must check `dominance_cold_start` before reading these fields.
- `prior_snapshot_age_sec` at extended detail is the primary tool for contextualizing dominance trend magnitude. A `"rising"` trend over 600 seconds is noise; over 14400 seconds it is structural. The `dominance_trend` string is the authoritative classification; the raw delta and age are audit fields.

**Cross-Source Verification:** No cross-source verification in v1. The BTC CVD validator was explicitly rejected as architecturally adversarial for a dominance-anchored metric. See Cross-Source Verification section above.

**Implementation Compromises:**
- **`cache_hint_sec: 14400` is too wide for fast-moving capitulation events.** The 4-hour call frequency recommendation is correct for structural regime detection (BTC-Led Expansion, Alt-Season, Institutional Build) where regime shifts take hours or days. During active Capitulation, the three sub-states (`negative_pressure` → `capitulation_price_stabilizing` → `abnormal_capitulation`) can cycle within 15–30 minutes. An agent respecting the 4-hour hint will miss intraday sub-state transitions during the most time-sensitive regime. The correct mitigation is to use `flow-tension` (1-hour TTL) as the fast-cycle companion during any Capitulation regime, and to reduce `mr` polling frequency only to the structural labels where 4 hours is appropriate. A future enhancement could emit a dynamic `cache_hint_sec` that varies by regime label — lower during Capitulation, higher during stable regimes. On cold start, the Neutral row fires by definition regardless of true market state. A developer or agent that acts on the first call's regime label without checking `dominance_cold_start` is operating on a default, not a measurement. The baseline behavior is deterministic and documented, but it cannot be avoided — it is an inherent property of a delta-based signal in a stateless CLI. The second call is the first actionable one.
- **Dominance delta is "since last fetch," not "24h."** The delta window is determined by the caller's polling interval and the cache TTL of `coingecko.global_market`, not by a fixed 24h window. A caller polling every 15 minutes will see small deltas; a caller respecting `cache_hint_sec: 14400` will see 4h deltas. The `dominance_trend` string is calibrated against the ±0.5pp dead-band and remains the authoritative signal. The raw delta is an audit field only.
- **`prior_snapshot_age_sec` requires cache ModTime access.** If the `internal/cache` package does not expose `Stat(key)`, this field cannot be computed. This is a prerequisite infrastructure change. Until it is implemented, `prior_snapshot_age_sec` may be omitted at extended detail and noted in `meta` as unavailable. `dominance_delta_since_last_fetch` can still be computed from the cached value content without the timestamp.
- **Structural Decay and Stagnation share a superficially similar appearance but are distinct signals.** Stagnation (Neutral+Narrow) means dominance is flat and the market is being ignored — no directional commitment from either side. Structural Decay (Falling+Narrow, low/normal conviction) means alts are bleeding harder than BTC on thin volume — a directional signal, just a slow one. Both produce the same agent posture (no panic entry, no forced exits), which is why they were originally collapsed into one label. They are separated because the `dominance_trend` field carries different forward-looking implications: Stagnation can resolve in either direction; Structural Decay has a directional lean. The `dominance_trend` field always disambiguates them for agents that need to distinguish.
- **Stablecoin influence on breadth score.** Stablecoins in the top-250 universe register near-zero price changes and create a slight conservative bias on the breadth score — the same limitation documented in `market-breadth`. The breadth threshold of 0.40/0.60 is calibrated against this bias being present.
- **BTC dominance is a lagging signal in fast-moving markets.** During rapid price dislocations (e.g., a flash crash), dominance percentage shifts may lag the actual capital flow by minutes due to CoinGecko's data aggregation pipeline. The modifier (BTC price direction) captures the intraday move faster than the dominance delta. In high-volatility conditions, treat the modifier as the leading signal and the dominance trend as the confirming signal.
- **`lp_ratio` inherits stablecoin volume noise from `liquidity-pulse`.** CoinGecko's `/global` total volume includes stablecoin trading volume without filtering. A stablecoin depegging event (e.g., a USDT or USDC peg scare) can spike global volume and push `lp_ratio` above 0.15, triggering `conviction: "high"` and potentially firing the Capitulation label without reflecting genuine broad market panic. No stablecoin volume filtering is applied in v1. Agents encountering a Capitulation label should cross-reference `flow-tension` CVD and OI to confirm whether selling pressure is real before acting.
- **`divergent_momentum` requires persistent metric state infrastructure.** Detecting a contradiction between `mr` and `md` requires `mr.Compute()` to read the most recent `md` output. This is implemented via a **persistent metric state file**: on every successful `Compute()`, each metric writes its final JSON output to `~/.cache/cryptospect/state_<metric>.json` (e.g., `state_md.json`). `mr` reads this file at compute time and checks the `data.classification.label` field. If the file is absent, unreadable, or its `updated_at` timestamp is older than 4 hours, the divergence check is skipped entirely and `"divergent_momentum"` is not appended. This write-on-success pattern is additive — it does not affect any existing cache infrastructure for API responses. It is a new, separate file per metric in the same cache directory. The `state_<metric>.json` file is not a cache of raw API data; it is a cache of computed metric output. Implementers must add the write step to the metric runner after a successful `Compute()` call, not inside `Compute()` itself.

**Future Enhancements:**
- **`delta_24h` on breadth score:** Percentage change in `market_breadth_score` from prior cached value. Required prerequisite for the Phase overlay (Early/Mid/Late) — "Early" and "Late" are defined by whether breadth is improving or declining, which is a trend question that cannot be answered from a point-in-time snapshot. To be implemented when historical cache is available, consistent with the same enhancement planned for `market-breadth` and `liquidity-pulse`.
- **True 24h dominance delta:** Requires a timestamped cache-history walker that confirms the prior snapshot is approximately 24h old. Prerequisite: timestamp stored alongside cached content (or derivable from file ModTime with age validation). Planned for v1.1.
- **`--top` flag for breadth universe:** Align with `market-breadth`'s `--top N` flag so users can run both metrics with a consistent coin universe. Requires coordination with the shared endpoint key and `per_page` parameter.
- **Phase overlay (Early/Mid/Late):** Deferred from v1 because "Early/Mid/Late" requires trend data (is the breadth score improving or declining?) rather than point-in-time snapshots. Requires `delta_24h` on breadth score, which itself requires historical cache. Planned for v1.1 when historical cache is available.
- **Cross-source dominance validation:** Compare CoinGecko BTC dominance against CoinMarketCap or an on-chain market cap computation. CoinMarketCap free tier is rate-limited to 30 calls/month — unsuitable for real-time CLI in v1.
- **Volume conviction per regime cell:** A future `conviction_trend` signal (is lp_ratio rising or falling within the current regime?) would improve early detection of Consolidation → breakout transitions.

**Agentic Logic (Strategic Notes)**

When an LLM or agent calls this tool, apply the following heuristics:

- **Always read `regime` + `modifier` together.** The regime label is the structural state; the modifier is the immediate directional pressure. "Institutional Build" alone is bullish; "Institutional Build" + `negative_pressure` is a warning that the build is happening in a down market — relative strength, not accumulation. Never act on regime label alone without checking modifier.

- **Check `notes` before acting.** If `"cold_start"` is present, `dominance_trend` is a default, not computed — the matrix fired on the Neutral row by definition. Do not treat a Neutral-row regime from a cold-start call with the same confidence as a computed Neutral. Make a second call after `cache_hint_sec` to get a populated delta.

- **Check `prior_snapshot_age_sec` (extended detail) before acting on dominance trend magnitude.** A `"rising"` trend with `prior_snapshot_age_sec: 900` (15 minutes) is noise-level. A `"rising"` trend with `prior_snapshot_age_sec: 14400` (4 hours) is structural. The `dominance_trend` string is the classification; the age contextualizes whether to treat it as a regime confirmation or a transient tick.

- **The Pressure Cooker pattern:** `regime: "Consolidation"` + `conviction: "high"` is the highest-urgency neutral state in the suite. High volume within flat dominance and mixed breadth means a large amount of capital is changing hands without a directional resolution. Cross-reference `flow-tension` OI: if `"building"`, the coil is tightening and a violent break is likely. Do not initiate directional positions — wait for regime to shift to an adjacent cell.

- **`"divergent_momentum"` note requires a third validator.** When this note is present, `mr` and `momentum-divergence` are giving opposing signals. Neither takes precedence. Use `stablecoin-power` (is dry powder present for a bull move?) and `flow-tension` (is leverage confirming or opposing?) as tiebreakers. Do not act on either signal alone.

- **Stagnation is the highest-risk alt-entry state.** It does not look dangerous — dominance is flat, breadth is just narrow. But it is the state with the least support for alt positions: no rotation signal, no participation depth, no directional anchor. Any BTC move amplifies into alts without a breadth floor to absorb it. Treat as equivalent to Flight to Safety for alt exposure decisions.

- **Capitulation is not a buy signal — it is a necessary condition for a buy signal.** Read the sub-state note first: `"capitulation_price_stabilizing"` (BTC stopped falling) is the shift from observation to cautious scaling; `"abnormal_capitulation"` (BTC actively reversing) is the urgency signal with lower confidence. Neither fires without `flow-tension` confirmation: OI unwinding + funding turning negative + `stablecoin-power` stable or expanding are the exhaustion signals required before treating any Capitulation sub-state as an entry. During fast-moving panics, cross-reference `flow-tension` (1h TTL) rather than waiting for `mr`'s 4h cache window to refresh.

- **Cross-metric named combinations:**
  - `BTC-Led Expansion` + `stablecoin-power High` + `flow-tension funding: "neutral"` → **Maximum Bull Setup.** Fuel available, broad participation, leverage not yet crowded. Strongest long configuration in the suite.
  - `Flight to Safety` + `stablecoin-power supply_trend_7d: "contracting"` → **Macro Risk-Off.** Capital fleeing both alts and crypto. Full defensive posture, not just alt avoidance.
  - `Alt-Season / Mania` + `modifier: "positive_momentum"` + `flow-tension funding: "overheated"` → **Blow-Off Warning.** Classic late-cycle peak configuration. Cross-reference `stablecoin-power` — if Low, the fuel is depleted and the blow-off is structurally unsupported.
  - `Consolidation` + `conviction: "high"` + `flow-tension OI: "building"` → **Pressure Cooker / Imminent Break.** Prepare for a regime shift; direction unknown but likely violent.
  - `Structural Decay` + `momentum-divergence: "flight_to_safety"` → **Confirmed Broad Selloff.** Both the macro matrix and the tier-rotation signal agree. Highest urgency for defensive repositioning across the alt tier.
  - `Capitulation` + `stablecoin-power High` + `flow-tension OI: "unwinding"` + `funding: "negative"` → **Capitulation Floor Confirmation.** All four conditions together are the closest the suite gets to a validated accumulation entry signal.