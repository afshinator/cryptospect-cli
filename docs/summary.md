update 5/24/26

cryptospect-cli — Summary
A Go CLI tool (single static binary, CGO_ENABLED=0) that fetches live crypto data, computes market regime metrics, and outputs agent-optimized JSON. Built with Cobra+Viper, keyless public APIs (CoinGecko, Binance US, DefiLlama, DBnomics, alternative.me). Every invocation writes exactly one JSON object to stdout — an CLIResponse envelope with a results array of MetricResult entries. Stderr is diagnostics only. Agents are meant to call individual metric commands and compose their own analysis.
All 10 metrics are production-implemented.
---
Implemented Metrics — Definitions & Implementation Fidelity
1. liquidity-pulse (lp) — v1.0.0
What it measures: Ratio of 24h trading volume to total market cap — a "conviction meter." High = strong trading relative to size; Low = stagnation or accumulation.
Formula: volume_to_mcap_ratio = total_volume_usd / total_market_cap_usd
Classification: High (>0.15), Normal (0.05–0.15), Low (<0.05)
Data sources: CoinGecko /global (primary), Binance US BTC spot CVD (validator). Cross-source validation with 20% discrepancy threshold. Confidence: high/medium/low based on validator agreement.
Implementation fidelity: HIGH. All classification thresholds, cross-source validation, discrepancy detection, and confidence gating implemented per spec. delta_24h is the only deferred field (requires historical cache).
2. stablecoin-power (sp) — v1.0.0
What it measures: "Ecosystem Fuel Gauge" — ratio of aggregate stablecoin market cap to volatile-asset market cap. High = dry powder available; Low = depleted. supply_trend_7d disambiguates two distinct Low scenarios (overextended vs. capital flight).
Formula: stable_power_ratio = stable_mcap_usd / (total_mcap_usd - stable_mcap_usd)
Classification: High (>0.15), Normal (0.07–0.15), Low (<0.07, split by supply_trend_7d)
Data sources: CoinGecko /global + /coins/markets?category=stablecoins (primary), DefiLlama stablecoin supply (validator). 15% discrepancy threshold. --top N flag (min 8, default 8).
Implementation fidelity: HIGH. Full multi-source verification, top clamping, supply_trend_7d disambiguation, per-stablecoin breakdown at --detail full, all implemented per spec. delta_24h deferred.
3. flow-tension (ft) — v1.0.0
What it measures: Kinetic energy — taker aggression (CVD), leverage accumulation/unwinding (OI 24h change), and cost of leverage (funding rate). Three co-equal signals, no composite score. Generates named verdicts like "Early Bull Phase," "Building Tension," "Long Squeeze Risk."
Formula (3 signals):
- cvd_ratio = (taker_buy - taker_sell) / (taker_buy + taker_sell) → -1, 1
- oi_change_pct = (oi_current - oi_cached) / oi_cached
- funding_rate (raw decimal per 8h cycle)
Classification: Per-signal hooks (e.g., CVD: aggressive_buy/neutral/aggressive_sell/low_confidence; OI: building/stable/unwinding; Funding: overheated/positive/neutral/negative). No single composite label.
Data sources: Binance US spot klines (CVD, keyless), CoinGecko public /derivatives (OI + funding, keyless). No cross-source validation in v1. OI 24h change via file cache (cold-start: stable default).
Implementation fidelity: HIGH. All three signals, thin-candle guard (<10 trades → low_confidence), OI cache-based 24h change, seven named verdict combinations, all hooks and thresholds implemented. delta_24h on CVD deferred.
4. market-breadth (mb) — v1.0.0
What it measures: Market participation — recency-biased weighted average of "% of top-N coins green" across 1h/24h/7d/30d. Ghost Rally divergence detector flags BTC >+2% while breadth <40%. Binance directional consensus validator.
Formula: breadth_score = 0.10 × green_1h + 0.30 × green_24h + 0.40 × green_7d + 0.20 × green_30d
Null-exclusion per timeframe (denominator is valid coins for that window, not len(coins)). Weight redistribution when per-timeframe TotalCount < 50. Global floor: <50 valid coins → status "degraded".
Classification: Broad (>0.60), Mixed (0.40–0.60), Narrow (<0.40). Overlay: divergence_detected (Ghost Rally).
Data sources: CoinGecko /coins/markets (primary, all timeframes in one call), Binance US BTC klines (validator — directional consensus, not quantitative). --top N flag (min 50, max 250).
Implementation fidelity: HIGH. Multi-timeframe null exclusion, weight redistribution, Ghost Rally divergence, directional consensus validator with staleness (>90min) and zero-Close guards, per-timeframe counts exposed in meta. 92.6% coverage. delta_24h deferred.
5. momentum-divergence (md) — v1.1.0
What it measures: Risk appetite gauge — segments top 200 coins into Large (1–10), Mid (11–50), Small (51–200) tiers by market_cap_rank. Market-cap weighted average 24h returns per tier. Positive mid-vs-large spread = capital rotating down risk curve (Risk-On). Negative = concentration into mega-caps (Top-Heavy or Flight-to-Safety). tail_extension boolean flags long-tail outperformance.
Formula:
tier_avg = sum(return_i × mcap_i) / sum(mcap_i)  for valid coins in tier
mid_vs_large = mid_avg - large_avg  (nil-safe via *float64)
Classification: risk_on (mid_vs_large > +5.0pp AND mid_avg > 1.0), top_heavy (mid_vs_large < -3.0pp AND large_avg > +0.5), flight_to_safety (mid_vs_large < -3.0pp AND large_avg < -0.5), neutral (all else, including ±0.5% dead band). tail_extension decoupled from label.
Data sources: CoinGecko /coins/markets (single call, same endpoint key as mb — cache-sharing). No cross-source validator (BTC CVD was explicitly rejected as adversarial). --segments flag (e.g., 10,50,200).
Implementation fidelity: HIGH. Market-cap weighted means (v1.1), nil-safe spreads, concentration dead band, tail extension decoupled from label, cache starvation guard (CoinCount < 250 → low confidence), configurable --segments. tier_detail at full detail. delta_24h and volume conviction deferred to v1.2.
6. china-m2 (cnm2) — v1.0.0
What it measures: China M2 money supply YoY % change — a macro liquidity indicator with historically strong correlation to Bitcoin price cycles. China's M2 (~$47T) is ~2.2× larger than US M2.
Formula: yoy_change_pct = ((current_value - value_12_months_ago) / value_12_months_ago) × 100
Classification: expanding (YoY > 8%), normal (4%–8%), slowing (YoY < 4%)
Data sources: DBnomics (National Bureau of Statistics of China series NBS/M_A0D01/A0D0101). Monthly frequency. Cold start: ≥13 months of history required for YoY; first run returns level only with confidence "medium."
Implementation fidelity: HIGH. YoY computation, cold-start handling, 30-day cache TTL, data_lag_days in meta. Monthly frequency correctly reflected in output.
7. dominance (dom) — v1.0.0
What it measures: BTC and ETH market cap dominance — percentage of total crypto market cap held by each. Trend detection via delta from prior cached snapshot.
Formula: btc_dominance = BTC.mcap / total_mcap; eth_dominance = ETH.mcap / total_mcap. Dominance trend via dead-banded delta: BTC ±0.5pp, ETH ±0.3pp.
Classification: btc_rising, eth_rising, neutral, capital_contracting (both falling). Cold start: trends default to "neutral" with cold_start: true.
Data sources: CoinGecko /global market_cap_percentage map (same endpoint as lp/sp). State cache key for prior-snapshot delta computation (48h TTL, 24h max snapshot age).
Implementation fidelity: HIGH. BTC and ETH dominance values, dead-banded trend classification, priority label selection, cold-start handling, eth_btc_ratio derived field. delta_24h deferred.
8. fear-greed-index (fgi) — v1.0.0
What it measures: Crypto Fear & Greed Index (0–100) from alternative.me — the industry-standard sentiment oscillator. Contrarian overlay to directional metrics.
Formula: Raw integer 0–100 value. 7-day MA and trend direction when ≥7 days of history available. Trend: improving/deteriorating/stable with ±2 dead band.
Classification: extreme_fear (0–25), fear (26–45), neutral (46–55), greed (56–75), extreme_greed (76–100). Classification computed from numeric value, not parsed from API string.
Data sources: alternative.me /fng/ endpoint. 7-day history fetched for MA + trend. 24h TTL (index updates once daily). Rate limit: 10/min.
Implementation fidelity: HIGH. 5-band classification, 7-day MA, trend direction with dead band, cold-start handling (<7 days history drops confidence to "medium").
9. volatility (vol) — v1.0.0
What it measures: Annualized realized volatility for BTC and ETH from hourly OHLC data, plus ETH/BTC volatility spread — a standard crypto volatility desk metric.
Formula: log_returns[i] = ln(close[i] / close[i-1]) for 24 hourly candles (23 returns). realized_vol = std(log_returns) × sqrt(8760). vol_spread = eth_vol / btc_vol.
Classification: subdued (spread < 0.8), normal (0.8–1.5), elevated (spread > 1.5). Individual vol bands (Low/Normal/High/Extreme) in meta at full detail.
Data sources: Binance US spot klines — BTCUSDT and ETHUSDT 1h candles (24 each). 1h TTL (tactical metric).
Implementation fidelity: HIGH. Sample stddev (n-1), annualization via sqrt(8760), vol spread with classification, per-asset vol band metadata. Realized vol is backward-looking (documented limitation).
10. market-regime (mr) — v1.0.0
What it measures: The macro "Master State" indicator — composite regime classification via a 2×3 matrix (Dominance Trend × Market Breadth), gated by BTC price direction (modifier) and liquidity conviction. Answers: "What structural phase is the market in?"
Formula: 2×3 matrix with 10 named regime labels. Signals: (1) BTC Dominance delta from cached prior snapshot with ±0.5pp dead band, (2) Market Breadth Score computed via mbv1.Compute() import (guarantees identical output to mb), (3) Liquidity Pulse ratio for conviction. Modifier: BTC 24h price change (+0.5pp dead band).
Classification (10 regimes): BTC-Led Expansion, Institutional Build, Flight to Safety, Steady Appreciation, Consolidation, Stagnation, Alt-Season / Mania, Capital Rotation, Structural Decay, Capitulation. Capitulation requires high conviction (lp_ratio > 0.15). Modifier is independent of regime label. Conviction and description branching for Consolidation/Stagnation Pressure Cooker variants.
Data sources: CoinGecko /global (dominance + lp_ratio) and /coins/markets (breadth + BTC price). Endpoint cache shared with lp, sp, mb, md. Dominance delta via state cache key marketregime_dominance_pct (prior snapshot comparison). No cross-source validator in v1 (BTC CVD explicitly rejected as adversarial for a dominance-anchored metric).
Implementation fidelity: HIGH. Direct mbv1.Compute() import for breadth consistency, cold-start handling (dominance_cold_start: true, notes: ["cold_start"], confidence: "medium"), weight redistribution detection, missing BTC reference fallback (modifier defaults to "neutral", confidence: "low"), summary reliability tokens ([SIGNAL_UNVERIFIED], [MISSING_BTC_REF], [BREADTH_PARTIAL]) at all detail levels, conviction-aware branching for all regime labels, Capitulation sub-states (abnormal_capitulation, capitulation_price_stabilizing), cache_hit inference, ttl_remaining_sec computed as min of both endpoint TTLs. cache_hint_sec: 14400 (structural metric). divergent_momentum and Phase overlay deferred to v1.1.