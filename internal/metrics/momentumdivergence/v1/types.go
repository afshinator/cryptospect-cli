package v1

import (
	"github.com/afshinator/cryptospect-cli/internal/api/coingecko"
)

// Threshold constants for the 6-stage momentum-divergence pipeline.
// See docs/metrics/momentum-divergence.md for the calibration rationale
// and the v1.1 recalibration plan.
const (
	RiskOnSpread          = 5.0
	TopHeavySpread        = -3.0
	MinPositivityGuard    = 1.0
	TailExtensionSpread   = 5.0
	TierFloorMinCoins     = 3
	SegmentsLargeMin      = 5
	SegmentsSmallMax      = 250
	ConcentrationDeadBand = 0.5

	DefaultLargeCeiling = 10
	DefaultMidCeiling   = 50
	DefaultSmallCeiling = 200

	BarbellMidVsLargeMin = 1.0

	LabelRiskOn         = "risk_on"
	LabelTopHeavy       = "top_heavy"
	LabelFlightToSafety = "flight_to_safety"
	LabelNeutral        = "neutral"
)

// Input is the pure compute input for momentum-divergence.
type Input struct {
	Coins                 []coingecko.CoinMarketsRankedCoin
	LargeCeiling          int
	MidCeiling            int
	SmallCeiling          int
	SegmentsClamped       bool
	SegmentsClampedReason string
}

// TierAverages holds the simple mean 24h return for each market-cap tier.
type TierAverages struct {
	Large float64 `json:"large"`
	Mid   float64 `json:"mid"`
	Small float64 `json:"small"`
}

// Spreads holds the percentage-point differences between tier averages.
// Fields are *float64 — nil means the spread could not be computed because
// one of the constituent tiers was absent. 0.0 is a valid real value
// (tiers performing identically).
type Spreads struct {
	MidVsLarge   *float64 `json:"mid_vs_large"`
	SmallVsLarge *float64 `json:"small_vs_large"`
	SmallVsMid   *float64 `json:"small_vs_mid"`
}

// Classification holds the primary label and human-readable description.
type Classification struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// Data is the output payload for the momentum-divergence metric.
type Data struct {
	TierAverages   TierAverages   `json:"tier_averages"`
	Spreads        Spreads        `json:"spreads"`
	TailExtension  bool           `json:"tail_extension"`
	Classification Classification `json:"classification"`
	Summary        string         `json:"summary"`
}

// TierCounts holds the number of valid (non-null) coins in each tier.
type TierCounts struct {
	Large int `json:"large"`
	Mid   int `json:"mid"`
	Small int `json:"small"`
}

// SegmentsUsed records the actual tier boundaries used after clamping.
type SegmentsUsed struct {
	LargeCeiling int `json:"large_ceiling"`
	MidCeiling   int `json:"mid_ceiling"`
	SmallCeiling int `json:"small_ceiling"`
}

// TierCoinDetail holds per-coin data for the tier_detail block at full detail.
type TierCoinDetail struct {
	ID        string   `json:"id"`
	Return24h *float64 `json:"return_24h"`
}

// TierDetail holds per-tier coin breakdowns for full-detail output.
type TierDetail struct {
	Large []TierCoinDetail `json:"large,omitempty"`
	Mid   []TierCoinDetail `json:"mid,omitempty"`
	Small []TierCoinDetail `json:"small,omitempty"`
}

// computedMeta holds metadata computed by the pure Compute function.
type computedMeta struct {
	Confidence       string
	TierCounts       TierCounts
	Thresholds       map[string]float64
	LabelDescription string
	TierDetail       *TierDetail
}
