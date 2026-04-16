package coingecko

import (
	"strings"
	"testing"
)

func TestCoinMarketsBreadthURL(t *testing.T) {
	url := CoinMarketsBreadthURL(250)
	if !strings.Contains(url, "/coins/markets") {
		t.Errorf("expected /coins/markets in URL, got %q", url)
	}
	if !strings.Contains(url, "per_page=250") {
		t.Errorf("expected per_page=250 in URL, got %q", url)
	}
	if !strings.Contains(url, "price_change_percentage=1h%2C24h%2C7d%2C30d") {
		t.Errorf("expected price_change_percentage param in URL, got %q", url)
	}
}

func TestCoinMarketsBreadthURL_CustomCount(t *testing.T) {
	url := CoinMarketsBreadthURL(100)
	if !strings.Contains(url, "per_page=100") {
		t.Errorf("expected per_page=100 in URL, got %q", url)
	}
}

// Fixture: 5 coins with mixed price changes across timeframes
const coinMarketsFixture = `[
  {"id":"bitcoin","symbol":"btc","price_change_percentage_1h_in_currency":2.5,"price_change_percentage_24h_in_currency":1.0,"price_change_percentage_7d_in_currency":-0.5,"price_change_percentage_30d_in_currency":5.0},
  {"id":"ethereum","symbol":"eth","price_change_percentage_1h_in_currency":-1.0,"price_change_percentage_24h_in_currency":3.0,"price_change_percentage_7d_in_currency":2.0,"price_change_percentage_30d_in_currency":-2.0},
  {"id":"tether","symbol":"usdt","price_change_percentage_1h_in_currency":0.01,"price_change_percentage_24h_in_currency":-0.02,"price_change_percentage_7d_in_currency":0.0,"price_change_percentage_30d_in_currency":0.01},
  {"id":"solana","symbol":"sol","price_change_percentage_1h_in_currency":5.0,"price_change_percentage_24h_in_currency":-3.0,"price_change_percentage_7d_in_currency":10.0,"price_change_percentage_30d_in_currency":20.0},
  {"id":"cardano","symbol":"ada","price_change_percentage_1h_in_currency":-0.5,"price_change_percentage_24h_in_currency":-2.0,"price_change_percentage_7d_in_currency":-5.0,"price_change_percentage_30d_in_currency":-10.0}
]`

func TestParseCoinMarketsBreadthResponse_Valid(t *testing.T) {
	result, err := ParseCoinMarketsBreadthResponse([]byte(coinMarketsFixture))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.CoinCount != 5 {
		t.Errorf("CoinCount: got %d, want 5", result.CoinCount)
	}
}

func TestParseCoinMarketsBreadthResponse_GreenFractions(t *testing.T) {
	result, err := ParseCoinMarketsBreadthResponse([]byte(coinMarketsFixture))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 1h: btc(+), tether(+), sol(+) = 3/5 = 0.60
	if abs(result.Green1h-0.60) > 0.001 {
		t.Errorf("Green1h: got %v, want 0.60", result.Green1h)
	}
	// 24h: btc(+), eth(+), tether(≈0, negative) = 2/5 = 0.40
	if abs(result.Green24h-0.40) > 0.001 {
		t.Errorf("Green24h: got %v, want 0.40", result.Green24h)
	}
	// 7d: eth(+), sol(+) = 2/5 = 0.40
	if abs(result.Green7d-0.40) > 0.001 {
		t.Errorf("Green7d: got %v, want 0.40", result.Green7d)
	}
	// 30d: btc(+), tether(+), sol(+) = 3/5 = 0.60
	if abs(result.Green30d-0.60) > 0.001 {
		t.Errorf("Green30d: got %v, want 0.60", result.Green30d)
	}
}

func TestParseCoinMarketsBreadthResponse_EmptyBody(t *testing.T) {
	_, err := ParseCoinMarketsBreadthResponse([]byte{})
	if err == nil {
		t.Error("expected error for empty body")
	}
}

func TestParseCoinMarketsBreadthResponse_InvalidJSON(t *testing.T) {
	_, err := ParseCoinMarketsBreadthResponse([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseCoinMarketsBreadthResponse_EmptyArray(t *testing.T) {
	_, err := ParseCoinMarketsBreadthResponse([]byte("[]"))
	if err == nil {
		t.Error("expected error for empty array")
	}
}

func TestParseCoinMarketsBreadthResponse_NullChanges(t *testing.T) {
	// Some coins may have null price change values — should be treated as not green
	fixture := `[
  {"id":"bitcoin","symbol":"btc","price_change_percentage_1h_in_currency":2.5,"price_change_percentage_24h_in_currency":null,"price_change_percentage_7d_in_currency":1.0,"price_change_percentage_30d_in_currency":1.0},
  {"id":"ethereum","symbol":"eth","price_change_percentage_1h_in_currency":-1.0,"price_change_percentage_24h_in_currency":3.0,"price_change_percentage_7d_in_currency":-2.0,"price_change_percentage_30d_in_currency":-2.0}
]`
	result, err := ParseCoinMarketsBreadthResponse([]byte(fixture))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 1h: btc(+) = 1/2 = 0.50
	if abs(result.Green1h-0.50) > 0.001 {
		t.Errorf("Green1h with null: got %v, want 0.50", result.Green1h)
	}
	// 24h: eth(+) = 1/2 = 0.50 (btc null counts as not green)
	if abs(result.Green24h-0.50) > 0.001 {
		t.Errorf("Green24h with null: got %v, want 0.50", result.Green24h)
	}
}

// --- Momentum Divergence ---

func TestCoinMarketsMomentumURL(t *testing.T) {
	url := CoinMarketsMomentumURL(200)
	if !strings.Contains(url, "/coins/markets") {
		t.Errorf("expected /coins/markets in URL, got %q", url)
	}
	if !strings.Contains(url, "per_page=200") {
		t.Errorf("expected per_page=200 in URL, got %q", url)
	}
	if !strings.Contains(url, "price_change_percentage=24h") {
		t.Errorf("expected price_change_percentage=24h in URL, got %q", url)
	}
}

func TestParseCoinMarketsMomentumResponse_Valid(t *testing.T) {
	fixture := `[
		{"id":"bitcoin","symbol":"btc","price_change_percentage_24h_in_currency":5.0,"total_volume":50000000000},
		{"id":"ethereum","symbol":"eth","price_change_percentage_24h_in_currency":-2.0,"total_volume":20000000000}
	]`
	result, err := ParseCoinMarketsMomentumResponse([]byte(fixture))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.PriceChanges) != 2 {
		t.Errorf("expected 2 entries, got %d", len(result.PriceChanges))
	}
	if result.PriceChanges[0] != 5.0 {
		t.Errorf("first price change: got %v, want 5.0", result.PriceChanges[0])
	}
	if result.PriceChanges[1] != -2.0 {
		t.Errorf("second price change: got %v, want -2.0", result.PriceChanges[1])
	}
	if result.Volumes[0] != 50000000000 {
		t.Errorf("first volume: got %v, want 50000000000", result.Volumes[0])
	}
}

func TestParseCoinMarketsMomentumResponse_NullChange(t *testing.T) {
	fixture := `[
		{"id":"bitcoin","symbol":"btc","total_volume":50000000000},
		{"id":"ethereum","symbol":"eth","price_change_percentage_24h_in_currency":3.0,"total_volume":20000000000}
	]`
	result, err := ParseCoinMarketsMomentumResponse([]byte(fixture))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PriceChanges[0] != 0 {
		t.Errorf("null change should be 0, got %v", result.PriceChanges[0])
	}
	if result.PriceChanges[1] != 3.0 {
		t.Errorf("second price change: got %v, want 3.0", result.PriceChanges[1])
	}
}

func TestParseCoinMarketsMomentumResponse_EmptyBody(t *testing.T) {
	_, err := ParseCoinMarketsMomentumResponse([]byte{})
	if err == nil {
		t.Error("expected error for empty body")
	}
}

func TestParseCoinMarketsMomentumResponse_InvalidJSON(t *testing.T) {
	_, err := ParseCoinMarketsMomentumResponse([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseCoinMarketsMomentumResponse_EmptyArray(t *testing.T) {
	_, err := ParseCoinMarketsMomentumResponse([]byte("[]"))
	if err == nil {
		t.Error("expected error for empty array")
	}
}
