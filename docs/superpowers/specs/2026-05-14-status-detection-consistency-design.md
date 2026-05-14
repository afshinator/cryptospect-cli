# Status Detection Consistency — Unify All 4 Metrics on `DetectStatus()`

Unify the 4 metrics (FT, MB, MD, MR) that bypass `DetectStatus()` to use the
standard helper, matching LP/SP's pattern. Also deduplicate `confidenceToFloat()`
into a shared helper.

## Problem

Design-Decisions.md Step 12.8 states: *"Each metric's Compute function calls
DetectStatus to set the MetricResult.Status field."* Only LP and SP follow this.
The other 4 metrics have ad-hoc status logic:

| Metric | Current status logic | Problem |
|--------|---------------------|---------|
| FT | `if !cgAvailable → degraded else ok` | Ignores confidence entirely. `meta.Confidence` set to status string (bug per issue #11) |
| MB | Pure function sets `MetricStatus` from data-sufficiency (coins, timeframes) | Independent of `ValidatorConfidence`. Bypasses confidence entirely |
| MD | `if compMeta.Confidence == "low" → degraded else ok` | No "unavailable" path; doesn't use `DetectStatus` |
| MR | `if breadthDegraded → degraded else ok` | Ignores the `Confidence` that MR's own pure `Compute()` already calculates |

## Solution

### 1. Shared helpers in `internal/metrics/helpers.go`

Extract the duplicated `confidenceToFloat` (LP line 50, SP line 323 — identical)
into a public `ConfidenceToFloat`, and add the inverse `FloatToConfidence`:

```go
// ConfidenceToFloat maps a confidence label to a float.
func ConfidenceToFloat(conf string) float64 {
    switch conf {
    case "high":   return 0.9
    case "medium": return 0.6
    case "low":    return 0.3
    default:       return 0.0
    }
}

// FloatToConfidence maps a float back to a confidence label (inverse of above).
func FloatToConfidence(f float64) string {
    switch {
    case f >= 0.8: return "high"
    case f >= 0.5: return "medium"
    default:       return "low"
    }
}
```

LP and SP replace their private `confidenceToFloat` with the shared `metrics.ConfidenceToFloat`.

### 2. Per-metric changes

#### MD (momentum-divergence)

Current (`provider.go`):
```go
status := "ok"
if compMeta.Confidence == "low" {
    status = "degraded"
}
```

New:
```go
status := metrics.DetectStatus(metrics.ConfidenceToFloat(compMeta.Confidence), false)
```

Same behavior for "high"/"low". No "unavailable" path from confidence
(Compute either returns Data or errors → provider calls `unavailable()` directly).

**Files:** `provider.go`

---

#### FT (flow-tension)

Current (`provider.go`):
```go
status := "ok"
if !cgAvailable {
    status = "degraded"
}
meta.Confidence = status  // "ok"/"degraded" — wrong, should be "high"/"medium"
```

New model — confidence reflects **signal completeness**:
- All 3 signals (CVD + OI + Funding via CoinGecko) → conf 0.9 → "high"
- Only CVD (CoinGecko unavailable, OI/Funding absent) → conf 0.6 → "medium"
- Thin data (NumTrades < MinTrades) → thinData=true → downgrade one level

```go
conf := 0.9
if !cgAvailable {
    conf = 0.6
}
thinData := klines.NumTrades < MinTrades
status := metrics.DetectStatus(conf, thinData)
meta.Confidence = metrics.FloatToConfidence(conf)
```

**Files:** `provider.go`, `provider_test.go`

---

#### MR (market-regime)

Current (`provider.go`):
```go
mrStatus := "ok"
if breadthDegraded {
    mrStatus = "degraded"
}
```

New — uses `computed.Confidence` from the pure Compute function routed through `DetectStatus`. Breadth degraded triggers degradation; cold start and weight redistribution produce medium confidence but remain "ok":

```go
conf := 0.9
switch {
case breadthDegraded:
    conf = 0.5 // breadth data insufficient → degraded
case computed.Confidence == ConfidenceMedium:
    conf = 0.8 // cold start or weight redistribution → ok with caveats
}
status := metrics.DetectStatus(conf, false)
```

**Files:** `provider.go`



#### MB (market-breadth) — Option A

MB is unique: `MetricStatus` (data sufficiency) and `ValidatorConfidence` (source
agreement) are intentionally separate axes. Under Option A, we derive a combined
confidence from both and route through `DetectStatus`.

No `thinData` is passed — the data-sufficiency concern is already baked into the
confidence via the cascading switch:

```go
var baseConf float64
switch {
case result.MetricStatus == "unavailable":
    baseConf = 0.3
case result.ValidatorConfidence == "low":
    baseConf = 0.5  // stale validator → floor at degraded
case result.MetricStatus == "degraded":
    baseConf = 0.5  // dropped timeframes / few coins → degraded
default:
    baseConf = metrics.ConfidenceToFloat(result.ValidatorConfidence)
}
status := metrics.DetectStatus(baseConf, false)
```

Trace:

| Scenario | MetricStatus | ValidatorConf | baseConf | Result |
|----------|-------------|---------------|----------|--------|
| All good | ok | high | 0.9 | ok ✓ |
| Validator stale | ok | low | 0.5 | degraded ✓ |
| Directional dispute | ok | medium | 0.6 | degraded ✓ |
| Few coins | degraded | high | 0.5 | degraded ✓ |
| Dropped timeframe | degraded | medium | 0.5 | degraded ✓ |
| All absent | unavailable | low | 0.3 | unavailable ✓ |

This preserves MB's dual-axis logic while routing through `DetectStatus`.

**Files:** `provider.go`, `provider_test.go`

### 3. Test impacts

Each metric's provider tests asserting status values should still pass — the
output status should be the same for normal inputs. Any test that checks the
status string stays unchanged unless the new logic produces a different status
for edge cases.

- **FT**: `TestFlowTension_*` — may need a new test for thin-data (NumTrades < 10) producing `"unavailable"`
- **MB**: `TestMarketBreadth_*` — the stale-kline case changes from `"ok"` → `"degraded"` status. Update assertions.
- **MD**: `TestMomentumDivergence_*` — same behavior, just different code path.
- **MR**: `TestMarketRegime_*` — same behavior for normal cases.

## Files Changed

| File | Change |
|------|--------|
| `internal/metrics/helpers.go` | Add `ConfidenceToFloat()`, `FloatToConfidence()` |
| `internal/metrics/liquiditypulse/v1/provider.go` | Replace private `confidenceToFloat` with shared `metrics.ConfidenceToFloat` |
| `internal/metrics/stablecoinpower/v1/provider.go` | Same |
| `internal/metrics/flowtension/v1/provider.go` | Compute conf float from sources, call `DetectStatus`, fix `meta.Confidence` |
| `internal/metrics/marketbreadth/v1/provider.go` | Derive combined status from MetricStatus + ValidatorConfidence via `DetectStatus` |
| `internal/metrics/momentumdivergence/v1/provider.go` | Replace ad-hoc check with `DetectStatus` |
| `internal/metrics/marketregime/v1/provider.go` | Use `computed.Confidence` via `DetectStatus` |
| `internal/metrics/*/v1/*_test.go` | Update assertions where status output changes |

## Verification

1. `go test -race -count=1 ./internal/metrics/... ./cmd/...` — all pass
2. `go build -o /dev/null ./cmd/cryptospect-cli/` — compiles
3. `make lint` — 0 issues
4. The JSON `status` field in output should remain the same for normal inputs
