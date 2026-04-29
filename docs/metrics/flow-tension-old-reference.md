# Flow-Tension Reference (from Original Implementation)

**Source:** `cryptospect-cli-original` project  
**Purpose:** Reference material for porting flow-tension to the new plugin architecture  
**Status:** This is documentation only — not the new implementation

---

## Metric Identity

**Name:** `flow-tension` (alias: `ft`)  
**Original description:** *"The 'Transmission' metric of the suite. While Stablecoin Power shows **potential** energy, Flow Tension shows **kinetic** energy — how aggressively traders are using leverage and moving assets onto exchanges to trade."*

**What it actually measures (3 signals):**
1. **Exchange Net Flow** (CVD proxy) — taker buy vs sell aggression from Binance US spot klines
2. **Open Interest** — leverage building/unwinding (requires API key)
3. **Funding Rate** — perp sentiment, longs vs shorts (requires API key)

**Status behavior:** Runs in `degraded` mode without API key (only CVD available). Full signals require `CRYPTOSPECT_COINGECKO_KEY` (CoinGecko Pro) or `CRYPTOSPECT_BINANCE_KEY` (Binance Futures).

---

## Files to Port (from `/project/cryptospect-cli-original/`)

### Core Implementation (3 files)

| Original File | Lines | Purpose | Porting Notes |
|--------------|-------|---------|----------------|
| `internal/metrics/flowtension/calc.go` | 197 | All compute logic + helpers | Must adapt to `Compute(ctx, map[string]json.RawMessage) (output.MetricResult, error)` signature |
| `internal/metrics/flowtension/types.go` | 47 | Input, Data, NarrativeHooks structs | Keep as-is or inline (new arch doesn't use types.go) |
| `internal/metrics/flowtension/calc_test.go` | 421 | Table-driven tests | Port to `_test.go` in new `v1/` package |

### Documentation (1 file)

| File | Lines | Purpose |
|------|-------|---------|
| `docs/metrics/flow-tension.md` | 306 | Output schemas (basic/extended/full), thresholds, narrative hook combinations, data sources, calibration notes |

---

## Key Constants & Thresholds (from original `calc.go`)

```go
// Thin-candle guard
minTrades = 10  // below this, flow hook = "low_confidence"

// CVD classification
flowAggressiveThreshold = 0.10  // > ±10% of volume = aggressive side

// Funding rate (decimal per 8h cycle, BTC perp)
fundingNegativeThreshold   = -0.0003  // <= -0.03%
fundingPositiveThreshold   =  0.0003  // >= +0.03% (and < overheated)
fundingOverheatedThreshold =  0.003   // > +0.30%

// Open Interest 24h change
oiBuildingThreshold  = 0.05   // > +5%
oiUnwindingThreshold = -0.05   // < -5%
```

---

## Data Flow Architecture

### Input → Compute → Output

**Input struct (original `types.go`):**
```go
type Input struct {
    // From Binance US spot klines (keyless)
    TakerBuyVolume  float64
    TakerSellVolume float64
    TotalVolume     float64
    NumTrades       int

    // From CoinGecko Pro or Binance Futures (requires API key)
    OpenInterest        *float64  // nil when unavailable
    OpenInterestPrev24h *float64  // nil when unavailable
    FundingRate         *float64  // nil when unavailable
}
```

**New architecture input:** `map[string]json.RawMessage` with keys from `Endpoints`:
- `api.BinanceSpotCVD_BTC_1h` → klines JSON → extract taker volumes
- `api.CoinGeckoDerivatives` → derivatives JSON → extract OI + funding

**Output (original `Data` struct):**
```go
type Data struct {
    FundingRate                    *float64
    FundingRateFormatted           *string
    OpenInterest                   *float64
    OpenInterestChange24h          *float64
    OpenInterestChange24hFormatted *string
    ExchangeNetFlow                float64
    ExchangeNetFlowFormatted       string
    NarrativeHooks                 NarrativeHooks
    Summary                        string
}
```

**New architecture output:** Wrap in `output.MetricResult` with `DetectStatus()` for status.

---

## Compute Functions to Port

| Function | Signature | Purpose | Test Coverage |
|----------|-----------|---------|----------------|
| `Compute` | `(input Input) (Data, error)` | Main entry point | `TestCompute_FullSignals`, `TestCompute_DegradedMode`, `TestCompute_ZeroTotalVolume` |
| `ComputeExchangeNetFlow` | `(takerBuy, takerSell, total float64) float64` | CVD = (buy-sell)/total | 4 tests (buyers/sellers/neutral/zero) |
| `ComputeFlowHook` | `(netFlow float64) string` | Classify CVD → buyers_aggressive / flow_neutral / sellers_aggressive | 5 tests (boundaries included) |
| `ComputeFundingRateHook` | `(rate float64) string` | Classify → negative/neutral/positive/overheated | 9 tests (realistic rates, boundaries) |
| `ComputeOIChange24h` | `(current, prev float64) float64` | % change in OI | 3 tests |
| `ComputeOIChangeHook` | `(change float64) string` | Classify → building/stable/unwinding | 5 tests (boundaries included) |
| `ComputeSummary` | `(hooks NarrativeHooks) string` | Narrative from hook combos | 6 tests (early bull, supply shock, tension, etc.) |

---

## Narrative Hook Combinations (from original docs)

| Funding | Open Interest | Flow | Summary |
|---------|-----------------|------|---------|
| `negative_funding` | `oi_building` | any | "Shorts paying longs while leverage builds — early bull phase, sellers exhausted." |
| `overheated_funding` | `oi_building` | any | "Leverage building with overheated longs — elevated liquidation risk." |
| any | `oi_building` | `buyers_aggressive` | "Leverage building with aggressive buying — tension coiling, breakout likely." |
| any | `oi_stable` | `sellers_aggressive` | "Assets staged on exchanges with aggressive selling — supply shock / top warning." |
| `oi_unwinding` | any | any | "Leverage unwinding — deleveraging in progress, likely post-liquidation." |
| `neutral_funding` | `oi_stable` | `flow_neutral` | "Flow tension neutral — no directional conviction." |
| degraded mode (funding/oi unavailable) | | | "Partial data — OI and funding rate unavailable without API key. Aggressive [buying/selling] detected." |
| `low_confidence` (thin candle) | | | "Thin candle — insufficient trades for reliable CVD signal. Flow direction unreliable." |

---

## API Endpoints Needed

| Endpoint Key | Source | Data | TTL | API Key |
|--------------|--------|------|-----|---------|
| `BinanceSpotCVD_BTC_1h` | Binance US `api/v3/klines` | symbol=BTCUSDT, interval=1h, limit=1 → takerBuyBaseVol, volume, numTrades | 3600s | ❌ Keyless |
| `CoinGeckoDerivatives` | CoinGecko Pro `/derivatives` | Filter BTC perpetual → open_interest, funding_rate | 3600s | ✅ Required |

**Original fallback:** Binance Futures (`fapi.binance.com`) if CoinGecko Pro key not set.

**Current new-arch constraint:** Binance US client is spot-only. Futures OI/funding need either:
1. Extend Binance client to support `fapi.binance.com`, OR
2. Rely solely on CoinGecko Pro derivatives endpoint

---

## Output Schema (from original docs)

**Basic (default):**
```json
{
  "metric": "flow-tension",
  "status": "ok",
  "data": {
    "fundingRate": 0.0003,
    "fundingRateFormatted": "+0.03%",
    "openInterest": 18500000000,
    "openInterestChange24h": 0.062,
    "openInterestChange24hFormatted": "+6.2%",
    "exchangeNetFlow": 0.14,
    "exchangeNetFlowFormatted": "+14.0%",
    "narrativeHooks": {
      "funding": "positive_funding",
      "openInterest": "oi_building",
      "flow": "buyers_aggressive"
    },
    "summary": "Leverage building with aggressive buying — tension coiling, breakout likely."
  },
  "meta": { "cacheHit": false, "ttl_remaining_sec": 3600 }
}
```

**Degraded mode** (no API key): `fundingRate`, `openInterest` fields = `null`, `status: "degraded"`, `meta.degradedReason` explains missing signals.

**Extended:** Adds `meta.sources` with per-source fetch metadata.

**Full:** Adds `data.thresholds` + `data.metricDescription`.

---

## Test Checklist (from original `calc_test.go`)

- [ ] `TestComputeExchangeNetFlow_BuyersAggressive` (70/30/100 → 0.40)
- [ ] `TestComputeExchangeNetFlow_SellersAggressive` (30/70/100 → -0.40)
- [ ] `TestComputeExchangeNetFlow_Neutral` (50/50/100 → 0.0)
- [ ] `TestComputeExchangeNetFlow_ZeroVolume` (0/0/0 → 0.0)
- [ ] `TestComputeFlowHook_*` (5 tests: boundaries + extremes)
- [ ] `TestComputeFundingRateHook_*` (9 tests: realistic CoinGecko rates, boundaries)
- [ ] `TestComputeOIChangeHook_*` (5 tests: building/stable/unwinding + boundaries)
- [ ] `TestComputeOIChange24h_*` (3 tests: normal/decline/zero-prev)
- [ ] `TestComputeSummary_*` (6 tests: all narrative combos)
- [ ] `TestCompute_FullSignals` (full input → all fields populated)
- [ ] `TestCompute_DegradedMode` (no API key → nil fields, unavailable hooks)
- [ ] `TestCompute_ZeroTotalVolume` (error case)
- [ ] `TestCompute_ThinCandle_*` (3 tests: low_confidence hook, summary, raw value still returned)
- [ ] `TestCompute_AtMinTradesThreshold_NotLowConfidence` (10 trades → normal hook)
- [ ] `TestCompute_BelowMinTradesThreshold_LowConfidence` (9 trades → low_confidence)

**Integration test:** `flow-tension_e2e_test.go` using `httptest.NewServer` to mock API responses, invoke via Cobra command.

---

## Calibration Notes (from original docs)

1. **BTC perpetual only** — most liquid, least manipulable instrument. Not coin-specific.
2. **Funding rate thresholds calibrated** against real CoinGecko data (range: -0.29% to +0.85%).
3. **Thin-candle guard** (`numTrades < 10`) prevents noise from low-activity periods.
4. **TTL = 3600s** — tactical signals, short staleness budget vs macro metrics.
5. **CVD is a proxy** — measures taker aggression within exchange, not true on-chain net flow.
6. **CoinGecko Pro preferred** over Binance Futures (broader market view, aggregated across exchanges).

---

## Porting Strategy to New Architecture

**Key differences:**
- Old: `func Compute(input Input) (Data, error)` → returns `Data` directly
- New: `func Compute(ctx context.Context, data map[string]json.RawMessage) (output.MetricResult, error)` → returns `MetricResult` envelope

**Steps:**
1. Copy threshold constants + helper functions from `calc.go` (they're pure math, no I/O)
2. In `Compute()`, unmarshal `data["binance.spot_cvd_btc_1h"]` and `data["coingecko.derivatives"]` into temp structs
3. Extract taker volumes, OI, funding rate from JSON
4. Call original compute helpers (adapted to work with extracted values)
5. Build `output.MetricResult` with `DetectStatus(confidence, thinData)` for status
6. Set `MetricResult.Data = json.Marshal(computedData)`

**Confidence calculation:** Use original logic — discrepancy detection not needed for this metric (single-exchange CVD primary, no cross-source validation in v1).

---

**End of reference document.**
