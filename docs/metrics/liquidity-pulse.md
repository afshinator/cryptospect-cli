# liquidity-pulse

**version:** v1.0.0
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

| Condition | Threshold |
|-----------|-----------|
| High | >= 0.15 |
| Normal | 0.05 - 0.15 |
| Low | < 0.05 |

## Data Source

- **Primary API:** CoinGecko
- **Endpoint:** `/global`
- **Fields used:** `total_volume["usd"]`, `total_market_cap["usd"]`
- **Secondary API:** Binance-US
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

Inherits all Global Flags; no unique sub-flags in v1.0.


## Output Schema

```json
{
    "metric":  "liquidity-pulse",
    "version": "v1.0.0",
    "status":  "string",   // "ok" / "degraded" / "unavailable"

    "data": {
        "volume_to_mcap_ratio": "float64",
        "volume_usd":           "float64",
        "market_cap_usd":       "float64",
        "classification": {
            "label":       "string",  // "high" / "normal" / "low"
            "description": "string"   // e.g. "Strong short-term conviction"
        },
        "summary": "string"
    },

    "meta": {
        // Omitted when --detail basic.
        // Present when --detail extended or full:
        "cache_hit":            "bool",
        "ttl_remaining_sec":    "int",
        "primary_source":       "coingecko",
        "validator_source":     "binance_us",
        "discrepancy_detected": "bool",
        "discrepancy_note":     "string",  // only if discrepancy_detected == true
        "confidence":           "string"   // "high" / "medium" / "low"
        // Additionally when --detail full:
        // "thresholds": { "high": 0.15, "low": 0.05 }
        // "description": "string"
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
- `delta_24h` with historical data source.  TBD: Is it the absolute difference (e.g., $0.12 - 0.10 = 0.02$) or the percentage change (e.g., a $20\%$ increase in turnover)? percentage change is usually higher signal for "momentum," while the absolute difference is better for "regime shifts." I'd suggest percentage change or providing both.
- Full anchor-asset validation (Binance US BTC spot CVD) ; note: Ensure secondary endpoint fetch is for 24h_volume or total_trading_volume for BTC, rather than just the CVD (net buy/sell pressure), to keep the "apples-to-apples" comparison with the primary metric's volume.
  -- *Implementation Detail*: Since Binance-US 24h_volume and CoinGecko's global_24h_volume are technically different scopes (one exchange vs. the whole world), your "Anchor Asset" comparison should specifically compare CoinGecko's BTC volume field against Binance-US's BTC volume field. Do not compare Binance-US BTC volume to CoinGecko Global Market volume; that will always trigger a discrepancy


**Agentic Logic (Strategic Notes)**
When an LLM or Agent calls this tool, it should use the following heuristic:

- Divergence Check: If Price is increasing but Liquidity Pulse is decreasing (Ratio < 0.05), the move is "hollow." Verdict: Bearish Divergence / Exit is Narrowing.

- Conviction Check: If Price breaks a key resistance and Liquidity Pulse spikes (> 0.15), the breakout is validated. Verdict: High Conviction Trend.

- Churn Check: If Liquidity Pulse is very high (> 0.20) but Price is sideways, big players are likely exiting into retail liquidity. Verdict: Distribution / Potential Top.

- The "Zombie Market" Risk: In a very low-volatility environment, both CoinGecko and Binance might report extremely low volumes that happen to be within 20% of each other. The tool will report confidence: high, but the actual liquidity might be so low that the metric itself is "fragile."  The LLM should be instructed (via agents.md) that confidence: high only means the data is accurate, not that the market is robust.