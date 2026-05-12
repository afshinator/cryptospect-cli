# market-regime Implementation Plan

> **For agentic workers:** Read this entire document before writing any code. The Design Decisions section captures choices already made — do not re-open them. Use `docs/metrics/market-regime.md` as the authoritative spec. Run all verification steps in order at the end.

**Goal:** Replace the 47-line `market-regime` scaffold with a full v1.0.0 implementation: 2×3 dominance-trend × breadth-band regime matrix, BTC modifier, lp_ratio conviction gate, Capitulation disambiguation, cold-start handling, and detail-level meta output.

**Spec:** `docs/metrics/market-regime.md`
**Companion docs:** `docs/mr-proposal1.md` (earlier draft, superseded), `docs/summary.md`

**Stack:** Go 1.25, stdlib `testing`/`httptest`, cobra/viper, `encoding/json`, `time`

**Pattern:** Follows `ft`/`mb`/`md` — separate `types.go`, `compute.go`, `provider.go` plus test files.

---

## Design Decisions (already made — do not re-open)

1. **`prior_snapshot_age_sec` uses `Entry.FetchedAt`, not file ModTime.** No `Stat()` method needed on the cache package. `FetchedAt` is immune to NFS/Docker clock drift.

2. **Breadth computation: import `marketbreadth/v1.Compute()` directly.** Do not duplicate the parser or null-exclusion logic. Pass `KlineAvailable: false` and `TopN: 250`. Discard `mbResult.ValidatorConfidence` — mr drives its own confidence entirely.

3. **`divergent_momentum` note deferred to v1.1.** v1 is self-contained. Spec section "Future Enhancements" documents the planned implementation.

4. **File structure:** `types.go`, `compute.go`, `provider.go` + test files — same as ft/mb/md.

5. **`Classification` struct:** defined fresh in `marketregime/v1/types.go` — not imported from mb. Two-field struct `{Label, Description string}`.

6. **`cache_hit` / `ttl_remaining_sec`:** implement at extended detail by reading the `api.CoinGeckoGlobalMarket` cache entry while the cache is already open for the state key. Not heavyweight — cache is opened once, two `Get()` calls.

---

## Known Types and Constants (verified pre-implementation)

### API endpoint keys (`internal/api/constants.go`)
```go
api.CoinGeckoGlobalMarket        = "coingecko.global_market"
api.CoinGeckoCoinMarketsBreadth  = "coingecko.coin_markets_breadth"
```

### CoinGecko global parser (`internal/api/coingecko/client.go`)
```go
// Parse volume/mcap fields:
func ParseGlobalResponse(body []byte) (GlobalData, error)
type GlobalData struct {
    TotalMarketCap map[string]float64  // key "usd"
    TotalVolume    map[string]float64  // key "usd"
}
func (g *GlobalData) GetVolumeUSD() (float64, bool)
func (g *GlobalData) GetMarketCapUSD() (float64, bool)

// Parse BTC dominance (separate re-parse of same bytes):
func ParseGlobalDominance(body []byte) *float64  // nil if absent
```

### CoinGecko coin markets parser (`internal/api/coingecko/client.go`)
```go
func ParseCoinMarketsBreadthResponse(body []byte) (CoinMarketsBreadthData, error)

type CoinMarketsBreadthData struct {
    TimeframeCounts map[string]TimeframeMetric
    CoinCount       int
    CoinsWithData   int  // json:"-"
    BTCReference    BTCReference
}
type BTCReference struct {
    PriceChange24h float64  // json:"price_change_24h_pct"
    Available      bool     // json:"-"
}
type TimeframeMetric struct {
    GreenCount int
    TotalCount int
}
```

### Cache package (`internal/cache/cache.go`)
```go
func Open(path string) (*Cache, error)
func (c *Cache) Get(endpoint string) (Entry, error)
func (c *Cache) Set(endpoint string, data []byte, ttl int) error
func (c *Cache) Close() error

type Entry struct {
    Data       []byte
    Found      bool
    Stale      bool
    FetchedAt  time.Time
    TTLSeconds int
}
// ExpiresAt = FetchedAt + TTLSeconds; compute ttl_remaining_sec as:
// int(entry.FetchedAt.Add(time.Duration(entry.TTLSeconds) * time.Second).Unix() - now.Unix())
// clamp to 0 if negative
```

### Config helpers (`internal/config/`)
```go
cfg, ok := config.FromContext(ctx)   // ok=false → skip cache ops, no crash
cfg.CacheDir() string
detail, ok := config.DetailFromContext(ctx)  // "basic" | "extended" | "full"
```

### Market-breadth compute (`internal/metrics/marketbreadth/v1/`)
```go
// Import alias: mbv1 "github.com/afshinator/cryptospect-cli/internal/metrics/marketbreadth/v1"

func Compute(input *Input) (ComputeResult, error)

type Input struct {
    TimeframeCounts map[string]coingecko.TimeframeMetric
    CoinsCounted    int       // use cgData.CoinsWithData
    BTCChange24h    float64   // use cgData.BTCReference.PriceChange24h
    BTCAvailable    bool      // use cgData.BTCReference.Available
    KlineClose      float64   // pass 0.0
    KlineOpen       float64   // pass 0.0
    KlineOpenTimeMs int64     // pass 0
    KlineAvailable  bool      // pass false — suppresses Binance validator
    TopN            int       // pass 250
    Now             time.Time
}

// Fields used from ComputeResult:
//   MarketBreadthScore metrics.MetricFloat  → float64(result.MarketBreadthScore)
//   Classification.Label string             → breadth band ("broad"/"mixed"/"narrow")
//   MetricStatus string                     → "ok"/"degraded"/"unavailable" (json:"-")
//   WeightsUsed map[string]float64          → check for 0.0 values to detect redistribution
// Field to discard:
//   ValidatorConfidence                     → ignored entirely

// Breadth band constants (re-export your own; do not import mb's consts):
// BroadThreshold  = 0.60  (mbv1.BroadThreshold)
// NarrowThreshold = 0.40  (mbv1.NarrowThreshold)
// StatisticalFloor = 50   (mbv1.StatisticalFloor)
```

### State cache key for prior dominance
```
key: "marketregime_dominance_pct"
TTL: 172800  (48h — survives normal 4h polling gaps)
value: JSON-encoded float64 (the current btc_dominance_pct)
```

### Flow-tension provider as reference for cache access pattern
See `internal/metrics/flowtension/v1/provider.go:86-108` — same `config.FromContext → cfg.CacheDir() → cache.Open → c.Get → c.Set → c.Close()` pattern used by mr.

---

## Files

| # | File | Action |
|---|------|--------|
| 1 | `internal/metrics/marketregime/v1/types.go` | Create |
| 2 | `internal/metrics/marketregime/v1/compute.go` | Create |
| 3 | `internal/metrics/marketregime/v1/compute_test.go` | Create |
| 4 | `internal/metrics/marketregime/v1/provider.go` | Replace scaffold |
| 5 | `internal/metrics/marketregime/v1/provider_test.go` | Replace scaffold |
| 6 | `cmd/cryptospect-cli/testdata/market_regime_global.json` | Create fixture |
| 7 | `cmd/cryptospect-cli/testdata/market_regime_coin_markets.json` | Create fixture |
| 8 | `cmd/cryptospect-cli/market_regime_e2e_test.go` | Create |

---

## Task 1: types.go

**File:** `internal/metrics/marketregime/v1/types.go`

- [ ] **Step 1: Package declaration and imports**

```go
// Package v1 implements the market-regime metric.
package v1

import "time"
```

- [ ] **Step 2: Metric identity constants**

```go
const (
    MetricName    = "market-regime"
    MetricVersion = "v1.0.0"

    stateKey = "marketregime_dominance_pct"
    stateTTL = 172800 // 48h
)
```

- [ ] **Step 3: Regime label constants**

```go
const (
    RegimeBTCLedExpansion    = "BTC-Led Expansion"
    RegimeInstitutionalBuild = "Institutional Build"
    RegimeFlightToSafety     = "Flight to Safety"
    RegimeSteadyAppreciation = "Steady Appreciation"
    RegimeConsolidation      = "Consolidation"
    RegimeStagnation         = "Stagnation"
    RegimeAltSeasonMania     = "Alt-Season / Mania"
    RegimeCapitalRotation    = "Capital Rotation"
    RegimeCapitulation       = "Capitulation"
    RegimeStructuralDecay    = "Structural Decay"
)
```

- [ ] **Step 4: Signal value constants**

```go
const (
    ModifierPositiveMomentum = "positive_momentum"
    ModifierNegativePressure = "negative_pressure"
    ModifierNeutral          = "neutral"

    ConvictionHigh   = "high"
    ConvictionNormal = "normal"
    ConvictionLow    = "low"

    TrendRising  = "rising"
    TrendFalling = "falling"
    TrendNeutral = "neutral"

    BreadthBroad  = "broad"
    BreadthMixed  = "mixed"
    BreadthNarrow = "narrow"
)
```

- [ ] **Step 5: Threshold constants**

```go
const (
    DomDeadBandPP     = 0.5   // percentage points
    BTCDirDeadBandPct = 0.5   // percent
    ConvictionHighMin = 0.15  // lp_ratio threshold for "high"
    ConvictionLowMax  = 0.07  // lp_ratio threshold for "low"
    BreadthBroadMin   = 0.60  // aligned with market-breadth
    BreadthNarrowMax  = 0.40  // aligned with market-breadth
    MaxSnapshotAgeSec = 86400 // 24h; prior snapshot older than this → cold start
    CacheHintSec      = 14400 // recommended agent call frequency
)
```

- [ ] **Step 6: Note key constants**

```go
const (
    NotesColdStart               = "cold_start"
    NotesWeightRedistribution    = "weight_redistribution"
    NotesAbnormalCapitulation    = "abnormal_capitulation"
    NotesCapitulationStabilizing = "capitulation_price_stabilizing"
    NotesMissingReferenceData    = "missing_reference_data"
)
```

- [ ] **Step 7: Input struct**

All fields are pre-parsed by the provider. Compute() is pure (no I/O).

```go
// Input holds all pre-parsed signals needed by Compute.
type Input struct {
    // Signal 1: BTC dominance
    BTCDominancePct     float64
    PriorDominancePct   *float64 // nil = cold start or stale snapshot
    PriorSnapshotAgeSec *int     // nil = cold start; bounds-checked by provider before passing

    // Signal 2: Market breadth (from marketbreadth/v1.Compute())
    BreadthScore        float64
    BreadthBand         string // "broad" | "mixed" | "narrow"
    BreadthStatus       string // "ok" | "degraded" | "unavailable"
    WeightsUsed         map[string]float64
    WeightRedistributed bool // true if any WeightsUsed value is 0.0

    // Signal 3: Conviction
    LPRatio float64 // zero-guard applied by provider; safe to use

    // Modifier
    BTCChange24h *float64 // nil = missing_reference_data

    Now time.Time
}
```

- [ ] **Step 8: ComputeResult struct**

```go
// ComputeResult is the full output of Compute. Provider uses it to build
// both the data and meta JSON sections.
type ComputeResult struct {
    // → data section
    Regime             string
    Modifier           string
    DominanceTrend     string
    Conviction         string
    MarketBreadthScore float64
    Classification     Classification
    Summary            string

    // → meta (basic and above)
    BTCDominancePct    float64
    BTCChange24h       *float64 // nil when missing_reference_data
    Confidence         string
    DominanceColdStart bool
    Notes              []string // always initialised; never nil
    CacheHintSec       int

    // → meta (extended and above)
    LPRatio             float64
    DominanceDelta      *float64 // nil on cold start
    PriorSnapshotAgeSec *int     // nil on cold start
    WeightsUsed         map[string]float64

    // → internal (provider uses for MetricResult.Status; not marshalled)
    MetricStatus string
}
```

- [ ] **Step 9: Data and Classification structs**

```go
// Data is the JSON-serialisable data section of the output envelope.
type Data struct {
    Regime             string         `json:"regime"`
    Modifier           string         `json:"modifier"`
    DominanceTrend     string         `json:"dominance_trend"`
    Conviction         string         `json:"conviction"`
    MarketBreadthScore float64        `json:"market_breadth_score"`
    Classification     Classification `json:"classification"`
    Summary            string         `json:"summary"`
}

// Classification holds the categorical output label and description.
type Classification struct {
    Label       string `json:"label"`
    Description string `json:"description"`
}
```

- [ ] **Step 10: Meta structs**

```go
// MetaBasic is always present (--detail basic and above).
type MetaBasic struct {
    BTCDominancePct    float64  `json:"btc_dominance_pct"`
    BTCChange24h       *float64 `json:"btc_24h_change,omitempty"`
    Confidence         string   `json:"confidence"`
    DominanceColdStart bool     `json:"dominance_cold_start"`
    Notes              []string `json:"notes"`
    CacheHintSec       int      `json:"cache_hint_sec"`
}

// MetaExtended adds fields at --detail extended.
type MetaExtended struct {
    MetaBasic
    CacheHit            bool               `json:"cache_hit"`
    TTLRemainingSec     int                `json:"ttl_remaining_sec"`
    PrimarySource       string             `json:"primary_source"`
    LPRatio             float64            `json:"lp_ratio"`
    DominanceDelta      *float64           `json:"dominance_delta_since_last_fetch,omitempty"`
    PriorSnapshotAgeSec *int               `json:"prior_snapshot_age_sec,omitempty"`
    WeightsUsed         map[string]float64 `json:"weights_used"`
}

// MetaFull adds fields at --detail full.
type MetaFull struct {
    MetaExtended
    Thresholds  map[string]float64 `json:"thresholds"`
    Description string             `json:"description"`
}
```

---

## Task 2: compute.go

**File:** `internal/metrics/marketregime/v1/compute.go`

**Imports:** `fmt`, `github.com/afshinator/cryptospect-cli/internal/metrics/marketbreadth/v1` (alias `mbv1`)

Note: the mbv1 import is for calling `mbv1.Compute()` from the provider, but `compute.go` itself does not call it — the provider does. `compute.go` only needs `fmt` (for summary strings). Remove the mbv1 import from compute.go if unused there.

- [ ] **Step 1: `Compute(input Input) (ComputeResult, error)` — main function**

Sequence:
1. Initialise `notes := []string{}`
2. Call `dominanceTrendAndDelta` → trend, delta, coldStart bool
3. If coldStart: set `DominanceColdStart: true`, append `NotesColdStart`
4. If `input.WeightRedistributed`: append `NotesWeightRedistribution`
5. Call `classifyConviction(input.LPRatio)` → conviction string
6. Call `matrixLookup(trend, input.BreadthBand, conviction)` → regime string
7. Call `classifyModifier(input.BTCChange24h)` → modifier string, missingRef bool
8. If missingRef: append `NotesMissingReferenceData`
9. If regime == Capitulation: apply sub-state note logic
10. Call `determineConfidence(...)` → confidence string
11. MetricStatus: "unavailable" if BreadthStatus=="unavailable", "degraded" if BreadthStatus=="degraded", else "ok"
12. Build Classification (label = regime, description from `classificationDescription`)
13. Build summary from `buildSummary`
14. Return ComputeResult

- [ ] **Step 2: `dominanceTrendAndDelta`**

```go
func dominanceTrendAndDelta(current float64, prior *float64, ageSec *int) (trend string, delta *float64, coldStart bool) {
    if prior == nil {
        return TrendNeutral, nil, true
    }
    // ageSec is already bounds-checked by provider (negative → nil, >86400 → nil)
    // so if prior != nil here, ageSec is valid
    d := current - *prior
    delta = &d
    switch {
    case d >= DomDeadBandPP:
        trend = TrendRising
    case d <= -DomDeadBandPP:
        trend = TrendFalling
    default:
        trend = TrendNeutral
    }
    return trend, delta, false
}
```

- [ ] **Step 3: `classifyConviction`**

```go
func classifyConviction(lpRatio float64) string {
    switch {
    case lpRatio > ConvictionHighMin:
        return ConvictionHigh
    case lpRatio >= ConvictionLowMax:
        return ConvictionNormal
    default:
        return ConvictionLow
    }
}
```

- [ ] **Step 4: `classifyModifier`**

```go
// Returns modifier string and whether BTC reference data was missing.
func classifyModifier(btcChange *float64) (string, bool) {
    if btcChange == nil {
        return ModifierNeutral, true
    }
    switch {
    case *btcChange >= BTCDirDeadBandPct:
        return ModifierPositiveMomentum, false
    case *btcChange <= -BTCDirDeadBandPct:
        return ModifierNegativePressure, false
    default:
        return ModifierNeutral, false
    }
}
```

- [ ] **Step 5: `matrixLookup`**

2×3 matrix with Capitulation disambiguation. Use a struct key or nested switch:

```go
func matrixLookup(trend, band, conviction string) string {
    switch trend {
    case TrendRising:
        switch band {
        case BreadthBroad:  return RegimeBTCLedExpansion
        case BreadthMixed:  return RegimeInstitutionalBuild
        default:            return RegimeFlightToSafety
        }
    case TrendNeutral:
        switch band {
        case BreadthBroad:  return RegimeSteadyAppreciation
        case BreadthMixed:  return RegimeConsolidation
        default:            return RegimeStagnation
        }
    default: // TrendFalling
        switch band {
        case BreadthBroad:  return RegimeAltSeasonMania
        case BreadthMixed:  return RegimeCapitalRotation
        default:
            // Capitulation disambiguation (Falling + Narrow only)
            if conviction == ConvictionHigh {
                return RegimeCapitulation
            }
            return RegimeStructuralDecay
        }
    }
}
```

- [ ] **Step 6: `classificationDescription`**

Static map; two conviction-conditional branches (Consolidation, Stagnation):

```go
func classificationDescription(regime, conviction string) string {
    switch regime {
    case RegimeConsolidation:
        if conviction == ConvictionHigh {
            return "Consolidation (Pressure Cooker) — high volume range-bound trading; imminent violent break likely"
        }
        return "Consolidation — market seeking direction, mixed participation"
    case RegimeStagnation:
        if conviction == ConvictionHigh {
            return "Stagnation (Pressure Cooker) — high volume, directionless market; explosive break likely in either direction"
        }
        return "Stagnation — flat dominance, narrow breadth, market ignored"
    case RegimeBTCLedExpansion:
        return "BTC-Led Expansion — broad participation with rising BTC dominance"
    case RegimeInstitutionalBuild:
        return "Institutional Build — BTC outperforming with mixed alt participation"
    case RegimeFlightToSafety:
        return "Flight to Safety — capital concentrating in BTC, alts bleeding"
    case RegimeSteadyAppreciation:
        return "Steady Appreciation — balanced market, broad participation, no dominance shift"
    case RegimeAltSeasonMania:
        return "Alt-Season / Mania — capital rotating down the risk curve with broad participation"
    case RegimeCapitalRotation:
        return "Capital Rotation — falling dominance with selective participation"
    case RegimeCapitulation:
        return "Capitulation — panic selling, high volume, alts collapsing"
    default: // RegimeStructuralDecay
        return "Structural Decay — slow bleed, falling dominance, thin volume"
    }
}
```

- [ ] **Step 7: `determineConfidence`**

Start at "high"; apply minimum across all applicable conditions:

```go
// Conditions that lower confidence. Evaluate all; take the minimum.
func determineConfidence(coldStart, weightRedistributed, missingRef, degraded bool, regime, modifier string) string {
    level := "high"
    lower := func(to string) {
        if to == "low" || (to == "medium" && level == "high") {
            level = to
        }
    }
    if coldStart           { lower("medium") }
    if weightRedistributed { lower("medium") }
    if missingRef          { lower("low") }
    if degraded            { lower("low") }
    if regime == RegimeCapitulation {
        switch modifier {
        case ModifierNeutral:          lower("medium")
        case ModifierPositiveMomentum: lower("medium")
        // ModifierNegativePressure keeps "high" — panic confirmed
        }
    }
    return level
}
```

- [ ] **Step 8: `buildSummary`**

Per-label NL synthesis. Must branch on conviction for Stagnation and Consolidation, and on modifier for Capitulation. Keep strings concise — one or two sentences. See spec `data.summary` examples for tone. Do not reference numeric deltas (delta is extended-only). Example skeletons:

```
BTC-Led Expansion + positive_momentum:
  "Dominance rising, breadth broad — BTC-Led Expansion with positive momentum. Capital flowing into BTC with broad alt participation."

Stagnation + high conviction:
  "Dominance neutral, breadth narrow, high conviction — Stagnation (Pressure Cooker). High volume in a directionless market; breakout likely in either direction."

Capitulation + capitulation_price_stabilizing:
  "Falling dominance, narrow breadth, high volume — Capitulation. BTC price stabilising against ongoing alt bleed; shift from observation toward cautious scaling."
```

Write a switch on `regime` with nested switches on `modifier` and `conviction` where needed. Cover all 10 labels.

---

## Task 3: compute_test.go

**File:** `internal/metrics/marketregime/v1/compute_test.go`

Table-driven, stdlib only (`testing`). Use `t.Run` sub-tests.

- [ ] **Step 1: Helper `mkInput`** — returns a valid base Input (non-cold-start, high breadth, normal conviction, positive momentum, all signals available). Tests override individual fields.

- [ ] **Step 2: All 10 regime labels** — one sub-test per cell. Verify `result.Regime`, `result.DominanceTrend`, `result.Conviction`, `result.Classification.Label`.

- [ ] **Step 3: Capitulation disambiguation** — two sub-tests: `conviction=="high"` → Capitulation; `conviction=="normal"` → Structural Decay; `conviction=="low"` → Structural Decay.

- [ ] **Step 4: Capitulation sub-state notes** — three sub-tests on modifier: negative_pressure (no note, confidence "high"); neutral ("capitulation_price_stabilizing", confidence "medium"); positive_momentum ("abnormal_capitulation", confidence "medium").

- [ ] **Step 5: Cold start** — `PriorDominancePct: nil` → `DominanceColdStart: true`, `"cold_start"` in notes, `DominanceDelta: nil`, trend "neutral", confidence "medium".

- [ ] **Step 6: Dead-band boundaries** — delta exactly `+0.5` → "rising"; `+0.49` → "neutral"; `-0.5` → "falling"; `-0.49` → "neutral". BTC change exactly `+0.5` → positive_momentum; `-0.5` → negative_pressure.

- [ ] **Step 7: Weight redistribution** — `WeightRedistributed: true` → `"weight_redistribution"` in notes, confidence ≤ "medium".

- [ ] **Step 8: Missing BTC reference** — `BTCChange24h: nil` → modifier "neutral", `"missing_reference_data"` in notes, `BTCChange24h` in result is nil, confidence "low".

- [ ] **Step 9: Pressure Cooker branches** — Stagnation (neutral+narrow) + high conviction → description contains "Pressure Cooker", summary mentions breakout. Consolidation (neutral+mixed) + high conviction → same.

- [ ] **Step 10: Degraded breadth** — `BreadthStatus: "degraded"` → `MetricStatus: "degraded"`, confidence "low".

- [ ] **Step 11: Multiple simultaneous conditions** — cold start + missing ref → confidence "low" (missing ref wins). Cold start + weight redistribution → confidence "medium".

- [ ] **Step 12: Notes always initialised** — fresh call with no conditions → `result.Notes` is non-nil empty slice, not nil.

---

## Task 4: provider.go

**File:** `internal/metrics/marketregime/v1/provider.go`

**Imports:** `context`, `encoding/json`, `fmt`, `time`, `github.com/afshinator/cryptospect-cli/internal/api`, `internal/api/coingecko`, `internal/cache`, `internal/config`, `internal/metrics`, `internal/output`, `internal/metrics/marketbreadth/v1` (alias `mbv1`)

- [ ] **Step 1: Package, init, Provider struct, Def()**

```go
func init() { metrics.MustRegister(&Provider{}) }

type Provider struct{}

func (p *Provider) Def() metrics.MetricDef {
    return metrics.MetricDef{
        Name:      MetricName,
        Namespace: metrics.CoreNamespace,
        Version:   MetricVersion,
        Aliases:   []string{"mr"},
        Endpoints: []string{
            api.CoinGeckoGlobalMarket,
            api.CoinGeckoCoinMarketsBreadth,
        },
        Description: "Composite macro regime classification: dominance trend × market breadth, gated by BTC direction and liquidity conviction.",
    }
}
```

- [ ] **Step 2: `computeErr` helper** — same pattern as ft/mb/md.

- [ ] **Step 3: Parse global market data**

```go
globalRaw, ok := data[api.CoinGeckoGlobalMarket]
if !ok || len(globalRaw) == 0 {
    return p.computeErr("missing global market data")
}
globalData, err := coingecko.ParseGlobalResponse(globalRaw)
if err != nil {
    return p.computeErr(fmt.Sprintf("parsing global response: %v", err))
}
volUSD, volOK := globalData.GetVolumeUSD()
mcapUSD, mcapOK := globalData.GetMarketCapUSD()
if !volOK || !mcapOK || volUSD == 0 || mcapUSD == 0 {
    return p.computeErr("global market data missing volume or market cap (zero-guard)")
}
lpRatio := volUSD / mcapUSD

btcDominance := coingecko.ParseGlobalDominance(globalRaw)
if btcDominance == nil {
    return p.computeErr("BTC dominance unavailable from global response")
}
```

- [ ] **Step 4: Parse coin markets breadth data**

```go
marketsRaw, ok := data[api.CoinGeckoCoinMarketsBreadth]
if !ok || len(marketsRaw) == 0 {
    return p.computeErr("missing coin markets breadth data")
}
cgData, err := coingecko.ParseCoinMarketsBreadthResponse(marketsRaw)
if err != nil {
    return p.computeErr(fmt.Sprintf("parsing coin markets response: %v", err))
}

var btcChange24h *float64
if cgData.BTCReference.Available {
    v := cgData.BTCReference.PriceChange24h
    btcChange24h = &v
}
```

- [ ] **Step 5: State cache — read prior dominance, write current**

```go
var priorDominance *float64
var priorSnapshotAgeSec *int
var cacheHit bool
var ttlRemainingSec int

cfg, hasCfg := config.FromContext(ctx)
if hasCfg {
    c, err := cache.Open(cfg.CacheDir())
    if err == nil {
        // Read prior dominance state
        stateEntry, err := c.Get(stateKey)
        if err == nil && stateEntry.Found {
            var stored float64
            if json.Unmarshal(stateEntry.Data, &stored) == nil {
                now := time.Now()
                age := int(now.Unix() - stateEntry.FetchedAt.Unix())
                if age < 0 {
                    // clock skew: treat as cold start
                } else if age > MaxSnapshotAgeSec {
                    // stale: treat as cold start
                } else {
                    priorDominance = &stored
                    priorSnapshotAgeSec = &age
                }
            }
        }
        // Write current dominance to state cache
        if curRaw, err := json.Marshal(*btcDominance); err == nil {
            _ = c.Set(stateKey, curRaw, stateTTL)
        }
        // Read global market endpoint entry for cache_hit / ttl_remaining_sec
        globalEntry, err := c.Get(api.CoinGeckoGlobalMarket)
        if err == nil && globalEntry.Found {
            cacheHit = !globalEntry.Stale
            remaining := int(globalEntry.FetchedAt.Add(
                time.Duration(globalEntry.TTLSeconds) * time.Second,
            ).Unix() - time.Now().Unix())
            if remaining < 0 {
                remaining = 0
            }
            ttlRemainingSec = remaining
        }
        _ = c.Close()
    }
}
```

- [ ] **Step 6: Compute market breadth via mb.Compute()**

```go
mbInput := mbv1.Input{
    TimeframeCounts: cgData.TimeframeCounts,
    CoinsCounted:    cgData.CoinsWithData,
    BTCChange24h:    cgData.BTCReference.PriceChange24h,
    BTCAvailable:    cgData.BTCReference.Available,
    KlineAvailable:  false,
    TopN:            250,
    Now:             time.Now(),
}
mbResult, err := mbv1.Compute(&mbInput)
if err != nil {
    return p.computeErr(fmt.Sprintf("breadth compute: %v", err))
}

weightRedistributed := false
for _, w := range mbResult.WeightsUsed {
    if w == 0.0 {
        weightRedistributed = true
        break
    }
}
```

- [ ] **Step 7: Build mr.Input and call Compute()**

```go
input := Input{
    BTCDominancePct:     *btcDominance,
    PriorDominancePct:   priorDominance,
    PriorSnapshotAgeSec: priorSnapshotAgeSec,

    BreadthScore:        float64(mbResult.MarketBreadthScore),
    BreadthBand:         mbResult.Classification.Label,
    BreadthStatus:       mbResult.MetricStatus,
    WeightsUsed:         mbResult.WeightsUsed,
    WeightRedistributed: weightRedistributed,

    LPRatio:      lpRatio,
    BTCChange24h: btcChange24h,
    Now:          time.Now(),
}
result, err := Compute(input)
if err != nil {
    return p.computeErr(fmt.Sprintf("compute: %v", err))
}
```

- [ ] **Step 8: Marshal data JSON**

```go
d := Data{
    Regime:             result.Regime,
    Modifier:           result.Modifier,
    DominanceTrend:     result.DominanceTrend,
    Conviction:         result.Conviction,
    MarketBreadthScore: result.MarketBreadthScore,
    Classification:     result.Classification,
    Summary:            result.Summary,
}
dJSON, err := json.Marshal(d)
if err != nil {
    return p.computeErr(fmt.Sprintf("marshalling data: %v", err))
}
```

- [ ] **Step 9: Marshal meta JSON at appropriate detail level**

```go
detail, _ := config.DetailFromContext(ctx)

basic := MetaBasic{
    BTCDominancePct:    result.BTCDominancePct,
    BTCChange24h:       result.BTCChange24h,
    Confidence:         result.Confidence,
    DominanceColdStart: result.DominanceColdStart,
    Notes:              result.Notes,
    CacheHintSec:       CacheHintSec,
}

var metaJSON []byte
switch detail {
case "full":
    m := MetaFull{
        MetaExtended: MetaExtended{
            MetaBasic:           basic,
            CacheHit:            cacheHit,
            TTLRemainingSec:     ttlRemainingSec,
            PrimarySource:       "coingecko",
            LPRatio:             result.LPRatio,
            DominanceDelta:      result.DominanceDelta,
            PriorSnapshotAgeSec: result.PriorSnapshotAgeSec,
            WeightsUsed:         result.WeightsUsed,
        },
        Thresholds: map[string]float64{
            "dom_dead_band_pp":      DomDeadBandPP,
            "breadth_broad":         BreadthBroadMin,
            "breadth_narrow":        BreadthNarrowMax,
            "btc_dir_dead_band_pct": BTCDirDeadBandPct,
            "conviction_high":       ConvictionHighMin,
            "conviction_low":        ConvictionLowMax,
            "capitulation_vol_min":  ConvictionHighMin,
        },
        Description: longDescription,
    }
    metaJSON, err = json.Marshal(m)
case "extended":
    m := MetaExtended{
        MetaBasic:           basic,
        CacheHit:            cacheHit,
        TTLRemainingSec:     ttlRemainingSec,
        PrimarySource:       "coingecko",
        LPRatio:             result.LPRatio,
        DominanceDelta:      result.DominanceDelta,
        PriorSnapshotAgeSec: result.PriorSnapshotAgeSec,
        WeightsUsed:         result.WeightsUsed,
    }
    metaJSON, err = json.Marshal(m)
default:
    metaJSON, err = json.Marshal(basic)
}
if err != nil {
    return p.computeErr(fmt.Sprintf("marshalling meta: %v", err))
}
```

- [ ] **Step 10: `longDescription` constant**

Add at package level — the Long Description text from `docs/metrics/market-regime.md` (the "High-level meaning and value" section), condensed to 2–3 sentences suitable for `--detail full` output. Example:

```go
const longDescription = "Market Regime is the structural context layer of the suite. " +
    "It classifies the market's current phase using a 2×3 matrix of BTC dominance trend " +
    "(rising/neutral/falling) against market breadth (broad/mixed/narrow), gated by BTC price " +
    "direction and liquidity conviction. Run market-regime first to establish macro context " +
    "before interpreting other suite metrics."
```

- [ ] **Step 11: Return MetricResult**

```go
return output.MetricResult{
    Metric:  MetricName,
    Version: MetricVersion,
    Status:  result.MetricStatus,
    Data:    json.RawMessage(dJSON),
    Meta:    json.RawMessage(metaJSON),
}, nil
```

---

## Task 5: provider_test.go

**File:** `internal/metrics/marketregime/v1/provider_test.go`

Follow the mock-injection pattern from `internal/metrics/marketbreadth/v1/provider_test.go` — inject raw JSON bytes directly into `data map[string]json.RawMessage`, call `p.Compute(ctx, data)`, assert on the unmarshalled result.

- [ ] **Step 1: Fixtures** — define minimal valid JSON strings for global and coin markets responses inline. Global must include `data.market_cap_percentage.btc`, `data.total_volume.usd`, `data.total_market_cap.usd`. Coin markets must include a BTC entry (`id: "bitcoin"`) with `price_change_percentage_24h` and all four price change timeframe fields for ≥50 coins (can be synthetic).

- [ ] **Step 2: Happy path** — prior dominance in state cache → status "ok", `dominance_cold_start: false` in meta.

- [ ] **Step 3: Cold start** — no prior state cache → status "ok", meta has `dominance_cold_start: true`, `"cold_start"` in notes.

- [ ] **Step 4: Zero-guard triggers unavailable** — global fixture with `total_volume: {}` → status "unavailable".

- [ ] **Step 5: Parse failure on global** → status "unavailable".

- [ ] **Step 6: Breadth degraded** — coin markets with <50 coins → status "degraded".

---

## Task 6: Fixtures

**Files:**
- `cmd/cryptospect-cli/testdata/market_regime_global.json`
- `cmd/cryptospect-cli/testdata/market_regime_coin_markets.json`

- [ ] **Step 1: `market_regime_global.json`**

Realistic CoinGecko `/global` response. Must include (nested under `"data"`):
- `"market_cap_percentage": {"btc": 52.41, ...}`
- `"total_volume": {"usd": 125000000000}`
- `"total_market_cap": {"usd": 2800000000000}`

Resulting lp_ratio ≈ 0.044 (conviction "low"). Adjust values to produce a specific regime in the happy-path E2E test.

- [ ] **Step 2: `market_regime_coin_markets.json`**

Synthetic array of 250 coin entries. Must include:
- Entry with `"id": "bitcoin"`, `"price_change_percentage_24h": 1.2`
- All four price change timeframe fields on each entry (use realistic mix of positive/negative)
- Enough non-null values per timeframe to avoid weight redistribution (≥50 per timeframe)

Reuse the mb fixture from `cmd/cryptospect-cli/testdata/` if it satisfies the above, or create a purpose-built one.

---

## Task 7: market_regime_e2e_test.go

**File:** `cmd/cryptospect-cli/market_regime_e2e_test.go`

Follow the hardened pattern from `cmd/cryptospect-cli/market_breadth_e2e_test.go` — httptest servers, status tolerant of "ok" or "error", bounds guards on `resp.Results[0]` before accessing.

- [ ] **Step 1: `TestMarketRegimeCommand`**

Status "ok" or "error", `len(resp.Results) > 0`, `resp.Results[0]` accessible. When status "ok" or "degraded": verify `data.regime` is non-empty string, `data.modifier` is one of the three valid values, `meta` is non-nil.

- [ ] **Step 2: `TestMarketRegimeAlias`**

Run with alias `mr`. Assert identical output shape to `market-regime`.

- [ ] **Step 3: `TestMarketRegimeDetailExtended`**

When status "ok" or "degraded" and `meta` non-nil: unmarshal meta; assert `weights_used` is a non-nil map with keys "1h", "24h", "7d", "30d". Assert `cache_hint_sec == 14400`.

- [ ] **Step 4: `TestMarketRegimeDetailFull`**

When status "ok" or "degraded" and `meta` non-nil: assert `thresholds` object is present and `thresholds["breadth_broad"] == 0.60`.

---

## Verification Sequence

Run these in order. All must pass clean before the task is complete.

```bash
# 1. Build
go build -o ./cryptospect-cli ./cmd/cryptospect-cli/

# 2. Unit + provider tests for the new metric
go test -race -count=1 ./internal/metrics/marketregime/...

# 3. E2E tests for the new metric only
go test -race -count=1 ./cmd/cryptospect-cli/ -run TestMarketRegime

# 4. Full test suite — no regressions
go test -race -count=1 ./...

# 5. Lint
golangci-lint run ./...

# 6. Format checks (both must produce no output)
gofumpt -l cmd/ internal/
goimports -l cmd/ internal/

# 7. Live smoke tests
./cryptospect-cli mr
./cryptospect-cli market-regime --detail extended
./cryptospect-cli market-regime --detail full
```

Expected live output:
- `mr`: JSON with `data.regime` (one of 10 labels), `data.modifier`, `data.conviction`, `data.market_breadth_score`
- `--detail extended`: meta includes `dominance_cold_start`, `weights_used`, `lp_ratio`; on first run `dominance_cold_start: true`
- `--detail full`: meta includes `thresholds` map with 7 keys

---

## Progress

- [ ] Task 1: types.go
- [ ] Task 2: compute.go
- [ ] Task 3: compute_test.go
- [ ] Task 4: provider.go
- [ ] Task 5: provider_test.go
- [ ] Task 6: Fixtures
- [ ] Task 7: market_regime_e2e_test.go
- [ ] Verification: all steps pass
