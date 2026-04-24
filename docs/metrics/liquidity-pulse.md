# liquidity-pulse

**Alias:** `lp`  
**Endpoints:** main: `coingecko.global_market`, supplementary: `binance_us.spot_cvd_btc_1h`

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
|-----------|-----------|---------|
| High | >= 0.15 | Strong short-term conviction |
| Normal | 0.05 - 0.15 | Healthy market |
| Low | < 0.05 | Low conviction / accumulation |

## Data Source

- **Primary API:** CoinGecko
- **Endpoint:** `/global`
- **Fields used:** `total_volume["usd"]`, `total_market_cap["usd"]`

## Cross-Source Verification

This metric uses the **Primary with Anchor-Asset Validation** pattern.

| Role | Source | Purpose |
|------|--------|---------|
| Primary | CoinGecko | Main metric computation (global market data) |
| Validator | Binance US | Anchor-asset (BTC) volume cross-check |

**Anchor asset:** BTC

**Discrepancy threshold:** 20%

**Behavior on mismatch:** Volume ratio computed from CoinGecko remains the metric value. Discrepancy is flagged in metadata with confidence adjusted to "medium" (<20%) or "low" (>=20%).

## CLI Flags

This metric has no CLI flags.

## Output Schema

**Base fields** (always present):
```json
{
  "data": {
    "volume_to_mcap_ratio": "float64",
    "volume_usd": "float64",
    "market_cap_usd": "float64",
    "classification": "string",
    "summary": "string"
  }
}
```

**Enhancements** (conditional — present when specific conditions are met):

| Field | Condition | Description |
|-------|-----------|-------------|
| `delta_24h` | Prior cache data exists | Change in ratio from 24h ago, indicating momentum |

## Usage

```bash
# Basic
cryptospect-cli liquidity-pulse

# With alias
cryptospect-cli lp

# With detail
cryptospect-cli liquidity-pulse --detail full
```

## Long Description

### High-level meaning and value
- This metric measures the "velocity" or "turnover" of money within the system. It is critical for identifying "mechanical integrity" — determining if a price move is a healthy, broad-based event or a "fake" move occurring on thin liquidity.
- It answers the question: "How aggressively is the market being traded relative to its size?"
- This metric is highly effective at identifying "speculative blow-off" risks when volume spikes excessively relative to market cap growth.

### Exact definition and data needs, logic
- A simple ratio: (24h Trading Volume) / (Total Market Cap) using CoinGecko's `/global` endpoint.
- It uses real-time volume data from CoinGecko against the total circulating value of all crypto assets.
- The ratio without directional bias — high means conviction, but direction depends on context.

### Possible values and associated verdicts
- **High (>0.15):** Expansion momentum OR exhaustion blow-off — monitor price to determine which
- **Normal (0.05–0.15):** Healthy market participation
- **Low (<0.05):** Apathy OR smart money accumulating — check other metrics

A high ratio indicates either:
- Bullish momentum (capital flowing in during uptrends)
- Risk-off panic (capital fleeing during selloffs)

Low ratios indicate either:
- Range-bound accumulation (smart money quietly accumulating)
- Distribution phase (smart money exiting)

The absolute value matters more than the direction—this is a conviction meter, not a directional indicator.

### Cross-Source Verification

CoinGecko's global volume can vary significantly from "Real Volume" (indices that filter out wash-trading). If the tool provides a ratio of 0.25 but 80% of that is wash-trading on unverified exchanges, the LLM might signal a "Strong Conviction" that doesn't exist.

**Solution:** Use Binance US BTC spot volume as an anchor-asset validator. BTC is liquid enough on major exchanges that its volume is relatively trustworthy. If BTC volume is wildly different between sources, the global metric is likely also skewed.

**Validation behavior:**
- Fetch BTC 1h CVD from Binance US in parallel with CoinGecko global data
- Compare anchor volumes; if discrepancy >20%, flag `discrepancy_detected: true`
- Metric value is still computed from CoinGecko primary — validator only reports
- `confidence` in metadata reflects validation result

**Why BTC vs. global volume?** Binance US only covers a fraction of global liquidity. Comparing total global volume to a single-exchange volume is apples-to-oranges. BTC is traded globally on Binance US, making it a reasonable anchor for cross-check.

### Other details

**Enhancements:**
- `delta_24h`: Change in ratio from previous calculation. Available on 2nd+ run (uses cached value). Not yet implemented — requires historical data source.

**Implementation Compromises:**
- Uses only CoinGecko for primary computation; cross-source validation planned for anchor asset
- No stablecoin filtering in v1 — stablecoin volume noise noted as known limitation

**Future Enhancements:**
- `--exclude-stables` flag to filter stablecoin volume
- `delta_24h` with historical data source
- Full anchor-asset validation (Binance US BTC spot CVD)