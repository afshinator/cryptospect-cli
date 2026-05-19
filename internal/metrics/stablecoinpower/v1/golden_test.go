package v1

import (
	"context"
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/metrics"
)

// TestGolden_Happy validates the "normal" stablecoin-power classification.
// Formula:  stableMcap  = Σ top-8 stables by market cap
//
//	volatileMcap = totalMcap - stableMcap
//	ratio = stableMcap / volatileMcap
//	classify: >= 0.15 → high, >= 0.07 → normal, < 0.07 → low
//
// Input:    totalMcap=3e12, 8 stables each 27e9 → stableMcap=216e9, DefiLlama cur=216e9 prev=215e9
// Expected: volatileMcap = 3e12 - 216e9 = 2.784e12
//
//	ratio = 216e9 / 2.784e12 = 0.077586… → 0.0776 (4dp)
//	0.0776 >= 0.07 → "normal", "Healthy Balance"
//	supply trend: (216e9-215e9)/215e9 ≈ 0.0047 < ±1% → "stable"
//	discrepancy: |216e9-216e9|/216e9 = 0 → confidence "high"
func TestGolden_Happy(t *testing.T) {
	dm := dataMap3(
		cgGlobalJSON(3e12),
		cgStablesJSON(8, 27e9),
		dlJSON(216e9, 215e9),
	)
	p := &Provider{}
	result, err := p.Compute(context.Background(), dm)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	metrics.AssertMatchesGolden(t, "../../testdata/golden/stablecoin-power/happy.golden", result.Data)
}

// TestGolden_High validates the "high" stablecoin-power classification.
// Formula:  stableMcap  = Σ top-8 stables by market cap
//
//	volatileMcap = totalMcap - stableMcap
//	ratio = stableMcap / volatileMcap
//	classify: >= 0.15 → high, >= 0.07 → normal, < 0.07 → low
//
// Input:    totalMcap=3e12, 8 stables each 80e9 → stableMcap=640e9, DefiLlama cur=640e9 prev=638e9
// Expected: volatileMcap = 3e12 - 640e9 = 2.36e12
//
//	ratio = 640e9 / 2.36e12 = 0.271186… → 0.2712 (4dp)
//	0.2712 >= 0.15 → "high", "Dry Powder Alert"
//	supply trend: (640e9-638e9)/638e9 ≈ 0.0031 < ±1% → "stable"
//	discrepancy: |640e9-640e9|/640e9 = 0 → confidence "high"
func TestGolden_High(t *testing.T) {
	dm := dataMap3(
		cgGlobalJSON(3e12),
		cgStablesJSON(8, 80e9),
		dlJSON(640e9, 638e9),
	)
	p := &Provider{}
	result, err := p.Compute(context.Background(), dm)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	metrics.AssertMatchesGolden(t, "../../testdata/golden/stablecoin-power/high.golden", result.Data)
}
