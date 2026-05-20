package v1

import "github.com/afshinator/cryptospect-cli/internal/metrics"

// ── Metric identity ──
const (
	MetricName    = "fear-greed-index"
	MetricVersion = "v1.0.0"
)

// ── Classification labels ──
const (
	LabelExtremeFear  = "extreme_fear"
	LabelFear         = "fear"
	LabelNeutral      = "neutral"
	LabelGreed        = "greed"
	LabelExtremeGreed = "extreme_greed"
)

// ── Trend labels ──
const (
	TrendImproving     = "improving"
	TrendDeteriorating = "deteriorating"
	TrendStable        = "stable"
)

// ── Classification bands ──
const (
	BandExtremeFearMax = 25
	BandFearMax        = 45
	BandNeutralMax     = 55
	BandGreedMax       = 75
	// > 75 → extreme_greed
)

// ── Trend dead band ──
const (
	TrendDeadBand = 2.0
)

// ── MA window ──
const (
	MAWindow = 7
)

// ── Confidence ──
const (
	ConfidenceHigh   = "high"
	ConfidenceMedium = "medium"
)

// ── Input ──

// Input holds FNG data points for pure Compute.
type Input struct {
	Values []int // most recent first
}

// ── Data ──

// Classification holds the primary label and description.
type Classification struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// Data is the output payload for the fear-greed-index metric.
type Data struct {
	Value          int            `json:"value"`
	Classification Classification `json:"classification"`
	Summary        string         `json:"summary"`
}

// ── Meta ──

// Meta holds extended and full-detail metadata.
// cache_hit and ttl_remaining_sec are injected by root.go.
type Meta struct {
	PrimarySource      string               `json:"primary_source"`
	Confidence         string               `json:"confidence"`
	Timestamp          string               `json:"timestamp,omitempty"`
	TimeUntilUpdateSec int                  `json:"time_until_update_sec,omitempty"`
	SevdMA             *metrics.MetricFloat `json:"sevd_ma,omitempty"` // null when < 7 data points
	Trend              string               `json:"trend,omitempty"`
	Thresholds         map[string]float64   `json:"thresholds,omitempty"`
	Description        string               `json:"description,omitempty"`
}

const metricDescription = "Fear & Greed Index measures crowd sentiment on a 0-100 scale " +
	"using alternative.me data. 0 = Extreme Fear (buying opportunity), 100 = Extreme Greed (correction risk). " +
	"Classification is computed from the raw value using the standard 5-band system. " +
	"A 7-day moving average and trend direction are computed when sufficient history is available."

// ── computedMeta ──

type computedMeta struct {
	Confidence         string
	SevdMA             *float64
	Trend              string
	Timestamp          string
	TimeUntilUpdateSec int
}
