package v1

import (
	"encoding/json"
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/metrics"
)

// TestGolden_BTCLedExpansion validates the 10-regime matrix lookup for rising BTC dominance
// with broad breadth and normal conviction.
//
// Dominance trend:  delta = 53.0 - 52.0 = +1.0 >= 0.5 (DomDeadBandPP) → "rising"
// Breadth band:     0.75 >= 0.60 (BreadthBroadThresh) → "broad"
// Conviction:       LPRatio=0.10 >= 0.07 (ConvictionLowThresh), < 0.15 (ConvictionHighThresh) → "normal"
// Modifier:         BTCChange24h=2.5 >= 0.5 (ModifierDeadBandPP) → "positive_momentum"
//
// Matrix: rising + broad → BTC-Led Expansion
// Confidence: no coldStart (PriorDominancePct+age provided), no weightRedist, no breadthDegraded,
//
//	no missingRef, no capNote → "high"
func TestGolden_BTCLedExpansion(t *testing.T) {
	priorDom := 52.0
	input := Input{
		BTCDominancePct:     53.0,
		PriorDominancePct:   &priorDom,
		PriorSnapshotAgeSec: ptr(86400),
		BreadthScore:        0.75,
		LPRatio:             0.10,
		BTCChange24h:        ptr(2.5),
	}
	result := Compute(input)

	d := Data{
		Regime:             result.Regime,
		Modifier:           result.Modifier,
		DominanceTrend:     result.DominanceTrend,
		Conviction:         result.Conviction,
		MarketBreadthScore: metrics.Ratio(input.BreadthScore),
		Classification: Classification{
			Label:       result.Regime,
			Description: classificationDescription(result.Regime, result.Conviction),
		},
		Summary: result.Summary,
	}
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	metrics.AssertMatchesGolden(t, "../../testdata/golden/market-regime/btc-led-expansion.golden", b)
}

// TestGolden_Stagnation validates the 10-regime matrix lookup for neutral dominance
// with narrow breadth and low conviction.
//
// Dominance trend:  delta = 52.0 - 52.3 = -0.3, within [-0.5, 0.5] deadband → "neutral"
// Breadth band:     0.35 < 0.40 (BreadthNarrowThresh) → "narrow"
// Conviction:       LPRatio=0.04 < 0.07 (ConvictionLowThresh) → "low"
// Modifier:         BTCChange24h=0.3, within [-0.5, 0.5] deadband → "neutral"
//
// Matrix: neutral + narrow → Stagnation (conviction "low" → not pressure-cooker variant)
// Confidence: no flags → "high"
func TestGolden_Stagnation(t *testing.T) {
	input := Input{
		BTCDominancePct:     52.0,
		PriorDominancePct:   ptr(52.3),
		PriorSnapshotAgeSec: ptr(86400),
		BreadthScore:        0.35,
		LPRatio:             0.04,
		BTCChange24h:        ptr(0.3),
	}
	result := Compute(input)

	d := Data{
		Regime:             result.Regime,
		Modifier:           result.Modifier,
		DominanceTrend:     result.DominanceTrend,
		Conviction:         result.Conviction,
		MarketBreadthScore: metrics.Ratio(input.BreadthScore),
		Classification: Classification{
			Label:       result.Regime,
			Description: classificationDescription(result.Regime, result.Conviction),
		},
		Summary: result.Summary,
	}
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	metrics.AssertMatchesGolden(t, "../../testdata/golden/market-regime/stagnation.golden", b)
}
