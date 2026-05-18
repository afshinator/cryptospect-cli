package v1

import (
	"encoding/json"
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/metrics"
)

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
