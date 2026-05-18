package v1

import (
	"encoding/json"
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/metrics"
)

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
