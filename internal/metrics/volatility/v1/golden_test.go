package v1

import (
	"encoding/json"
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/metrics"
)

// TestGolden_Normal validates normal spread classification.
// BTC: slow steady uptrend (50000→51050), ETH: similar slow uptrend (3000→3120).
// Both have low, similar volatility → spread close to 1.0 → "normal".
func TestGolden_Normal(t *testing.T) {
	btcCloses := []float64{
		50000, 50100, 50200, 50150, 50080, 50120, 50250, 50300,
		50280, 50400, 50350, 50500, 50480, 50600, 50550, 50700,
		50680, 50800, 50750, 50900, 50880, 51000, 50950, 51050,
	}
	ethCloses := []float64{
		3000, 3010, 3020, 3015, 3008, 3012, 3030, 3040,
		3035, 3050, 3045, 3060, 3055, 3070, 3065, 3080,
		3075, 3090, 3085, 3100, 3095, 3110, 3105, 3120,
	}

	data, _, err := Compute(Input{BTCCloses: btcCloses, ETHCloses: ethCloses})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}

	actual, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	metrics.AssertMatchesGolden(t, "../../testdata/golden/volatility/normal.golden", actual)
}

// TestGolden_Elevated validates elevated spread classification.
// BTC: very stable (~50000 ± small noise), ETH: oscillating wildly (3000→4300→1900).
// High ETH vol, low BTC vol → spread > 1.5 → "elevated".
func TestGolden_Elevated(t *testing.T) {
	btcCloses := []float64{
		50000, 50010, 49990, 50005, 50000, 50015, 49995, 50010,
		50000, 50020, 49990, 50005, 50000, 50010, 50000, 50005,
		49995, 50010, 50000, 50005, 49990, 50010, 50000, 50005,
	}
	ethCloses := []float64{
		3000, 3200, 2900, 3300, 2800, 3400, 2700, 3500,
		2600, 3600, 2500, 3700, 2400, 3800, 2300, 3900,
		2200, 4000, 2100, 4100, 2000, 4200, 1900, 4300,
	}

	data, _, err := Compute(Input{BTCCloses: btcCloses, ETHCloses: ethCloses})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}

	actual, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	metrics.AssertMatchesGolden(t, "../../testdata/golden/volatility/elevated.golden", actual)
}
