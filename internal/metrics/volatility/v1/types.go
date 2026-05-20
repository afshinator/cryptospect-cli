package v1

import "github.com/afshinator/cryptospect-cli/internal/metrics"

// ── Metric identity ──
const (
	MetricName    = "volatility"
	MetricVersion = "v1.0.0"
)

// ── Classification labels ──
const (
	LabelSubdued  = "subdued"
	LabelNormal   = "normal"
	LabelElevated = "elevated"
)

// ── Spread thresholds ──
const (
	SpreadSubduedMax  = 0.8
	SpreadElevatedMin = 1.5
)

// ── Confidence ──
const (
	ConfidenceHigh   = "high"
	ConfidenceMedium = "medium"
)

// ── Candles ──
const (
	CandlesRequired = 24
	AnnualFactor    = 8760.0 // 365 days × 24 hours
)

// ── Input ──

// Input holds BTC and ETH kline data for pure Compute.
type Input struct {
	BTCCloses []float64 // 24 hourly close prices
	ETHCloses []float64 // 24 hourly close prices
}

// ── Data ──

// Classification holds the primary label and description.
type Classification struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// Data is the output payload for the volatility metric.
type Data struct {
	BTCRealizedVol metrics.MetricFloat `json:"btc_realized_vol"`
	ETHRealizedVol metrics.MetricFloat `json:"eth_realized_vol"`
	VolSpread      metrics.MetricFloat `json:"vol_spread"`
	Classification Classification      `json:"classification"`
	Summary        string              `json:"summary"`
}

// ── Meta ──

// Meta holds extended and full-detail metadata.
// cache_hit and ttl_remaining_sec are injected by root.go.
type Meta struct {
	PrimarySource string             `json:"primary_source"`
	Confidence    string             `json:"confidence"`
	BTCCandles    int                `json:"btc_candles"`
	ETHCandles    int                `json:"eth_candles"`
	Thresholds    map[string]float64 `json:"thresholds,omitempty"`
	Description   string             `json:"description,omitempty"`
}

const metricDescription = "Volatility measures annualized realized volatility for BTC and ETH " +
	"from 24 hourly OHLC candles. The ETH/BTC volatility spread identifies when ETH is " +
	"disproportionately volatile relative to BTC — a standard crypto volatility desk signal. " +
	"Spread below 0.8 indicates subdued ETH vol; above 1.5 indicates elevated ETH speculation."

// ── computedMeta ──

// computedMeta holds metadata computed by the pure Compute function.
type computedMeta struct {
	Confidence string
}
