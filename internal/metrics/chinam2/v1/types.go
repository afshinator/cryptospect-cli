package v1

import "github.com/afshinator/cryptospect-cli/internal/metrics"

// ── Metric identity ──
const (
	MetricName    = "china-m2"
	MetricVersion = "v1.0.0"
)

// ── Classification labels ──
const (
	LabelExpanding = "expanding"
	LabelNormal    = "normal"
	LabelSlowing   = "slowing"
)

// ── Thresholds ──
const (
	YoYExpandingMin = 8.0
	YoYSlowingMax   = 4.0
)

// ── Confidence ──
const (
	ConfidenceHigh   = "high"
	ConfidenceMedium = "medium"
)

// ── Data requirements ──
const (
	MinHistoryMonths = 13 // need 12 months back for YoY
)

// ── Input ──

// Observation holds a single period's M2 data.
type Observation struct {
	Period string
	Value  float64 // in 100 million yuan
}

// Input holds China M2 observations for pure Compute.
type Input struct {
	Observations []Observation // most recent first
}

// ── Data ──

// Classification holds the primary label and description.
type Classification struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// Data is the output payload for the china-m2 metric.
type Data struct {
	M2LevelCNYBillion metrics.MetricFloat  `json:"m2_level_cny_billion"`
	YoYChangePct      *metrics.MetricFloat `json:"yoy_change_pct"` // null when insufficient history
	Period            string               `json:"period"`
	Classification    Classification       `json:"classification"`
	Summary           string               `json:"summary"`
}

// ── Meta ──

// Meta holds extended and full-detail metadata.
// cache_hit and ttl_remaining_sec are injected by root.go.
type Meta struct {
	PrimarySource string             `json:"primary_source"`
	Confidence    string             `json:"confidence"`
	DataFrequency string             `json:"data_frequency"`
	DataLagDays   int                `json:"data_lag_days,omitempty"`
	Units         string             `json:"units"`
	Thresholds    map[string]float64 `json:"thresholds,omitempty"`
	Description   string             `json:"description,omitempty"`
}

const metricDescription = "China M2 money supply from the National Bureau of Statistics of China " +
	"via DBnomics. Values in CNY billion (original data in 100 million yuan, divided by 10). " +
	"Monthly frequency. YoY change compares current value to value from 12 months prior. " +
	"Expanding (>8% YoY) signals strong liquidity tailwind; slowing (<4% YoY) signals tightening."

// ── computedMeta ──

type computedMeta struct {
	Confidence  string
	YoYChange   *float64
	DataLagDays int
}
