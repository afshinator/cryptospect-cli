# Metric Expansion Analysis — 2026-05-18 (updated 2026-05-18)

Competitive landscape review and gap analysis of what metrics would complement the existing
6-metric cryptospect-cli suite. Three immediate candidates and six macro/finance candidates
are evaluated below.

**Decisions finalized 2026-05-18:**
- Altcoin Season Index dropped (invalidated per user).
- BTC/ETH Dominance = standalone metric (`dom`).
- China M2 = standalone metric (`cnm2`), sourced from DBnomics (free, keyless, confirmed working).
- DXY + Fed Funds Rate folded into US Macro Liquidity composite (`usml`).
- US Macro Liquidity = 3-signal composite (M2SL + DTWEXBGS + FEDFUNDS) from FRED (free key).

---

## Existing Suite (6 metrics)

| Metric | Function | Data source |
|--------|----------|-------------|
| Liquidity Pulse | Volume/mcap turnover conviction meter | CoinGecko /global |
| Stablecoin Power | Dry powder vs volatile market | CoinGecko /global, /stables; DefiLlama stablecoins |
| Flow Tension | CVD + OI change + funding rate (leverage kinetics) | Binance klines; CoinGecko /derivatives |
| Market Breadth | % coins green across 4 timeframes (participation proxy) | CoinGecko /coins/markets |
| Momentum Divergence | Tier rotation analysis (risk appetite gauge) | CoinGecko /coins/markets |
| Market Regime | 10-regime matrix (macro master state) | Aggregator: calls LP/SP/FT/MB compute |

**Current data sources:** CoinGecko (public/demo), Binance US (keyless spot), DefiLlama (keyless), CoinDesk (keyless).

---

## Part A: Three Immediate Candidates

### A1. Fear & Greed Index — `fgi`

- **Why:** Largest conceptual gap. We have no sentiment metric. Every major analyst uses
  Fear & Greed as a contrarian overlay. Natural complement to market-regime: mr tells
  you what kind of market you're in, fgi tells you how the crowd feels about it.
- **What:** 0-100 integer. 0 = Extreme Fear, 100 = Extreme Greed. Classification bands:
  0-25 Extreme Fear, 26-45 Fear, 46-55 Neutral, 56-75 Greed, 76-100 Extreme Greed.
- **Industry use:** alternative.me's index is the industry standard. CoinMarketCap,
  Binance, and most crypto dashboards surface it. Contrarian: extreme fear = buying
  opportunity, extreme greed = correction risk.
- **Data source:** `https://api.alternative.me/fng/` — free, keyless REST API.
  - Endpoint: GET `/fng/?limit=1`
  - Response: `{"value": "28", "value_classification": "Fear", "timestamp": "..."}`
  - Historical: `?limit=0` returns all available data (daily values back to 2018).
  - Rate limit: 10/min for free tier (documented).
- **Integration effort:** Trivial. 1 new API client (`internal/api/alternativeme/`).
  Single endpoint, single integer. Fits existing metric pattern exactly.
- **Classification:** 5 bands (extreme_fear, fear, neutral, greed, extreme_greed).
  Classification label and description.
- **Cache TTL:** 24 hours recommended (index updates once daily). Can be cached
  aggressively.
- **Risks/limitations:**
  - Sentiment is a contrarian indicator — best used alongside directional metrics
    (market-regime, flow-tension), not standalone.
  - Composition is a black box: methodology weights (volatility 25%, volume 25%,
    social 15%, surveys 15%, dominance 10%, trends 10%) are documented but custom.
  - Daily update only — not intraday.
- **Open questions for review:**
  - Should we compute a 7-day moving average alongside the raw value?
  - Should the classification include trend direction (fear decreasing vs increasing)?

### A2. DeFi Ecosystem Health — `defi-health`

- **Why:** We track stablecoin supply but nothing else about DeFi. TVL (Total Value
  Locked) is the industry's universal health indicator for on-chain capital. DEX
  volume and protocol fees tell you whether DeFi is expanding or contracting —
  orthogonal to our exchange-based metrics.
- **What:** Multi-signal metric:
  - Total DeFi TVL across all chains.
  - TVL 7d/30d percentage change (trend).
  - Dominant chain TVL ranking (top 5 chains with share %).
  - Aggregate DEX volume 24h.
  - Aggregate protocol fees 24h.
- **Industry use:** DefiLlama is the primary open-source aggregator. CoinGecko has a
  `/global/de-fi` endpoint. Token Metrics, Messari, Nansen all track DeFi TVL.
- **Data sources:**
  - DefiLlama (already integrated): `/charts` for aggregate TVL, `/protocols` for
    per-protocol, `/fees` for fee data. All keyless.
  - CoinGecko (already integrated): `/global/de-fi` for DeFi market cap/volume.
    On public/demo tier.
- **Integration effort:** Moderate. Extend existing DefiLlama client with new endpoints.
  New metric provider package. Multi-signal output similar to flow-tension structure.
- **Cache TTL:** 4 hours recommended (TVL changes slowly, fees more dynamic).
- **Risks/limitations:**
  - TVL is denominated in USD at current token prices — price drops reduce TVL even
    if deposits are steady. Need to note this in interpretation.
  - "Double-counting" in TVL: tokens deposited in L1 → bridged to L2 → deposited in
    protocol are counted at each layer. DefiLlama attempts to deduplicate but not
    perfectly.
  - Memecoin/hype driven DEX volume can spike without genuine ecosystem growth.
  - Fee data is per-protocol and may miss newer/smaller protocols.
- **Open questions for review:**
  - Single aggregate score vs. multi-signal output (like flow-tension)?
  - Should we include yield/APY data? Available from DefiLlama but noisy.
  - Classification: expanding/stable/contracting based on TVL trend?

### A3. Market Volatility — `vol`

- **Why:** We measure participation, rotation, conviction, and regime — but nothing
  about absolute risk level. Realized volatility tells agents whether to size
  positions up or down, and flags breakout/compression conditions. Token Metrics
  ranks volatility tools as essential.
- **What:** Three-signal volatility metric:
  - BTC realized volatility (annualized): `std(log_returns) × sqrt(365)`.
  - ETH realized volatility (annualized): same formula.
  - ETH/BTC volatility spread: `eth_vol / btc_vol` — a known trading signal.
    Spread > 1.5 = elevated ETH speculation (altcoin risk-on). Spread < 0.8 =
    ETH unusually calm (capital parked in BTC, alts dormant).
- **Industry use:** Every major platform includes volatility. CME Bitcoin Volatility
  Index. CoinGlass ATR. TradingView volatility indicators. The ETH/BTC vol spread
  is a standard crypto volatility desk metric.
- **Data sources:** Minimal new API footprint.
  - Binance BTC klines (existing `spot_cvd_btc_1h` endpoint, already fetched).
  - Binance ETH klines — new endpoint key: `binance.spot_cvd_eth_1h`, fetching
    `KlinesURL("ETHUSDT", "1h", 24)` — 24 candles of 1h bars. Same parser.
  - Pure compute from OHLC data: `std(log_returns) × sqrt(365)`.
- **Integration effort:** Trivial. New endpoint key (one line in fetcher switch).
  Pure compute function. No new API client — reuses existing Binance parser.
- **Classification:** Based on historical percentile bands:
  - Low: annualized < 30% (typical bear/ranging)
  - Normal: 30-60% (typical crypto baseline)
  - High: 60-100% (elevated, position sizing alert)
  - Extreme: > 100% (panic/euphoria, reduce exposure)
  - ETH/BTC vol spread: subdued (<0.8) / normal (0.8-1.5) / elevated (>1.5)
- **Output shape:**
  ```
  volatility:
    btc_realized_vol: 0.45         (45% annualized)
    eth_realized_vol: 0.72         (72% annualized)
    vol_spread: 1.6               (ETH 1.6× BTC vol)
    classification: elevated_eth
    summary: "BTC vol 45% (normal), ETH vol 72% (elevated), spread 1.6× — heightened altcoin speculation."
  ```
- **Cache TTL:** 1 hour (tactical metric like flow-tension). ETH klines cached
  independently from BTC klines.
- **Risks/limitations:**
  - Realized vol is backward-looking — doesn't predict volatility spikes.
  - Threshold calibration needs historical data. The bands above are rough heuristics.
  - ETH/BTC vol spread can be distorted by single-asset idiosyncratic events.
- **Design decision (2026-05-18):** BTC + ETH realized vol with vol spread. Realized
  vol (not ATR) — more standard for risk assessment.

---

## Part B: Macro/Finance Metrics

### B1. BTC & ETH Dominance — `dom`

- **Why:** BTC dominance (BTC.D) is already in market-regime but only as one axis of
  the matrix. ETH dominance (ETH.D) and the ETH/BTC ratio are the other half of the
  rotation story — BTC.D rising + ETH.D falling = safety retreat; BTC.D falling +
  ETH.D rising = risk-on rotation into the largest alt before trickle-down to
  mid/small caps. Both dominance values are universally tracked by every major
  crypto analytics platform (CMC, CoinGecko, TradingView, Glassnode).
- **What:** Two dominance values in one metric:
  - BTC Dominance: `BTC.mcap / total_mcap` (already computed in market-regime).
  - ETH Dominance: `ETH.mcap / total_mcap`.
  - ETH/BTC Ratio: `ETH.mcap / BTC.mcap` (or `ETH.price / BTC.price`).
  - Dominance trend: rising/falling/neutral for each, based on delta vs prior
    cached value (same pattern as MR's dominance delta).
- **Data sources:** All from existing CoinGecko endpoints.
  - `/global` — `market_cap_percentage` map already contains `btc` and `eth`.
    No new API calls needed.
- **Integration effort:** Low. Pure compute from existing data. No new API client.
  New state cache for storing prior dominance values to compute delta.
- **Classification:** Per-asset dominance trend:
  - BTC: rising (>+0.5pp) / neutral / falling (<-0.5pp).
  - ETH: rising (>+0.3pp) / neutral / falling (<-0.3pp) — tighter band since
    ETH.D is typically 15-20% vs BTC.D's 45-65%.
- **Summary examples:**
  - *"BTC dominance 52.3% (falling, -0.8pp), ETH dominance 18.1% (rising, +0.6pp) — capital rotating out of BTC into ETH."*
  - *"BTC dominance 58.2% (rising, +1.5pp), ETH dominance 15.4% (falling, -1.1pp) — safety retreat, BTC outperforming."*
- **Design decision (2026-05-18):** Standalone metric. Not folded into market-regime.
  Altcoin Season Index dropped — user considers concept invalidated.
- **Risks/limitations:**
  - ETH dominance moving doesn't guarantee alt rotation — ETH can decouple from the
    rest of the alt market (e.g., ETF-driven flows favoring ETH over other alts).
  - Dominance deltas require a prior cached snapshot (cold start on first run,
    same pattern as market-regime's dominance delta).
- **Cache TTL:** 4 hours (same as market-regime — structural metric).

### B2. US Macro Liquidity — `usml`

- **Why:** The strongest documented macro driver of crypto prices. CF Benchmarks'
  research shows 0.71-0.90 R-squared between Bitcoin and M2 during 2022-2024.
  Bitcoin's price has historically reverted toward its M2-implied fair value across
  every major divergence. When M2 expands and the dollar weakens with dovish Fed
  policy, risk assets including crypto catch a strong tailwind. This combines three
  reinforcing signals into a single composite metric.
- **What:** Three-signal composite metric (similar to flow-tension structure):
  - US M2 Money Supply YoY % change (M2SL from FRED).
  - US Dollar strength trend (DTWEXBGS broad trade-weighted dollar index from FRED).
  - Federal Funds Rate stance (FEDFUNDS from FRED — current rate, last change direction).
- **Data sources:**
  - **FRED (St. Louis Fed):** `https://api.stlouisfed.org/fred/series/observations`
    - Series: M2SL (US M2), DTWEXBGS (Broad Dollar Index), FEDFUNDS (Fed Funds Rate).
    - Requires free API key signup (`CRYPTOSPECT_FRED_KEY` — already planned per
      knowledge file). Single client, three series.
    - Rate limit: 120 requests/minute.
    - Data lag: M2 weekly (~1 week), DXY daily (real-time), Fed Funds daily.
- **Signals and classification:**

  | Signal | Data | Classification |
  |--------|------|---------------|
  | M2 Growth | US M2 YoY% change | expanding (>5%) / stable (0-5%) / contracting (<0%) |
  | Dollar Strength | DXY 30-day trend | strengthening (>+1%) / neutral / weakening (<-1%) |
  | Fed Stance | Fed Funds Rate + last change | hawkish (hiking) / neutral (holding) / dovish (cutting) |

- **Summary examples:**
  - *"Expanding liquidity + weakening dollar + dovish Fed — strong risk-asset tailwind."*
  - *"Contracting M2 + strengthening dollar + hawkish Fed — liquidity headwind, defensive posture."*
- **Integration effort:** Medium. New FRED API client. Three series pulled from one
  provider, combined into one composite metric. New state cache for storing prior
  M2/DXY/FedFunds values to compute deltas.
- **Risks/limitations:**
  - M2 data is lagged (weekly publication, ~1 week delay). Cannot be used for
    intraday/tactical decisions.
  - Correlation is structural but not mechanical — M2 expansion doesn't guarantee
    crypto inflow (2025-2026 divergence documented by CF Benchmarks).
  - DXY-crypto correlation is unstable (ranged from -0.8 to +0.4 over 2020-2026).
  - Requires FRED API key (free signup, but not keyless).
- **Open questions for review:**
  - Should we compute an "M2 fair value gap" (BTC actual vs M2-implied fair value)?
  - DXY trend window: 30-day or 90-day moving average?

### B2b. China M2 Money Supply — `cnm2`

- **Why:** CF Benchmarks research shows 0.71-0.90 R² between Bitcoin and global M2.
  Multiple analysts (Wedson, TradingView, Binance Research) have documented a
  specifically strong correlation between **China's** M2 expansion and BTC price
  cycles. China's M2 is now ~2.2× larger than US M2 (~$47T vs ~$21T). Historical
  pattern: when China M2 overtook US M2 (circa 2009), Bitcoin was born into a new
  global liquidity regime. Whenever China M2 growth accelerates, BTC has historically
  followed with a lag measured in months.
- **What:** China M2 money supply level (CNY) and YoY % change. Optionally,
  China M2 / US M2 ratio as a comparative liquidity signal.
- **Data source (confirmed 2026-05-18):**
  - **DBnomics:** `https://api.db.nomics.world/v22/series/NBS/M_A0D01/A0D0101`
    - Provider: National Bureau of Statistics of China (via DBnomics).
    - Series: "Money and Quasi-Money (M2) Supply, period-end."
    - Free, keyless, structured JSON REST API.
    - Latest confirmed data: February 2026 — 3,492,200 (100M yuan) = 34.92T CNY.
    - Frequency: Monthly. Values in 100 million yuan (divide by 10 for CNY Billion).
  - **Trading Economics** (HTML scrape fallback):
    - `https://tradingeconomics.com/china/money-supply-m2`
    - Previously used as "flash anchor" in cryptoSpect project. DBnomics is now fresher.
    - Not needed for v1 — DBnomics is sufficient.
- **Integration effort:** Low. New DBnomics API client — one endpoint, structured
  JSON, no key. Simpler than FRED. New state cache for storing prior M2 values
  to compute delta and YoY change.
- **Classification:** Based on China M2 YoY growth rate:
  - Expanding: YoY > 8% — strong liquidity tailwind (typical for China).
  - Normal: YoY 4-8% — steady expansion.
  - Slowing: YoY < 4% — tightening (rare for China, last seen 2014-2015).
- **Cache TTL:** 30 days (data updates monthly).
- **Risks/limitations:**
  - Data is monthly, not real-time. Not actionable for intraday decisions.
  - Correlation is directional but not mechanical — China M2 expansion doesn't
    guarantee immediate BTC inflow. The transmission channel is indirect (Chinese
    capital controls → offshore capital → crypto).
  - Currency denomination: China M2 is in CNY. For comparison with US M2, need
    USD/CNY exchange rate conversion (or compare YoY percentages directly).
- **Design decision (2026-05-18):** Standalone metric, not folded into US Macro
  Liquidity. China M2 and US M2 are structurally different — China's M2 is
  ~2.2× larger and has different transmission dynamics. Keeping them separate
  lets agents compare independently.

### B3. US Dollar Index (DXY) — Folded into B2

> **Note (2026-05-18):** DXY is now a sub-signal within the US Macro Liquidity (`usml`)
> composite metric (B2). It is not implemented as a standalone metric. The rationale
> below is retained for reference.

- **Why:** Historically negative correlation with crypto, though the relationship
  has weakened post-ETF (2024-2026 showed periods of positive correlation). DXY
  captures the dollar's strength against a basket of major currencies — a rising
  dollar typically means tighter global liquidity conditions, which is a headwind
  for risk assets including crypto. It's the most cited macro indicator in crypto
  trading discourse alongside Fed rate decisions.
- **What:** DXY index value and direction (rising/falling/neutral trend).
- **Data sources:**
  - FRED series DTWEXBGS (Broad Trade-Weighted Dollar Index) — same key requirement.
  - Alpha Vantage: `FX_MONTHLY` or `CURRENCY_EXCHANGE_RATE` for USD/EUR as proxy.
  - No keyless real-time DXY source found. TradingView has it but no public API.
  - Some forex APIs offer limited free tiers (e.g., exchangerate-api.com).
- **Integration effort:** Medium. Depends on data source availability. If FRED is
  used, same client as M2.
- **Classification:** Directional (> +1% = strengthening, < -1% = weakening,
  else neutral). Trend based on 30-day moving average.
- **Risks/limitations:**
  - DXY-crypto correlation is unstable (ranged from -0.8 to +0.4 over 2020-2026).
    The correlation regime itself is what matters — a positive correlation period
    means crypto is trading like a risk-on tech asset; negative means it's trading
    as a dollar hedge.
  - DXY is a relative measure — it can rise because USD is strong OR because
    EUR/JPY/GBP are weak. This ambiguity matters for interpretation.
  - Keyless data access is the primary blocker.
- **Open questions for review:**
  - Is DXY worth adding given the unstable correlation? The metric's value may be
    in detecting correlation regime shifts, not the DXY level itself.
  - Should we track DXY trend + its correlation with BTC as a combined signal?

### B4. Federal Funds Rate — Folded into B2

> **Note (2026-05-18):** Fed Funds Rate is now a sub-signal within the US Macro
> Liquidity (`usml`) composite metric (B2). It is not implemented as a standalone
> metric. The rationale below is retained for reference.

- **Why:** The single most important macro variable for all risk assets. Rate cuts
  = liquidity expansion = tailwind. Rate hikes = contraction = headwind. The current
  rate, the direction of change, and market-implied future rate expectations all matter.
- **What:** Current Federal Funds Effective Rate. Direction (last change magnitude
  and date). Optionally, CME FedWatch probability of next meeting change
  (requires scraping — no API).
- **Data sources:**
  - FRED series FEDFUNDS — daily, same key requirement.
  - Alternative: scrape CME FedWatch tool (no API, HTML parsing required).
- **Integration effort:** Medium with FRED key. High without (scraping).
- **Risks/limitations:**
  - Rate changes are infrequent (6-8 meetings/year). Updates are event-driven.
  - Market-implied expectations (FedWatch) are more actionable than the current
    rate, but require scraping.

---

## Feasibility Summary

| Metric | API needed | Keyless? | Integration effort | Signal frequency |
|--------|-----------|----------|-------------------|-----------------|
| Fear & Greed (fgi) | alternative.me | ✅ Yes | Trivial | Daily |
| DeFi Health (defi) | DefiLlama + CoinGecko (existing) | ✅ Yes | Moderate | 4h |
| Volatility (vol) | None (existing data) | ✅ Yes | Trivial | 1h |
| BTC/ETH Dominance (dom) | CoinGecko (existing) | ✅ Yes | Low | 4h |
| China M2 (cnm2) | DBnomics | ✅ Yes | Low | Monthly |
| US Macro Liquidity (usml) | FRED (3 series) | ⚠️ Free key | Medium | Daily/Weekly |

---

## Recommended Implementation Order

1. **Phase 1 (keyless, trivial-to-low, high impact):**
   - Fear & Greed + Volatility + BTC/ETH Dominance + China M2
   - Four metrics, ~1 session each. No new API providers for 2 of 4. One new
     keyless provider each for fgi (alternative.me) and cnm2 (DBnomics).
   - Fills sentiment, risk, dominance, and macro (China M2) gaps.
   - Total: 3-4 sessions.

2. **Phase 2 (keyless, moderate, extends existing):**
   - DeFi Health
   - Extends DefiLlama client with TVL/fees endpoints. Multi-signal metric.
   - Total: 1-2 sessions.

3. **Phase 3 (macro, requires FRED key):**
   - US Macro Liquidity (M2SL + DTWEXBGS + FEDFUNDS)
   - Single FRED client, three series, one composite metric output.
   - Requires user to sign up for free FRED API key (`CRYPTOSPECT_FRED_KEY`).
   - Total: 2-3 sessions.

---

## Open Decisions

| # | Question | Answer (2026-05-18) |
|---|----------|---------------------|
| 1 | BTC/ETH Dominance: standalone or folded into market-regime? | **Standalone** — separate `dom` metric |
| 2 | China M2: standalone or folded into US Macro Liquidity? | **Standalone** — different transmission dynamics |
| 3 | Fear & Greed: raw daily value only, or also 7d MA + trend direction? | **Enriched** — raw value + 7d MA + trend direction |
| 4 | Volatility: BTC-only or top-N aggregate? Realized vol or ATR? | **BTC + ETH realized vol** — annualized, with ETH/BTC vol spread as third signal. Realized vol (not ATR). |
| 5 | FRED API key: acceptable dependency? | **Accept** — free signup, gated to Phase 3 |
| 6 | DeFi Health: single composite score or multi-signal (like flow-tension)? | **Multi-signal** — TVL, DEX volume, fees are distinct signals |
