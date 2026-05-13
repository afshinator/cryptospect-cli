package v1

// ── Dominance trend labels ──
const (
	TrendRising  = "rising"
	TrendFalling = "falling"
	TrendNeutral = "neutral"
)

// ── Conviction labels ──
const (
	ConvictionHigh   = "high"
	ConvictionNormal = "normal"
	ConvictionLow    = "low"
)

// ── Modifier labels ──
const (
	ModifierPositiveMomentum = "positive_momentum"
	ModifierNegativePressure = "negative_pressure"
	ModifierNeutral          = "neutral"
)

// ── Regime labels ──
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

// ── Confidence labels ──
const (
	ConfidenceHigh   = "high"
	ConfidenceMedium = "medium"
	ConfidenceLow    = "low"
)

// ── Thresholds ──
const (
	DomDeadBandPP          = 0.5
	ConvictionHighThresh   = 0.15
	ConvictionLowThresh    = 0.07
	BreadthBroadThresh     = 0.60
	BreadthNarrowThresh    = 0.40
	ModifierDeadBandPP     = 0.5
	CacheHintSec           = 14400
	StateKey               = "marketregime_dominance_pct"
	StateTTLSec            = 172800
	MaxSnapshotAgeSec      = 86400
	DegradedCoinsThreshold = 50
)

// ── Input ──

// Input holds all data needed by the pure Compute function.
type Input struct {
	BTCDominancePct     float64
	PriorDominancePct   *float64
	PriorSnapshotAgeSec *int
	BreadthScore        float64
	LPRatio             float64
	BTCChange24h        *float64
	WeightRedistributed bool
	BreadthDegraded     bool
}

// ── ComputeResult ──

// ComputeResult holds the full output of the pure compute function.
type ComputeResult struct {
	Regime         string
	Modifier       string
	DominanceTrend string
	Conviction     string
	BreadthScore   float64
	ColdStart      bool
	Confidence     string
	Notes          []string

	Summary string
	// Internal — provider uses for meta construction
	MissingReferenceData bool
	CapitulationNote     string
}

// ── Data ──

// Data is the output payload for the market-regime metric.
type Data struct {
	Regime             string         `json:"regime"`
	Modifier           string         `json:"modifier"`
	DominanceTrend     string         `json:"dominance_trend"`
	Conviction         string         `json:"conviction"`
	MarketBreadthScore float64        `json:"market_breadth_score"`
	Classification     Classification `json:"classification"`
	Summary            string         `json:"summary"`
}

// Classification holds the categorical output.
type Classification struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// ── Meta ──

// WeightsUsed holds the fixed-struct breadth weights output.
type WeightsUsed struct {
	OneHour    float64 `json:"1h"`
	TwentyFour float64 `json:"24h"`
	SevenDay   float64 `json:"7d"`
	ThirtyDay  float64 `json:"30d"`
}

// MetaExtended holds extended-detail metadata.
type MetaExtended struct {
	CacheHit                     bool        `json:"cache_hit"`
	TTLRemainingSec              int         `json:"ttl_remaining_sec"`
	PrimarySource                string      `json:"primary_source"`
	BTCDominancePct              float64     `json:"btc_dominance_pct"`
	BTC24hChange                 *float64    `json:"btc_24h_change,omitempty"`
	Confidence                   string      `json:"confidence"`
	DominanceColdStart           bool        `json:"dominance_cold_start"`
	Notes                        []string    `json:"notes"`
	CacheHintSec                 int         `json:"cache_hint_sec"`
	LPRatio                      float64     `json:"lp_ratio"`
	DominanceDeltaSinceLastFetch *float64    `json:"dominance_delta_since_last_fetch,omitempty"`
	PriorSnapshotAgeSec          *int        `json:"prior_snapshot_age_sec,omitempty"`
	WeightsUsed                  WeightsUsed `json:"weights_used"`
}

// MetaFull holds full-detail metadata (extends MetaExtended).
type MetaFull struct {
	MetaExtended
	Thresholds  map[string]float64 `json:"thresholds"`
	Description string             `json:"description"`
}
