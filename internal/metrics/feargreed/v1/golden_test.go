package v1

import (
	"encoding/json"
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/metrics"
)

// TestGolden_ExtremeFear validates extreme_fear classification with MA.
// Value: 10. MA of 7 values [10,15,20,12,18,14,22] = 111/7 ≈ 15.86.
// 10 < 15.86 − 2 → deteriorating.
func TestGolden_ExtremeFear(t *testing.T) {
	data, _, err := Compute(Input{Values: []int{10, 15, 20, 12, 18, 14, 22}})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	actual, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	metrics.AssertMatchesGolden(t, "../../testdata/golden/fear-greed-index/extreme_fear.golden", actual)
}

// TestGolden_Greed validates greed classification with insufficient history.
// Only 3 data points — no MA, no trend, confidence medium.
// 65 > 55 AND ≤ 75 → greed.
func TestGolden_Greed(t *testing.T) {
	data, _, err := Compute(Input{Values: []int{65, 60, 68}})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	actual, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	metrics.AssertMatchesGolden(t, "../../testdata/golden/fear-greed-index/greed.golden", actual)
}
