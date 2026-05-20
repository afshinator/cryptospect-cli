package v1

import (
	"fmt"
	"math"

	"github.com/afshinator/cryptospect-cli/internal/metrics"
)

// Compute performs the pure dominance classification.
// It is side-effect-free: no I/O, no state mutation.
func Compute(in Input) (Data, computedMeta, error) {
	meta := computedMeta{}
	coldStart := in.PriorBTCDominance == nil || in.PriorETHDominance == nil

	var btcDelta, ethDelta *float64
	var btcTrend, ethTrend string

	if coldStart {
		btcTrend = TrendNeutral
		ethTrend = TrendNeutral
		meta.ColdStart = true
		meta.Confidence = ConfidenceMedium
	} else {
		dBtc := in.BTCDominancePct - *in.PriorBTCDominance
		dEth := in.ETHDominancePct - *in.PriorETHDominance
		btcDelta = &dBtc
		ethDelta = &dEth

		btcTrend = classifyTrend(dBtc, BTCDeadBandPP)
		ethTrend = classifyTrend(dEth, ETHDeadBandPP)

		// Verify stale snapshot doesn't produce false signals
		ageOk := in.PriorSnapshotAgeSec != nil && *in.PriorSnapshotAgeSec <= MaxSnapshotAgeSec
		if !ageOk {
			// Prior snapshot too old — treat as cold start
			btcTrend = TrendNeutral
			ethTrend = TrendNeutral
			btcDelta = nil
			ethDelta = nil
			meta.ColdStart = true
			meta.Confidence = ConfidenceMedium
		} else {
			meta.ColdStart = false
			meta.Confidence = ConfidenceHigh
		}
	}

	// Primary classification label
	label := selectLabel(btcTrend, ethTrend, btcDelta, ethDelta)
	desc := classificationDescription(label, btcDelta, ethDelta)

	classification := Classification{
		Label:       label,
		Description: desc,
	}

	// ETH/BTC ratio
	ethBTCRatio := 0.0
	if in.BTCDominancePct > 0 {
		ethBTCRatio = in.ETHDominancePct / in.BTCDominancePct
	}

	// Build asset dominance structs
	btc := convertToAssetDominance(in.BTCDominancePct, btcTrend, btcDelta)
	eth := convertToAssetDominance(in.ETHDominancePct, ethTrend, ethDelta)

	summary := buildSummary(btc, eth, label, coldStart)

	data := Data{
		BTC:            btc,
		ETH:            eth,
		ETHBTCRatio:    metrics.Ratio(ethBTCRatio),
		Classification: classification,
		Summary:        summary,
	}

	return data, meta, nil
}

// classifyTrend returns the trend label for a delta against a dead band.
func classifyTrend(delta, deadBand float64) string {
	switch {
	case delta > deadBand:
		return TrendRising
	case delta < -deadBand:
		return TrendFalling
	default:
		return TrendNeutral
	}
}

// selectLabel picks the primary classification label based on the most significant movement.
func selectLabel(btcTrend, ethTrend string, btcDelta, ethDelta *float64) string {
	bothNeutral := btcTrend == TrendNeutral && ethTrend == TrendNeutral
	if bothNeutral {
		return LabelNeutral
	}

	btcRising := btcTrend == TrendRising
	ethRising := ethTrend == TrendRising
	btcFalling := btcTrend == TrendFalling
	ethFalling := ethTrend == TrendFalling

	// Cross-direction: most interesting cases
	if btcRising && ethFalling {
		return LabelBTCRising
	}
	if ethRising && btcFalling {
		return LabelETHRising
	}

	// Both falling
	if btcFalling && ethFalling {
		return LabelCapitalContracting
	}

	// Both rising or one rising + other neutral — pick the larger delta
	absBtc := 0.0
	absEth := 0.0
	if btcDelta != nil {
		absBtc = math.Abs(*btcDelta)
	}
	if ethDelta != nil {
		absEth = math.Abs(*ethDelta)
	}

	if absBtc >= absEth && btcTrend != TrendNeutral {
		return LabelBTCRising
	}
	if ethTrend != TrendNeutral {
		return LabelETHRising
	}
	return LabelNeutral
}

// classificationDescription returns a human-readable description for the primary label.
func classificationDescription(label string, btcDelta, ethDelta *float64) string {
	switch label {
	case LabelBTCRising:
		return fmt.Sprintf("BTC dominance %s — capital rotating into BTC",
			deltaDesc("rising", btcDelta))
	case LabelETHRising:
		return fmt.Sprintf("ETH dominance %s — capital rotating into ETH",
			deltaDesc("rising", ethDelta))
	case LabelCapitalContracting:
		return "Both BTC and ETH dominance falling — capital leaving dominant assets"
	case LabelNeutral:
		return "No significant dominance trend detected"
	default:
		return label
	}
}

func deltaDesc(direction string, delta *float64) string {
	if delta == nil {
		return direction
	}
	return fmt.Sprintf("%s (%+.1fpp)", direction, *delta)
}

// buildSummary constructs a one-line natural-language summary.
func buildSummary(btc, eth AssetDominance, label string, coldStart bool) string {
	btcDeltaStr := ""
	ethDeltaStr := ""

	if btc.DeltaPP != nil {
		btcDeltaStr = fmt.Sprintf(", %+.1fpp", btc.DeltaPP.Value())
	}
	if eth.DeltaPP != nil {
		ethDeltaStr = fmt.Sprintf(", %+.1fpp", eth.DeltaPP.Value())
	}

	base := fmt.Sprintf("BTC dominance %.1f%% (%s%s), ETH dominance %.1f%% (%s%s)",
		btc.Dominance.Value(), btc.Trend, btcDeltaStr,
		eth.Dominance.Value(), eth.Trend, ethDeltaStr)

	switch label {
	case LabelBTCRising:
		base += " — capital rotating into BTC."
	case LabelETHRising:
		base += " — capital rotating into ETH."
	case LabelCapitalContracting:
		base += " — both losing share."
	case LabelNeutral:
		if coldStart {
			base += " [UNVERIFIED — first run, no prior snapshot]."
		} else {
			base += " — no significant trend."
		}
	}

	return base
}
