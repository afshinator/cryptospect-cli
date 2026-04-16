package coingecko

import (
	"strings"
	"testing"
)

func TestGlobalURL(t *testing.T) {
	url := GlobalURL()
	if !strings.Contains(url, "/global") {
		t.Errorf("expected /global in URL, got %q", url)
	}
}

const globalFixture = `{
  "data": {
    "total_market_cap": {
      "usd": 2500000000000
    },
    "total_volume": {
      "usd": 80000000000
    },
    "market_cap_change_percentage_24h_usd": 2.5
  }
}`

func TestParseGlobalResponse_Valid(t *testing.T) {
	data, err := ParseGlobalResponse([]byte(globalFixture))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cap, ok := data.GetMarketCapUSD(); !ok || cap != 2500000000000 {
		t.Errorf("GetMarketCapUSD: got %v (%v), want 2.5e12", cap, ok)
	}
	if vol, ok := data.GetVolumeUSD(); !ok || vol != 80000000000 {
		t.Errorf("GetVolumeUSD: got %v (%v), want 8e10", vol, ok)
	}
	if data.MarketCapChangePercentage24hUsd != 2.5 {
		t.Errorf("MarketCapChangePercentage24hUsd: got %v, want 2.5", data.MarketCapChangePercentage24hUsd)
	}
}

func TestParseGlobalResponse_EmptyBody(t *testing.T) {
	_, err := ParseGlobalResponse([]byte{})
	if err == nil {
		t.Error("expected error for empty body")
	}
}

func TestParseGlobalResponse_InvalidJSON(t *testing.T) {
	_, err := ParseGlobalResponse([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseGlobalDominance_Valid(t *testing.T) {
	fixture := `{
		"data": {
			"market_cap_percentage": {
				"btc": 52.5,
				"eth": 18.2
			}
		}
	}`
	dom := ParseGlobalDominance([]byte(fixture))
	if dom == nil {
		t.Fatal("expected non‑nil dominance")
	}
	if *dom != 52.5 {
		t.Errorf("dominance: got %v, want 52.5", *dom)
	}
}

func TestParseGlobalDominance_NoBTC(t *testing.T) {
	fixture := `{
		"data": {
			"market_cap_percentage": {
				"eth": 100
			}
		}
	}`
	dom := ParseGlobalDominance([]byte(fixture))
	if dom != nil {
		t.Errorf("expected nil dominance, got %v", *dom)
	}
}

func TestParseGlobalDominance_EmptyBody(t *testing.T) {
	dom := ParseGlobalDominance([]byte{})
	if dom != nil {
		t.Errorf("expected nil dominance for empty body, got %v", *dom)
	}
}

func TestParseGlobalDominance_InvalidJSON(t *testing.T) {
	dom := ParseGlobalDominance([]byte("not json"))
	if dom != nil {
		t.Errorf("expected nil dominance for invalid JSON, got %v", *dom)
	}
}

func TestStablesMarketsURL(t *testing.T) {
	url := StablesMarketsURL()
	if !strings.Contains(url, "/coins/markets") {
		t.Errorf("expected /coins/markets in URL, got %q", url)
	}
	if !strings.Contains(url, "ids=") {
		t.Errorf("expected ids param in URL, got %q", url)
	}
	// Should contain at least tether
	if !strings.Contains(url, "tether") {
		t.Errorf("expected tether in ids list, got %q", url)
	}
}

const stablesFixture = `[
  {
    "id": "tether",
    "symbol": "usdt",
    "market_cap": 100000000000,
    "total_volume": 50000000000,
    "market_cap_change_percentage_24h": 0.1,
    "current_price": 1.0
  },
  {
    "id": "usd-coin",
    "symbol": "usdc",
    "market_cap": 30000000000,
    "total_volume": 20000000000,
    "market_cap_change_percentage_24h": -0.05,
    "current_price": 1.0
  }
]`

func TestParseStablesMarketsResponse_Valid(t *testing.T) {
	markets, err := ParseStablesMarketsResponse([]byte(stablesFixture))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(markets) != 2 {
		t.Errorf("expected 2 markets, got %d", len(markets))
	}
	if markets[0].ID != "tether" {
		t.Errorf("first market ID: got %q, want tether", markets[0].ID)
	}
	if markets[0].MarketCap != 100000000000 {
		t.Errorf("first market cap: got %v, want 1e11", markets[0].MarketCap)
	}
	if markets[1].McapChange24h != -0.05 {
		t.Errorf("second 24h change: got %v, want -0.05", markets[1].McapChange24h)
	}
}

func TestParseStablesMarketsResponse_EmptyBody(t *testing.T) {
	_, err := ParseStablesMarketsResponse([]byte{})
	if err == nil {
		t.Error("expected error for empty body")
	}
}

func TestParseStablesMarketsResponse_InvalidJSON(t *testing.T) {
	_, err := ParseStablesMarketsResponse([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseTopGainers_Valid(t *testing.T) {
	fixture := `[
		{"symbol":"btc","price_change_percentage_24h_in_currency":10.5},
		{"symbol":"eth","price_change_percentage_24h_in_currency":-2.0},
		{"symbol":"sol","price_change_percentage_24h_in_currency":25.0}
	]`
	gainers, err := ParseTopGainers([]byte(fixture), 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gainers) != 2 {
		t.Errorf("expected 2 gainers (limit), got %d", len(gainers))
	}
	if gainers[0].Symbol != "SOL" {
		t.Errorf("first gainer symbol: got %q, want SOL", gainers[0].Symbol)
	}
	if gainers[0].Change24h == nil || *gainers[0].Change24h != 25.0 {
		t.Errorf("first gainer change: got %v, want 25.0", gainers[0].Change24h)
	}
	if gainers[1].Symbol != "BTC" {
		t.Errorf("second gainer symbol: got %q, want BTC", gainers[1].Symbol)
	}
}

func TestParseTopGainers_NullChanges(t *testing.T) {
	fixture := `[
		{"symbol":"btc"},
		{"symbol":"eth","price_change_percentage_24h_in_currency":5.0}
	]`
	gainers, err := ParseTopGainers([]byte(fixture), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gainers) != 2 {
		t.Errorf("expected 2 gainers, got %d", len(gainers))
	}
	if gainers[0].Symbol != "ETH" {
		t.Errorf("first gainer symbol: got %q, want ETH", gainers[0].Symbol)
	}
	if gainers[0].Change24h == nil || *gainers[0].Change24h != 5.0 {
		t.Errorf("first gainer change: got %v, want 5.0", gainers[0].Change24h)
	}
	if gainers[1].Symbol != "BTC" {
		t.Errorf("second gainer symbol: got %q, want BTC", gainers[1].Symbol)
	}
	if gainers[1].Change24h == nil || *gainers[1].Change24h != 0.0 {
		t.Errorf("second gainer change: got %v, want 0.0", gainers[1].Change24h)
	}
}

func TestParseTopGainers_EmptyBody(t *testing.T) {
	_, err := ParseTopGainers([]byte{}, 5)
	if err == nil {
		t.Error("expected error for empty body")
	}
}

func TestParseTopGainers_InvalidJSON(t *testing.T) {
	_, err := ParseTopGainers([]byte("not json"), 5)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
