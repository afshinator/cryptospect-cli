# liquidity-pulse

**Alias:** `lp`  
**Endpoint:** `coingecko.global_market`

## Overview

Measures the ratio of 24h trading volume to total market cap. Indicates how actively the crypto market is being traded relative to its size. High values suggest strong short-term conviction and tight market depth; low values suggest stagnation or distribution.

## Formula

```
liquidity_pulse = total_volume_usd / total_market_cap_usd
```

## Interpretation

- **High (>0.15):** Strong trading activity relative to size — potential bullish momentum or panic selling
- **Normal (0.05–0.15):** Healthy market conditions
- **Low (<0.05):** Low conviction, range-bound or accumulating phase

## Classification

| Condition | Threshold | Meaning |
|-----------|-----------|--------|
| High | >= 0.15 | Strong short-term conviction |
| Normal | 0.05 - 0.15 | Healthy market |
| Low | < 0.05 | Low conviction / accumulation |

## Data Source

- **API:** CoinGecko
- **Endpoint:** `/global`
- **Fields used:** `total_volume["usd"]`, `total_market_cap["usd"]`

## Output Schema

```json
{
  "data": {
    "volume_to_mcap_ratio": "float64",
    "volume_usd": "float64",
    "market_cap_usd": "float64",
    "classification": "string",
    "summary": "string"
  },
  "meta": {
    "sources": ["coingecko.global_market"],
    "thresholds": {
      "high": 0.15,
      "low": 0.05
    }
  }
}
```

## Usage

```bash
# Basic
cryptospect-cli liquidity-pulse

# With alias
cryptospect-cli lp

# With detail
cryptospect-cli lp --detail full
```

## Long Description

Liquidity-pulse measures the ratio of 24-hour global trading volume to total crypto market capitalization. It answers the question: "How aggressively is the market being traded relative to its size?"

This metric captures market conviction intensity. When traders allocate significant capital relative to total market cap, it signals strong short-term belief in price direction—whether up or down. A high ratio indicates either:
- Bullish momentum (capital flowing in during uptrends)
- Risk-off panic (capital fleeing during selloffs)

Low ratios indicate either:
- Range-bound accumulation (smart money quietly accumulating)
- Distribution phase (smart money exiting)

The absolute value matters more than the direction—this is a conviction meter, not a directional indicator.

### Use Cases
- **Bull markets:** High ratio (>0.15) confirms strong momentum—ride the trend
- **Bear markets:** Very high ratio (>0.20) often marks local tops (panic selling)
- **Range-bound:** Low ratio (<0.05) suggests accumulation phase—patience