package coingecko

import (
	"strings"
	"testing"
)

// DerivativesURL tests

func TestDerivativesURL_NoKey(t *testing.T) {
	url := DerivativesURL("")
	if !strings.Contains(url, "/derivatives") {
		t.Errorf("expected /derivatives in URL, got %q", url)
	}
	if strings.Contains(url, "x_cg_demo_api_key") {
		t.Errorf("expected no key param when key is empty, got %q", url)
	}
}

func TestDerivativesURL_WithKey(t *testing.T) {
	url := DerivativesURL("test-key")
	if !strings.Contains(url, "x_cg_demo_api_key=test-key") {
		t.Errorf("expected key param in URL, got %q", url)
	}
}

// ParseDerivativesResponse tests

// Fixture: 3 BTC perpetuals + 1 BTC futures (should be filtered out) + 1 ETH perpetual
const derivativesFixture = `[
  {"market":"Binance (Futures)","symbol":"BTCUSDT","index_id":"BTC","contract_type":"perpetual","funding_rate":-0.002924,"open_interest":5873824027.52},
  {"market":"Bitmart Futures","symbol":"BTCUSDT","index_id":"BTC","contract_type":"perpetual","funding_rate":0.00852,"open_interest":6486026274.58},
  {"market":"Gate (Futures)","symbol":"BTCUSDT","index_id":"BTC","contract_type":"perpetual","funding_rate":0.0023,"open_interest":4017902098.97},
  {"market":"Binance","symbol":"BTC-26JUN26","index_id":"BTC","contract_type":"futures","funding_rate":0,"open_interest":100000000},
  {"market":"Binance (Futures)","symbol":"ETHUSDT","index_id":"ETH","contract_type":"perpetual","funding_rate":-0.001,"open_interest":2000000000}
]`

func TestParseDerivativesResponse_Valid(t *testing.T) {
	result, err := ParseDerivativesResponse([]byte(derivativesFixture))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalOpenInterest == 0 {
		t.Error("expected non-zero TotalOpenInterest")
	}
	if result.MedianFundingRate == 0 {
		t.Error("expected non-zero MedianFundingRate")
	}
}

func TestParseDerivativesResponse_FiltersPerpetualOnly(t *testing.T) {
	result, err := ParseDerivativesResponse([]byte(derivativesFixture))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 3 BTC perpetuals + 1 ETH perpetual = 4 perpetual entries, but we only count BTC
	// Total OI should be sum of 3 BTC perpetuals only
	wantOI := 5873824027.52 + 6486026274.58 + 4017902098.97
	if abs(result.TotalOpenInterest-wantOI) > 1 {
		t.Errorf("TotalOpenInterest: got %.2f, want %.2f", result.TotalOpenInterest, wantOI)
	}
}

func TestParseDerivativesResponse_FiltersBTCOnly(t *testing.T) {
	result, err := ParseDerivativesResponse([]byte(derivativesFixture))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// ETH entry should not be included in BTC aggregation
	// Median funding rate should be median of [-0.002924, 0.00852, 0.0023]
	// Sorted: [-0.002924, 0.0023, 0.00852] → median = 0.0023
	wantMedian := 0.0023
	if abs(result.MedianFundingRate-wantMedian) > 1e-6 {
		t.Errorf("MedianFundingRate: got %.6f, want %.6f", result.MedianFundingRate, wantMedian)
	}
}

func TestParseDerivativesResponse_EmptyBody(t *testing.T) {
	_, err := ParseDerivativesResponse([]byte{})
	if err == nil {
		t.Error("expected error for empty body")
	}
}

func TestParseDerivativesResponse_InvalidJSON(t *testing.T) {
	_, err := ParseDerivativesResponse([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseDerivativesResponse_NoBTCPerpetuals(t *testing.T) {
	// Only ETH entries — no BTC data
	onlyETH := `[{"market":"Binance (Futures)","symbol":"ETHUSDT","index_id":"ETH","contract_type":"perpetual","funding_rate":-0.001,"open_interest":2000000000}]`
	result, err := ParseDerivativesResponse([]byte(onlyETH))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalOpenInterest != 0 {
		t.Errorf("expected zero OI when no BTC perpetuals, got %v", result.TotalOpenInterest)
	}
	if result.MedianFundingRate != 0 {
		t.Errorf("expected zero median when no BTC perpetuals, got %v", result.MedianFundingRate)
	}
}

func TestParseDerivativesResponse_SingleEntry(t *testing.T) {
	single := `[{"market":"Binance (Futures)","symbol":"BTCUSDT","index_id":"BTC","contract_type":"perpetual","funding_rate":-0.002924,"open_interest":5873824027.52}]`
	result, err := ParseDerivativesResponse([]byte(single))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if abs(result.TotalOpenInterest-5873824027.52) > 1 {
		t.Errorf("TotalOpenInterest: got %.2f, want 5873824027.52", result.TotalOpenInterest)
	}
	if abs(result.MedianFundingRate-(-0.002924)) > 1e-6 {
		t.Errorf("MedianFundingRate: got %.6f, want -0.002924", result.MedianFundingRate)
	}
}

func TestParseDerivativesResponse_EvenCountMedian(t *testing.T) {
	// 2 entries → median is average of the two
	two := `[
  {"market":"A","symbol":"BTCUSDT","index_id":"BTC","contract_type":"perpetual","funding_rate":0.001,"open_interest":1000},
  {"market":"B","symbol":"BTCUSDT","index_id":"BTC","contract_type":"perpetual","funding_rate":0.003,"open_interest":2000}
]`
	result, err := ParseDerivativesResponse([]byte(two))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Median of [0.001, 0.003] = 0.002
	if abs(result.MedianFundingRate-0.002) > 1e-6 {
		t.Errorf("MedianFundingRate: got %.6f, want 0.002", result.MedianFundingRate)
	}
}
