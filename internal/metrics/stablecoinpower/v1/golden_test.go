package v1

import (
	"context"
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/metrics"
)

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
