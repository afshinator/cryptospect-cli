// Package v1 implements the flow-tension metric.
//
// Flow tension measures the kinetic energy of the market — how aggressively
// traders are using leverage and moving assets onto exchanges. It tracks three
// distinct signals: taker aggression in spot markets (CVD), leverage accumulation
// or unwinding (Open Interest), and the cost of holding leveraged positions
// (Funding Rate).
//
// All three signals are sourced from keyless public APIs — no API key required.
package v1

import "github.com/afshinator/cryptospect-cli/internal/metrics"

// ┌─────────────────────────────────────────────┐
// │                Constants                     │
// └─────────────────────────────────────────────┘

// Thresholds for the flow-tension metric.
const (
	MinTrades                  = 10      // thin-candle guard: below this, CVD hook = low_confidence
	FlowAggressiveThreshold    = 0.10    // CVD ratio ±threshold for aggressive classification
	FundingNegativeThreshold   = -0.0003 // ≤ this → negative_funding
	FundingPositiveThreshold   = 0.0003  // ≥ this → positive_funding (and < overheated)
	FundingOverheatedThreshold = 0.003   // > this → overheated_funding
	OIBuildingThreshold        = 0.05    // > this → oi_building
	OIUnwindingThreshold       = -0.05   // < this → oi_unwinding
)

// Hook label constants for each signal.
const (
	// CVD hooks
	HookAggressiveBuy  = "aggressive_buy"
	HookNeutral        = "neutral"
	HookAggressiveSell = "aggressive_sell"
	HookLowConfidence  = "low_confidence"

	// OI hooks
	HookOIBuilding  = "building"
	HookOIStable    = "stable"
	HookOIUnwinding = "unwinding"

	// Funding hooks
	HookNegative   = "negative"
	HookNeutralFR  = "neutral"
	HookPositive   = "positive"
	HookOverheated = "overheated"
)

// ┌─────────────────────────────────────────────┐
// │                Input struct                  │
// └─────────────────────────────────────────────┘

// Input holds the raw signal values for the pure compute function.
// OI change requires a prior cached value — PrevOI is nil on first run.
type Input struct {
	// From Binance-US spot klines (keyless)
	TakerBuyVolume  float64
	TakerSellVolume float64
	TotalVolume     float64
	NumTrades       int

	// From CoinGecko public /derivatives (keyless)
	TotalOpenInterest float64
	PrevOI            *float64 // nil on first run (no cache history)
	FundingRate       float64  // Binance Futures BTC perpetual funding rate
	ExchangeCount     int      // number of exchanges contributing BTC perpetual data
}

// ┌─────────────────────────────────────────────┐
// │                Output structs                │
// └─────────────────────────────────────────────┘

// Data is the core metric output.
type Data struct {
	Signals Signals `json:"signals"`
	Summary string  `json:"summary"`
}

// Signals holds all three flow-tension sub-signals.
type Signals struct {
	CVD          SignalCVD `json:"cvd"`
	OpenInterest SignalOI  `json:"open_interest"`
	FundingRate  SignalFR  `json:"funding_rate"`
}

// SignalCVD holds the CVD ratio and hook.
type SignalCVD struct {
	Ratio metrics.MetricFloat `json:"ratio"`
	Hook  string              `json:"hook"` // aggressive_buy | neutral | aggressive_sell | low_confidence
}

// SignalOI holds the current aggregated OI, 24h change (when available), and hook.
type SignalOI struct {
	CurrentUSD    metrics.MetricFloat  `json:"current_usd"`
	ChangePct24h  *metrics.MetricFloat `json:"change_pct_24h,omitempty"` // omitted on first run
	ExchangeCount int                  `json:"exchange_count"`
	Hook          string               `json:"hook"` // building | stable | unwinding
}

// SignalFR holds the Binance Futures funding rate and hook.
type SignalFR struct {
	Rate metrics.MetricFloat `json:"rate"`
	Hook string              `json:"hook"` // negative | neutral | positive | overheated
}

// ┌─────────────────────────────────────────────┐
// │                Meta struct                    │
// └─────────────────────────────────────────────┘

// Meta holds extended and full-detail metadata.
type Meta struct {
	PrimarySources  []string       `json:"primary_sources"`
	Confidence      string         `json:"confidence"`
	CvdSampleTrades int            `json:"cvd_sample_trades"`
	OIExchangeCount int            `json:"oi_exchange_count"`
	Instrument      string         `json:"instrument"`
	Thresholds      map[string]any `json:"thresholds,omitempty"`
	Description     string         `json:"description,omitempty"`
}
