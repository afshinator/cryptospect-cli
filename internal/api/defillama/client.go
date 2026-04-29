// defillama/client.go — DefiLlama stablecoins API client.
package defillama

import (
	"encoding/json"
	"fmt"
)

// BaseURL is the base URL for the DefiLlama stablecoins API.
const BaseURL = "https://stablecoins.llama.fi"

// Circulating holds the USD-equivalent circulating supply for a stablecoin.
// Non-USD stablecoins (e.g., EURC) may have PeggedUSD == 0 if DefiLlama does
// not convert their peg to USD in this field; they are excluded from the sum.
type Circulating struct {
	PeggedUSD float64 `json:"peggedUSD"`
}

// PeggedAsset is a single stablecoin entry from the /stablecoins endpoint.
// Only the fields used by stablecoin-power are declared; the full response
// also includes per-chain breakdowns and other metadata.
type PeggedAsset struct {
	ID                  string      `json:"id"`
	Symbol              string      `json:"symbol"`
	Name                string      `json:"name"`
	Circulating         Circulating `json:"circulating"`
	CirculatingPrevWeek Circulating `json:"circulatingPrevWeek"`
}

// StablecoinsResponse is the top-level response from /stablecoins.
type StablecoinsResponse struct {
	PeggedAssets []PeggedAsset `json:"peggedAssets"`
}

// StablecoinsURL returns the URL for the DefiLlama /stablecoins endpoint.
func StablecoinsURL() string {
	return fmt.Sprintf("%s/stablecoins", BaseURL)
}

// ParseStablecoinsResponse unmarshals the raw JSON body from /stablecoins.
func ParseStablecoinsResponse(body []byte) (StablecoinsResponse, error) {
	if len(body) == 0 {
		return StablecoinsResponse{}, fmt.Errorf("empty response body")
	}
	var resp StablecoinsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return StablecoinsResponse{}, fmt.Errorf("parsing stablecoins response: %w", err)
	}
	return resp, nil
}

// AggregateSupply sums peggedUSD across all assets for current and prev-week supply.
// Assets with zero PeggedUSD (non-USD stablecoins without a USD conversion) are
// included as zero — consistent with how CoinGecko denominates all caps in USD.
func AggregateSupply(resp StablecoinsResponse) (current, prevWeek float64) {
	for _, a := range resp.PeggedAssets {
		current += a.Circulating.PeggedUSD
		prevWeek += a.CirculatingPrevWeek.PeggedUSD
	}
	return current, prevWeek
}

const (
	// TrendExpanding signals stablecoin supply grew > +1% over 7 days.
	TrendExpanding = "expanding"
	// TrendStable signals stablecoin supply changed within ±1% over 7 days.
	TrendStable = "stable"
	// TrendContracting signals stablecoin supply shrank > 1% over 7 days.
	TrendContracting = "contracting"

	trendThreshold = 0.01 // ±1%
)

// ClassifyTrend returns the 7-day supply trend given current and prevWeek totals.
// Returns "stable" when prevWeek is zero to avoid a division-by-zero on newly
// tracked assets where historical data is absent.
func ClassifyTrend(current, prevWeek float64) string {
	if prevWeek == 0 {
		return TrendStable
	}
	pct := (current - prevWeek) / prevWeek
	switch {
	case pct >= trendThreshold:
		return TrendExpanding
	case pct <= -trendThreshold:
		return TrendContracting
	default:
		return TrendStable
	}
}
