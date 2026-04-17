package binance

import (
	"testing"
)

// Fixture captured from live API: api.binance.us/api/v3/klines?symbol=BTCUSDT&interval=1h&limit=1
// Fields: [openTime, open, high, low, close, volume, closeTime, quoteVol, trades, takerBuyBaseVol, takerBuyQuoteVol, ignore]
const klinesFixture = `[[1775088000000,"68114.37000000","68304.58000000","68065.69000000","68304.58000000","0.04166000",1775091599999,"2838.36038640",48,"0.01687000","1149.81361760","0"]]`

const klinesEmptyFixture = `[]`

const klinesInvalidFixture = `not json`

func TestParseKlinesResponse_Valid(t *testing.T) {
	result, err := ParseKlinesResponse([]byte(klinesFixture))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalVolume == 0 {
		t.Error("expected non-zero TotalVolume")
	}
	if result.TakerBuyVolume == 0 {
		t.Error("expected non-zero TakerBuyVolume")
	}
	// takerSellVol = totalVol - takerBuyVol; must be >= 0
	if result.TakerSellVolume < 0 {
		t.Errorf("TakerSellVolume must be >= 0, got %v", result.TakerSellVolume)
	}
	// takerBuyVol must be <= totalVol
	if result.TakerBuyVolume > result.TotalVolume {
		t.Errorf("TakerBuyVolume (%v) > TotalVolume (%v)", result.TakerBuyVolume, result.TotalVolume)
	}
}

func TestParseKlinesResponse_CorrectValues(t *testing.T) {
	result, err := ParseKlinesResponse([]byte(klinesFixture))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// volume = 0.04166, takerBuyBaseVol = 0.01687
	wantTotal := 0.04166
	wantBuy := 0.01687
	wantSell := wantTotal - wantBuy

	if abs(result.TotalVolume-wantTotal) > 1e-8 {
		t.Errorf("TotalVolume: expected %.5f, got %.5f", wantTotal, result.TotalVolume)
	}
	if abs(result.TakerBuyVolume-wantBuy) > 1e-8 {
		t.Errorf("TakerBuyVolume: expected %.5f, got %.5f", wantBuy, result.TakerBuyVolume)
	}
	if abs(result.TakerSellVolume-wantSell) > 1e-8 {
		t.Errorf("TakerSellVolume: expected %.5f, got %.5f", wantSell, result.TakerSellVolume)
	}
}

func TestParseKlinesResponse_Empty(t *testing.T) {
	_, err := ParseKlinesResponse([]byte(klinesEmptyFixture))
	if err == nil {
		t.Error("expected error for empty klines array")
	}
}

func TestParseKlinesResponse_InvalidJSON(t *testing.T) {
	_, err := ParseKlinesResponse([]byte(klinesInvalidFixture))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseKlinesResponse_EmptyBody(t *testing.T) {
	_, err := ParseKlinesResponse([]byte{})
	if err == nil {
		t.Error("expected error for empty body")
	}
}

func TestKlinesURL(t *testing.T) {
	url := KlinesURL("BTCUSDT", "1h", 1)
	if url == "" {
		t.Error("expected non-empty URL")
	}
	// Must contain base URL and required params
	for _, want := range []string{BaseURL, "klines", "BTCUSDT", "1h"} {
		if !contains(url, want) {
			t.Errorf("URL %q missing expected substring %q", url, want)
		}
	}
}

// --- ParseKlinesVolumesResponse ---

const klinesMultiFixture = `[
	[1775088000000,"68114.37","68304.58","68065.69","68304.58","0.04166000",1775091599999,"2838.36",48,"0.01687","1149.81","0"],
	[1775091600000,"68304.58","68500.00","68200.00","68450.00","0.05234000",1775095199999,"3578.92",62,"0.02891","1978.45","0"]
]`

func TestParseKlinesVolumesResponse_Valid(t *testing.T) {
	result, err := ParseKlinesVolumesResponse([]byte(klinesMultiFixture))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Volumes) != 2 {
		t.Fatalf("expected 2 volumes, got %d", len(result.Volumes))
	}
	if abs(result.Volumes[0]-0.04166) > 1e-8 {
		t.Errorf("first volume: got %v, want 0.04166", result.Volumes[0])
	}
	if abs(result.Volumes[1]-0.05234) > 1e-8 {
		t.Errorf("second volume: got %v, want 0.05234", result.Volumes[1])
	}
}

func TestParseKlinesVolumesResponse_SingleKline(t *testing.T) {
	result, err := ParseKlinesVolumesResponse([]byte(klinesFixture))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Volumes) != 1 {
		t.Fatalf("expected 1 volume, got %d", len(result.Volumes))
	}
}

func TestParseKlinesVolumesResponse_Empty(t *testing.T) {
	_, err := ParseKlinesVolumesResponse([]byte(klinesEmptyFixture))
	if err == nil {
		t.Error("expected error for empty array")
	}
}

func TestParseKlinesVolumesResponse_InvalidJSON(t *testing.T) {
	_, err := ParseKlinesVolumesResponse([]byte(klinesInvalidFixture))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseKlinesVolumesResponse_EmptyBody(t *testing.T) {
	_, err := ParseKlinesVolumesResponse([]byte{})
	if err == nil {
		t.Error("expected error for empty body")
	}
}

// helpers

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
