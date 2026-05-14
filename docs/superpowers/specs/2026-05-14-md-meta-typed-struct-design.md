# MD Meta: Typed Struct Refactor

Convert momentum-divergence's `map[string]interface{}` meta to a typed Go struct, following the project convention established by market-breadth.

## Problem

`internal/metrics/momentumdivergence/v1/provider.go:103-127` builds metadata as:

```go
metaMap := map[string]interface{}{"primary_source": ..., "confidence": ...}
metaBytes, err := json.Marshal(metaMap)
```

This has no compile-time field safety, fragile string keys, and violates the project convention of typed Meta structs (used by LP, SP, FT, MB, MR).

## Solution

### New Meta struct in `types.go`

```go
type Meta struct {
	PrimarySource         string             `json:"primary_source"`
	Confidence            string             `json:"confidence"`
	TierCounts            TierCounts         `json:"tier_counts"`
	SegmentsUsed          SegmentsUsed       `json:"segments_used"`
	WeightingMethod       string             `json:"weighting_method"`
	SegmentsClamped       bool               `json:"segments_clamped,omitempty"`
	SegmentsClampedReason string             `json:"segments_clamped_reason,omitempty"`
	Thresholds            map[string]float64 `json:"thresholds,omitempty"`
	Description           string             `json:"description,omitempty"`
	TierDetail            *TierDetail        `json:"tier_detail,omitempty"`
}
```

### What's removed vs current map

| Current map key | Decision | Rationale |
|---|---|---|
| `cache_hit` | **Removed** | Hardcoded `false` — misleading. MB/FT don't populate it either. |
| `ttl_remaining_sec` | **Removed** | Hardcoded `0` — misleading. Only MR populates it correctly. |
| `data_timestamp` | **Removed** | Redundant with `CLIResponse.ts`. No other metric includes this. |
| `weighting_method` | **Kept** (constant) | Informative. |
| `primary_source` | **Kept** | Standard across all metrics. |
| `confidence` | **Kept** | Standard across all metrics. |
| `tier_counts` | **Kept** | Informative. Typed as `TierCounts`. |
| `segments_used` | **Kept** | Informative. Typed as `SegmentsUsed`. |
| `segments_clamped` | **Kept** | Informative. `omitempty`. |
| `segments_clamped_reason` | **Kept** | Informative. `omitempty`. |
| `thresholds` | **Kept** | Full-detail. `omitempty`. |
| `description` | **Repurposed** | Currently set to classification label (e.g., "Risk-On Rotation") — this is already in `Data.Classification.Description`. Meta `Description` gets a proper methodology constant. |
| `tier_detail` | **Kept** | Full-detail. `omitempty`. |

### Methodology description constant (in `types.go`)

```go
const metricDescription = "Momentum divergence measures capital rotation across market-cap tiers " +
	"by computing market-cap weighted 24h returns for large, mid, and small cap segments. " +
	"A positive mid-vs-large spread above 5pp with positive mid-cap returns signals Risk-On rotation; " +
	"a negative spread below -3pp with positive large-cap returns signals Top-Heavy concentration."
```

## Files Changed

| File | Change |
|------|--------|
| `types.go` | Add `Meta` struct + `metricDescription` constant |
| `provider.go` | Build `Meta{...}` instead of `map[string]interface{}`; marshal with `json.Marshal` |
| `provider_test.go` | Update tests that inspect meta via `map[string]interface{}` to use the typed struct |

## Verification

1. `go test -race -cover ./internal/metrics/momentumdivergence/v1/...` — all existing tests pass with updated meta assertions
2. `go build -o /dev/null ./cmd/cryptospect-cli/` — compiles
3. The JSON output schema should be unchanged except `cache_hit`, `ttl_remaining_sec`, and `data_timestamp` are no longer emitted
