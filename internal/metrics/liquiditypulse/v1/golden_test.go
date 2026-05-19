package v1

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/api"
	"github.com/afshinator/cryptospect-cli/internal/metrics"
)

// TestGolden_Happy validates the "normal" classification boundary.
// Formula:  ratio = volume_24h / market_cap
//
//	classify: >= 0.15 → high, >= 0.05 → normal, < 0.05 → low
//
// Input:    volume = 1.0e9, mcap = 8.0e9
// Expected: ratio = 1.0e9 / 8.0e9 = 0.125 → 0.125 >= 0.05, < 0.15 → "normal"
// Summary:  Volume/MCap: 12.50% | Conviction: normal
func TestGolden_Happy(t *testing.T) {
	p := &Provider{}
	result, err := p.Compute(context.Background(), map[string]json.RawMessage{
		api.CoinGeckoGlobalMarket: json.RawMessage(`{
			"data": {
				"total_volume": {"usd": 1000000000},
				"total_market_cap": {"usd": 8000000000}
			}
		}`),
	})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	metrics.AssertMatchesGolden(t, "../../testdata/golden/liquidity-pulse/happy.golden", result.Data)
}

// TestGolden_High validates the "high" classification boundary.
// Formula:  ratio = volume_24h / market_cap
//
//	classify: >= 0.15 → high, >= 0.05 → normal, < 0.05 → low
//
// Input:    volume = 1.6e9, mcap = 8.0e9
// Expected: ratio = 1.6e9 / 8.0e9 = 0.20 → 0.20 >= 0.15 → "high"
// Summary:  Volume/MCap: 20.00% | Conviction: high
func TestGolden_High(t *testing.T) {
	p := &Provider{}
	result, err := p.Compute(context.Background(), map[string]json.RawMessage{
		api.CoinGeckoGlobalMarket: json.RawMessage(`{
			"data": {
				"total_volume": {"usd": 1600000000},
				"total_market_cap": {"usd": 8000000000}
			}
		}`),
	})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	metrics.AssertMatchesGolden(t, "../../testdata/golden/liquidity-pulse/high.golden", result.Data)
}
