# volatility

**version:** `v1.0.0`
**Alias:** `vol`
**Endpoints:** `binance.spot_cvd_btc_1h`, `binance.spot_cvd_eth_1h`

## Overview

Measures annualized realized volatility for BTC and ETH from hourly OHLC data, plus
the ETH/BTC volatility spread — a standard crypto volatility desk metric. Answers:
_"How risky is this market right now, and is ETH speculation elevated relative to BTC?"_

Realized volatility helps agents size positions: low vol supports larger exposure;
high/extreme vol warrants reduction. The ETH/BTC vol spread flags when ETH is
disproportionately volatile — a known signal for altcoin risk-on conditions.

## Formula

```
Realized volatility (annualized):
  For each asset (BTC, ETH):
    log_returns[i] = ln(close[i] / close[i-1])  for i = 1..23  (24 candles → 23 returns)
    realized_vol  = std(log_returns) × sqrt(8760)

    // 24 hourly candles (1 day of data). Annualized via sqrt(365 × 24) = sqrt(8760).
    // std uses sample standard deviation (n-1 denominator).

Vol spread:
  vol_spread = eth_realized_vol / btc_realized_vol

Classification (from vol_spread):
  spread < 0.8    → "subdued"    (ETH unusually calm, capital parked in BTC)
  0.8 ≤ spread ≤ 1.5 → "normal"    (typical ETH/BTC vol relationship)
  spread > 1.5    → "elevated"   (heightened ETH speculation / altcoin risk-on)
```

**Threshold rationale:** 0.8/1.5 boundaries are standard crypto volatility desk
thresholds. Spread below 0.8 indicates ETH is unusually calm relative to BTC —
often precedes altcoin underperformance. Spread above 1.5 indicates ETH is
disproportionately volatile — historically coincides with altcoin speculation
and risk-on rotation.

## Output Schema

```json
{
  "metric": "volatility",
  "version": "v1.0.0",
  "status": "string",

  "data": {
    "btc_realized_vol": 0.45,
    "eth_realized_vol": 0.72,
    "vol_spread": 1.6,
    "classification": {
      "label": "elevated",
      "description": "Elevated ETH volatility relative to BTC — heightened altcoin speculation"
    },
    "summary": "BTC vol 45.0% (annualized), ETH vol 72.0%, spread 1.6× — elevated ETH speculation"
  },

  "meta": {
    "primary_source": "binance_us",
    "confidence": "high",
    "btc_candles": 24,
    "eth_candles": 24,
    "thresholds": {
      "vol_subdued_max": 0.8,
      "vol_elevated_min": 1.5,
      "annualization_factor": 365
    },
    "description": "Volatility measures annualized realized volatility..."
  }
}
```

**Individual vol bands** (informational, in meta for full detail):

| Band    | BTC range | ETH range |
| ------- | --------- | --------- |
| Low     | < 30%     | < 30%     |
| Normal  | 30–60%    | 30–60%    |
| High    | 60–100%   | 60–100%   |
| Extreme | > 100%    | > 100%    |

## Classification

| Label      | Condition              | Description                                 |
| ---------- | ---------------------- | ------------------------------------------- |
| `subdued`  | vol_spread < 0.8       | ETH unusually calm; capital parked in BTC   |
| `normal`   | 0.8 ≤ vol_spread ≤ 1.5 | Typical ETH/BTC volatility relationship     |
| `elevated` | vol_spread > 1.5       | Heightened ETH speculation; altcoin risk-on |

## Data Sources

- **BTC klines:** Existing `binance.spot_cvd_btc_1h` endpoint — `KlinesURL("BTCUSDT", "1h", 24)`. Returns 24 hourly candles. Already fetched by liquidity-pulse and flow-tension.
- **ETH klines:** New endpoint `binance.spot_cvd_eth_1h` — `KlinesURL("ETHUSDT", "1h", 24)`. Same parser, different symbol.
- **Parser:** New `ParseMultiKlinesResponse` in binance client returning all kline data (Close, Open, OpenTime, Volume for each candle).
- **TTL:** 1 hour (3600s) — tactical metric like flow-tension.

## Interpretation

- **subdued (spread < 0.8):** ETH volatility is compressed relative to BTC.
  Often precedes ETH underperformance. If BTC vol is also low, market may be
  coiling for a breakout.

- **normal (0.8–1.5):** Typical relationship. No volatility-based edge.

- **elevated (spread > 1.5):** ETH is disproportionately volatile. Historically
  coincides with altcoin speculation and risk-on rotation. Cross-check with
  momentum-divergence: if md is also risk_on, confirms alt rotation. If md is
  neutral or top_heavy, ETH vol may be an idiosyncratic event (ETF flow, network
  upgrade) rather than broad speculation.

- **BTC vol > 100% (extreme):** Panic or euphoria. Cross-check fgi: extreme fear
  - extreme vol = capitulation; extreme greed + extreme vol = blow-off top.

- **BTC vol < 30% (low):** Compression. May support larger position sizing.
  Low vol in a trending regime (market-regime = BTC-Led Expansion) is stronger
  than low vol in stagnation.

## Limitations

- **Realized vol is backward-looking.** Does not predict volatility spikes.
- **Single-exchange data (Binance US).** Vol estimates from one exchange.
- **24-hour window only.** Short-term vol can be noisy; a 7-day variant would
  be smoother but less reactive.
- **Threshold calibration is heuristic.** The 0.8/1.5 spread bands are standard
  crypto desk thresholds, not empirically calibrated to this data source.
