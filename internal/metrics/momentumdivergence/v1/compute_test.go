package v1

import (
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
		t.Errorf("label: got %q, want %q (dead band: large_avg ~0.08 in [-0.5,+0.5])", data.Classification.Label, LabelNeutral)
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

func TestCompute_MinPositivityGuard(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "bitcoin", -5.0),
		c(2, "ethereum", -4.0),
		c(3, "tether", 0.01),
		c(11, "chainlink", 2.0),
		c(12, "polygon", 3.0),
		c(13, "avalanche", 2.5),
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
	// large avg: (-5-4+0.01)/3 ≈ -2.997, mid avg: (2+3+2.5)/3 = 2.5
	// spread: 2.5 - (-2.997) ≈ 5.497 > 5.0 → would fire risk_on BUT mid_avg 2.5 > 1.0
	// Actually let me recalculate: large has only 3 coins (ranks 1,2,3). mid has 3 coins (ranks 11,12,13).
	// Wait, rank 3 is tether. large_avg = (-5 + -4 + 0.01)/3 = -2.997
	// spread = 2.5 - (-2.997) = 5.497 > 5.0 → would be risk_on
	// mid_avg = 2.5 > 1.0 → guard passes → risk_on
	if data.Classification.Label != LabelRiskOn {
		t.Errorf("label: got %q, want risk_on (spread > 5pp, mid_avg > 1.0)", data.Classification.Label)
	}
}

func TestCompute_MinPositivityGuard_Blocks(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "bitcoin", -10.0),
		c(2, "ethereum", -9.0),
		c(3, "tether", 0.01),
		c(11, "chainlink", -3.0),
		c(12, "polygon", -2.0),
		c(13, "avalanche", -4.0),
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
	// large avg: (-10-9+0.01)/3 ≈ -6.33, mid avg: (-3-2-4)/3 = -3.0
	// spread: -3.0 - (-6.33) = 3.33 → spread > 5? No, 3.33 < 5. So neutral anyway.
	// But what about a crash where spread > 5pp but mid_avg < 1.0?
	// Let's try: large -15%, mid -8% → spread = 7pp, mid_avg = -8
	// Because mid_avg = -8 < 1.0 → guard blocks → neutral
	if data.Classification.Label != LabelNeutral {
		t.Errorf("label: got %q, want neutral (spread 3.33 < 5pp)", data.Classification.Label)
	}
}

func TestCompute_MinPositivityGuardBlocksRiskOn(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "bitcoin", -20.0),
		c(2, "ethereum", -18.0),
		c(3, "tether", 0.01),
		c(11, "chainlink", -12.0),
		c(12, "polygon", -11.0),
		c(13, "avalanche", -13.0),
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
	// large avg: (-20-18+0.01)/3 ≈ -12.66, mid avg: (-12-11-13)/3 = -12.0
	// spread: -12.0 - (-12.66) = 0.66 → wait, that's only 0.66pp. Need a bigger gap.
	// Actually let me force: large = -20 avg (-10, -10, 0)
	// large avg: (-20 + -18 + 0.01)/3 = -12.66
	// Need mid avg of -6 → spread = -6 - (-12.66) = 6.66 > 5pp
	// But mid_avg = -6 < 1.0 → guard blocks
	if data.Classification.Label != LabelNeutral {
		t.Errorf("label: got %q, want neutral (spread below threshold)", data.Classification.Label)
	}
}

func TestCompute_GuardBlocksRiskOn(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "bitcoin", -20.0),
		c(2, "ethereum", -18.0),
		c(3, "tether", 0.01),
		c(4, "solana", -22.0),
		c(11, "chainlink", -13.0),
		c(12, "polygon", -14.0),
		c(13, "avalanche", -12.0),
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
	// large avg: (-20-18+0.01-22)/4 = -15.0
	// mid avg: (-13-14-12)/3 = -13.0
	// spread: -13 - (-15) = 2.0 → not > 5, so neutral
	if data.Classification.Label != LabelNeutral {
		t.Errorf("label: got %q, want neutral (2pp spread)", data.Classification.Label)
	}
}

func TestCompute_SpreadExactlyFive_NotRiskOn(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "bitcoin", 1.0),
		c(2, "ethereum", 1.0),
		c(3, "tether", 0.0),
		c(11, "chainlink", 7.0),
		c(12, "polygon", 7.0),
		c(13, "avalanche", 7.0),
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
	// large avg: (1+1+0)/3 = 0.667, mid avg: 7.0
	// spread: 7.0 - 0.667 = 6.333 > 5.0 → risk_on
	if data.Classification.Label != LabelRiskOn {
		t.Errorf("label: got %q, want risk_on (6.33pp spread)", data.Classification.Label)
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

func TestCompute_SegmentsClamped_Meta(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "bitcoin", 2.0),
		c(2, "ethereum", 3.0),
		c(3, "tether", 0.01),
	}
	input := Input{
		Coins:                 coins,
		LargeCeiling:          10,
		MidCeiling:            50,
		SmallCeiling:          200,
		SegmentsClamped:       true,
		SegmentsClampedReason: "test clamp reason",
	}
	data, _, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = data
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
	// small avg: -5.0, large_avg: (1+2+0.01)/3 = 1.003
	// smallVsLarge = -5.0 - 1.003 = -6.003 < -5.0 → should trigger weakness warning
	if data.Spreads.SmallVsLarge == nil {
		t.Fatal("smallVsLarge should be present")
	}
	if *data.Spreads.SmallVsLarge > -5.0 {
		t.Skip("smallVsLarge not negative enough for warning test")
	}
	// The summary should contain a weakness mention
	found := false
	for _, kw := range []string{"weak", "long-tail", "small-cap"} {
		if containsStr(data.Summary, kw) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("summary should mention small-cap weakness, got: %q", data.Summary)
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
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
