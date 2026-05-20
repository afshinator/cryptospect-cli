package v1

import (
	"fmt"
	"math"

	"github.com/afshinator/cryptospect-cli/internal/metrics"
)

// Compute performs the pure volatility calculation.
// It is side-effect-free: no I/O, no state mutation.
func Compute(in Input) (Data, computedMeta, error) {
	if len(in.BTCCloses) < 2 || len(in.ETHCloses) < 2 {
		return Data{}, computedMeta{}, fmt.Errorf("insufficient candle data: need at least 2 closes per asset")
	}

	btcVol := realizedVol(in.BTCCloses)
	ethVol := realizedVol(in.ETHCloses)

	var spread float64
	if btcVol > 0 {
		spread = ethVol / btcVol
	}

	// Classification from spread
	label := classifySpread(spread)
	desc := classificationDescription(label)

	classification := Classification{
		Label:       label,
		Description: desc,
	}

	// Confidence
	conf := ConfidenceHigh
	btcCandles := len(in.BTCCloses)
	ethCandles := len(in.ETHCloses)
	if btcCandles < CandlesRequired || ethCandles < CandlesRequired {
		conf = ConfidenceMedium
	}

	summary := buildSummary(btcVol, ethVol, spread, label)

	data := Data{
		BTCRealizedVol: metrics.Ratio(btcVol),
		ETHRealizedVol: metrics.Ratio(ethVol),
		VolSpread:      metrics.Ratio(spread),
		Classification: classification,
		Summary:        summary,
	}

	meta := computedMeta{
		Confidence: conf,
	}

	return data, meta, nil
}

// realizedVol computes annualized realized volatility from close prices.
// Formula: std(log_returns) × sqrt(365)
func realizedVol(closes []float64) float64 {
	if len(closes) < 2 {
		return 0
	}

	logReturns := make([]float64, len(closes)-1)
	for i := 1; i < len(closes); i++ {
		if closes[i-1] <= 0 || closes[i] <= 0 {
			return 0
		}
		logReturns[i-1] = math.Log(closes[i] / closes[i-1])
	}

	std := stdDev(logReturns)
	return std * math.Sqrt(AnnualFactor)
}

// stdDev computes sample standard deviation (n-1 denominator).
func stdDev(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}

	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))

	sumSq := 0.0
	for _, v := range values {
		d := v - mean
		sumSq += d * d
	}

	return math.Sqrt(sumSq / float64(len(values)-1))
}

// classifySpread returns the volatility spread classification.
func classifySpread(spread float64) string {
	switch {
	case spread < SpreadSubduedMax:
		return LabelSubdued
	case spread > SpreadElevatedMin:
		return LabelElevated
	default:
		return LabelNormal
	}
}

// classificationDescription returns a human-readable description for the label.
func classificationDescription(label string) string {
	switch label {
	case LabelSubdued:
		return "Subdued ETH volatility relative to BTC — capital parked in BTC"
	case LabelElevated:
		return "Elevated ETH volatility relative to BTC — heightened altcoin speculation"
	default:
		return "Normal ETH/BTC volatility relationship"
	}
}

// buildSummary constructs a one-line natural-language summary.
func buildSummary(btcVol, ethVol, spread float64, label string) string {
	base := fmt.Sprintf("BTC vol %.1f%% (annualized), ETH vol %.1f%%, spread %.2f×",
		btcVol*100, ethVol*100, spread)

	switch label {
	case LabelSubdued:
		base += " — subdued ETH speculation."
	case LabelElevated:
		base += " — elevated ETH speculation."
	default:
		base += " — normal."
	}

	return base
}
