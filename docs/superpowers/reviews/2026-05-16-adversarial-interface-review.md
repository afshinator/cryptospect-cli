# Adversarial Interface Review — 2026-05-16

Full adversarial review of metric interfaces, CLI help output, output envelope, and LLM agent consumption patterns across all 6 metrics. No code changes — findings only.

## Summary

All 6 metrics follow the same `MetricProvider` interface and output contract. The core architecture (registry, dispatcher, detail levels, DetectStatus) is solid. However, there are **1 🔴 bug**, **6 🟡 cross-metric inconsistencies**, and **6 🔵 agent-experience gaps** that make the tool harder for LLM agents to consume predictably.

---

## 🔴 Bugs

### B1. `--version` violates output contract — raw text on stdout

**File:** `cmd/cryptospect-cli/root.go:51-57`

```go
if len(args) > 0 && args[0] == "--version" {
    _, err := fmt.Fprintf(cmd.OutOrStdout(), "Version: %s\n", cmd.Version)
```

This prints raw text to stdout. Cobra's built-in `--version` handler (activated by `cmd.Version` being set) also prints raw text:
```
cryptospect-cli version: dev
```

**Contract violation:** `Design-Decisions.md Step 8` states: "stdout: ONLY the JSON envelope (CLIResponse). One object per invocation." The `--version` flag outputs raw text to stdout, breaking any agent, linter, or guard that expects valid JSON on stdout.

**Fix:** Either intercept `--version` and output a `CLIResponse` envelope (e.g., `{"status":"ok","ts":...,"results":[]}` with version in a structured field), or disable cobra's built-in version handler via `cmd.SetVersionTemplate("")` and redirect to stderr.

---

## 🟡 Cross-Metric Inconsistencies

### I1. FT uses `primary_sources` (plural, `[]string`) — all others use `primary_source` (singular, `string`)

| Metric | Meta field | Type |
|--------|-----------|------|
| LP | `primary_source` | `string` |
| SP | `primary_source` | `string` |
| FT | `primary_sources` | `[]string` |
| MB | `primary_source` | `string` |
| MD | `primary_source` | `string` |
| MR | `primary_source` | `string` |

**Impact:** LLM agents must branch parsing logic — `meta.primary_source` is a string for 5 metrics and an array for 1. If an agent calls both `ft` and `lp` in the same session, it gets two different shapes for what is semantically the same metadata field.

**Files:** `internal/metrics/flowtension/v1/types.go:112` (`PrimarySources []string`)

---

### I2. FT thresholds are `map[string]any` — all others are `map[string]float64`

| Metric | Meta thresholds type |
|--------|---------------------|
| LP | `map[string]float64` |
| SP | `map[string]float64` |
| FT | `map[string]any` (nested maps inside) |
| MB | `map[string]float64` |
| MD | `map[string]float64` |
| MR | `map[string]float64` |

FT embeds nested threshold maps (`cvd`, `oi`, `funding` each as `map[string]float64`) inside the top-level thresholds. This forces agents to handle `any`/`interface{}` deserialization for FT alone.

**Files:** `internal/metrics/flowtension/v1/types.go:117` — `Thresholds map[string]any`

**Fix options:**
- Option A: Use a typed `FTThresholds` struct with named sub-fields (`cvd`, `oi`, `funding`) — consistent Go typing, still produces nested JSON, but at least schema-aware.
- Option B: Flatten thresholds into `map[string]float64` with dotted keys (`"cvd.aggressive"`, `"oi.building"`, etc.).
- Option C: Separate `thresholds` from the `map[string]any` pattern — document that FT is intentionally different.

---

### I3. MD's `TierAverages` and `Spreads` use raw `float64` — other metrics use `metrics.MetricFloat`

**Files:** `internal/metrics/momentumdivergence/v1/types.go:42-55`

```go
type TierAverages struct {
    Large float64 `json:"large"`
    Mid   float64 `json:"mid"`
    Small float64 `json:"small"`
}
type Spreads struct {
    MidVsLarge   *float64 `json:"mid_vs_large"`
    ...
}
```

Issue #9 of the 2026-05-13 review fixed MR's `market_breadth_score` from `float64` to `metrics.MetricFloat` (4 decimal places). MD's TierAverages and Spreads were missed. This means MD values may serialize with up to 17 significant digits (e.g., `3.141592653589793`) while all other metric scores serialize with 4 decimal places (e.g., `3.1416`).

**Impact:** Token waste for LLM agents (~7 extra chars per value), precision inconsistency across metrics.

---

### I4. MR's `MetaExtended` has native `CacheHit`/`TTLRemainingSec` — root.go overwrites them

**File:** `internal/metrics/marketregime/v1/types.go:124-139`

MR's `MetaExtended` struct has `CacheHit bool` (json: `"cache_hit"`) and `TTLRemainingSec int` (json: `"ttl_remaining_sec"`) as native fields. MR's `provider.go:144-146` computes these from explicit cache reads of two endpoint keys.

Meanwhile, `cmd/cryptospect-cli/root.go:191-196` calls `aggregateFetchMeta()` and then `postProcessMeta()` which injects `cache_hit` and `ttl_remaining_sec` at line 55-56:
```go
meta["cache_hit"] = cacheHit
meta["ttl_remaining_sec"] = ttlRemaining
```

This unconditionally overwrites MR's own values. MR's cache-freshness computation in `provider.go` lines 142-155 is **dead code** — its results are always replaced. The two computations can also diverge (MR reads the cache directly; root.go derives from `fetchMetas` which reflect the fetcher layer).

**Files:** `internal/metrics/marketregime/v1/provider.go:142-155`, `cmd/cryptospect-cli/meta_processing.go:54-56`

---

### I5. Summary string format is radically different across metrics

| Metric | Summary example |
|--------|----------------|
| LP | `"Volume/MCap:  8.23% | Conviction: normal"` |
| SP | `"SP Ratio: 0.1234 | Dry Powder Alert"` or `"SP Ratio: 0.0456 | Capital Flight | Supply: contracting"` |
| FT | `"Leverage building with aggressive buying — tension coiling, breakout likely."` (full sentence) |
| MB | `"Market breadth 72% (broad): 7d at 80% green, 1h at 65%. No divergence."` |
| MD | `"[risk_on] Mid-caps outpacing mega-caps by +5.2pp (mid avg +6.1%); full alt-season rotation detected (small_vs_large +8.3pp)."` |
| MR | `"BTC dominance rising, breadth broad — BTC-Led Expansion with positive momentum. Capital flowing into BTC with broad alt participation."` |

LP and SP use compact `Key: Value | Key: Value` format. FT and MR use full prose paragraphs. MD uses bracketed prefix `[label]`. MB uses percentage-heavy format with parentheticals. No two metrics format their summary the same way.

**Impact:** Agents cannot reliably parse summaries without metric-specific logic.

---

### I6. MR description meta field is `Capital_Pascal_Case` while all others are plain text

**File:** `internal/metrics/marketregime/v1/types.go:27-38`

MR's regime labels use title-case proper nouns with spaces and special chars:
- `"BTC-Led Expansion"`
- `"Alt-Season / Mania"`
- `"Flight to Safety"`

All other metric classification labels are flat lowercase:
- LP: `"high"`, `"normal"`, `"low"`
- SP: `"high"`, `"normal"`, `"low"`
- FT: `"aggressive_buy"`, `"neutral"`, etc. (snake_case)
- MB: `"broad"`, `"mixed"`, `"narrow"`
- MD: `"risk_on"`, `"top_heavy"`, `"flight_to_safety"`, `"neutral"` (snake_case)

MR's labels are human-readable proper nouns. LP/SP/MB use simple adjectives. FT/MD use snake_case machine codes.

**Note:** This was flagged as issue #14 in the 2026-05-13 review and intentionally deferred ("Design choice — deferred"). Re-stated here for completeness.

---

## 🔵 Agent Experience Issues

### A1. No JSON schema discovery mechanism

The `list-metrics` command returns metric names, versions, aliases, endpoints, and descriptions — but **not** the JSON schema of each metric's `data` field. An LLM agent calling this tool for the first time cannot discover the output shape without:
1. Calling each metric at `--detail full` (expensive in tokens)
2. Reading documentation manually
3. Trial-and-error parsing

**Suggestion:** Add an optional `--schema` flag to `list-metrics` or a dedicated `describe <metric>` command that outputs the JSON schema of each metric's `data` and `meta` fields.

---

### A2. No token-budget guidance

Agents can choose `--detail basic|extended|full` but have no way to estimate token cost before calling. A `basic` LP response is ~200 tokens; `full` MR can exceed 2000 tokens.

**Suggestion:** Document approximate token sizes per metric/detail level in `list-metrics` response or in the metric description.

---

### A3. Unavailable metrics return data as `{"error":"..."}` — inconsistent with ok/degraded data shape

When status is `"unavailable"`, the `data` field is `{"error": "some message"}`. When status is `"ok"` or `"degraded"`, the `data` field is the metric-specific struct. An agent that naively accesses `data.volume_to_mcap_ratio` will get `undefined`/`nil` when the metric is unavailable — and must check `result.status` before parsing `data`.

This is documented behavior and all metrics follow it consistently. But it's a footgun for agents that don't read status first.

**Suggestion:** Consider a `data_variant` discriminator or an `error` field at the result level (not data level) for unavailable metrics.

---

### A4. No envelope-level health summary

When an agent calls multiple metrics, the top-level `CLIResponse.status` is `"ok"` even if every individual metric result has `status: "unavailable"`. The envelope status only reflects whether the CLI command itself succeeded (exit 0), not whether the metrics produced useful data.

**Impact:** An agent must iterate all results to check per-metric status. No fast-path for "everything is fine" vs "nothing is fine."

**Suggestion:** Add an optional envelope-level `metrics_status` summary (e.g., `{"ok": 4, "degraded": 1, "unavailable": 1}`) or derive top-level `status` from worst result.

---

### A5. Missing `namespace` in individual `MetricResult`

The `MetricResult` struct has `metric` and `version` but no `namespace`. The namespace is visible in `list-metrics` output and in CLI `--help` long text, but not in individual metric command results. If a fork registers a metric under a different namespace, the agent cannot distinguish namespaces from the result alone.

**File:** `internal/output/envelope.go:19-31`

```go
type MetricResult struct {
    Metric  string          `json:"metric"`
    Version string          `json:"version"`
    Status  string          `json:"status"`
    Data    json.RawMessage `json:"data"`
    Meta    json.RawMessage `json:"meta,omitempty"`
}
```

No `Namespace` field.

**Note:** Issue #4 of the 2026-05-13 review removed namespace from MetricResult because "all core metrics use namespace 'cryptospec'" — but this means the output format is locked to a single namespace, which contradicts the plugin architecture's fork support.

---

### A6. No `--help` for per-metric `data` fields

`cryptospect-cli liquidity-pulse --help` shows flags and description but nothing about what data fields to expect. An agent that calls `--help` on a metric command (common LLM tool-exploration pattern) gets no schema information.

**Suggestion:** Extend `Long` description of each metric command to include a brief output schema (field names and types).

---

## 🔵 Implementation Observations

### O1. `unavailable()` helper duplicated verbatim across 6 providers

All metrics have an identical 6-line `unavailable()` method. Consider extracting to a shared helper in `internal/metrics/helpers.go`:

```go
func UnavailableResult(name, version, msg string) (output.MetricResult, error) {
    errMsg, _ := json.Marshal(map[string]string{"error": msg})
    return output.MetricResult{
        Metric:  name,
        Version: version,
        Status:  "unavailable",
        Data:    json.RawMessage(errMsg),
    }, nil
}
```

**Files:** All 6 `provider.go` files — each duplicates lines 192-200 (LP), 291-299 (SP), 47-55 (FT), 172-180 (MB), 181-189 (MD), 257-265 (MR).

---

### O2. FT's `prevOI` nil-guard uses `cached > 0` — silently drops valid zero

**File:** `internal/metrics/flowtension/v1/provider.go:96`

```go
if err := json.Unmarshal(entry.Data, &cached); err == nil && cached > 0 {
```

If a previous run stored `totalOI = 0` (impossible in practice, but possible if CoinGecko returns zero OI), the cached value is dropped and treated as a cold start. The `> 0` should arguably be `>= 0` (though OI is practically never zero).

---

### O3. `list-metrics` hardcodes its own `Version` as `"v1.0.0"` — not from registry

**File:** `cmd/cryptospect-cli/list.go:26`

```go
result := output.MetricResult{
    Metric:  "list-metrics",
    Version: "v1.0.0",
```

If the list-metrics output format changes, this version string won't reflect it unless manually updated.

---

### O4. MB's `Compute` takes `*Input` (pointer) — all other `Compute` functions take `Input` (value)

| Metric | Compute signature |
|--------|------------------|
| LP | Not split — everything in provider |
| SP | Not split — everything in provider |
| FT | `Compute(input Input) (Data, error)` |
| MB | `Compute(input *Input) (ComputeResult, error)` ← pointer |
| MD | `Compute(input Input) (Data, computedMeta, error)` |
| MR | `Compute(in Input) ComputeResult` |

MB takes `*Input`. FT/MD/MR take value. This is minor but MB is the outlier. The pointer is arguably unnecessary — MB doesn't modify Input, and Input is passed by value from the caller anyway.

---

## Metrics Interface Summary Table

| Aspect | LP | SP | FT | MB | MD | MR |
|--------|-----|-----|-----|-----|-----|-----|
| Classification in Data | ✓ | ✓ | ✗ (hooks) | ✓ | ✓ | ✓ |
| Summary in Data | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| MetricFloat for scores | ✓ | ✓ | ✓ | ✓ | ✗ (raw) | ✓ |
| PrimarySource (singular) | ✓ | ✓ | ✗ (plural) | ✓ | ✓ | ✓ |
| Validator source | ✓ (opt) | ✓ (always) | ✗ | ✓ (opt) | ✗ | ✗ |
| Thresholds `map[string]float64` | ✓ | ✓ | ✗ (any) | ✓ | ✓ | ✓ |
| Custom flags | ✗ | `--top` | ✗ | `--top` | `--segments` | ✗ |
| `unavailable()` helper | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `DetectStatus()` call | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Metadata injected by root.go | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Compute (pure, no I/O) | ✗ (inline) | ✗ (inline) | ✓ | ✓ | ✓ | ✓ |
| testdata/ fixtures | N/A | N/A | ✓ | ✓ | ✓ | ✓ |

---

## Priority Recommendations

### Must-fix (🔴)
1. **B1**: Fix `--version` to output valid JSON on stdout

### Should-fix (🟡)
2. **I1**: Unify FT's `primary_sources` → `primary_source` (singular string), or make all metrics use the plural form with a single-element array
3. **I2**: Give FT a typed thresholds struct instead of `map[string]any`
4. **I3**: Convert MD's TierAverages/Spreads to `metrics.MetricFloat`
5. **I4**: Remove MR's dead-code cacheHit computation, or remove root.go's overwrite for MR specifically

### Consider (🔵)
6. **O1**: Extract shared `unavailable()` helper to `internal/metrics/helpers.go`
7. **A5**: Add `namespace` to `MetricResult` for fork compatibility
8. **A1**: Add schema discovery mechanism for agents
9. **I5**: Standardize summary format across metrics (or document per-metric format explicitly)
10. **A4**: Add envelope-level metrics health summary

### Deferred (already acknowledged)
- **I6**: MR's title-case regime labels (design choice, issue #14)
