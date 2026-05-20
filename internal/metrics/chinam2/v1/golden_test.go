package v1

import (
	"encoding/json"
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/metrics"
)

// TestGolden_Expanding validates expanding classification with real data.
// 13 months from real DBnomics data (Feb 2025 - Feb 2026).
// Latest: 34,922, 12mo ago: 32,051.74 → YoY ≈ 8.95% → "expanding".
func TestGolden_Expanding(t *testing.T) {
	obs := []Observation{
		{Period: "2026-02", Value: 34922.0},
		{Period: "2026-01", Value: 34718.6},
		{Period: "2025-12", Value: 34029.48},
		{Period: "2025-11", Value: 33698.9},
		{Period: "2025-10", Value: 33513.12},
		{Period: "2025-09", Value: 33537.71},
		{Period: "2025-08", Value: 33198.31},
		{Period: "2025-07", Value: 32994.29},
		{Period: "2025-06", Value: 33028.68},
		{Period: "2025-05", Value: 32578.38},
		{Period: "2025-04", Value: 32517.39},
		{Period: "2025-03", Value: 32605.55},
		{Period: "2025-02", Value: 32051.74},
	}

	data, _, err := Compute(Input{Observations: obs})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	actual, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	metrics.AssertMatchesGolden(t, "../../testdata/golden/china-m2/expanding.golden", actual)
}

// TestGolden_InsufficientHistory validates insufficient history output.
// Single data point — no YoY, confidence medium.
func TestGolden_InsufficientHistory(t *testing.T) {
	obs := []Observation{
		{Period: "2026-02", Value: 34922.0},
	}

	data, _, err := Compute(Input{Observations: obs})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	actual, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	metrics.AssertMatchesGolden(t, "../../testdata/golden/china-m2/insufficient_history.golden", actual)
}
