# fear-greed-index

**version:** `v1.0.0`
**Alias:** `fgi`
**Endpoints:** `alternativeme.fng`

## Overview

The Crypto Fear & Greed Index from alternative.me — the industry-standard sentiment
oscillator. Measures crowd emotion on a 0-100 scale: 0 = Extreme Fear (potential
buying opportunity), 100 = Extreme Greed (correction risk). Serves as a contrarian
overlay to the directional metrics in the suite.

Complements market-regime: mr tells you what kind of market you're in, fgi tells
you how the crowd feels about it.

## Formula

```
Raw value: integer 0-100 from alternative.me /fng/ endpoint.

7-day moving average (when ≥ 7 days of history available):
  7d_ma = sum(last_7_values) / 7

Trend direction (requires ≥ 7 days of history):
  value > 7d_ma + 2  → "improving"
  value < 7d_ma − 2  → "deteriorating"
  otherwise          → "stable"

  // ±2 dead band prevents oscillation when value hovers near MA.

Classification bands (computed from raw value):
  0–25   → extreme_fear
  26–45  → fear
  46–55  → neutral
  56–75  → greed
  76–100 → extreme_greed

  // Classification is COMPUTED from the numeric value, not parsed from the
  // API's "value_classification" string — consistent with all other metrics.
```

**Threshold rationale:** The 5-band classification matches alternative.me's documented
methodology (volatility 25%, volume 25%, social 15%, surveys 15%, dominance 10%,
trends 10%). Bands are symmetric and widely recognized across the crypto industry.
The ±2 dead band on trend is a heuristic to suppress noise.

## Output Schema

```json
{
  "metric": "fear-greed-index",
  "version": "v1.0.0",
  "status": "string",

  "data": {
    "value": 25,
    "classification": {
      "label": "extreme_fear",
      "description": "Extreme Fear — historically a buying opportunity"
    },
    "summary": "Fear & Greed: 25/100 (Extreme Fear), sentiment deteriorating (below 7d MA of 32)"
  },

  "meta": {
    "primary_source": "alternative.me",
    "confidence": "high",
    "timestamp": "2026-05-19T12:00:00Z",
    "time_until_update_sec": 5325,
    "sevd_ma": 32.0,
    "trend": "deteriorating",
    "thresholds": {
      "extreme_fear_max": 25,
      "fear_max": 45,
      "neutral_max": 55,
      "greed_max": 75,
      "trend_dead_band": 2
    },
    "description": "Fear & Greed Index measures crowd sentiment on 0-100 scale..."
  }
}
```

**Enhancements:**

| Field     | Condition           | Description                              |
| --------- | ------------------- | ---------------------------------------- |
| `sevd_ma` | ≥ 7 days of history | 7-day simple moving average              |
| `trend`   | ≥ 7 days of history | "improving" / "deteriorating" / "stable" |

When fewer than 7 days of history are available (e.g., first run), `sevd_ma` and `trend`
are `null` and `confidence` drops to `"medium"`.

## Classification

| Label           | Range  | Description                                                       |
| --------------- | ------ | ----------------------------------------------------------------- |
| `extreme_fear`  | 0–25   | Historically a buying opportunity; contrarian long signal         |
| `fear`          | 26–45  | Cautious sentiment; market undervalued                            |
| `neutral`       | 46–55  | Balanced sentiment; no contrarian signal                          |
| `greed`         | 56–75  | Elevated sentiment; market may be overvalued                      |
| `extreme_greed` | 76–100 | Historically a correction risk; contrarian short/defensive signal |

## Data Source

- **API:** alternative.me — `https://api.alternative.me/fng/`
- **Endpoint key:** `alternativeme.fng`
- **Endpoint URL:** `/fng/?limit=7` (7 days for MA + trend computation)
- **Rate limit:** 10/min (free tier)
- **Response shape:** `{"name":"Fear and Greed Index","data":[{"value":"25","value_classification":"Extreme Fear","timestamp":"1779148800","time_until_update":"5325"}],"metadata":{"error":null}}`
- **TTL:** 24 hours (86400s) — index updates once daily

## Interpretation

- **extreme_fear (0–25):** Crowd is panicking. Historically a buying opportunity.
  Cross-check with market-regime — extreme fear in a BTC-Led Expansion is a stronger
  signal than extreme fear in Capitulation.

- **fear (26–45):** Cautious but not panicked. Market likely undervalued. Trend
  direction matters: deteriorating fear → building capitulation; improving fear →
  early recovery.

- **neutral (46–55):** No strong contrarian signal. Wait for directional movement.

- **greed (56–75):** Elevated sentiment. Not yet a sell signal but warrants caution
  on new longs. If trend is improving → FOMO building; watch for extreme_greed.

- **extreme_greed (76–100):** Crowd is euphoric. Historically a correction warning.
  Not a timing signal — markets can stay greedy longer than expected. Cross-check
  flow-tension funding rate and OI for leverage confirmation.

**Important:** Fear & Greed is a contrarian indicator. It works best alongside
directional metrics, not standalone. Extreme fear doesn't mean "buy now" — it means
"the crowd is selling, check if fundamentals support buying." Extreme greed doesn't
mean "sell now" — it means "the crowd is buying, check if momentum is exhausted."
