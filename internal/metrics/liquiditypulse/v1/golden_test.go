package v1

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/api"
	"github.com/afshinator/cryptospect-cli/internal/metrics"
)

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
