# flow-tension

**version:** `v1.0.0`
**Alias:** `ft`
**Endpoints:** main: `binance_us.spot_klines_btc_1h`, supplementary: `coingecko_pro.derivatives_btc_oi`, `coingecko_pro.derivatives_btc_funding`

## Overview

Measures the kinetic energy of the market — how aggressively traders are using leverage and moving assets onto exchanges to trade. While `stablecoin-power` shows potential energy (dry powder available), `flow-tension` shows whether that powder is actually igniting. It tracks three distinct signals: taker aggression in spot markets (CVD), leverage accumulation or unwinding (Open Interest), and the cost of holding leveraged positions (Funding Rate). Together, these reveal whether price moves are conviction-driven or fragile, and whether the current regime favors longs or shorts.

This metric operates in two modes depending on API key availability: **full mode** (all three signals) and **degraded mode** (CVD only, no API key required).

## Formula (or how to compute)

This is a multi-signal metric. There is no single composite score — three signals are computed independently and reported together. The `summary` field synthesizes the combination into a verdict string.

**Signal 1 — Exchange Net Flow (CVD proxy)**
```
cvd_ratio = (taker_buy_volume - taker_sell_volume) / (taker_buy_volume + taker_sell_volume)
```
Result is a normalized ratio in [-1, 1]. Positive = net buyer aggression; negative = net seller aggression.
Source: Binance-US BTC/USDT 1h spot klines. Keyless.

**Signal 2 — Open Interest 24h Change**
```
oi_change_pct = (oi_current - oi_24h_ago) / oi_24h_ago
```
Result is a percentage change. Positive = leverage building; negative = leverage unwinding.
Source: CoinGecko Pro derivatives endpoint. Requires API key.

**Signal 3 — Funding Rate (current 8h cycle)**
```
funding_rate = raw funding rate for BTC perpetual (decimal per 8h cycle)
```
Reported as a decimal (e.g., 0.0003 = 0.03% per 8h). Sign indicates direction: positive = longs paying shorts; negative = shorts paying longs.
Source: CoinGecko Pro derivatives endpoint. Requires API key.

## Interpretation

Each signal carries independent meaning. Their combination produces a regime verdict:

**CVD Ratio**
- Positive (> +0.10): Buyers are aggressive — taker demand is absorbing offers. Bullish short-term pressure.
- Neutral (-0.10 to +0.10): Balanced order flow. No strong directional aggression.
- Negative (< -0.10): Sellers are aggressive — takers hitting bids. Bearish short-term pressure.

**Open Interest Change**
- Building (> +5%): Leverage is accumulating. Volatility is loading — breakout likely in either direction.
- Stable (-5% to +5%): Leverage is flat. Market is not amplifying current price action.
- Unwinding (< -5%): Leverage is being removed. Capitulation or profit-taking in progress.

**Funding Rate**
- Overheated (> +0.30% per 8h): Longs are paying an extreme premium to stay open. Historically associated with blow-off tops; long squeeze risk is elevated.
- Positive (+0.03% to +0.30%): Longs paying shorts — bullish sentiment is the dominant funded position.
- Neutral (-0.03% to +0.03%): Balanced funding; neither side is paying a meaningful premium.
- Negative (≤ -0.03%): Shorts paying longs — bearish sentiment is the dominant funded position. Often precedes reversals when combined with positive CVD.

**Signal combinations that produce named verdicts:**
- Funding shifts from negative → neutral/positive + CVD turning positive: **"Early Bull Phase"** — sellers exhausted, buyers regaining control.
- High exchange inflows (strong negative CVD, assets moving to exchanges) + price flat: **"Supply Shock / Top Warning"** — assets being staged for potential sell-off.
- OI building rapidly + price stable + neutral CVD: **"Building Tension"** — leverage accumulating without directional resolution; volatility breakout is likely.
- Funding overheated + CVD fading or turning negative: **"Long Squeeze Risk"** — crowded longs at risk of forced unwind.
- OI unwinding + negative CVD: **"Deleveraging"** — leverage being removed, often a flush in progress.

## Classification

Flow Tension does not use a single composite classification label. Each signal has its own categorical hook:

**CVD Hook**

| Label | Threshold |
|-------|-----------|
| `aggressive_buy` | cvd_ratio > +0.10 |
| `neutral` | -0.10 ≤ cvd_ratio ≤ +0.10 |
| `aggressive_sell` | cvd_ratio < -0.10 |
| `low_confidence` | fewer than 10 trades in sample window |

**OI Hook** *(requires API key; `null` in degraded mode)*

| Label | Threshold |
|-------|-----------|
| `building` | oi_change_pct > +5% |
| `stable` | -5% ≤ oi_change_pct ≤ +5% |
| `unwinding` | oi_change_pct < -5% |

**Funding Hook** *(requires API key; `null` in degraded mode)*

| Label | Threshold |
|-------|-----------|
| `overheated` | funding_rate > +0.003 (> +0.30% per 8h) |
| `positive` | +0.0003 ≤ funding_rate ≤ +0.003 |
| `neutral` | -0.0003 < funding_rate < +0.0003 |
| `negative` | funding_rate ≤ -0.0003 |

**Threshold rationale:** Funding rate thresholds are calibrated against observed BTC perpetual data (historical range: approximately -0.29% to +0.85% per 8h). The ±0.03% neutral band captures the typical noise floor; +0.30% marks the upper tail where funding cost becomes punishing for longs and historical correlation with short-term tops strengthens. OI ±5% captures intraday noise while flagging genuine leverage accumulation. CVD ±10% of total volume separates decisive taker imbalance from balanced churn.

## Data Source(s)

- **Primary API (keyless):** Binance-US
  - **Endpoint:** `/api/v3/klines` (BTC/USDT, 1h interval)
  - **Fields used:** `takerBuyBaseAssetVolume`, computed taker sell volume (total volume minus taker buy volume), `numberOfTrades`
  - **Instrument:** BTC/USDT spot

- **Primary API (keyed):** CoinGecko Pro
  - **Endpoint:** `/derivatives/exchanges` or equivalent derivatives endpoint
  - **Fields used:** BTC perpetual `open_interest_usd` (current + 24h prior), `funding_rate`
  - **Requires:** `CRYPTOSPECT_COINGECKO_KEY` environment variable

**Why CoinGecko Pro over Binance Futures for OI and Funding Rate:**
CoinGecko Pro aggregates OI and funding data across multiple derivatives exchanges, not just Binance. This produces a more representative picture of the global perp market. Binance Futures is a single-exchange view and is subject to Binance-specific dynamics (e.g., Binance liquidation cascades that don't reflect aggregate market health). Rate-limit budget is also a factor: since CoinGecko is already the primary source for macro metrics (`liquidity-pulse`, `stablecoin-power`), routing derivatives data through CoinGecko Pro consolidates API key usage and rate-limit headroom into a single provider relationship.

**Why Binance-US for CVD (rather than CoinGecko Pro):**
CVD requires raw trade-level kline data (taker buy vs. sell volume per candle). CoinGecko does not expose this granularity — it provides aggregate volume, not taker-side breakdown. Binance-US klines are the most accessible source of taker-disaggregated spot volume without an API key, making them the correct source for the keyless CVD signal.

## Cross-Source Verification

No cross-source verification in v1.

Each of the three signals originates from a genuinely distinct data type (spot kline taker volume, aggregate derivatives OI, perpetual funding rate). There is no second source that exposes all three with compatible methodology and scope. Comparing Binance-US CVD to another exchange's CVD would measure exchange-specific differences rather than validate data quality. Comparing CoinGecko Pro OI to a single exchange's OI would be structurally misleading (aggregate vs. point source).

Cross-source validation is noted as a future enhancement; see Future Enhancements.

## CLI Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--instrument` | `string` | `btc` | Target instrument for all three signals. Only `btc` is supported in v1; passing any other value returns an error. Reserved for future multi-asset support. |

**Note on `--instrument`:** BTC-only is a calibration constraint, not an arbitrary limitation. The classification thresholds for funding rate and OI change are calibrated specifically against BTC perpetual historical data. Applying these thresholds to ETH or SOL perps without re-calibration would produce unreliable hooks. Multi-asset support requires per-asset threshold calibration and is queued as a future enhancement.

## Output Schema

```json
{
    "metric":  "flow-tension",
    "version": "v1.0.0",
    "status":  "string",  // "ok" / "degraded" / "unavailable"
                          // "degraded" = CoinGecko Pro key absent; CVD computed, OI/funding null
                          // "unavailable" = Binance-US fetch failed; no signals available

    "data": {
        "signals": {
            "cvd": {
                "ratio":  "float64",  // normalized [-1, 1]; positive = net buy aggression
                "hook":   "string"    // "aggressive_buy" / "neutral" / "aggressive_sell" / "low_confidence"
            },
            "open_interest": {
                "change_pct_24h": "float64 | null",  // null in degraded mode
                "hook":           "string | null"     // "building" / "stable" / "unwinding" / null
            },
            "funding_rate": {
                "rate":  "float64 | null",  // decimal per 8h cycle; null in degraded mode
                "hook":  "string | null"    // "overheated" / "positive" / "neutral" / "negative" / null
            }
        },
        "summary": "string"  // NL verdict combining active signals, e.g.:
                             // "BTC perp: aggressive buying (CVD +0.14), OI building (+7.2%), funding positive (0.045%/8h) — Building Tension with bullish lean."
                             // In degraded mode: "BTC spot: neutral taker flow (CVD +0.02). OI and funding unavailable — API key required for full signal set."
    },

    "meta": {
        // Omitted when --detail basic.
        // Present when --detail extended or full:
        "cache_hit":         "bool",
        "ttl_remaining_sec": "int",        // 3600s (1h) — tactical staleness budget
        "primary_sources":   ["binance_us", "coingecko_pro"],  // coingecko_pro omitted in degraded mode
        "keyed_signals":     "bool",       // true if OI + funding were computed; false in degraded mode
        "api_key_env_var":   "string",     // "CRYPTOSPECT_COINGECKO_KEY" — documents which key unlocks full mode
        "instrument":        "string",     // "btc" (always in v1)
        "cvd_sample_trades": "int",        // number of trades in CVD sample; < 10 triggers low_confidence hook
        "confidence":        "string"      // "high" (all signals present) / "degraded" (CVD only)
                                           // Note: no discrepancy detection in v1; confidence reflects signal completeness
        // Additionally when --detail full:
        // "thresholds": {
        //     "cvd": { "aggressive": 0.10 },
        //     "oi":  { "building": 0.05, "unwinding": -0.05 },
        //     "funding": { "negative": -0.0003, "positive": 0.0003, "overheated": 0.003 }
        // }
        // "description": "string"  // full Long Description text
    }
}
```

**Note on `confidence` field:** For this metric, `confidence` reflects signal completeness rather than cross-source discrepancy (no cross-source validation exists in v1). `"high"` means all three signals were computed. `"degraded"` means only CVD was available. This is a semantic divergence from the other metrics where `confidence` reflects validator agreement — documented here to prevent agent misinterpretation.

**Enhancements** (conditional — present when specific conditions are met):

| Field | Condition | Description |
|-------|-----------|-------------|
| `cvd.hook: "low_confidence"` | `cvd_sample_trades < 10` | Thin-candle guard: fewer than 10 trades in the sample window means CVD ratio is statistically unreliable. Hook is overridden to `low_confidence` regardless of ratio value. |
| `open_interest.*: null` | API key absent (`status: "degraded"`) | OI fields are present in schema but set to `null`. Agent should check `keyed_signals` in meta to determine cause. |
| `funding_rate.*: null` | API key absent (`status: "degraded"`) | Funding rate fields are present in schema but set to `null`. Same cause as OI nulls. |
| `delta_24h` (not yet implemented) | Prior cache data exists | Percentage change in CVD ratio from 24h prior. Planned for v1.1 when historical cache is available. |

## Usage

```bash
# Basic (keyless — CVD only, OI and funding null)
cryptospect-cli flow-tension

# With alias
cryptospect-cli ft

# Full detail (requires CRYPTOSPECT_COINGECKO_KEY for complete signal set)
cryptospect-cli flow-tension --detail full

# Extended detail
cryptospect-cli flow-tension --detail extended

# Instrument flag (v1: btc only)
cryptospect-cli flow-tension --instrument btc
```

**Setting the API key for full mode:**
```bash
export CRYPTOSPECT_COINGECKO_KEY=your_key_here
cryptospect-cli flow-tension
```

## Long Description

### High-level meaning and value

Flow Tension is the "Transmission" metric of the suite — it measures whether the potential energy in `stablecoin-power` is being converted into actual market activity, and if so, in what direction and with how much conviction.

The three signals it tracks answer three distinct questions:

1. **CVD (Exchange Net Flow proxy):** Are buyers or sellers currently the aggressor in spot markets? Takers initiate trades by hitting bids or lifting offers — their behavior reveals who has urgency. A taker-dominated buy side signals conviction; a taker-dominated sell side signals distribution or panic.

2. **Open Interest (24h change):** Is leverage accumulating or being removed? Rising OI means more contracts are open — the market is loading for a potential move. Falling OI means positions are being closed — either capitulation or profit-taking. Crucially, high OI without price movement is a volatility coil: the tension is building and will resolve.

3. **Funding Rate (current 8h cycle):** What is the market's aggregate directional bet, and how expensive is it to maintain that bet? Positive funding means the market is collectively long and longs are paying shorts to stay open. Negative funding means the reverse. The overheated zone (> +0.30% per 8h) is historically associated with crowded-long conditions and elevated short-term correction risk.

The metric does not collapse these into a single score because their combination carries meaning that a scalar would destroy. "CVD aggressive buy + OI building + funding neutral" is an early bull regime. "CVD neutral + OI building + funding overheated" is a blow-off warning. These are opposite verdicts that would average to the same composite number.

### Exact definition and data needs, logic

**CVD computation:**
Binance-US BTC/USDT 1h klines expose `takerBuyBaseAssetVolume`. Taker sell volume is computed as `totalVolume - takerBuyBaseAssetVolume`. The ratio is `(buy - sell) / (buy + sell)`, normalized to [-1, 1]. A thin-candle guard rejects candles with fewer than 10 trades (`numberOfTrades < 10`) and overrides the hook to `low_confidence`, preventing noise from low-activity periods from producing misleading signals.

**OI change computation:**
CoinGecko Pro returns current BTC perpetual OI in USD. The 24h change is computed against the prior cached value or a secondary API call for the prior value. Result is expressed as a percentage change.

**Funding rate:**
CoinGecko Pro returns the current funding rate for BTC perpetual as a decimal per 8h cycle. No transformation is applied — the raw value is reported alongside its hook classification.

**TTL:** 3600 seconds (1 hour). This is a tactical-signals metric. Unlike macro metrics (`stablecoin-power`, `liquidity-pulse`) which operate on 24h+ data rhythms, funding rate and OI are meaningful at the 1h resolution. A 4–6h TTL would risk serving stale tactical signals.

### Possible values and associated verdicts

**"Early Bull Phase"**
Funding shifts from negative → neutral/positive; CVD turns positive. Sellers who were paying to short are closing positions; buyers are becoming the aggressor. This regime often precedes sustained upward moves when confirmed by `stablecoin-power` showing available dry powder.

**"Building Tension"**
OI rising > +5% with price flat and CVD neutral. Leverage is accumulating without a directional resolution. Volatility breakout is likely in either direction. This is not a directional signal — it is a warning that the coil is tightening.

**"Supply Shock / Top Warning"**
Strong negative CVD (sellers aggressive) with flat price. Assets are being moved to exchanges and sold into bids that are absorbing the flow. If this condition precedes a price breakdown, the setup was telegraphed. Often corresponds with high exchange inflow data.

**"Long Squeeze Risk"**
Funding overheated (> +0.30%) with CVD fading or turning negative. Longs are paying an unsustainable premium and taker aggression is shifting to the sell side. Forced liquidations of crowded longs become the tail risk.

**"Deleveraging"**
OI falling > -5% with negative CVD. Positions are being force-closed or voluntarily unwound while sellers are aggressive. Typically a flush in progress — can be a cleansing event or the start of a deeper move depending on `stablecoin-power`.

**Degraded mode verdicts:**
In degraded mode (CVD only), the `summary` will produce CVD-only verdicts ("Aggressive buying," "Neutral flow," "Aggressive selling," "Low confidence — thin candle") without OI or funding context. The `summary` string will explicitly note that OI and funding are unavailable and reference the API key requirement.

### Other details

**CLI Flags:**
`--instrument btc` is the only valid value in v1 and is the default. The flag is exposed to establish the interface for future multi-asset support, not because it does anything different from the default. Passing any non-`btc` value returns a validation error with a message explaining that only BTC is calibrated in v1.

**Enhancements:**
- `cvd.hook: "low_confidence"` is a data-quality guard, not a signal. Agents should treat it as a null equivalent for CVD and rely only on OI and funding if those are available.
- `delta_24h` on CVD is planned for v1.1. Percentage change form preferred over absolute difference — a CVD shift from -0.05 to +0.08 is more meaningful as a directional regime change than as an absolute +0.13 difference.

**Cross-Source Verification:**
No cross-source verification in v1. The three signals are sourced from different APIs because no single API exposes all three with appropriate granularity. The absence of a validator means `confidence` reflects signal completeness (all signals present vs. degraded) rather than inter-source agreement. This is explicitly noted in the schema.

**Implementation Compromises:**
- **CVD is a proxy, not true net flow.** Binance-US spot klines measure taker aggression within a single exchange. This is not the same as global exchange net inflow (coins moving from cold wallets to exchange hot wallets). A full exchange net flow signal would require on-chain data (e.g., Glassnode). The CVD proxy is a reasonable substitute at the tactical timeframe but should not be described to agents as "coins moving onto exchanges" — it is taker buy/sell imbalance within existing on-exchange liquidity.
- **OI is BTC perpetual aggregate, not spot-specific.** Derivatives OI reflects the leveraged market's positioning, which can diverge from spot market dynamics. A large derivatives position can exist independently of spot buying pressure.
- **Funding rate is current 8h cycle only.** The metric does not track funding rate trend (whether it is rising or falling toward the overheated zone). A funding rate of +0.25% that was +0.05% yesterday is a very different situation from one that was +0.28% yesterday. Trend tracking is a future enhancement.
- **No stablecoin-pair filtering for CVD.** Binance-US BTC/USDT klines include USDT as the quote currency. USDT-specific events (e.g., a USDT de-peg scare) could influence CVD without reflecting genuine BTC sentiment. This is an accepted approximation for v1.

**Future Enhancements:**
- **Multi-asset support (`--instrument eth`, etc.):** Requires per-asset threshold re-calibration. ETH and SOL perpetuals have different typical funding ranges and OI dynamics than BTC. Shipping calibrated thresholds per asset is the prerequisite.
- **Funding rate trend (`funding_trend_1h`):** Direction of funding rate change over the past 1–4 hours. Rising-toward-overheated is a different signal from falling-from-overheated. Percentage change or slope preferred over absolute value for trend.
- **`delta_24h` on CVD ratio:** Requires historical cache. Percentage change form preferred.
- **True exchange net flow:** Replace CVD proxy with on-chain exchange inflow/outflow data (Glassnode or equivalent). Materially more accurate for the "assets moving to exchanges" interpretation but requires an additional API and likely an API key.
- **Cross-source OI validation:** Compare CoinGecko Pro aggregate OI against a second derivatives data provider (e.g., Coinalyze or CoinGlass) to detect data anomalies. Low priority in v1 given CoinGecko Pro's multi-exchange aggregation already reduces single-source risk.

**Agentic Logic (Strategic Notes)**

When an LLM or agent calls this tool, it should apply the following heuristics:

- **Check `status` first.** If `status: "degraded"`, only CVD is available. OI and funding hooks will be `null`. A CVD-only verdict has lower confidence — it should inform but not drive a trading decision without corroboration from other metrics.

- **Check `cvd.hook` for `"low_confidence"`.** If the thin-candle guard triggered, treat CVD as unavailable. This typically occurs during off-hours or very low-activity windows. Do not act on a `low_confidence` CVD.

- **`confidence` means signal completeness here, not data agreement.** Unlike `liquidity-pulse` and `stablecoin-power` where `confidence` reflects cross-source validator agreement, here `confidence: "high"` means all three signals were computed — not that they agree with each other. The signals can and do diverge; that divergence is itself informative.

- **Regime combination logic:**
  - `funding: "negative"` + `cvd: "aggressive_buy"` + `oi: "building"` → strong early bull signal. Shorts are being squeezed while buyers are aggressive and leverage is accumulating.
  - `funding: "overheated"` + `cvd: "aggressive_sell"` + `oi: "unwinding"` → long squeeze in progress. Exit or hedge.
  - `funding: "overheated"` + `cvd: "neutral"` + `oi: "building"` → dangerous coil. Crowded longs + accumulating leverage + no decisive taker activity. Breakout could flush longs violently.
  - `cvd: "aggressive_buy"` + `oi: "stable"` + `funding: "neutral"` → genuine spot demand without perp leverage. Structurally healthier than leverage-driven moves.

- **Cross-metric use:** Flow Tension is the kinetic complement to `stablecoin-power`. The high-signal combinations:
  - High stablecoin power + aggressive buy CVD + building OI + neutral/positive funding → ideal bull setup: dry powder available, spot buyers aggressive, leverage building, funding not yet crowded.
  - Low stablecoin power + building OI + overheated funding → most dangerous configuration: fuel depleted, leverage crowded, conditions for a violent unwind.
  - High stablecoin power + low-confidence CVD + stable OI + neutral funding → dry powder present but market dormant. Accumulation phase, not yet actionable.