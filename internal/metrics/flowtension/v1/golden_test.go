package v1

import (
	"encoding/json"
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/metrics"
)

// TestGolden_FullSignals validates all three flow-tension sub-signals with complete data.
//
// CVD:  ratio = (takerBuy - takerSell) / totalVolume = (70 - 30) / 100 = 0.40
//
//	hook: 0.40 >= 0.10 (FlowAggressiveThreshold) → "aggressive_buy"
//
// OI:   change = (curr - prev) / prev = (18.5e9 - 17.4e9) / 17.4e9 = 0.06322…
//
//	→ 0.0632 (4dp). hook: 0.0632 > 0.05 (OIBuildingThreshold) → "building"
//
// Funding: 0.0003 >= 0.0003 (FundingPositiveThreshold), < 0.003 (Overheated) → "positive"
//
// Summary: building + aggressive_buy → "Leverage building with aggressive buying — tension coiling, breakout likely."
func TestGolden_FullSignals(t *testing.T) {
	input := Input{
		TakerBuyVolume:    70,
		TakerSellVolume:   30,
		TotalVolume:       100,
		NumTrades:         50,
		TotalOpenInterest: 18500000000,
		PrevOI:            ptr(17400000000),
		FundingRate:       0.0003,
	}
	data, err := Compute(input)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	metrics.AssertMatchesGolden(t, "../../testdata/golden/flow-tension/full-signals.golden", b)
}

// TestGolden_Degraded validates flow-tension with aggressive-selling CVD and missing OI/funding data.
//
// CVD:  ratio = (30 - 70) / 100 = -0.40
//
//	hook: -0.40 <= -0.10 (FlowAggressiveThreshold) → "aggressive_sell"
//
// OI:   zero TotalOpenInterest → stable hook, no change_pct_24h field
//
// Funding: 0 → hook "neutral" (within [-0.0003, 0.0003] deadband)
//
// Summary: aggressive_sell + stable → "Assets staged on exchanges with aggressive selling — supply shock / top warning."
func TestGolden_Degraded(t *testing.T) {
	input := Input{
		TakerBuyVolume:  30,
		TakerSellVolume: 70,
		TotalVolume:     100,
		NumTrades:       50,
	}
	data, err := Compute(input)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	metrics.AssertMatchesGolden(t, "../../testdata/golden/flow-tension/degraded.golden", b)
}
