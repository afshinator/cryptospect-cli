# dominance

**version:** `v1.0.0`
**Alias:** `dom`
**Endpoints:** `coingecko.global_market`

## Overview

Tracks BTC and ETH market cap dominance — the percentage of total crypto market cap
held by BTC and ETH respectively. Measures the direction of capital flow between
the two dominant assets and the rest of the market.

BTC dominance rising + ETH dominance falling = safety retreat into BTC. BTC dominance
falling + ETH dominance rising = risk-on rotation into ETH (often precedes broader
alt rotation). Both dominance values are universally tracked by every major crypto
analytics platform.

## Formula

```
BTC dominance: BTC.mcap / total_mcap  (from CoinGecko /global market_cap_percentage map)
ETH dominance: ETH.mcap / total_mcap  (same)

ETH/BTC ratio: ETH.mcap / BTC.mcap    (derived)

Dominance trend (delta from prior cached snapshot, requires ≥ 2 runs):
  BTC: delta > +0.5pp → "rising"
       delta < −0.5pp → "falling"
       otherwise       → "neutral"
  ETH: delta > +0.3pp → "rising"
       delta < −0.3pp → "falling"
       otherwise       → "neutral"

  // BTC dead band wider (±0.5pp) because BTC.D normally oscillates ±0.2pp intraday.
  // ETH dead band narrower (±0.3pp) because ETH.D has a smaller absolute range (~15-20%).
  // Cold start (first run): both trends are "neutral" with cold_start: true in meta.

Primary classification label (most significant movement):
  both neutral                            → "neutral"
  BTC rising + ETH falling                → "btc_rising"
  ETH rising + BTC falling                → "eth_rising"
  both rising, |btc_delta| > |eth_delta|  → "btc_rising"
  both rising, |eth_delta| ≥ |btc_delta|  → "eth_rising"
  both falling                            → "capital_contracting"
```

**Threshold rationale:** BTC.D operates in a 40-70% range with typical daily moves
of ±0.2pp. A ±0.5pp dead band filters noise while catching genuine directional shifts.
ETH.D operates in a 10-25% range with smaller absolute moves; ±0.3pp catches meaningful
shifts while filtering noise. These thresholds match market-regime's existing BTC
dominance delta classification (DomDeadBandPP = 0.5).

## Output Schema

```json
{
  "metric": "dominance",
  "version": "v1.0.0",
  "status": "string",

  "data": {
    "btc": {
      "dominance": 52.3,
      "trend": "rising",
      "delta_pp": 1.2
    },
    "eth": {
      "dominance": 18.1,
      "trend": "falling",
      "delta_pp": -0.6
    },
    "eth_btc_ratio": 0.346,
    "classification": {
      "label": "btc_rising",
      "description": "BTC dominance rising (+1.2pp), ETH dominance falling (-0.6pp) — capital rotating into BTC"
    },
    "summary": "BTC dominance 52.3% (rising, +1.2pp), ETH dominance 18.1% (falling, -0.6pp) — capital rotating into BTC."
  },

  "meta": {
    "primary_source": "coingecko",
    "confidence": "high",
    "cold_start": false,
    "prior_snapshot_age_sec": 14400,
    "thresholds": {
      "btc_dead_band_pp": 0.5,
      "eth_dead_band_pp": 0.3
    },
    "description": "Dominance tracks BTC and ETH market cap share..."
  }
}
```

**Cold start output** (first run, no prior snapshot):

```json
{
  "btc": {
    "dominance": 52.3,
    "trend": "neutral",
    "delta_pp": null
  },
  "eth": {
    "dominance": 18.1,
    "trend": "neutral",
    "delta_pp": null
  },
  "classification": {
    "label": "neutral",
    "description": "Dominance trends unverified — first run, no prior snapshot"
  },
  "summary": "BTC dominance 52.3%, ETH dominance 18.1% [UNVERIFIED — first run, no prior snapshot]."
}
```

## Classification

| Label                 | Condition                                  | Description                                                |
| --------------------- | ------------------------------------------ | ---------------------------------------------------------- |
| `btc_rising`          | BTC.D rising, ETH.D falling or rising less | Safety retreat — capital rotating into BTC                 |
| `eth_rising`          | ETH.D rising, BTC.D falling or rising less | Risk-on rotation — capital rotating into ETH               |
| `neutral`             | Both trends neutral or cold start          | No directional signal                                      |
| `capital_contracting` | Both BTC.D and ETH.D falling               | Both losing share to other assets (stablecoins, other L1s) |

Individual `btc.trend` and `eth.trend` fields are always available for finer-grained
reads regardless of the primary label.

## Data Source

- **API:** CoinGecko `/global` — already fetched by liquidity-pulse, stablecoin-power, market-regime
- **Endpoint key:** `coingecko.global_market`
- **Fields used:** `market_cap_percentage.btc`, `market_cap_percentage.eth`
- **Parser:** Extend existing `ParseGlobalDominance` to extract both BTC and ETH
- **State cache:** `dom_btc_dominance_pct` and `dom_eth_dominance_pct` keys. TTL: 48h (172800s). Max snapshot age: 24h (86400s).
- **TTL:** 4 hours (14400s) — structural metric

## Interpretation

- **btc_rising:** Capital rotating out of alts into BTC. If market-breadth is also
  narrow → Flight to Safety. If breadth is broad → BTC-Led Expansion (BTC leading
  but alts participating).

- **eth_rising:** Capital rotating into ETH. Often the first stage of broader alt
  rotation (ETH leads, then mid-caps follow). Cross-check momentum-divergence: if
  md is also risk_on → full rotation in progress.

- **neutral:** No strong directional shift. Cold start if first run. Within dead
  band if subsequent run.

- **capital_contracting:** Both BTC and ETH losing market share. Could indicate
  stablecoin accumulation (cross-check stablecoin-power) or rotation into smaller
  alts (cross-check momentum-divergence tail_extension).

## Limitations

- **Cold start:** First run has no delta — trends are unverified.
- **Dominance is zero-sum by construction.** BTC.D + ETH.D + others = 100%.
  A rising BTC.D mathematically implies falling "others" share.
- **ETH.D movement doesn't guarantee alt rotation.** ETH can decouple from the
  broader alt market (e.g., ETF-driven flows).
- **Single-snapshot deltas** can be noisy. A sustained trend across multiple
  calls is more reliable than a single delta reading.
