package v1

import (
	"encoding/json"
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/metrics"
)

// TestGolden_ColdStart validates cold start output (no prior dominance snapshot).
// BTC: 52.3%, ETH: 18.1%. No prior data → trends neutral, deltas null.
// ETH/BTC ratio: 18.1 / 52.3 = 0.34608...
// Classification: neutral (unverified).
func TestGolden_ColdStart(t *testing.T) {
	in := Input{
		BTCDominancePct: 52.3,
		ETHDominancePct: 18.1,
		// No prior data
	}

	data, _, err := Compute(in)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}

	actual, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	metrics.AssertMatchesGolden(t, "../../testdata/golden/dominance/cold_start.golden", actual)
}

// TestGolden_BTCRising validates btc_rising classification.
// BTC: 51.0% → 52.3% (+1.3pp → rising), ETH: 18.7% → 18.1% (−0.6pp → falling).
// Label: btc_rising. ETH/BTC ratio: 18.1/52.3 = 0.346...
func TestGolden_BTCRising(t *testing.T) {
	in := Input{
		BTCDominancePct:     52.3,
		ETHDominancePct:     18.1,
		PriorBTCDominance:   ptr(51.0),
		PriorETHDominance:   ptr(18.7),
		PriorSnapshotAgeSec: ptr(3600),
	}

	data, _, err := Compute(in)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}

	actual, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	metrics.AssertMatchesGolden(t, "../../testdata/golden/dominance/btc_rising.golden", actual)
}
