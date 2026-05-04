package v1

import (
	"testing"
	"time"

	"github.com/afshinator/cryptospect-cli/internal/api/coingecko"
)

func makeCounts(g1h, t1h, g24h, t24h, g7d, t7d, g30d, t30d int) map[string]coingecko.TimeframeMetric {
	return map[string]coingecko.TimeframeMetric{
		"1h":  {GreenCount: g1h, TotalCount: t1h},
		"24h": {GreenCount: g24h, TotalCount: t24h},
		"7d":  {GreenCount: g7d, TotalCount: t7d},
		"30d": {GreenCount: g30d, TotalCount: t30d},
	}
}

func freshKline(closePrice, openPrice float64) (float64, float64, int64, bool) {
	return closePrice, openPrice, time.Now().UnixMilli(), true
}

func TestCompute_BroadScore(t *testing.T) {
	close, open_, openTime, kAvail := freshKline(68500, 68100)

	input := Input{
		TimeframeCounts: makeCounts(180, 250, 160, 250, 170, 250, 140, 250),
		CoinsCounted:    250,
		BTCChange24h:    5.0,
		BTCAvailable:    true,
		KlineClose:      close,
		KlineOpen:       open_,
		KlineOpenTimeMs: openTime,
		KlineAvailable:  kAvail,
		TopN:            250,
		Now:             time.Now(),
	}

	result, err := Compute(&input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Classification.Label != ClassificationBroad {
		t.Errorf("label: got %s, want %s", result.Classification.Label, ClassificationBroad)
	}
	if result.DivergenceDetected {
		t.Error("divergence should not be detected with broad score")
	}
	if result.ValidatorConfidence != "high" {
		t.Errorf("confidence: got %s, want high", result.ValidatorConfidence)
	}
	if result.MetricStatus != "ok" {
		t.Errorf("status: got %s, want ok", result.MetricStatus)
	}
}

func TestCompute_GhostRally(t *testing.T) {
	close, open_, openTime, kAvail := freshKline(68500, 68100)

	input := Input{
		TimeframeCounts: makeCounts(80, 250, 70, 250, 60, 250, 65, 250),
		CoinsCounted:    250,
		BTCChange24h:    5.0,
		BTCAvailable:    true,
		KlineClose:      close,
		KlineOpen:       open_,
		KlineOpenTimeMs: openTime,
		KlineAvailable:  kAvail,
		TopN:            250,
		Now:             time.Now(),
	}

	result, err := Compute(&input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.DivergenceDetected {
		t.Error("divergence should be detected: BTC > 2% and breadth < 40%")
	}
	if result.Classification.Label != ClassificationNarrow {
		t.Errorf("label: got %s, want %s", result.Classification.Label, ClassificationNarrow)
	}
}

func TestCompute_StaleCandle(t *testing.T) {
	// Candle opened 2 hours ago
	staleOpenTime := time.Now().UnixMilli() - 7200_000

	input := Input{
		TimeframeCounts: makeCounts(180, 250, 160, 250, 170, 250, 140, 250),
		CoinsCounted:    250,
		BTCChange24h:    5.0,
		BTCAvailable:    true,
		KlineClose:      68500,
		KlineOpen:       68100,
		KlineOpenTimeMs: staleOpenTime,
		KlineAvailable:  true,
		TopN:            250,
		Now:             time.Now(),
	}

	result, err := Compute(&input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.DiscrepancyDetected {
		t.Error("discrepancy should be true when candle stale")
	}
	if result.ValidatorConfidence != "low" {
		t.Errorf("confidence: got %s, want low for stale candle", result.ValidatorConfidence)
	}
	if result.MetricStatus != "ok" {
		t.Errorf("status: got %s, want ok (stale affects confidence, not status)", result.MetricStatus)
	}
}

func TestCompute_ZeroClose(t *testing.T) {
	input := Input{
		TimeframeCounts: makeCounts(180, 250, 160, 250, 170, 250, 140, 250),
		CoinsCounted:    250,
		BTCChange24h:    5.0,
		BTCAvailable:    true,
		KlineClose:      0.0,
		KlineOpen:       68100,
		KlineOpenTimeMs: time.Now().UnixMilli(),
		KlineAvailable:  true,
		TopN:            250,
		Now:             time.Now(),
	}

	result, err := Compute(&input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.DiscrepancyDetected {
		t.Error("discrepancy should be true when close is zero")
	}
	if result.ValidatorConfidence != "low" {
		t.Errorf("confidence: got %s, want low for zero close", result.ValidatorConfidence)
	}
}

func TestCompute_DiscrepancyDirectionalDisagreement(t *testing.T) {
	// CG says BTC -3% (bearish), Binance close=69000 > open=68000 (bullish)
	input := Input{
		TimeframeCounts: makeCounts(180, 250, 160, 250, 170, 250, 140, 250),
		CoinsCounted:    250,
		BTCChange24h:    -3.0,
		BTCAvailable:    true,
		KlineClose:      69000,
		KlineOpen:       68000,
		KlineOpenTimeMs: time.Now().UnixMilli(),
		KlineAvailable:  true,
		TopN:            250,
		Now:             time.Now(),
	}

	result, err := Compute(&input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.DiscrepancyDetected {
		t.Error("discrepancy should be detected: CG bearish (-3%), Binance bullish (close > open)")
	}
	if result.ValidatorConfidence != "medium" {
		t.Errorf("confidence: got %s, want medium for directional disagreement", result.ValidatorConfidence)
	}
}

func TestCompute_WeightRedistribution(t *testing.T) {
	// 1h has only 30 coins (< floor 50) — weight should redistribute
	input := Input{
		TimeframeCounts: makeCounts(15, 30, 160, 250, 170, 250, 140, 250),
		CoinsCounted:    250,
		BTCChange24h:    1.0,
		BTCAvailable:    true,
		KlineClose:      68500,
		KlineOpen:       68100,
		KlineOpenTimeMs: time.Now().UnixMilli(),
		KlineAvailable:  true,
		TopN:            250,
		Now:             time.Now(),
	}

	result, err := Compute(&input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if w, ok := result.WeightsUsed["1h"]; !ok || w != 0.0 {
		t.Errorf("1h effective weight: got %v, want 0.0 (redistributed)", w)
	}
	w24h := result.WeightsUsed["24h"]
	if w24h <= Weight24h {
		t.Errorf("24h effective weight should be > nominal %.2f after redistribution, got %.4f", Weight24h, w24h)
	}
	if result.MetricStatus != "degraded" {
		t.Errorf("status: got %s, want degraded (per-timeframe floor triggered)", result.MetricStatus)
	}
}

func TestCompute_BTCUnavailable(t *testing.T) {
	close, open_, openTime, _ := freshKline(68500, 68100)

	input := Input{
		TimeframeCounts: makeCounts(180, 250, 160, 250, 170, 250, 140, 250),
		CoinsCounted:    250,
		BTCChange24h:    0.0,
		BTCAvailable:    false,
		KlineClose:      close,
		KlineOpen:       open_,
		KlineOpenTimeMs: openTime,
		KlineAvailable:  true,
		TopN:            250,
		Now:             time.Now(),
	}

	result, err := Compute(&input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.DivergenceDetected {
		t.Error("divergence should be false when BTC unavailable")
	}
	if result.ValidatorConfidence != "medium" {
		t.Errorf("confidence: got %s, want medium for BTC unavailable", result.ValidatorConfidence)
	}
}

func TestCompute_KlineUnavailable(t *testing.T) {
	input := Input{
		TimeframeCounts: makeCounts(180, 250, 160, 250, 170, 250, 140, 250),
		CoinsCounted:    250,
		BTCChange24h:    3.0,
		BTCAvailable:    true,
		KlineClose:      0,
		KlineOpen:       0,
		KlineOpenTimeMs: 0,
		KlineAvailable:  false,
		TopN:            250,
		Now:             time.Now(),
	}

	result, err := Compute(&input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.DiscrepancyDetected {
		t.Error("discrepancy should be true when kline unavailable")
	}
	if result.ValidatorConfidence != "low" {
		t.Errorf("confidence: got %s, want low for unavailable kline", result.ValidatorConfidence)
	}
	if result.MetricStatus != "ok" {
		t.Errorf("status: got %s, want ok", result.MetricStatus)
	}
}

func TestCompute_GlobalFloor(t *testing.T) {
	// Only 30 total valid coins — below floor 50
	input := Input{
		TimeframeCounts: makeCounts(20, 30, 20, 30, 20, 30, 20, 30),
		CoinsCounted:    30,
		BTCChange24h:    1.0,
		BTCAvailable:    true,
		KlineClose:      68500,
		KlineOpen:       68100,
		KlineOpenTimeMs: time.Now().UnixMilli(),
		KlineAvailable:  true,
		TopN:            250,
		Now:             time.Now(),
	}

	result, err := Compute(&input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.MetricStatus != "degraded" {
		t.Errorf("status: got %s, want degraded (global floor)", result.MetricStatus)
	}
}

func TestCompute_AllTimeframesAbsent(t *testing.T) {
	input := Input{
		TimeframeCounts: makeCounts(0, 0, 0, 0, 0, 0, 0, 0),
		CoinsCounted:    0,
		BTCChange24h:    0.0,
		BTCAvailable:    false,
		KlineClose:      0,
		KlineOpen:       0,
		KlineOpenTimeMs: 0,
		KlineAvailable:  false,
		TopN:            250,
		Now:             time.Now(),
	}

	result, err := Compute(&input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.MetricStatus != "unavailable" {
		t.Errorf("status: got %s, want unavailable (all timeframes absent)", result.MetricStatus)
	}
}

func TestCompute_ExactBoundaries(t *testing.T) {
	close, open_, openTime, kAvail := freshKline(68500, 68100)

	tests := []struct {
		name                                 string
		green1h, total1h, green24h, total24h int
		green7d, total7d, green30d, total30d int
		btcChange                            float64
		wantLabel                            string
		wantDivergence                       bool
	}{
		{
			name:    "broad — all 1.0",
			green1h: 250, total1h: 250, green24h: 250, total24h: 250,
			green7d: 250, total7d: 250, green30d: 250, total30d: 250,
			btcChange: 1.0, wantLabel: ClassificationBroad,
		},
		{
			name:    "exactly narrow boundary (0.40) — mixed",
			green1h: 100, total1h: 250, green24h: 100, total24h: 250,
			green7d: 100, total7d: 250, green30d: 100, total30d: 250,
			btcChange: 1.0, wantLabel: ClassificationMixed,
		},
		{
			name:    "below narrow (0.39)",
			green1h: 97, total1h: 250, green24h: 97, total24h: 250,
			green7d: 97, total7d: 250, green30d: 97, total30d: 250,
			btcChange: 1.0, wantLabel: ClassificationNarrow,
		},
		{
			name:    "divergence boundary — BTC exactly 2.0 (not >)",
			green1h: 80, total1h: 250, green24h: 80, total24h: 250,
			green7d: 80, total7d: 250, green30d: 80, total30d: 250,
			btcChange: 2.0, wantLabel: ClassificationNarrow, wantDivergence: false,
		},
		{
			name:    "divergence fires — BTC 2.1",
			green1h: 80, total1h: 250, green24h: 80, total24h: 250,
			green7d: 80, total7d: 250, green30d: 80, total30d: 250,
			btcChange: 2.1, wantLabel: ClassificationNarrow, wantDivergence: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := Input{
				TimeframeCounts: makeCounts(tt.green1h, tt.total1h, tt.green24h, tt.total24h, tt.green7d, tt.total7d, tt.green30d, tt.total30d),
				CoinsCounted:    250,
				BTCChange24h:    tt.btcChange,
				BTCAvailable:    true,
				KlineClose:      close,
				KlineOpen:       open_,
				KlineOpenTimeMs: openTime,
				KlineAvailable:  kAvail,
				TopN:            250,
				Now:             time.Now(),
			}

			result, err := Compute(&input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Classification.Label != tt.wantLabel {
				t.Errorf("label: got %s, want %s", result.Classification.Label, tt.wantLabel)
			}
			if result.DivergenceDetected != tt.wantDivergence {
				t.Errorf("divergence: got %v, want %v", result.DivergenceDetected, tt.wantDivergence)
			}
		})
	}
}
