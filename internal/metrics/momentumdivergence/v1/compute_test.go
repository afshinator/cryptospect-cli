package v1

import (
	"strings"
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/api/coingecko"
)

func c(rank int, id string, change float64) coingecko.CoinMarketsRankedCoin {
	return coingecko.CoinMarketsRankedCoin{ID: id, MarketCapRank: rank, Change24h: &change}
}

func cn(rank int, id string) coingecko.CoinMarketsRankedCoin {
	return coingecko.CoinMarketsRankedCoin{ID: id, MarketCapRank: rank, Change24h: nil}
}

func TestCompute_RiskOn(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "bitcoin", 2.0),
		c(2, "ethereum", 3.0),
		c(3, "tether", 0.01),
		c(4, "solana", 1.5),
		c(5, "bnb", 2.5),
		c(11, "chainlink", 8.0),
		c(12, "polygon", 7.0),
		c(13, "avalanche", 9.0),
		c(51, "gmx", 12.0),
		c(52, "dydx", 10.0),
		c(53, "inj", 14.0),
	}
	input := Input{
		Coins:        coins,
		LargeCeiling: 10,
		MidCeiling:   50,
		SmallCeiling: 200,
	}
	data, _, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Classification.Label != LabelRiskOn {
		t.Errorf("label: got %q, want %q", data.Classification.Label, LabelRiskOn)
	}
	if data.Spreads.MidVsLarge == nil {
		t.Fatal("midVsLarge should not be nil")
	}
	// mid avg: 8.0, large avg: (2+3+0.01+1.5+2.5)/5 = 1.802 → spread ~6.198
	if *data.Spreads.MidVsLarge <= RiskOnSpread {
		t.Errorf("midVsLarge: got %v, want > %v", *data.Spreads.MidVsLarge, RiskOnSpread)
	}
}

func TestCompute_TopHeavy(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "bitcoin", 5.0),
		c(2, "ethereum", 4.0),
		c(3, "tether", 0.01),
		c(4, "solana", 6.0),
		c(5, "bnb", 3.0),
		c(11, "chainlink", -2.0),
		c(12, "polygon", -1.0),
		c(13, "avalanche", -3.0),
		c(51, "gmx", -5.0),
		c(52, "dydx", -4.0),
		c(53, "inj", -6.0),
	}
	input := Input{
		Coins:        coins,
		LargeCeiling: 10,
		MidCeiling:   50,
		SmallCeiling: 200,
	}
	data, _, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Classification.Label != LabelTopHeavy {
		t.Errorf("label: got %q, want %q", data.Classification.Label, LabelTopHeavy)
	}
	if data.Spreads.MidVsLarge == nil {
		t.Fatal("midVsLarge should not be nil")
	}
	if *data.Spreads.MidVsLarge >= TopHeavySpread {
		t.Errorf("midVsLarge: got %v, want < %v", *data.Spreads.MidVsLarge, TopHeavySpread)
	}
}

func TestCompute_FlightToSafety(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "bitcoin", -2.0),
		c(2, "ethereum", -1.5),
		c(3, "tether", 0.01),
		c(4, "solana", -3.0),
		c(5, "bnb", -1.0),
		c(11, "chainlink", -10.0),
		c(12, "polygon", -9.0),
		c(13, "avalanche", -12.0),
		c(51, "gmx", -20.0),
		c(52, "dydx", -18.0),
		c(53, "inj", -22.0),
	}
	input := Input{
		Coins:        coins,
		LargeCeiling: 10,
		MidCeiling:   50,
		SmallCeiling: 200,
	}
	data, _, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Classification.Label != LabelFlightToSafety {
		t.Errorf("label: got %q, want %q", data.Classification.Label, LabelFlightToSafety)
	}
}

func TestCompute_Neutral_ConcentrationDeadBand(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "bitcoin", 0.2),
		c(2, "ethereum", -0.1),
		c(3, "tether", 0.01),
		c(4, "solana", 0.3),
		c(5, "bnb", -0.2),
		c(11, "chainlink", -8.0),
		c(12, "polygon", -7.0),
		c(13, "avalanche", -9.0),
	}
	input := Input{
		Coins:        coins,
		LargeCeiling: 10,
		MidCeiling:   50,
		SmallCeiling: 200,
	}
	data, _, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Classification.Label != LabelNeutral {
		t.Errorf("label: got %q, want %q (dead band: large_avg ~0.08 in (-0.5,+0.5])", data.Classification.Label, LabelNeutral)
	}
}

func TestCompute_TailExtension(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "bitcoin", 2.0),
		c(2, "ethereum", 3.0),
		c(3, "tether", 0.01),
		c(11, "chainlink", 8.0),
		c(12, "polygon", 7.0),
		c(13, "avalanche", 9.0),
		c(51, "gmx", 15.0),
		c(52, "dydx", 14.0),
		c(53, "inj", 16.0),
	}
	input := Input{
		Coins:        coins,
		LargeCeiling: 10,
		MidCeiling:   50,
		SmallCeiling: 200,
	}
	data, _, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Classification.Label != LabelRiskOn {
		t.Errorf("label: got %q, want %q", data.Classification.Label, LabelRiskOn)
	}
	if !data.TailExtension {
		t.Error("tail_extension should be true when small_vs_large > +5pp")
	}
}

func TestCompute_TailExtension_WithoutRiskOn(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "bitcoin", 2.0),
		c(2, "ethereum", 3.0),
		c(3, "tether", 0.01),
		c(11, "chainlink", 4.0),
		c(12, "polygon", 3.5),
		c(13, "avalanche", 4.5),
		c(51, "gmx", 15.0),
		c(52, "dydx", 14.0),
		c(53, "inj", 16.0),
	}
	input := Input{
		Coins:        coins,
		LargeCeiling: 10,
		MidCeiling:   50,
		SmallCeiling: 200,
	}
	data, _, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Classification.Label != LabelNeutral {
		t.Errorf("label: got %q, want neutral (midVsLarge ~2pp, below 5pp threshold)", data.Classification.Label)
	}
	if !data.TailExtension {
		t.Error("tail_extension should be true even when primary label is neutral")
	}
	// Barbell check: neutral + tail_extension + midVsLarge > 1.0
	if data.Spreads.MidVsLarge != nil && *data.Spreads.MidVsLarge > BarbellMidVsLargeMin {
		if data.Classification.Description != "Barbell — Speculative Tail Extension" {
			t.Errorf("description: got %q, want Barbell", data.Classification.Description)
		}
	}
}

func TestCompute_TailExtension_WithTopHeavy(t *testing.T) {
	// Unusual config: top_heavy label (large rallying, mid lagging) but small outperforming.
	// large avg 5.0, mid avg 1.0 → spread -4.0 < -3.0, large_avg 5.0 > 0.5 → top_heavy
	// small avg 12.0 → small_vs_large 7.0 > 5.0 → tail_extension true
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "bitcoin", 5.0),
		c(2, "ethereum", 5.0),
		c(3, "tether", 5.0),
		c(11, "chainlink", 1.0),
		c(12, "polygon", 1.0),
		c(13, "avalanche", 1.0),
		c(51, "gmx", 12.0),
		c(52, "dydx", 12.0),
		c(53, "inj", 12.0),
	}
	input := Input{
		Coins:        coins,
		LargeCeiling: 10,
		MidCeiling:   50,
		SmallCeiling: 200,
	}
	data, _, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Classification.Label != LabelTopHeavy {
		t.Errorf("label: got %q, want top_heavy", data.Classification.Label)
	}
	if !data.TailExtension {
		t.Error("tail_extension should be true (small_vs_large 7pp > 5pp) even with top_heavy label")
	}
}

func TestCompute_MissingTier_NilSpread(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "bitcoin", 2.0),
		c(2, "ethereum", 3.0),
		c(3, "tether", 0.01),
		c(51, "gmx", 15.0),
		c(52, "dydx", 14.0),
		c(53, "inj", 12.0),
	}
	input := Input{
		Coins:        coins,
		LargeCeiling: 10,
		MidCeiling:   50,
		SmallCeiling: 200,
	}
	data, _, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Spreads.MidVsLarge != nil {
		t.Error("midVsLarge should be nil when tier_mid is absent")
	}
	if data.Spreads.SmallVsMid != nil {
		t.Error("smallVsMid should be nil when tier_mid is absent")
	}
	if data.Classification.Label != LabelNeutral {
		t.Errorf("label: got %q, want neutral (nil midVsLarge)", data.Classification.Label)
	}
}

func TestCompute_OnlyLargeTierPresent(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "bitcoin", 2.0),
		c(2, "ethereum", 3.0),
		c(3, "tether", 0.01),
	}
	input := Input{
		Coins:        coins,
		LargeCeiling: 10,
		MidCeiling:   50,
		SmallCeiling: 200,
	}
	data, meta, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Confidence != "low" {
		t.Errorf("confidence: got %q, want low (mid and small absent)", meta.Confidence)
	}
	if data.Spreads.MidVsLarge != nil {
		t.Error("midVsLarge should be nil when tier_mid is absent")
	}
	if data.Spreads.SmallVsLarge != nil {
		t.Error("smallVsLarge should be nil when tier_small is absent")
	}
	if data.Spreads.SmallVsMid != nil {
		t.Error("smallVsMid should be nil when both mid and small absent")
	}
	if data.Classification.Label != LabelNeutral {
		t.Errorf("label: got %q, want neutral (nil midVsLarge)", data.Classification.Label)
	}
}

func TestCompute_AllAbsent_Error(t *testing.T) {
	input := Input{
		Coins:        []coingecko.CoinMarketsRankedCoin{},
		LargeCeiling: 10,
		MidCeiling:   50,
		SmallCeiling: 200,
	}
	_, _, err := Compute(input)
	if err == nil {
		t.Error("expected error when all tiers are absent")
	}
}

func TestCompute_Confidence_Low(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "bitcoin", 2.0),
		c(2, "ethereum", 3.0),
		c(3, "tether", 0.01),
		c(11, "chainlink", 8.0),
		c(12, "polygon", 7.0),
	}
	input := Input{
		Coins:        coins,
		LargeCeiling: 10,
		MidCeiling:   50,
		SmallCeiling: 200,
	}
	_, meta, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Confidence != "low" {
		t.Errorf("confidence: got %q, want low (tier_mid has only 2 coins < floor 3)", meta.Confidence)
	}
}

func TestCompute_Confidence_High(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "bitcoin", 2.0),
		c(2, "ethereum", 3.0),
		c(3, "tether", 0.01),
		c(11, "chainlink", 4.0),
		c(12, "polygon", 3.0),
		c(13, "avalanche", 5.0),
		c(51, "gmx", 6.0),
		c(52, "dydx", 7.0),
		c(53, "inj", 8.0),
	}
	input := Input{
		Coins:        coins,
		LargeCeiling: 10,
		MidCeiling:   50,
		SmallCeiling: 200,
	}
	_, meta, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Confidence != "high" {
		t.Errorf("confidence: got %q, want high", meta.Confidence)
	}
}

func TestCompute_NullSpreadsNotZero(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "bitcoin", 2.0),
		c(2, "ethereum", 3.0),
		c(3, "tether", 0.01),
		c(51, "gmx", 15.0),
		c(52, "dydx", 14.0),
		c(53, "inj", 12.0),
	}
	input := Input{
		Coins:        coins,
		LargeCeiling: 10,
		MidCeiling:   50,
		SmallCeiling: 200,
	}
	data, _, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Spreads.MidVsLarge != nil {
		t.Error("midVsLarge should be nil when tier_mid is absent, not 0.0")
	}
	if data.Spreads.SmallVsLarge == nil {
		t.Error("smallVsLarge should be computed (both tiers present)")
	}
}

// TestCompute_MinPositivityGuard verifies the guard passes when mid_avg is above 1.0:
// spread 5.5pp and mid_avg 2.5 → risk_on fires.
func TestCompute_MinPositivityGuard(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "bitcoin", -5.0),
		c(2, "ethereum", -4.0),
		c(3, "tether", 0.01),
		c(11, "chainlink", 2.0),
		c(12, "polygon", 3.0),
		c(13, "avalanche", 2.5),
	}
	// large_avg = (-5 + -4 + 0.01) / 3 = -2.997
	// mid_avg   = (2 + 3 + 2.5) / 3     = 2.5
	// spread    = 2.5 - (-2.997)         = 5.497 > 5.0, mid_avg 2.5 > 1.0 → risk_on
	input := Input{
		Coins:        coins,
		LargeCeiling: 10,
		MidCeiling:   50,
		SmallCeiling: 200,
	}
	data, _, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Classification.Label != LabelRiskOn {
		t.Errorf("label: got %q, want risk_on (spread > 5pp, mid_avg 2.5 > 1.0)", data.Classification.Label)
	}
}

// TestCompute_MinPositivityGuard_Blocks verifies the guard blocks risk_on when
// spread > 5pp but mid_avg is below 1.0 (market-wide crash scenario).
func TestCompute_MinPositivityGuard_Blocks(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "bitcoin", -13.0),
		c(2, "ethereum", -14.0),
		c(3, "tether", -15.0),
		c(11, "chainlink", -6.0),
		c(12, "polygon", -7.0),
		c(13, "avalanche", -8.0),
	}
	// large_avg = -14.0, mid_avg = -7.0
	// spread    = -7.0 - (-14.0) = 7.0 > 5.0 → would fire risk_on
	// mid_avg -7.0 < 1.0 → guard blocks → neutral
	input := Input{
		Coins:        coins,
		LargeCeiling: 10,
		MidCeiling:   50,
		SmallCeiling: 200,
	}
	data, _, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Classification.Label != LabelNeutral {
		t.Errorf("label: got %q, want neutral (guard blocks: mid_avg -7.0 < 1.0 despite 7pp spread)", data.Classification.Label)
	}
}

// TestCompute_MinPositivityGuard_BoundaryBelow verifies the guard blocks when
// mid_avg is just below 1.0 (0.9), even with spread > 5pp.
func TestCompute_MinPositivityGuard_BoundaryBelow(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "bitcoin", -5.0),
		c(2, "ethereum", -5.0),
		c(3, "tether", -5.3),
		c(11, "chainlink", 0.9),
		c(12, "polygon", 0.9),
		c(13, "avalanche", 0.9),
	}
	// large_avg = (-5 + -5 + -5.3) / 3 = -5.1
	// mid_avg   = 0.9
	// spread    = 0.9 - (-5.1)           = 6.0 > 5.0 → would fire risk_on
	// mid_avg 0.9 < 1.0 → guard blocks → neutral
	input := Input{
		Coins:        coins,
		LargeCeiling: 10,
		MidCeiling:   50,
		SmallCeiling: 200,
	}
	data, _, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Classification.Label != LabelNeutral {
		t.Errorf("label: got %q, want neutral (guard blocks: mid_avg 0.9 < 1.0 despite 6pp spread)", data.Classification.Label)
	}
}

// TestCompute_MinPositivityGuard_BoundaryAbove verifies the guard passes when
// mid_avg is just above 1.0 (1.1) with spread > 5pp → risk_on fires.
func TestCompute_MinPositivityGuard_BoundaryAbove(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "bitcoin", -5.0),
		c(2, "ethereum", -5.0),
		c(3, "tether", -5.3),
		c(11, "chainlink", 1.1),
		c(12, "polygon", 1.1),
		c(13, "avalanche", 1.1),
	}
	// large_avg = -5.1, mid_avg = 1.1
	// spread = 1.1 - (-5.1) = 6.2 > 5.0, mid_avg 1.1 > 1.0 → risk_on
	input := Input{
		Coins:        coins,
		LargeCeiling: 10,
		MidCeiling:   50,
		SmallCeiling: 200,
	}
	data, _, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Classification.Label != LabelRiskOn {
		t.Errorf("label: got %q, want risk_on (mid_avg 1.1 > 1.0, spread 6.2pp)", data.Classification.Label)
	}
}

// TestCompute_SpreadAboveFive_RiskOn confirms spread > 5.0pp with mid_avg > 1.0 → risk_on.
func TestCompute_SpreadAboveFive_RiskOn(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "bitcoin", 1.0),
		c(2, "ethereum", 1.0),
		c(3, "tether", 0.0),
		c(11, "chainlink", 7.0),
		c(12, "polygon", 7.0),
		c(13, "avalanche", 7.0),
	}
	// large_avg = (1+1+0)/3 = 0.667, mid_avg = 7.0
	// spread = 7.0 - 0.667 = 6.333 > 5.0 → risk_on
	input := Input{
		Coins:        coins,
		LargeCeiling: 10,
		MidCeiling:   50,
		SmallCeiling: 200,
	}
	data, _, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Classification.Label != LabelRiskOn {
		t.Errorf("label: got %q, want risk_on (6.33pp spread)", data.Classification.Label)
	}
}

// TestCompute_SpreadExactlyFive_NotRiskOn confirms the threshold is strict > 5.0:
// a spread of exactly 5.0pp does not fire risk_on.
func TestCompute_SpreadExactlyFive_NotRiskOn(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "bitcoin", 2.0),
		c(2, "ethereum", 2.0),
		c(3, "tether", 2.0),
		c(11, "chainlink", 7.0),
		c(12, "polygon", 7.0),
		c(13, "avalanche", 7.0),
	}
	// large_avg = 2.0, mid_avg = 7.0, spread = 5.0
	// 5.0 is NOT strictly > 5.0 → neutral
	input := Input{
		Coins:        coins,
		LargeCeiling: 10,
		MidCeiling:   50,
		SmallCeiling: 200,
	}
	data, _, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Classification.Label != LabelNeutral {
		t.Errorf("label: got %q, want neutral (spread exactly 5.0 is not > 5.0)", data.Classification.Label)
	}
}

// TestCompute_DeadBand_NegativeBoundary confirms large_avg == -0.5 triggers flight_to_safety
// (open interval: -0.5 is excluded from the neutral dead band).
func TestCompute_DeadBand_NegativeBoundary(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "a", -0.5),
		c(2, "b", -0.5),
		c(3, "c", -0.5),
		c(11, "d", -4.5),
		c(12, "e", -4.5),
		c(13, "f", -4.5),
	}
	// large_avg = -0.5, mid_avg = -4.5, spread = -4.0 < -3.0
	// large_avg == -ConcentrationDeadBand (-0.5) → flight_to_safety (boundary inclusive)
	input := Input{Coins: coins, LargeCeiling: 10, MidCeiling: 50, SmallCeiling: 200}
	data, _, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Classification.Label != LabelFlightToSafety {
		t.Errorf("label: got %q, want flight_to_safety (large_avg == -0.5 <= boundary)", data.Classification.Label)
	}
}

// TestCompute_DeadBand_PositiveBoundary confirms large_avg == +0.5 stays neutral
// (closed interval: +0.5 is inside the dead band, not top_heavy which requires > +0.5).
func TestCompute_DeadBand_PositiveBoundary(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "a", 0.5),
		c(2, "b", 0.5),
		c(3, "c", 0.5),
		c(11, "d", -3.5),
		c(12, "e", -3.5),
		c(13, "f", -3.5),
	}
	// large_avg = 0.5, mid_avg = -3.5, spread = -4.0 < -3.0
	// large_avg == +ConcentrationDeadBand (+0.5) → neutral (need strictly > 0.5 for top_heavy)
	input := Input{Coins: coins, LargeCeiling: 10, MidCeiling: 50, SmallCeiling: 200}
	data, _, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Classification.Label != LabelNeutral {
		t.Errorf("label: got %q, want neutral (large_avg == +0.5 in dead band, need > 0.5 for top_heavy)", data.Classification.Label)
	}
}

// TestCompute_TierBoundaryExact confirms coins at exactly rank == ceiling values
// are assigned to the correct tier (boundaries are inclusive: rank <= ceiling).
func TestCompute_TierBoundaryExact(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "a", 1.0), c(2, "b", 1.0), c(3, "c", 1.0),
		c(10, "large_boundary", 1.0), // rank == LargeCeiling → large tier
		c(11, "d", 1.0), c(12, "e", 1.0),
		c(50, "mid_boundary", 1.0), // rank == MidCeiling → mid tier
		c(51, "f", 1.0), c(52, "g", 1.0),
		c(200, "small_boundary", 1.0), // rank == SmallCeiling → small tier
	}
	input := Input{Coins: coins, LargeCeiling: 10, MidCeiling: 50, SmallCeiling: 200}
	_, meta, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.TierCounts.Large != 4 {
		t.Errorf("TierCounts.Large: got %d, want 4 (ranks 1,2,3,10)", meta.TierCounts.Large)
	}
	if meta.TierCounts.Mid != 3 {
		t.Errorf("TierCounts.Mid: got %d, want 3 (ranks 11,12,50)", meta.TierCounts.Mid)
	}
	if meta.TierCounts.Small != 3 {
		t.Errorf("TierCounts.Small: got %d, want 3 (ranks 51,52,200)", meta.TierCounts.Small)
	}
}

func TestCompute_NullChangeExcluded(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "bitcoin", 5.0),
		cn(2, "ethereum"),
		c(3, "tether", 0.01),
		c(4, "solana", 2.0),
		c(11, "chainlink", 10.0),
		c(12, "polygon", 12.0),
		cn(13, "avalanche"),
		c(14, "uni", 9.0),
	}
	input := Input{
		Coins:        coins,
		LargeCeiling: 10,
		MidCeiling:   50,
		SmallCeiling: 200,
	}
	data, _, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Spreads.MidVsLarge == nil {
		t.Fatal("midVsLarge should be present")
	}
	diff := *data.Spreads.MidVsLarge - 7.9967
	if diff < -0.01 || diff > 0.01 {
		t.Errorf("midVsLarge: got %v, want ~7.9967 (null changes excluded)", *data.Spreads.MidVsLarge)
	}
	if data.Classification.Label != LabelRiskOn {
		t.Errorf("label: got %q, want risk_on", data.Classification.Label)
	}
}

func TestCompute_TierCounts(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "bitcoin", 2.0),
		c(2, "ethereum", 3.0),
		c(3, "tether", 0.01),
		c(4, "solana", 1.0),
		c(11, "chainlink", 4.0),
		c(12, "polygon", 3.0),
		c(13, "avalanche", 5.0),
		c(14, "uni", 4.5),
		c(51, "gmx", 6.0),
		c(52, "dydx", 7.0),
	}
	input := Input{
		Coins:        coins,
		LargeCeiling: 10,
		MidCeiling:   50,
		SmallCeiling: 200,
	}
	_, meta, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.TierCounts.Large != 4 {
		t.Errorf("TierCounts.Large: got %d, want 4", meta.TierCounts.Large)
	}
	if meta.TierCounts.Mid != 4 {
		t.Errorf("TierCounts.Mid: got %d, want 4", meta.TierCounts.Mid)
	}
	if meta.TierCounts.Small != 2 {
		t.Errorf("TierCounts.Small: got %d, want 2", meta.TierCounts.Small)
	}
}

func TestCompute_SmallVsMid(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "bitcoin", 2.0),
		c(2, "ethereum", 3.0),
		c(3, "tether", 0.01),
		c(11, "chainlink", 8.0),
		c(12, "polygon", 7.0),
		c(13, "avalanche", 9.0),
		c(51, "gmx", 10.0),
		c(52, "dydx", 9.0),
		c(53, "inj", 11.0),
	}
	input := Input{
		Coins:        coins,
		LargeCeiling: 10,
		MidCeiling:   50,
		SmallCeiling: 200,
	}
	data, _, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Spreads.SmallVsMid == nil {
		t.Fatal("smallVsMid should be present")
	}
	if *data.Spreads.SmallVsMid != 2.0 {
		t.Errorf("smallVsMid: got %v, want 2.0 (small avg 10 - mid avg 8)", *data.Spreads.SmallVsMid)
	}
}

func TestCompute_SummaryString(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "bitcoin", 2.0),
		c(2, "ethereum", 3.0),
		c(3, "tether", 0.01),
		c(4, "solana", 1.5),
		c(5, "bnb", 2.5),
		c(11, "chainlink", 8.0),
		c(12, "polygon", 7.0),
		c(13, "avalanche", 9.0),
		c(51, "gmx", 12.0),
		c(52, "dydx", 10.0),
		c(53, "inj", 14.0),
	}
	input := Input{
		Coins:        coins,
		LargeCeiling: 10,
		MidCeiling:   50,
		SmallCeiling: 200,
	}
	data, _, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Summary == "" {
		t.Error("summary should not be empty")
	}
	if data.Summary[:1] != "[" {
		t.Error("summary should start with [LABEL] prefix")
	}
}

func TestCompute_SummaryContainsWarningForSmallWeakness(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "bitcoin", 1.0),
		c(2, "ethereum", 2.0),
		c(3, "tether", 0.01),
		c(11, "chainlink", 8.0),
		c(12, "polygon", 7.0),
		c(13, "avalanche", 9.0),
		c(51, "gmx", -5.0),
		c(52, "dydx", -6.0),
		c(53, "inj", -4.0),
	}
	input := Input{
		Coins:        coins,
		LargeCeiling: 10,
		MidCeiling:   50,
		SmallCeiling: 200,
	}
	data, _, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Classification.Label != LabelRiskOn {
		t.Errorf("label: got %q, want risk_on", data.Classification.Label)
	}
	if data.Spreads.SmallVsLarge == nil {
		t.Fatal("smallVsLarge should be present")
	}
	if *data.Spreads.SmallVsLarge > -5.0 {
		t.Skip("smallVsLarge not negative enough for warning test")
	}
	found := strings.Contains(data.Summary, "weak") ||
		strings.Contains(data.Summary, "long-tail") ||
		strings.Contains(data.Summary, "small-cap")
	if !found {
		t.Errorf("summary should mention small-cap weakness, got: %q", data.Summary)
	}
}

func TestCompute_CustomSegments_TighterLarge(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "bitcoin", 10.0),
		c(2, "ethereum", 9.0),
		c(3, "tether", 0.01),
		c(4, "solana", 2.0),
		c(5, "bnb", 1.0),
		c(6, "xrp", 0.5),
		c(11, "chainlink", 7.0),
		c(12, "polygon", 8.0),
		c(13, "avalanche", 9.0),
	}
	input := Input{
		Coins:        coins,
		LargeCeiling: 3,
		MidCeiling:   10,
		SmallCeiling: 50,
	}
	data, _, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Classification.Label != LabelTopHeavy {
		t.Errorf("label: got %q, want top_heavy (tight large tier 1-3 at +6.3%%, mid 4-10 at +1.2%%)", data.Classification.Label)
	}
}
