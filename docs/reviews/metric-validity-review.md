# Metric Validity Review — 2026-05-18

Sanity check of all 6 metric formulas against real-world crypto/finance standards.
Each metric is assessed on: formula correctness, threshold calibration, known approximations,
and whether the docs honestly disclose limitations.

## Group 1: Structurally sound — no issues

### Liquidity Pulse

- **Formula:** `volume_24h / total_market_cap`
- **Assessment:** Standard turnover/velocity ratio. Thresholds (0.05/0.15) are reasonable
  crypto-level heuristics — crypto has higher turnover than equities, so 15% daily is
  plausible for "high conviction" while 5% floor captures stagnation.
- **Binance validator pattern:** Sensible — BTC anchor-volume cross-check to detect
  wash-trading inflation in CoinGecko's global aggregate. Documented in metric doc.
- **Known limitation:** No stablecoin filtering in v1. USDT volume can inflate the ratio.

### Flow Tension

- **Formula:** Three independent signals — CVD proxy `(buy - sell) / total`, OI change
  `(curr - prev) / prev`, funding rate (raw from Binance Futures BTC perpetual).
- **Assessment:** Industry-standard derivatives metrics. Every major crypto trading
  desk and analytics platform tracks these same three signals. Thresholds are
  well-calibrated to BTC perpetual historical ranges (OI ±5% captures intraday noise;
  funding 0.03%/0.30% maps to observed BTC perpetual cycles; CVD ±10% filters balanced churn).
- **Known limitations (documented):** CVD is a single-exchange proxy (Binance US), not true
  exchange net flow. OI 24h change has cold-start delay on first run. No funding rate
  trend tracking (rising toward overheated vs already overheated). No cross-source
  validation for OI/funding.

## Group 2: Correct concept, implementation tradeoffs acknowledged

### Momentum Divergence

- **Formula:** Market-cap weighted tier averages → spread matrix → classification.
- **Assessment:** Same methodology as equity small-cap vs large-cap rotation analysis
  (Russell 2000 vs S&P 500). Market-cap weighting (v1.1) is correct — BTC/ETH should
  dominate large-cap tier. Nil-safe spread matrix (pointer types) correctly distinguishes
  "absent" from "genuine zero." Asymmetric thresholds make sense: +5pp risk_on is
  conservative (requires consistent directional bias across 40 mid-cap coins); -3pp
  top_heavy is tighter because concentration patterns are structurally more specific.
  Dead band (±0.5%) for top_heavy/flight_to_safety split prevents label oscillation on
  near-zero large-cap averages.
- **Threshold calibration gap:** The ±5pp / –3pp / 1pp guard thresholds were calibrated
  against v1 simple-mean values and carried forward unchanged to v1.1 market-cap weighting.
  Under market-cap weighting, large-tier averages are more BTC/ETH-dominated → spreads may
  be narrower in absolute value. Recalibration is a data-accumulation task.
- **Other documented limitations:** USDT in large tier dampens large_avg (near-zero 24h
  return × $150B market cap). Conservative 5pp threshold means slow rotations in
  low-volatility markets go undetected (design choice, not defect). No stale-data guard
  for zero/near-zero payloads from API degradation.

### Market Regime

- **Formula:** 2×3 matrix (dominance trend × breadth band) producing 10 named regimes,
  with conviction (lp_ratio) disambiguating Capitulation vs Structural Decay, and modifier
  (BTC 24h direction) as an independent overlay.
- **Assessment:** Internally coherent framework. The axes (BTC dominance, market breadth,
  liquidity conviction) are the correct dimensions for crypto regime classification — they
  capture the direction of capital flow, the breadth of participation, and the intensity
  of activity. The matrix itself is a design artifact, not derived from backtesting or
  an external standard. No objective source says "rising dom + broad breadth = BTC-Led
  Expansion" — this is an opinion, not a fact.
- **Capitulation/Structural Decay split:** Well-reasoned. High volume + falling dom +
  narrow breadth = panic selling (Capitulation). Low volume + same = slow bleed
  (Structural Decay). The conviction threshold (lp_ratio > 0.15 = high) requiring
  strictly greater provides a clean boundary with no ambiguous fall-through.
- **Capitulation sub-states:** The modifier-based sub-states
  (negative_pressure = confirmed panic, neutral = price stabilizing,
  positive_momentum = V-bottom/short squeeze) add useful granularity but are a
  qualitative overlay, not a quantitative classification.
- **Documented limitations:** First run produces neutral dominance (cold start) — regime
  label from first call is not a signal. Dominance delta requires prior cached snapshot.
  Relies on input metrics (LP, MB) which have their own approximations.

## Group 3: Simplified vs industry standard

### Market Breadth

- **Formula:** Weighted composite `0.10×1h + 0.30×24h + 0.40×7d + 0.20×30d` of
  % coins with positive price change per timeframe.
- **Industry standard equivalent:** "% of assets above their X-day moving average" —
  e.g., "% of S&P 500 constituents above their 50-day MA."
- **Key difference:** Our proxy uses "positive X-day price change" as a substitute for
  "above X-day MA." These are NOT equivalent. A coin that fell -80% over 30 days then
  rallied +3% today is "green on 24h" and counts as participating. True MA breadth
  would NOT count it as "above its 50-day MA." During sharp bear-market rallies, our
  metric overstates breadth. During V-recoveries, our metric lags (24-48h documented)
  because the 7d/30d windows are still red.
- **Mitigations documented in metric doc:** Multi-timeframe composite with recency bias
  (1h at only 10% weight, 7d at 40%) partially compensates. The 1h-vs-7d spread read
  in Agentic Logic is designed to catch early recovery signals. V-shape lag caveat is
  explicitly documented. True MA-based breadth is explicitly deferred — requires
  historical price storage per coin, not feasible given public API constraints.
- **Why this proxy is reasonable:** The CoinGecko public API exposes pre-computed
  percentage changes (1h, 24h, 7d, 30d) rather than raw OHLCV history for all coins.
  The proxy uses what's available in a single API call. For a free, keyless CLI tool,
  the approximation is defensible. The documentation is honest about the gap.

### Stablecoin Power

- **Formula:** `stable_mcap / (total_mcap - stable_mcap)`, with top-N selection,
  DefiLlama cross-validation, and 7d supply trend disambiguation.
- **Assessment:** A creative metric with no exact real-world equivalent. The formula
  choice (stables / volatile, not stables / total) is correct — it prevents stablecoin
  supply growth from inflating the denominator and masking genuine dry powder increase.
  The cross-source validation (DefiLlama on-chain vs CoinGecko exchange-feed) uses
  genuinely different collection methodologies — a proper independent validation.
- **Two structural limitations that inflate the signal:**
  1. **Deployed vs idle:** Stablecoins locked in DeFi (Curve pools, Aave, Uniswap LP
     positions) count as "dry powder" but are not freely deployable. The metric has no
     way to distinguish idle from deployed — it likely overstates true available dry
     powder, especially during high DeFi activity. Fixable with DefiLlama TVL data
     (documented as future enhancement).
  2. **Non-USD stables:** EURC, GBPT, XSGD included at USD-equivalent market cap.
     Their supply can grow for non-crypto reasons (e.g., MiCA compliance driving EURC
     adoption). A supply expansion of EURC could push the ratio higher without any
     genuine increase in crypto-specific dry powder. Currently minor (~3-4% of supply)
     but may grow. `--usd-only` flag documented as future enhancement.
- **Other documented gotchas:** Algorithmic/synthetic stables (USDe, DAI) included at
  market price — if any depeg, market price < $1, understating true supply. Top-N
  (not total supply) — top 8 captures ~93-95%, acceptable but not complete.
  DefiLlama uses CoinGecko for pricing (shared layer) — discrepancies reflect coverage
  scope differences, not data errors.

## Summary

| Metric | Formula quality | Industry alignment | Biggest gap |
|--------|:-:|:-:|---|
| Liquidity Pulse | ✅ | ✅ Direct match | Stablecoin volume noise |
| Flow Tension | ✅ | ✅ Direct match | Single-exchange CVD proxy |
| Momentum Divergence | ✅ | ✅ Good adaptation | Thresholds un-recalibrated for v1.1 weighting |
| Market Regime | ✅ | ⚠️ Proprietary | No external validation of regime assignments |
| Market Breadth | ✅ | ⚠️ Proxy, not standard | % positive ≠ above MA; overstates during bear rallies |
| Stablecoin Power | ✅ | ⚠️ Novel metric | Deployed stables counted as dry powder; non-USD stable noise |

## Recommendations for future improvements

1. **Market Breadth — true MA proxy:** Consider adding the CoinGecko 7d/30d percentage
   change as a two-point proxy for "above/below MA" by comparing sign of shorter vs
   longer window (e.g., 24h positive but 7d negative = below medium-term reference).
   Not a true MA, but closer than raw % positive.

2. **Stablecoin Power — DeFi TVL subtraction:** Use DefiLlama TVL data (already
   available from the same API) to subtract stablecoins locked in major DeFi protocols
   from the numerator. Transforms "dry powder" from approximation to estimate.

3. **Market Regime — sensitivity flag:** Add a `--sensitivity` flag like MD, allowing
   users to tighten/widen dead bands for dominance delta and modifier. Makes the
   "opinion" nature of the matrix explicit and user-configurable.

4. **Momentum Divergence — threshold recalibration:** After accumulating real v1.1
   output data, backtest whether the ±5pp / –3pp thresholds remain appropriate under
   market-cap weighting. USDT dampening may make risk_on harder to trigger
   (large_avg pulled toward zero), requiring a lower threshold.

5. **Flow Tension — second exchange for CVD:** Add a second Binance endpoint
   (Binance Global if API key wired) or OKX public API to cross-validate CVD.
   Reduces single-exchange distortion risk.

6. **Liquidity Pulse — stablecoin volume filter:** Add `--exclude-stables` flag.
   Removes USDT/USDC volume from the numerator for a pure volatile-asset turnover ratio.
