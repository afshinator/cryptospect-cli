// Package v1 implements the market-breadth metric.
package v1

import (
	"time"

	"github.com/afshinator/cryptospect-cli/internal/api/coingecko"
	"github.com/afshinator/cryptospect-cli/internal/metrics"
)

// ── Constants ──

// Metric configuration constants for market-breadth.
const (
	DefaultTopN = 250
	MinTopN     = 50
	MaxTopN     = 250

	Weight1h  = 0.10
	Weight24h = 0.30
	Weight7d  = 0.40
	Weight30d = 0.20

	BroadThreshold  = 0.60
	NarrowThreshold = 0.40

	DivergenceBTCChangeMin = 2.0 // CoinGecko returns percentage directly

	StalenessThresholdSec = 5400 // 90 minutes in seconds

	StatisticalFloor = 50 // min coins for a timeframe to contribute
)

// ── Classification constants ──

// Classification labels for market breadth.
const (
	ClassificationBroad  = "broad"
	ClassificationMixed  = "mixed"
	ClassificationNarrow = "narrow"
)

// ── Input ──

// Input holds all data needed by the pure Compute function.
type Input struct {
	TimeframeCounts map[string]coingecko.TimeframeMetric
	CoinsCounted    int // from parser: coins with >=1 non-null timeframe field
	BTCChange24h    float64
	BTCAvailable    bool
	KlineClose      float64
	KlineOpen       float64 // open price for sign(close-open) discrepancy check
	KlineOpenTimeMs int64
	KlineAvailable  bool
	TopN            int
	Now             time.Time
}

// ── ComputeResult ──

// ComputeResult holds the full output of the pure compute function.
type ComputeResult struct {
	MarketBreadthScore metrics.MetricFloat `json:"market_breadth_score"`
	CoinsCounted       int                 `json:"coins_counted"`
	TimeframeBreadth   TimeframeFractions  `json:"timeframe_breadth"`
	DivergenceDetected bool                `json:"divergence_detected"`
	BTCChange24h       metrics.MetricFloat `json:"btc_change_24h_pct"`
	Classification     Classification      `json:"classification"`
	Summary            string              `json:"summary"`

	// Internal — provider uses for meta/status/confidence
	DiscrepancyDetected bool               `json:"-"`
	DiscrepancyNote     string             `json:"-"`
	ValidatorConfidence string             `json:"-"` // "high" / "medium" / "low"
	MetricStatus        string             `json:"-"` // "ok" / "degraded" / "unavailable"
	WeightsUsed         map[string]float64 `json:"-"`
	GreenCounts         map[string]int     `json:"-"`
	TotalCounts         map[string]int     `json:"-"`
}

// TimeframeFractions holds per-timeframe green_pct values.
type TimeframeFractions struct {
	OneHour    metrics.MetricFloat `json:"1h"`
	TwentyFour metrics.MetricFloat `json:"24h"`
	SevenDay   metrics.MetricFloat `json:"7d"`
	ThirtyDay  metrics.MetricFloat `json:"30d"`
}

// ── Classification ──

// Classification holds the categorical output.
type Classification struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// ── Meta ──

// Meta holds extended and full-detail metadata.
type Meta struct {
	PrimarySource       string                               `json:"primary_source"`
	ValidatorSource     string                               `json:"validator_source"`
	DiscrepancyDetected bool                                 `json:"discrepancy_detected"`
	DiscrepancyNote     string                               `json:"discrepancy_note,omitempty"`
	Confidence          string                               `json:"confidence"`
	TopClamped          bool                                 `json:"top_clamped,omitempty"`
	TopClampedReason    string                               `json:"top_clamped_reason,omitempty"`
	WeightsUsed         map[string]float64                   `json:"weights_used"`
	TimeframeCounts     map[string]coingecko.TimeframeMetric `json:"timeframe_counts"`
	Thresholds          map[string]float64                   `json:"thresholds,omitempty"`
	Description         string                               `json:"description,omitempty"`
}
