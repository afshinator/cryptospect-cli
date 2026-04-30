# flow-tension

**version:** `v1.0.0`
**Alias:** `ft`
**Endpoints:** main: `binance.spot_cvd_btc_1h`, `coingecko.derivatives`

## Overview

Measures the kinetic energy of the market — how aggressively traders are using leverage and moving assets onto exchanges to trade. While `stablecoin-power` shows potential energy (dry powder available), `flow-tension` shows whether that powder is actually igniting. It tracks three distinct signals: taker aggression in spot markets (CVD), leverage accumulation or unwinding (Open Interest), and the cost of holding leveraged positions (Funding Rate). Together, these reveal whether price moves are conviction-driven or fragile, and whether the current regime favors longs or shorts.

All three signals are sourced from keyless public APIs — no API key required. This is a deliberate design improvement over the original implementation which required CoinGecko Pro for OI and funding data.

## Formula (or how to compute)

This is a multi-signal metric. There is no single composite score — three signals are computed independently and reported together. The `summary` field synthesizes the combination into a verdict string.

**Signal 1 — Exchange Net Flow (CVD proxy)**
```
cvd_ratio = (taker_buy_volume - taker_sell_volume) / (taker_buy_volume + taker_sell_volume)
```
Result is a normalized ratio in [-1, 1]. Positive = net buyer aggression; negative = net seller aggression.
Source: Binance-US BTC/USDT 1h spot klines. Keyless.

**Signal 2 — Open Interest (current + 24h change)**
```
oi_current = sum of open_interest across all BTC perpetual entries in /derivatives response
oi_change_pct = (oi_current - oi_cached) / oi_cached
```
Current OI is reported directly from CoinGecko. 24h change is computed against a cached value from a prior fetch (persisted via the file cache). Positive = leverage building; negative = leverage unwinding.
Source: CoinGecko public `/derivatives` endpoint (keyless). OI is aggregated across all exchanges (179+ BTC perpetual entries) for a global picture.

**Signal 3 — Funding Rate (current 8h cycle)**
```
funding_rate = raw funding rate for BTC perpetual (decimal per 8h cycle)
```
Reported as a decimal (e.g., 0.0003 = 0.03% per 8h). Sign indicates direction: positive = longs paying shorts; negative = shorts paying longs.
Source: CoinGecko public `/derivatives` endpoint (keyless). Uses Binance Futures BTC perpetual funding rate as the primary signal (most liquid exchange).

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

**OI Hook**

| Label | Threshold |
|-------|-----------|
| `building` | oi_change_pct > +5% |
| `stable` | -5% ≤ oi_change_pct ≤ +5% |
| `unwinding` | oi_change_pct < -5% |
| *no cache history* | `stable` (default until 24h comparison available) |

**Funding Hook**

| Label | Threshold |
|-------|-----------|
| `overheated` | funding_rate > +0.003 (> +0.30% per 8h) |
| `positive` | +0.0003 ≤ funding_rate ≤ +0.003 |
| `neutral` | -0.0003 < funding_rate < +0.0003 |
| `negative` | funding_rate ≤ -0.0003 |

**Threshold rationale:** Funding rate thresholds are calibrated against observed BTC perpetual data (historical range: approximately -0.29% to +0.85% per 8h). The ±0.03% neutral band captures the typical noise floor; +0.30% marks the upper tail where funding cost becomes punishing for longs and historical correlation with short-term tops strengthens. OI ±5% captures intraday noise while flagging genuine leverage accumulation. CVD ±10% of total volume separates decisive taker imbalance from balanced churn.

## Data Source(s)

Data source selection is guided by two principles: **(1)** all signals must be available from keyless public APIs, and **(2)** where possible, avoid introducing new API clients.

| Role | Source | Endpoint | Key | Fields Used |
|------|--------|----------|-----|-------------|
| CVD | Binance-US (spot) | `/api/v3/klines` (BTC/USDT, 1h) | ❌ Keyless | `takerBuyBaseAssetVolume`, `volume`, `numberOfTrades` |
| OI | CoinGecko (public) | `/derivatives` | ❌ Keyless | `open_interest` (aggregated across all BTC perpetual entries) |
| Funding Rate | CoinGecko (public) | `/derivatives` | ❌ Keyless | `funding_rate` from Binance Futures BTC perpetual entry |

**Why the same API for OI and Funding?** CoinGecko's `/derivatives` endpoint returns both `open_interest` and `funding_rate` per exchange per ticker in a single response. This is a public, keyless endpoint — no CoinGecko Pro subscription needed. It covers 179+ BTC perpetual entries across all major exchanges. OI is summed across all entries for a global aggregate picture; funding rate is taken from Binance Futures (the most liquid exchange) as the canonical signal.

**Why CoinGecko public `/derivatives` over OKX or Bybit APIs for OI/Funding:**
- Already have the CoinGecko client integrated — no new API client needed
- Broadcasting endpoint: one call retrieves OI and funding for all 179+ exchanges simultaneously
- OI aggregation across all exchanges gives a genuinely global picture, better than any single-exchange source
- Binance Futures (`fapi.binance.com`) was evaluated but is geo-restricted from some regions

**Why Binance-US for CVD (rather than using CoinGecko):**
CVD requires raw trade-level kline data (taker buy vs. sell volume per candle). CoinGecko does not expose this granularity — it provides aggregate volume, not taker-side breakdown. Binance-US klines are the most accessible source of taker-disaggregated spot volume without an API key, making them the correct source for the CVD signal.

## Cross-Source Verification

No cross-source verification in v1.

Each of the three signals originates from a genuinely distinct data type (spot kline taker volume, aggregate derivatives OI, perpetual funding rate). There is no second source that exposes all three with compatible methodology and scope. Comparing Binance-US CVD to another exchange's CVD would measure exchange-specific differences rather than validate data quality.

However, the OI signal has a natural self-validator: the `/derivatives` response returns per-exchange OI values. Significant divergence between Binance Futures OI and the sum of all other exchanges' OI could indicate exchange-specific anomalies. This is noted as a future enhancement — for v1, the aggregated OI sum is used directly without per-exchange validation.

Cross-source validation is noted as a future enhancement; see Future Enhancements.

## CLI Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
No per-metric CLI flags in v1. `--instrument btc` is the only supported scope and is implicit — no flag is exposed. The classification thresholds for funding rate and OI change are calibrated specifically against BTC perpetual historical data; multi-asset support requires per-asset threshold calibration and is queued as a future enhancement.

## Output Schema

```json
{
    "metric":  "flow-tension",
    "version": "v1.0.0",
    "status":  "string",  // "ok" / "degraded" / "unavailable"
                          // "degraded" = transient API failure (OI/funding fetch failed); CVD still reported
                          // "unavailable" = Binance-US fetch failed; no signals available

    "data": {
        "signals": {
            "cvd": {
                "ratio":  "float64",  // normalized [-1, 1]; positive = net buy aggression
                "hook":   "string"    // "aggressive_buy" / "neutral" / "aggressive_sell" / "low_confidence"
            },
            "open_interest": {
                "current_usd":  "float64",        // current aggregated OI across all exchanges
                "change_pct_24h": "float64?",      // percentage change vs cached value; omitted on first run
                "hook":         "string"           // "building" / "stable" / "unwinding"
                                                     // defaults to "stable" when no cache history
            },
            "funding_rate": {
                "rate":  "float64",  // decimal per 8h cycle (Binance Futures)
                "hook":  "string"    // "overheated" / "positive" / "neutral" / "negative"
            }
        },
        "summary": "string"  // NL verdict combining all three signals, e.g.:
                             // "BTC perp: aggressive buying (CVD +0.14), OI building (+7.2%), funding positive (0.045%/8h) — Building Tension with bullish lean."
    },

    "meta": {
        // Omitted when --detail basic.
        // Present when --detail extended or full:
        "cache_hit":         "bool",
        "ttl_remaining_sec": "int",        // 3600s (1h) — tactical staleness budget
        "primary_sources":   ["binance_us", "coingecko"],
        "instrument":        "string",     // "btc" (always in v1)
        "cvd_sample_trades": "int",        // number of trades in CVD sample; < 10 triggers low_confidence hook
        "oi_exchanges_count": "int",        // number of exchanges aggregated for OI
        "confidence":        "string"      // "high" (all signals present) / "degraded" (CVD only, OI/funding transient failure)
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

**Note on `confidence` field:** For this metric, `confidence` reflects signal completeness rather than cross-source discrepancy (no cross-source validation exists in v1). `"high"` means all three signals were computed. `"degraded"` means only CVD was available due to a transient CoinGecko API failure. This is a semantic divergence from the other metrics where `confidence` reflects validator agreement — documented here to prevent agent misinterpretation.

**Enhancements** (conditional — present when specific conditions are met):

| Field | Condition | Description |
|-------|-----------|-------------|
| `cvd.hook: "low_confidence"` | `cvd_sample_trades < 10` | Thin-candle guard: fewer than 10 trades in the sample window means CVD ratio is statistically unreliable. Hook is overridden to `low_confidence` regardless of ratio value. |
| `open_interest.change_pct_24h` omitted | First run (no cache history) | OI 24h change requires a cached prior value. On first run, current OI is reported but no change percentage is available. Hook defaults to `"stable"`. |
| `delta_24h` on CVD (not yet implemented) | Prior cache data exists | Percentage change in CVD ratio from 24h prior. Planned for v1.1 when historical cache is available. |

## Usage

```bash
# All signals available — no API key needed
cryptospect-cli flow-tension

# With alias
cryptospect-cli ft

# Full detail (thresholds, description, exchange count)
cryptospect-cli flow-tension --detail full

# Extended detail
cryptospect-cli flow-tension --detail extended
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

**OI computation:**
CoinGecko's public `/derivatives` endpoint returns per-exchange OI for each BTC perpetual entry. OI is aggregated by summing `open_interest` across all entries in the response (179+ exchanges). The 24h change is computed against a cached value from a prior successful fetch, stored via the existing file cache infrastructure. On first run (no cache history), current OI is reported but no change percentage is available — the OI hook defaults to `"stable"` until a 24h comparison window exists.

**Funding rate:**
CoinGecko's public `/derivatives` endpoint returns `funding_rate` per exchange per ticker. The metric uses Binance Futures BTC perpetual funding rate as the canonical signal (most liquid exchange). No transformation is applied — the raw decimal value is reported alongside its hook classification.

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

**Transient degraded mode:**
If CoinGecko's `/derivatives` endpoint fails transiently (e.g., rate limit, timeout), CVD is still reported from Binance with OI and funding set to their raw values but the metric status set to `"degraded"`. This is a runtime-operational concern, not a permanent mode — retry will restore full signals.

### Other details

**No per-metric CLI flags in v1.** `--instrument btc` is the only supported scope and is implicit. The classification thresholds are calibrated against BTC perpetual data only; multi-asset support requires per-asset re-calibration and is queued as a future enhancement.

**Enhancements:**
- `cvd.hook: "low_confidence"` is a data-quality guard, not a signal. Agents should treat it as a null equivalent for CVD and rely only on OI and funding if those are available.
- `delta_24h` on CVD is planned for v1.1. Percentage change form preferred over absolute difference — a CVD shift from -0.05 to +0.08 is more meaningful as a directional regime change than as an absolute +0.13 difference.

**Cross-Source Verification:**
No cross-source verification in v1. All three signals are sourced from two APIs (Binance-US for CVD, CoinGecko for OI/Funding). The per-exchange breakdown in `/derivatives` provides an internal consistency check (e.g., Binance Futures OI vs aggregate OI) but is not validated against an independent source in v1.

**Implementation Compromises:**
- **CVD is a proxy, not true net flow.** Binance-US spot klines measure taker aggression within a single exchange. This is not the same as global exchange net inflow (coins moving from cold wallets to exchange hot wallets). A full exchange net flow signal would require on-chain data (e.g., Glassnode). The CVD proxy is a reasonable substitute at the tactical timeframe but should not be described to agents as "coins moving onto exchanges" — it is taker buy/sell imbalance within existing on-exchange liquidity.
- **OI is aggregated across all exchanges, not per-exchange.** The `/derivatives` endpoint returns OI per exchange; the metric sums them for a global picture. This gives a broader view than any single exchange but loses exchange-specific granularity (e.g., a spike in Binance OI alone could be masked by flat OI elsewhere).
- **OI 24h change has a cold-start delay.** On first run, no cached value exists for 24h comparison. The OI hook defaults to `"stable"` and change is omitted from output until a second successful fetch confirms the window. Agents should not rely on OI change on initial calls.
- **Funding rate is current 8h cycle only.** The metric does not track funding rate trend (whether it is rising or falling toward the overheated zone). A funding rate of +0.25% that was +0.05% yesterday is a very different situation from one that was +0.28% yesterday. Trend tracking is a future enhancement.
- **No stablecoin-pair filtering for CVD.** Binance-US BTC/USDT klines include USDT as the quote currency. USDT-specific events (e.g., a USDT de-peg scare) could influence CVD without reflecting genuine BTC sentiment. This is an accepted approximation for v1.

**Future Enhancements:**
- **Multi-asset support (`--instrument eth`, etc.):** Requires per-asset threshold re-calibration. ETH and SOL perpetuals have different typical funding ranges and OI dynamics than BTC. Shipping calibrated thresholds per asset is the prerequisite.
- **Funding rate trend (`funding_trend_1h`):** Direction of funding rate change over the past 1–4 hours. Rising-toward-overheated is a different signal from falling-from-overheated. Percentage change or slope preferred over absolute value for trend.
- **`delta_24h` on CVD ratio:** Requires historical cache. Percentage change form preferred.
- **True exchange net flow:** Replace CVD proxy with on-chain exchange inflow/outflow data (Glassnode or equivalent). Materially more accurate for the "assets moving to exchanges" interpretation but requires an additional API and likely an API key.
- **Cross-source OI validation:** Compare CoinGecko aggregate OI against a second source (e.g., OKX public API or CoinGlass) to detect data anomalies.
- **Per-exchange OI breakdown in meta:** Report individual exchange OI values (Binance, Bybit, OKX, etc.) in full detail for debugging and advanced analysis.

**Agentic Logic (Strategic Notes)**

When an LLM or agent calls this tool, it should apply the following heuristics:

- **Check `status` first.** All three signals are always available under normal conditions (no API key needed). If `status: "degraded"`, a transient CoinGecko failure means OI/funding may be stale — check `cvd_sample_trades` to confirm CVD reliability.

- **Check `cvd.hook` for `"low_confidence"`.** If the thin-candle guard triggered, treat CVD as unavailable. This typically occurs during off-hours or very low-activity windows. Do not act on a `low_confidence` CVD — OI and funding are still valid but cannot triangulate with CVD.

- **Check `open_interest.change_pct_24h` for omission.** On first run, no 24h change is available. The OI hook defaults to `"stable"` which may understate the true state. Make a second call after 1h+ for a populated OI change.

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