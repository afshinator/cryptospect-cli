update 5/11/26

cryptospect-cli — Summary
A Go CLI tool (single static binary, CGO_ENABLED=0) that fetches live crypto data, computes market regime metrics, and outputs agent-optimized JSON. Built with Cobra+Viper, keyless public APIs (CoinGecko, Binance US, DefiLlama). Every invocation writes exactly one JSON object to stdout — an CLIResponse envelope with a results array of MetricResult entries. Stderr is diagnostics only. Agents are meant to call individual metric commands and compose their own analysis.
5 of 6 metrics are production-implemented. Market-regime (mr) is the last remaining scaffold.
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
---
market-regime (mr) — The Last Metric
Current state: 47-line scaffold. Compute() returns "metric not yet implemented: market-regime" with status "unavailable". No compute.go, no types.go, no metric doc at docs/metrics/market-regime.md.
What it should be (inferred from design): Intended as the composite/macro metric — the one that synthesizes outputs from the other five metrics into a single market regime classification. The orchestration playbook says "run regime first to establish macro context." Its endpoints are defined in the scaffold: coingecko.global_market and coingecko.coin_markets_breadth.
Key design question for implementation: Should market-regime be a pure aggregator that calls other metric Compute() functions internally (importing lp/sp/ft/mb/md packages), or should it re-fetch raw data and compute its own signals + classify? The playbook says "regime first" which implies it could be a lightweight aggregator that reads other metrics' outputs and produces a synthesis. But the scaffold has its own endpoints, suggesting an independent computation.
Dependencies on other metrics (for the aggregator approach): liquidity-pulse's volume_to_mcap_ratio, stablecoin-power's stable_power_ratio + supply_trend_7d, flow-tension's CVD/OI/funding hooks, market-breadth's score + divergence_detected, momentum-divergence's label + tail_extension. These would inform a regime classification like "Bull Trend," "Risk-On Expansion," "Defensive/Accumulation," "Bear Trend," etc.