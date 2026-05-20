package v1

import "github.com/afshinator/cryptospect-cli/internal/metrics"

// ── Metric identity ──
const (
	MetricName    = "dominance"
	MetricVersion = "v1.0.0"
)

// ── Trend labels ──
const (
	TrendRising  = "rising"
	TrendFalling = "falling"
	TrendNeutral = "neutral"
)

// ── Classification labels ──
const (
	LabelBTCRising          = "btc_rising"
	LabelETHRising          = "eth_rising"
	LabelNeutral            = "neutral"
	LabelCapitalContracting = "capital_contracting"
)

// ── Dead band thresholds (percentage points) ──
const (
	BTCDeadBandPP = 0.5
	ETHDeadBandPP = 0.3
)

// ── Confidence labels ──
const (
	ConfidenceHigh   = "high"
	ConfidenceMedium = "medium"
)

// ── State cache ──
const (
	StateKeyBTCDom    = "dom_btc_dominance_pct"
	StateKeyETHDom    = "dom_eth_dominance_pct"
	StateTTLSec       = 172800 // 48 hours
	MaxSnapshotAgeSec = 86400  // 24 hours
)

// ── Input ──

// Input holds all data needed by the pure Compute function.
type Input struct {
	BTCDominancePct     float64
	ETHDominancePct     float64
	PriorBTCDominance   *float64
	PriorETHDominance   *float64
	PriorSnapshotAgeSec *int
}

// ── AssetDominance ──

// AssetDominance holds dominance data for a single asset (BTC or ETH).
type AssetDominance struct {
	Dominance metrics.MetricFloat  `json:"dominance"`
	Trend     string               `json:"trend"`
	DeltaPP   *metrics.MetricFloat `json:"delta_pp"` // null on cold start
}

// ── Data ──

// Classification holds the primary label and human-readable description.
type Classification struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// Data is the output payload for the dominance metric.
type Data struct {
	BTC            AssetDominance      `json:"btc"`
	ETH            AssetDominance      `json:"eth"`
	ETHBTCRatio    metrics.MetricFloat `json:"eth_btc_ratio"`
	Classification Classification      `json:"classification"`
	Summary        string              `json:"summary"`
}

// ── Meta ──

// Meta holds extended and full-detail metadata.
// cache_hit and ttl_remaining_sec are injected by root.go's postProcessMeta.
type Meta struct {
	PrimarySource       string             `json:"primary_source"`
	Confidence          string             `json:"confidence"`
	ColdStart           bool               `json:"cold_start"`
	PriorSnapshotAgeSec *int               `json:"prior_snapshot_age_sec,omitempty"`
	Thresholds          map[string]float64 `json:"thresholds,omitempty"`
	Description         string             `json:"description,omitempty"`
}

// metricDescription is a human-readable description of the methodology.
const metricDescription = "Dominance tracks BTC and ETH market cap share as a percentage " +
	"of total crypto market cap. Trends are computed from delta vs a prior cached snapshot " +
	"with ±0.5pp dead band for BTC and ±0.3pp dead band for ETH. " +
	"Rising BTC dominance signals safety retreat; rising ETH dominance signals risk-on rotation."

// ── computedMeta ──

// computedMeta holds metadata computed by the pure Compute function.
type computedMeta struct {
	Confidence string
	ColdStart  bool
	BTCDeltaPP *float64
	ETHDeltaPP *float64
}

// convertToAssetDominance builds an AssetDominance from dominance value, trend, and delta.
func convertToAssetDominance(dom float64, trend string, delta *float64) AssetDominance {
	ad := AssetDominance{
		Dominance: metrics.Ratio(dom),
		Trend:     trend,
	}
	if delta != nil {
		d := metrics.Ratio(*delta)
		ad.DeltaPP = &d
	}
	return ad
}
