# china-m2

**version:** `v1.0.0`
**Alias:** `cnm2`
**Endpoints:** `dbnomics.china_m2`

## Overview

China M2 money supply — a macro liquidity indicator with historically strong
correlation to Bitcoin price cycles. China's M2 (~$47T) is ~2.2× larger than US M2
(~$21T) and multiple analysts have documented a specifically strong relationship
between China's M2 expansion and BTC bull cycles.

Operates on a monthly frequency — not actionable for intraday decisions, but provides
structural macro context for the directional metrics in the suite.

## Formula

```
China M2 level: value from DBnomics series NBS/M_A0D01/A0D0101
  (Money and Quasi-Money (M2) Supply, period-end)
  Units: 100 million yuan → divide by 10 for CNY billion.

YoY % change: ((current_value - value_12_months_ago) / value_12_months_ago) × 100

  // Requires ≥ 13 months of history. On cold start (first run, insufficient history):
  // yoy_change = null, confidence drops to "medium".

Classification (from YoY % change):
  YoY > 8%   → "expanding"   (strong liquidity tailwind, typical for China)
  4% ≤ YoY ≤ 8% → "normal"     (steady expansion)
  YoY < 4%   → "slowing"     (tightening, rare for China, last seen 2014-2015)
```

**Threshold rationale:** China's M2 has historically grown 8-12% annually.
Below 8% indicates slowing expansion; below 4% indicates genuine tightening
(rare — historically associated with economic slowdowns). Above 8% is the
normal expansion range for China's credit-driven economy.

## Output Schema

```json
{
  "metric": "china-m2",
  "version": "v1.0.0",
  "status": "string",

  "data": {
    "m2_level_cny_billion": 34922.0,
    "yoy_change_pct": 8.7,
    "period": "2026-02",
    "classification": {
      "label": "expanding",
      "description": "China M2 expanding (+8.7% YoY) — strong liquidity tailwind"
    },
    "summary": "China M2: 34,922B CNY (Feb 2026), +8.7% YoY — expanding, strong liquidity tailwind."
  },

  "meta": {
    "primary_source": "dbnomics",
    "confidence": "high",
    "data_frequency": "monthly",
    "data_lag_days": 78,
    "units": "CNY billion",
    "thresholds": {
      "expanding_min_yoy": 8.0,
      "slowing_max_yoy": 4.0
    },
    "description": "China M2 money supply from National Bureau of Statistics of China via DBnomics..."
  }
}
```

**Cold start / insufficient history:**

```json
{
  "m2_level_cny_billion": 34922.0,
  "yoy_change_pct": null,
  "period": "2026-02",
  "classification": {
    "label": "normal",
    "description": "Insufficient history for YoY calculation — showing latest level only"
  },
  "summary": "China M2: 34,922B CNY (Feb 2026) [INSUFFICIENT_HISTORY — YoY unavailable, need 13+ months]."
}
```

## Classification

| Label       | Condition     | Description                                             |
| ----------- | ------------- | ------------------------------------------------------- |
| `expanding` | YoY > 8%      | Strong liquidity tailwind; historically bullish for BTC |
| `normal`    | 4% ≤ YoY ≤ 8% | Steady expansion; neutral macro signal                  |
| `slowing`   | YoY < 4%      | Tightening; historically bearish for BTC (rare)         |

When YoY is unavailable (cold start / insufficient history): `"normal"` with
`confidence: "medium"` and a note in description.

## Data Source

- **API:** DBnomics — `https://api.db.nomics.world/v22/series/NBS/M_A0D01/A0D0101`
- **Endpoint key:** `dbnomics.china_m2`
- **Endpoint URL:** `/series/NBS/M_A0D01/A0D0101?observations=true&limit=13`
- **Provider:** National Bureau of Statistics of China
- **Frequency:** Monthly (data lags ~1-2 months behind current month)
- **Response shape:** `{"series":{"docs":[{"period":["2025-03",...,"2026-02"],"value":[...]}]}}`
- **TTL:** 30 days (2592000s) — data updates monthly

## Interpretation

- **expanding (YoY > 8%):** China's credit engine is running. Historically
  coincides with BTC bull cycles. The transmission channel is indirect (Chinese
  capital controls → offshore capital → crypto) but the correlation is well-documented.
  Cross-check: expanding China M2 + rising market-breadth = strong macro tailwind
  confirmed by on-chain participation.

- **normal (4-8%):** Steady expansion. No strong macro signal. Default state
  for China's economy.

- **slowing (YoY < 4%):** China's credit expansion is decelerating. Historically
  associated with economic slowdowns and BTC bear phases (2014-2015 was the last
  sub-4% period). Cross-check with US macro context: China slowing + Fed hawkish
  = maximum macro headwind.

- **Data lag:** China M2 data is published with ~1-2 month lag. The `period` field
  indicates the month the data refers to, not when it was fetched. The `data_lag_days`
  field in meta tells agents how stale the data is.

## Limitations

- **Monthly frequency.** Not actionable for intraday/tactical decisions.
- **Correlation is directional, not mechanical.** China M2 expansion doesn't
  guarantee immediate BTC inflow. The transmission channel is indirect.
- **Data lag.** Published 1-2 months after the reference period.
- **Currency denomination.** Values are in CNY. For comparison with US M2,
  USD/CNY exchange rate conversion is needed (deferred to usml in Phase 3).
- **China M2 methodology differences.** China's M2 definition includes deposits
  that Western definitions would exclude, making direct US/China M2 comparisons
  approximate.
