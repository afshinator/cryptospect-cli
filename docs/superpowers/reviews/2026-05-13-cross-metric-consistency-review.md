# Cross-Metric Consistency Review — 2026-05-13

Full review of all 6 metrics: code vs specs, cross-metric consistency, documentation gaps.

## Summary

All 6 metrics implement their individual specs correctly — thresholds, classification logic, and core formulas match documentation. The main issues are **cross-metric inconsistencies** in meta structures, status detection, confidence handling, and detail-level filtering.

---

## Issues by Priority

### 🔴 Blocking

#### 1. MD uses `map[string]interface{}` for meta — all other metrics use typed structs
**File:** `internal/metrics/momentumdivergence/v1/provider.go:109-140`
- LP, SP, FT, MB, MR: each has a typed Meta struct with compile-time guarantees
- MD: builds meta as `map[string]interface{}` with string keys
- Consequences: no field name safety, fragile interaction with root.go detail-level filter (relies on magic strings), violates project conventions

#### 2. Status detection bypasses `DetectStatus()` in 4 of 6 metrics
**Helper:** `internal/metrics/helpers.go` — `DetectStatus(confidence float64, thinData bool)`
- Only LP and SP call `DetectStatus()`
- **FT** (`provider.go:175`): hardcoded `"ok"` / `"degraded"` based on API availability
- **MB** (`compute.go`): status computed in pure function as `"ok"` / `"degraded"` / `"unavailable"` — bypasses confidence entirely
- **MD** (`provider.go:147`): `if compMeta.Confidence == "low" → "degraded" else "ok"`
- **MR** (`provider.go:2852`): `if breadthDegraded → "degraded" else "ok"` — ignores computed confidence

Design-Decisions.md Step 12.8: "Each metric's Compute function calls DetectStatus to set the MetricResult.Status field."

#### 3. MD's `description` meta field is misused
**File:** `internal/metrics/momentumdivergence/v1/provider.go:138`
- MD sets `metaMap["description"]` to `compMeta.LabelDescription` (e.g., "Risk-On Rotation")
- All other metrics put a long methodology description in this field
- root.go treats `description` as a full-detail field to strip at extended level
- MD's classification description should live in `data.classification.description`, not meta

#### 4. MR status ignores computed confidence
**File:** `internal/metrics/marketregime/v1/provider.go:2852`
- MR's pure Compute function calculates a weighted `Confidence` string (high/medium/low) considering cold start, weight redistribution, breadth degradation, missing ref data, capitulation notes
- Provider ignores this entirely for status — only checks `breadthDegraded`
- Example: cold start + weight redistribution + missing BTC ref = confidence "low" but status still "ok"

---

### 🟡 Suggestions

#### 5. `cache_hit` / `ttl_remaining_sec` only populated by MR
- `output/meta.go` defines `MetaBasic{CacheHit, TTLRemaining}` — not used by any metric
- Only MR populates actual cache_hit/ttl_remaining values
- LP, SP, MB: omit these fields entirely
- FT: omits entirely
- MD: hardcodes `false` / `0` — misleading
- Spec docs for LP, SP, MB mention these fields as part of output schema

#### 6. Duplicate `confidenceToFloat()` in LP and SP
- `liquiditypulse/v1/provider.go:60-71` and `stablecoinpower/v1/provider.go:276-284`
- Identical private function
- Should be in `internal/metrics/helpers.go`

#### 7. MR opens cache twice in provider
- `marketregime/v1/provider.go:2729` — opens cache for state read/write
- `provider.go:2753` — opens cache again for freshness metadata
- Should be a single open/close cycle

#### 8. `output/meta.go` types are dead code
- `MetaBasic`, `MetaExtended`, `MetaFull`, `SourceMeta` defined but never used
- Every metric defines its own Meta struct locally
- Either use via composition or remove

#### 9. MR uses `float64` for `market_breadth_score` in Data; MB uses `metrics.MetricFloat`
- `marketregime/v1/types.go:2251`: `MarketBreadthScore float64` → JSON: `{"market_breadth_score": 0.65}`
- `marketbreadth/v1/types.go:179`: `MarketBreadthScore metrics.MetricFloat` → JSON: `{"market_breadth_score": {"value": 0.65, "type": "ratio"}}`
- Loss of type info when MR outputs the same field name

#### 10. Detail-level filter in root.go is fragile
**File:** `cmd/cryptospect-cli/root.go:178-188`
- Strips `thresholds`, `description`, `top_n_stablecoins`, `tier_detail` from meta at extended level
- Hardcodes field names from specific metrics — won't catch renamed or differently structured fields
- Misses `weights_used` and `timeframe_counts` from MB (arguably full-detail)
- Should delegate filtering to providers or use typed interfaces

#### 11. FT Confidence field is set to status, not actual confidence
**File:** `internal/metrics/flowtension/v1/provider.go:182`
- `meta.Confidence = status` (where status is "ok" or "degraded")
- "degraded" is not a confidence level — should be "high"/"medium"/"low"

---

### 🔵 Nitpicks

#### 12. Inconsistent `unavailable()` method names
- LP, SP, MB, MR: `unavailable()`
- FT: `computeErr()`

#### 13. SP thresholds map duplicate values
- `stablecoinpower/v1/provider.go:491-495`: `"normal_min": 0.07` and `"low": 0.07` point to same value

#### 14. MR Classification.Label is regime name, not a categorical level
- LP: `"high"` / `"normal"` / `"low"` — categorical
- MB: `"broad"` / `"mixed"` / `"narrow"` — categorical
- MR: `"BTC-Led Expansion"` etc. — proper noun, not a level

#### 15. MD hardcodes `cache_hit: false, ttl_remaining_sec: 0`
- Always false/0 — either implement real cache lookups or omit

#### 16. Spec doc vs CLAUDE.md: "7-regime matrix" vs actual 10 regimes
- CLAUDE.md and Design-Decisions.md say "7-regime classification matrix"
- Actual code implements 10 regimes (3×3 dominance/breadth matrix + conviction-split on Capitulation)
- Spec doc `market-regime.md` correctly lists all 10

---

## Cross-Metric Meta Structure Comparison

| Metric | Meta type | Has cache_hit? | Has ttl_remaining? | Has PrimarySource? | Has Confidence? | Status method |
|--------|-----------|----------------|--------------------|--------------------|-----------------|---------------|
| LP | typed struct | ❌ | ❌ | ✅ | ✅ | DetectStatus |
| SP | typed struct | ❌ | ❌ | ✅ | ✅ | DetectStatus |
| FT | typed struct | ❌ | ❌ | ✅ (list) | ✅ (set to status) | manual |
| MB | typed struct | ❌ | ❌ | ✅ | ✅ (validator) | compute-internal |
| MD | map[string]any | fake | fake | ✅ | ✅ | manual |
| MR | typed struct (embedded) | ✅ | ✅ | ✅ | ✅ (compute-internal) | manual |

---

## Verdict

**Needs minor changes.** No single-metric spec is broken. Systemic inconsistency in meta structures, status detection, and confidence handling is the primary technical debt. MD's untyped meta map is the most urgent fix.
