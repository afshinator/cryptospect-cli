package v1

import (
	"testing"
)

// ┌─────────────────────────────────────────────┐
// │            ComputeExchangeNetFlow            │
// └─────────────────────────────────────────────┘

func TestComputeExchangeNetFlow_BuyersAggressive(t *testing.T) {
	result := ComputeExchangeNetFlow(70, 30, 100)
	want := 0.40
	if result != want {
		t.Errorf("got %v, want %v", result, want)
	}
}

func TestComputeExchangeNetFlow_SellersAggressive(t *testing.T) {
	result := ComputeExchangeNetFlow(30, 70, 100)
	want := -0.40
	if result != want {
		t.Errorf("got %v, want %v", result, want)
	}
}

func TestComputeExchangeNetFlow_Neutral(t *testing.T) {
	result := ComputeExchangeNetFlow(50, 50, 100)
	want := 0.0
	if result != want {
		t.Errorf("got %v, want %v", result, want)
	}
}

func TestComputeExchangeNetFlow_ZeroVolume(t *testing.T) {
	result := ComputeExchangeNetFlow(0, 0, 0)
	want := 0.0
	if result != want {
		t.Errorf("got %v, want %v", result, want)
	}
}

// ┌─────────────────────────────────────────────┐
// │               ComputeFlowHook                │
// └─────────────────────────────────────────────┘

func TestComputeFlowHook_BuyersAggressive(t *testing.T) {
	if got := ComputeFlowHook(0.15); got != HookAggressiveBuy {
		t.Errorf("got %q, want %q", got, HookAggressiveBuy)
	}
}

func TestComputeFlowHook_SellersAggressive(t *testing.T) {
	if got := ComputeFlowHook(-0.15); got != HookAggressiveSell {
		t.Errorf("got %q, want %q", got, HookAggressiveSell)
	}
}

func TestComputeFlowHook_Neutral(t *testing.T) {
	if got := ComputeFlowHook(0.05); got != HookNeutral {
		t.Errorf("got %q, want %q", got, HookNeutral)
	}
}

func TestComputeFlowHook_AtPositiveBoundary(t *testing.T) {
	if got := ComputeFlowHook(0.10); got != HookAggressiveBuy {
		t.Errorf("got %q at boundary, want %q", got, HookAggressiveBuy)
	}
}

func TestComputeFlowHook_AtNegativeBoundary(t *testing.T) {
	if got := ComputeFlowHook(-0.10); got != HookAggressiveSell {
		t.Errorf("got %q at boundary, want %q", got, HookAggressiveSell)
	}
}

// ┌─────────────────────────────────────────────┐
// │            ComputeFundingRateHook            │
// └─────────────────────────────────────────────┘

func TestComputeFundingRateHook_ZeroIsNeutral(t *testing.T) {
	if got := ComputeFundingRateHook(0); got != HookNeutralFR {
		t.Errorf("got %q, want %q", got, HookNeutralFR)
	}
}

func TestComputeFundingRateHook_Negative(t *testing.T) {
	// -0.29% — realistic negative funding
	if got := ComputeFundingRateHook(-0.002924); got != HookNegative {
		t.Errorf("got %q for -0.29%%, want %q", got, HookNegative)
	}
}

func TestComputeFundingRateHook_NegativeAtBoundary(t *testing.T) {
	// Exactly at negative threshold
	if got := ComputeFundingRateHook(-0.0003); got != HookNegative {
		t.Errorf("got %q at boundary -0.0003, want %q", got, HookNegative)
	}
}

func TestComputeFundingRateHook_NearZeroPositive(t *testing.T) {
	// +0.01% — below positive threshold, should be neutral
	if got := ComputeFundingRateHook(0.0001); got != HookNeutralFR {
		t.Errorf("got %q for +0.01%%, want %q", got, HookNeutralFR)
	}
}

func TestComputeFundingRateHook_PositiveTypical(t *testing.T) {
	// +0.23% — typical positive funding
	if got := ComputeFundingRateHook(0.0023); got != HookPositive {
		t.Errorf("got %q for +0.23%%, want %q", got, HookPositive)
	}
}

func TestComputeFundingRateHook_PositiveAtBoundary(t *testing.T) {
	// Exactly at positive threshold
	if got := ComputeFundingRateHook(0.0003); got != HookPositive {
		t.Errorf("got %q at boundary +0.0003, want %q", got, HookPositive)
	}
}

func TestComputeFundingRateHook_PositiveNearOverheated(t *testing.T) {
	// +0.10% — below overheated threshold, should be positive
	if got := ComputeFundingRateHook(0.001); got != HookPositive {
		t.Errorf("got %q at +0.10%%, want %q", got, HookPositive)
	}
}

func TestComputeFundingRateHook_Overheated(t *testing.T) {
	// +0.31% — just over overheated threshold
	if got := ComputeFundingRateHook(0.0031); got != HookOverheated {
		t.Errorf("got %q for +0.31%%, want %q", got, HookOverheated)
	}
}

func TestComputeFundingRateHook_OverheatedAtBoundary(t *testing.T) {
	// Exactly at overheated threshold (not over — still positive)
	if got := ComputeFundingRateHook(0.003); got != HookPositive {
		t.Errorf("got %q at boundary +0.003, want %q", got, HookPositive)
	}
}

// ┌─────────────────────────────────────────────┐
// │              ComputeOIChange24h              │
// └─────────────────────────────────────────────┘

func TestComputeOIChange24h_Normal(t *testing.T) {
	got := ComputeOIChange24h(11000, 10000)
	want := 0.10
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestComputeOIChange24h_Decline(t *testing.T) {
	got := ComputeOIChange24h(9000, 10000)
	want := -0.10
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestComputeOIChange24h_ZeroPrev(t *testing.T) {
	got := ComputeOIChange24h(10000, 0)
	want := 0.0
	if got != want {
		t.Errorf("got %v for zero prev, want %v", got, want)
	}
}

func TestComputeOIChange24h_NoChange(t *testing.T) {
	got := ComputeOIChange24h(10000, 10000)
	want := 0.0
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// ┌─────────────────────────────────────────────┐
// │              ComputeOIChangeHook             │
// └─────────────────────────────────────────────┘

func TestComputeOIChangeHook_Building(t *testing.T) {
	if got := ComputeOIChangeHook(0.06); got != HookOIBuilding {
		t.Errorf("got %q, want %q", got, HookOIBuilding)
	}
}

func TestComputeOIChangeHook_Stable(t *testing.T) {
	if got := ComputeOIChangeHook(0.02); got != HookOIStable {
		t.Errorf("got %q, want %q", got, HookOIStable)
	}
}

func TestComputeOIChangeHook_Unwinding(t *testing.T) {
	if got := ComputeOIChangeHook(-0.08); got != HookOIUnwinding {
		t.Errorf("got %q, want %q", got, HookOIUnwinding)
	}
}

func TestComputeOIChangeHook_AtBuildingBoundary(t *testing.T) {
	if got := ComputeOIChangeHook(0.05); got != HookOIBuilding {
		t.Errorf("got %q at boundary +0.05, want %q", got, HookOIBuilding)
	}
}

func TestComputeOIChangeHook_AtUnwindingBoundary(t *testing.T) {
	if got := ComputeOIChangeHook(-0.05); got != HookOIUnwinding {
		t.Errorf("got %q at boundary -0.05, want %q", got, HookOIUnwinding)
	}
}

func TestComputeOIChangeHook_Zero(t *testing.T) {
	if got := ComputeOIChangeHook(0); got != HookOIStable {
		t.Errorf("got %q for zero, want %q", got, HookOIStable)
	}
}

// ┌─────────────────────────────────────────────┐
// │               ComputeSummary                 │
// └─────────────────────────────────────────────┘

func TestComputeSummary_EarlyBullPhase(t *testing.T) {
	got := ComputeSummary(HookNegative, HookOIBuilding, HookNeutral)
	if got == "" {
		t.Fatal("summary must not be empty")
	}
	if contains(got, "early bull") == false {
		t.Errorf("summary %q should mention early bull phase", got)
	}
}

func TestComputeSummary_BuildingTensionWithBuyers(t *testing.T) {
	got := ComputeSummary(HookNeutralFR, HookOIBuilding, HookAggressiveBuy)
	if got == "" {
		t.Fatal("summary must not be empty")
	}
	if contains(got, "tension") == false && contains(got, "breakout") == false {
		t.Errorf("summary %q should mention tension or breakout", got)
	}
}

func TestComputeSummary_SupplyShock(t *testing.T) {
	got := ComputeSummary(HookPositive, HookOIStable, HookAggressiveSell)
	if got == "" {
		t.Fatal("summary must not be empty")
	}
	if contains(got, "supply shock") == false && contains(got, "top warning") == false {
		t.Errorf("summary %q should mention supply shock or top warning", got)
	}
}

func TestComputeSummary_Neutral(t *testing.T) {
	got := ComputeSummary(HookNeutralFR, HookOIStable, HookNeutral)
	if got == "" {
		t.Fatal("summary must not be empty")
	}
	if contains(got, "no directional conviction") == false {
		t.Errorf("summary %q should mention no directional conviction", got)
	}
}

func TestComputeSummary_LowConfidenceCandle(t *testing.T) {
	got := ComputeSummary(HookNeutralFR, HookOIStable, HookLowConfidence)
	if got == "" {
		t.Fatal("summary must not be empty")
	}
	if contains(got, "thin candle") == false && contains(got, "insufficient") == false {
		t.Errorf("summary %q should mention thin candle", got)
	}
}

func TestComputeSummary_OverheatedLongs(t *testing.T) {
	got := ComputeSummary(HookOverheated, HookOIBuilding, HookNeutral)
	if got == "" {
		t.Fatal("summary must not be empty")
	}
	if contains(got, "liquidation") == false && contains(got, "overheated") == false {
		t.Errorf("summary %q should mention liquidation risk or overheated", got)
	}
}

func TestComputeSummary_Deleveraging(t *testing.T) {
	got := ComputeSummary(HookNeutralFR, HookOIUnwinding, HookNeutral)
	if got == "" {
		t.Fatal("summary must not be empty")
	}
	if contains(got, "deleveraging") == false && contains(got, "unwinding") == false {
		t.Errorf("summary %q should mention deleveraging or unwinding", got)
	}
}

func TestComputeSummary_MixedSignals(t *testing.T) {
	got := ComputeSummary(HookPositive, HookOIBuilding, HookAggressiveSell)
	if got == "" {
		t.Fatal("summary must not be empty")
	}
	// This combination doesn't match any named verdict — should fall back to mixed
	if contains(got, "Mixed") == false {
		t.Errorf("summary %q should mention Mixed signals", got)
	}
}

// ┌─────────────────────────────────────────────┐
// │                Compute (full)                │
// └─────────────────────────────────────────────┘

func TestCompute_FullSignals(t *testing.T) {
	input := Input{
		TakerBuyVolume:    70,
		TakerSellVolume:   30,
		TotalVolume:       100,
		NumTrades:         50,
		TotalOpenInterest: 18500000000,
		PrevOI:            ptr(17400000000),
		FundingRate:       0.0003,
	}

	data, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Signals.CVD.Hook != HookAggressiveBuy {
		t.Errorf("CVD hook = %q, want %q", data.Signals.CVD.Hook, HookAggressiveBuy)
	}
	if data.Signals.FundingRate.Hook != HookPositive {
		t.Errorf("Funding hook = %q, want %q", data.Signals.FundingRate.Hook, HookPositive)
	}
	if data.Signals.OpenInterest.Hook != HookOIBuilding {
		t.Errorf("OI hook = %q, want %q", data.Signals.OpenInterest.Hook, HookOIBuilding)
	}
	if data.Signals.OpenInterest.ChangePct24h == nil {
		t.Error("OI ChangePct24h should be present when PrevOI is set")
	}
	if data.Summary == "" {
		t.Error("summary must not be empty")
	}
}

func TestCompute_DegradedMode(t *testing.T) {
	// Only CVD data, zero OI and funding (transient failure)
	input := Input{
		TakerBuyVolume:  30,
		TakerSellVolume: 70,
		TotalVolume:     100,
		NumTrades:       50,
	}

	data, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Signals.CVD.Hook != HookAggressiveSell {
		t.Errorf("CVD hook = %q, want %q", data.Signals.CVD.Hook, HookAggressiveSell)
	}
	if data.Signals.OpenInterest.ChangePct24h != nil {
		t.Error("OI ChangePct24h should be nil when PrevOI is nil")
	}
	if data.Summary == "" {
		t.Error("summary must not be empty")
	}
}

func TestCompute_ZeroTotalVolume(t *testing.T) {
	input := Input{
		TakerBuyVolume:  0,
		TakerSellVolume: 0,
		TotalVolume:     0,
		NumTrades:       0,
	}
	if _, err := Compute(input); err == nil {
		t.Error("expected error for zero total volume")
	}
}

func TestCompute_ThinCandle_LowConfidenceHook(t *testing.T) {
	input := Input{
		TakerBuyVolume:  0.001,
		TakerSellVolume: 0,
		TotalVolume:     0.001,
		NumTrades:       2,
	}
	data, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Signals.CVD.Hook != HookLowConfidence {
		t.Errorf("CVD hook = %q, want %q", data.Signals.CVD.Hook, HookLowConfidence)
	}
}

func TestCompute_ThinCandle_RawValueStillReturned(t *testing.T) {
	input := Input{
		TakerBuyVolume:  0.001,
		TakerSellVolume: 0,
		TotalVolume:     0.001,
		NumTrades:       2,
	}
	data, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Signals.CVD.Ratio.Value() == 0 {
		t.Error("expected raw CVD ratio to be returned even in low_confidence mode")
	}
}

func TestCompute_AtMinTradesThreshold(t *testing.T) {
	// Exactly at min Trades — should produce normal directional hook
	input := Input{
		TakerBuyVolume:  70,
		TakerSellVolume: 30,
		TotalVolume:     100,
		NumTrades:       10,
	}
	data, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Signals.CVD.Hook == HookLowConfidence {
		t.Error("expected directional hook at minimum trade threshold, got low_confidence")
	}
}

func TestCompute_BelowMinTradesThreshold(t *testing.T) {
	input := Input{
		TakerBuyVolume:  70,
		TakerSellVolume: 30,
		TotalVolume:     100,
		NumTrades:       9,
	}
	data, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Signals.CVD.Hook != HookLowConfidence {
		t.Errorf("expected low_confidence below min threshold, got %q", data.Signals.CVD.Hook)
	}
}

func TestCompute_NoPrevOI_ChangeOmitted(t *testing.T) {
	// PrevOI is nil — OI change should be omitted, hook defaults to stable
	input := Input{
		TakerBuyVolume:    50,
		TakerSellVolume:   50,
		TotalVolume:       100,
		NumTrades:         25,
		TotalOpenInterest: 18500000000,
		FundingRate:       0.0003,
	}
	data, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Signals.OpenInterest.ChangePct24h != nil {
		t.Error("OI ChangePct24h should be nil when PrevOI is nil")
	}
	if data.Signals.OpenInterest.Hook != HookOIStable {
		t.Errorf("OI hook = %q, want %q (default stable)", data.Signals.OpenInterest.Hook, HookOIStable)
	}
}

func TestCompute_ZeroOI_StillComputes(t *testing.T) {
	// Zero OI (no BTC perpetuals in response) — should not block metric
	input := Input{
		TakerBuyVolume:  50,
		TakerSellVolume: 50,
		TotalVolume:     100,
		NumTrades:       25,
		FundingRate:     0.0001,
	}
	data, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Signals.OpenInterest.Hook != HookOIStable {
		t.Errorf("OI hook = %q, want stable for zero OI", data.Signals.OpenInterest.Hook)
	}
	if data.Signals.CVD.Hook != HookNeutral {
		t.Errorf("CVD hook = %q, want neutral", data.Signals.CVD.Hook)
	}
}

// ┌─────────────────────────────────────────────┐
// │                 Helpers                      │
// └─────────────────────────────────────────────┘

func ptr(f float64) *float64 { return &f }

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
