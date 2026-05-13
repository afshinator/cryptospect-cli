package v1

import (
	"slices"
	"strings"
	"testing"
)

func TestCompute_BTCLedExpansion(t *testing.T) {
	priorDom := 52.0
	priorAge := 86400
	input := Input{
		BTCDominancePct:     53.0,
		PriorDominancePct:   &priorDom,
		PriorSnapshotAgeSec: &priorAge,
		BreadthScore:        0.75,
		LPRatio:             0.10,
		BTCChange24h:        ptr(2.5),
	}

	result := Compute(input)

	if result.DominanceTrend != TrendRising {
		t.Errorf("DominanceTrend = %q, want %q (delta +1.0pp >= +0.5)", result.DominanceTrend, TrendRising)
	}
	if result.Conviction != ConvictionNormal {
		t.Errorf("Conviction = %q, want %q (lp_ratio 0.10)", result.Conviction, ConvictionNormal)
	}
	if result.Modifier != ModifierPositiveMomentum {
		t.Errorf("Modifier = %q, want %q (BTC+2.5%%)", result.Modifier, ModifierPositiveMomentum)
	}
	if result.Regime != RegimeBTCLedExpansion {
		t.Errorf("Regime = %q, want %q (rising + broad)", result.Regime, RegimeBTCLedExpansion)
	}
	if result.ColdStart {
		t.Error("ColdStart should be false")
	}
	if result.Confidence != ConfidenceHigh {
		t.Errorf("Confidence = %q, want %q", result.Confidence, ConfidenceHigh)
	}
}

// ── Dominance trend boundaries ──

func TestCompute_DominanceTrend_RisingBoundary(t *testing.T) {
	// Exactly +0.5 → rising
	result := Compute(Input{
		BTCDominancePct: 52.5, PriorDominancePct: ptr(52.0),
		PriorSnapshotAgeSec: ptr(3600), BreadthScore: 0.75, LPRatio: 0.10, BTCChange24h: ptr(0.0),
	})
	if result.DominanceTrend != TrendRising {
		t.Errorf("DominanceTrend = %q, want %q at delta +0.5", result.DominanceTrend, TrendRising)
	}
}

func TestCompute_DominanceTrend_RisingAboveBoundary(t *testing.T) {
	result := Compute(Input{
		BTCDominancePct: 53.0, PriorDominancePct: ptr(52.3),
		PriorSnapshotAgeSec: ptr(3600), BreadthScore: 0.75, LPRatio: 0.10, BTCChange24h: ptr(0.0),
	})
	if result.DominanceTrend != TrendRising {
		t.Errorf("DominanceTrend = %q, want %q at delta +0.7", result.DominanceTrend, TrendRising)
	}
}

func TestCompute_DominanceTrend_FallingBoundary(t *testing.T) {
	// Exactly -0.5 → falling
	result := Compute(Input{
		BTCDominancePct: 51.5, PriorDominancePct: ptr(52.0),
		PriorSnapshotAgeSec: ptr(3600), BreadthScore: 0.75, LPRatio: 0.10, BTCChange24h: ptr(0.0),
	})
	if result.DominanceTrend != TrendFalling {
		t.Errorf("DominanceTrend = %q, want %q at delta -0.5", result.DominanceTrend, TrendFalling)
	}
}

func TestCompute_DominanceTrend_NeutralInDeadBand(t *testing.T) {
	result := Compute(Input{
		BTCDominancePct: 52.3, PriorDominancePct: ptr(52.0),
		PriorSnapshotAgeSec: ptr(3600), BreadthScore: 0.75, LPRatio: 0.10, BTCChange24h: ptr(0.0),
	})
	if result.DominanceTrend != TrendNeutral {
		t.Errorf("DominanceTrend = %q, want %q at delta +0.3", result.DominanceTrend, TrendNeutral)
	}
}

func TestCompute_DominanceTrend_ColdStart(t *testing.T) {
	result := Compute(Input{
		BTCDominancePct:   52.0,
		PriorDominancePct: nil,
		BreadthScore:      0.75,
		LPRatio:           0.10,
		BTCChange24h:      ptr(0.0),
	})
	if result.DominanceTrend != TrendNeutral {
		t.Errorf("DominanceTrend = %q, want %q on cold start", result.DominanceTrend, TrendNeutral)
	}
	if !result.ColdStart {
		t.Error("ColdStart should be true when prior is nil")
	}
	if !slices.Contains(result.Notes, "cold_start") {
		t.Error("Notes should contain cold_start")
	}
	if result.Confidence != ConfidenceMedium {
		t.Errorf("Confidence = %q, want %q on cold start", result.Confidence, ConfidenceMedium)
	}
}

// ── Conviction boundaries ──

func TestCompute_Conviction_High(t *testing.T) {
	result := Compute(Input{
		BTCDominancePct: 52.0, PriorDominancePct: ptr(52.0),
		PriorSnapshotAgeSec: ptr(3600), BreadthScore: 0.75, LPRatio: 0.20, BTCChange24h: ptr(0.0),
	})
	if result.Conviction != ConvictionHigh {
		t.Errorf("Conviction = %q, want %q at lp_ratio 0.20", result.Conviction, ConvictionHigh)
	}
}

func TestCompute_Conviction_HighBoundaryExclusive(t *testing.T) {
	// 0.15 is NOT high — boundary is exclusive (>0.15)
	result := Compute(Input{
		BTCDominancePct: 52.0, PriorDominancePct: ptr(52.0),
		PriorSnapshotAgeSec: ptr(3600), BreadthScore: 0.75, LPRatio: 0.15, BTCChange24h: ptr(0.0),
	})
	if result.Conviction != ConvictionNormal {
		t.Errorf("Conviction = %q, want %q at lp_ratio 0.15 (boundary is normal)", result.Conviction, ConvictionNormal)
	}
}

func TestCompute_Conviction_NormalLowBoundary(t *testing.T) {
	result := Compute(Input{
		BTCDominancePct: 52.0, PriorDominancePct: ptr(52.0),
		PriorSnapshotAgeSec: ptr(3600), BreadthScore: 0.75, LPRatio: 0.07, BTCChange24h: ptr(0.0),
	})
	if result.Conviction != ConvictionNormal {
		t.Errorf("Conviction = %q, want %q at lp_ratio 0.07 (boundary is normal)", result.Conviction, ConvictionNormal)
	}
}

func TestCompute_Conviction_Low(t *testing.T) {
	result := Compute(Input{
		BTCDominancePct: 52.0, PriorDominancePct: ptr(52.0),
		PriorSnapshotAgeSec: ptr(3600), BreadthScore: 0.75, LPRatio: 0.05, BTCChange24h: ptr(0.0),
	})
	if result.Conviction != ConvictionLow {
		t.Errorf("Conviction = %q, want %q at lp_ratio 0.05", result.Conviction, ConvictionLow)
	}
}

// ── Matrix mapping: all 10 cells ──

func TestCompute_Matrix_AllCells(t *testing.T) {
	tests := []struct {
		name         string
		trend        string
		breadthScore float64
		convLPR      float64
		want         string
	}{
		{"BTC-Led Expansion", "rising", 0.75, 0.10, RegimeBTCLedExpansion},
		{"Institutional Build", "rising", 0.50, 0.10, RegimeInstitutionalBuild},
		{"Flight to Safety", "rising", 0.20, 0.10, RegimeFlightToSafety},
		{"Steady Appreciation", "neutral", 0.75, 0.10, RegimeSteadyAppreciation},
		{"Consolidation", "neutral", 0.50, 0.10, RegimeConsolidation},
		{"Stagnation", "neutral", 0.20, 0.10, RegimeStagnation},
		{"Alt-Season / Mania", "falling", 0.75, 0.10, RegimeAltSeasonMania},
		{"Capital Rotation", "falling", 0.50, 0.10, RegimeCapitalRotation},
		{"Structural Decay", "falling", 0.20, 0.05, RegimeStructuralDecay},
		{"Capitulation", "falling", 0.20, 0.20, RegimeCapitulation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dom, priorDom float64
			switch tt.trend {
			case "rising":
				dom, priorDom = 53.0, 52.0
			case "falling":
				dom, priorDom = 51.0, 52.0
			default: // neutral
				dom, priorDom = 52.2, 52.0
			}

			result := Compute(Input{
				BTCDominancePct:     dom,
				PriorDominancePct:   &priorDom,
				PriorSnapshotAgeSec: ptr(3600),
				BreadthScore:        tt.breadthScore,
				LPRatio:             tt.convLPR,
				BTCChange24h:        ptr(0.0),
			})
			if result.Regime != tt.want {
				t.Errorf("Regime = %q, want %q", result.Regime, tt.want)
			}
		})
	}
}

// ── Breadth band boundaries ──

func TestCompute_BreadthBand_BroadBoundary(t *testing.T) {
	result := Compute(Input{
		BTCDominancePct: 52.0, PriorDominancePct: ptr(52.0),
		PriorSnapshotAgeSec: ptr(3600), BreadthScore: 0.60, LPRatio: 0.10, BTCChange24h: ptr(0.0),
	})
	if result.Regime != RegimeSteadyAppreciation {
		t.Errorf("Regime = %q, want %q at breadth 0.60 (broad boundary)", result.Regime, RegimeSteadyAppreciation)
	}
}

func TestCompute_BreadthBand_MixedBoundary(t *testing.T) {
	result := Compute(Input{
		BTCDominancePct: 52.0, PriorDominancePct: ptr(52.0),
		PriorSnapshotAgeSec: ptr(3600), BreadthScore: 0.40, LPRatio: 0.10, BTCChange24h: ptr(0.0),
	})
	if result.Regime != RegimeConsolidation {
		t.Errorf("Regime = %q, want %q at breadth 0.40 (mixed boundary)", result.Regime, RegimeConsolidation)
	}
}

// ── Capitulation disambiguation ──

func TestCompute_Capitulation_OnlyWithHighConviction(t *testing.T) {
	// Falling + narrow + normal conviction → Structural Decay
	result := Compute(Input{
		BTCDominancePct: 51.0, PriorDominancePct: ptr(52.0),
		PriorSnapshotAgeSec: ptr(3600), BreadthScore: 0.20, LPRatio: 0.15, BTCChange24h: ptr(0.0),
	})
	if result.Regime != RegimeStructuralDecay {
		t.Errorf("Regime = %q, want %q (normal conviction at boundary)", result.Regime, RegimeStructuralDecay)
	}
}

func TestCompute_Capitulation_HighConviction(t *testing.T) {
	result := Compute(Input{
		BTCDominancePct: 51.0, PriorDominancePct: ptr(52.0),
		PriorSnapshotAgeSec: ptr(3600), BreadthScore: 0.20, LPRatio: 0.20, BTCChange24h: ptr(-2.0),
	})
	if result.Regime != RegimeCapitulation {
		t.Errorf("Regime = %q, want %q", result.Regime, RegimeCapitulation)
	}
	if result.Notes == nil {
		t.Fatal("Notes must not be nil")
	}
	if slices.Contains(result.Notes, "abnormal_capitulation") {
		t.Error("Should NOT have abnormal_capitulation with negative modifier")
	}
	if slices.Contains(result.Notes, "capitulation_price_stabilizing") {
		t.Error("Should NOT have capitulation_price_stabilizing with negative modifier")
	}
	if result.Confidence != ConfidenceHigh {
		t.Errorf("Confidence = %q, want %q for standard capitulation", result.Confidence, ConfidenceHigh)
	}
}

// ── Capitulation sub-states ──

func TestCompute_Capitulation_NeutralModifier(t *testing.T) {
	result := Compute(Input{
		BTCDominancePct: 51.0, PriorDominancePct: ptr(52.0),
		PriorSnapshotAgeSec: ptr(3600), BreadthScore: 0.20, LPRatio: 0.20, BTCChange24h: ptr(0.0),
	})
	if result.Regime != RegimeCapitulation {
		t.Fatalf("Regime = %q, want %q", result.Regime, RegimeCapitulation)
	}
	if !slices.Contains(result.Notes, "capitulation_price_stabilizing") {
		t.Error("Notes should contain capitulation_price_stabilizing")
	}
	if result.Confidence != ConfidenceMedium {
		t.Errorf("Confidence = %q, want %q for stabilizing", result.Confidence, ConfidenceMedium)
	}
}

func TestCompute_Capitulation_PositiveMomentumModifier(t *testing.T) {
	result := Compute(Input{
		BTCDominancePct: 51.0, PriorDominancePct: ptr(52.0),
		PriorSnapshotAgeSec: ptr(3600), BreadthScore: 0.20, LPRatio: 0.20, BTCChange24h: ptr(3.0),
	})
	if result.Regime != RegimeCapitulation {
		t.Fatalf("Regime = %q, want %q", result.Regime, RegimeCapitulation)
	}
	if !slices.Contains(result.Notes, "abnormal_capitulation") {
		t.Error("Notes should contain abnormal_capitulation")
	}
	if result.Confidence != ConfidenceMedium {
		t.Errorf("Confidence = %q, want %q for abnormal", result.Confidence, ConfidenceMedium)
	}
}

// ── Modifier boundaries ──

func TestCompute_Modifier_PositiveBoundary(t *testing.T) {
	result := Compute(Input{
		BTCDominancePct: 52.0, PriorDominancePct: ptr(52.0),
		PriorSnapshotAgeSec: ptr(3600), BreadthScore: 0.75, LPRatio: 0.10, BTCChange24h: ptr(0.5),
	})
	if result.Modifier != ModifierPositiveMomentum {
		t.Errorf("Modifier = %q, want %q at +0.5%%", result.Modifier, ModifierPositiveMomentum)
	}
}

func TestCompute_Modifier_NegativeBoundary(t *testing.T) {
	result := Compute(Input{
		BTCDominancePct: 52.0, PriorDominancePct: ptr(52.0),
		PriorSnapshotAgeSec: ptr(3600), BreadthScore: 0.75, LPRatio: 0.10, BTCChange24h: ptr(-0.5),
	})
	if result.Modifier != ModifierNegativePressure {
		t.Errorf("Modifier = %q, want %q at -0.5%%", result.Modifier, ModifierNegativePressure)
	}
}

func TestCompute_Modifier_NeutralDeadBand(t *testing.T) {
	result := Compute(Input{
		BTCDominancePct: 52.0, PriorDominancePct: ptr(52.0),
		PriorSnapshotAgeSec: ptr(3600), BreadthScore: 0.75, LPRatio: 0.10, BTCChange24h: ptr(0.3),
	})
	if result.Modifier != ModifierNeutral {
		t.Errorf("Modifier = %q, want %q in dead band", result.Modifier, ModifierNeutral)
	}
}

// ── Missing BTC reference ──

func TestCompute_MissingBTCReference(t *testing.T) {
	result := Compute(Input{
		BTCDominancePct: 52.0, PriorDominancePct: ptr(52.0),
		PriorSnapshotAgeSec: ptr(3600), BreadthScore: 0.75, LPRatio: 0.10, BTCChange24h: nil,
	})
	if result.Modifier != ModifierNeutral {
		t.Errorf("Modifier = %q, want %q (fallback)", result.Modifier, ModifierNeutral)
	}
	if !result.MissingReferenceData {
		t.Error("MissingReferenceData should be true")
	}
	if !slices.Contains(result.Notes, "missing_reference_data") {
		t.Error("Notes should contain missing_reference_data")
	}
	if result.Confidence != ConfidenceLow {
		t.Errorf("Confidence = %q, want %q for missing reference", result.Confidence, ConfidenceLow)
	}
}

// ── Confidence precedence ──

func TestCompute_Confidence_ColdStart(t *testing.T) {
	result := Compute(Input{
		BTCDominancePct: 52.0, PriorDominancePct: nil,
		BreadthScore: 0.75, LPRatio: 0.10, BTCChange24h: ptr(0.0),
	})
	if result.Confidence != ConfidenceMedium {
		t.Errorf("Confidence = %q, want %q (cold start)", result.Confidence, ConfidenceMedium)
	}
}

func TestCompute_Confidence_ColdStartPlusMissingRef(t *testing.T) {
	// cold start (medium) + missing ref (low) → low wins
	result := Compute(Input{
		BTCDominancePct: 52.0, PriorDominancePct: nil,
		BreadthScore: 0.75, LPRatio: 0.10, BTCChange24h: nil,
	})
	if result.Confidence != ConfidenceLow {
		t.Errorf("Confidence = %q, want %q (cold+missing → low wins)", result.Confidence, ConfidenceLow)
	}
}

func TestCompute_Confidence_DegradedForcesLow(t *testing.T) {
	result := Compute(Input{
		BTCDominancePct: 52.0, PriorDominancePct: ptr(52.0),
		PriorSnapshotAgeSec: ptr(3600), BreadthScore: 0.75, LPRatio: 0.10, BTCChange24h: ptr(0.0),
		BreadthDegraded: true,
	})
	if result.Confidence != ConfidenceLow {
		t.Errorf("Confidence = %q, want %q (degraded)", result.Confidence, ConfidenceLow)
	}
}

func TestCompute_Confidence_WeightRedistribution(t *testing.T) {
	result := Compute(Input{
		BTCDominancePct: 52.0, PriorDominancePct: ptr(52.0),
		PriorSnapshotAgeSec: ptr(3600), BreadthScore: 0.75, LPRatio: 0.10, BTCChange24h: ptr(0.0),
		WeightRedistributed: true,
	})
	if result.Confidence != ConfidenceMedium {
		t.Errorf("Confidence = %q, want %q (weight redist)", result.Confidence, ConfidenceMedium)
	}
	if !slices.Contains(result.Notes, "weight_redistribution") {
		t.Error("Notes should contain weight_redistribution")
	}
}

// ── Notes accumulation ──

func TestCompute_Notes_MultipleConditions(t *testing.T) {
	result := Compute(Input{
		BTCDominancePct: 52.0, PriorDominancePct: nil,
		BreadthScore: 0.20, LPRatio: 0.05, BTCChange24h: nil,
	})
	// cold_start + missing_reference_data should both be present
	if !slices.Contains(result.Notes, "cold_start") {
		t.Error("missing cold_start note")
	}
	if !slices.Contains(result.Notes, "missing_reference_data") {
		t.Error("missing missing_reference_data note")
	}
}

// ── Summary generation ──

func TestCompute_Summary_Basic(t *testing.T) {
	// Non-cold, non-missing, non-redistributed → no reliability tokens
	result := Compute(Input{
		BTCDominancePct: 53.0, PriorDominancePct: ptr(52.0),
		PriorSnapshotAgeSec: ptr(3600), BreadthScore: 0.75, LPRatio: 0.10, BTCChange24h: ptr(2.5),
	})
	if result.Regime == "" {
		t.Fatal("regime should not be empty")
	}
	if strings.Contains(result.Summary, "[SIGNAL_UNVERIFIED]") {
		t.Error("summary should not contain cold start token")
	}
	if strings.Contains(result.Summary, "[MISSING_BTC_REF]") {
		t.Error("summary should not contain missing ref token")
	}
	if strings.Contains(result.Summary, "[BREADTH_PARTIAL]") {
		t.Error("summary should not contain redistribution token")
	}
}

func TestCompute_Summary_ColdStartToken(t *testing.T) {
	result := Compute(Input{
		BTCDominancePct: 52.0, PriorDominancePct: nil,
		BreadthScore: 0.50, LPRatio: 0.10, BTCChange24h: ptr(0.0),
	})
	if !strings.HasPrefix(result.Summary, "[SIGNAL_UNVERIFIED] ") {
		t.Errorf("summary should start with [SIGNAL_UNVERIFIED], got %q", result.Summary)
	}
}

func TestCompute_Summary_MissingRefToken(t *testing.T) {
	result := Compute(Input{
		BTCDominancePct: 52.0, PriorDominancePct: ptr(52.0),
		PriorSnapshotAgeSec: ptr(3600), BreadthScore: 0.50, LPRatio: 0.10, BTCChange24h: nil,
	})
	if !strings.Contains(result.Summary, " [MISSING_BTC_REF]") {
		t.Errorf("summary should end with [MISSING_BTC_REF], got %q", result.Summary)
	}
}

func TestCompute_Summary_RedistributionToken(t *testing.T) {
	result := Compute(Input{
		BTCDominancePct: 52.0, PriorDominancePct: ptr(52.0),
		PriorSnapshotAgeSec: ptr(3600), BreadthScore: 0.50, LPRatio: 0.10, BTCChange24h: ptr(0.0),
		WeightRedistributed: true,
	})
	if !strings.Contains(result.Summary, " [BREADTH_PARTIAL]") {
		t.Errorf("summary should contain [BREADTH_PARTIAL], got %q", result.Summary)
	}
}

func TestCompute_Summary_AllTokens(t *testing.T) {
	result := Compute(Input{
		BTCDominancePct: 52.0, PriorDominancePct: nil,
		BreadthScore: 0.50, LPRatio: 0.10, BTCChange24h: nil,
		WeightRedistributed: true,
	})
	s := result.Summary
	if !strings.HasPrefix(s, "[SIGNAL_UNVERIFIED] ") {
		t.Errorf("missing cold token prefix: %q", s)
	}
	if !strings.Contains(s, " [MISSING_BTC_REF]") {
		t.Errorf("missing ref token: %q", s)
	}
	if !strings.Contains(s, " [BREADTH_PARTIAL]") {
		t.Errorf("missing redistribution token: %q", s)
	}
}

// ── Summary conviction-aware branching ──

func TestCompute_Summary_StagnationPressureCooker(t *testing.T) {
	// Neutral + narrow + high conviction → "Pressure Cooker"
	result := Compute(Input{
		BTCDominancePct: 52.2, PriorDominancePct: ptr(52.0),
		PriorSnapshotAgeSec: ptr(3600), BreadthScore: 0.20, LPRatio: 0.20, BTCChange24h: ptr(0.0),
	})
	if result.Regime != RegimeStagnation {
		t.Fatalf("Regime = %q, want %q", result.Regime, RegimeStagnation)
	}
	if !strings.Contains(strings.ToLower(result.Summary), "pressure cooker") {
		t.Errorf("Stagnation + high conviction summary should mention Pressure Cooker: %q", result.Summary)
	}
}

func TestCompute_Summary_ConsolidationPressureCooker(t *testing.T) {
	// Neutral + mixed + high conviction → "Pressure Cooker"
	result := Compute(Input{
		BTCDominancePct: 52.2, PriorDominancePct: ptr(52.0),
		PriorSnapshotAgeSec: ptr(3600), BreadthScore: 0.50, LPRatio: 0.20, BTCChange24h: ptr(0.0),
	})
	if result.Regime != RegimeConsolidation {
		t.Fatalf("Regime = %q, want %q", result.Regime, RegimeConsolidation)
	}
	if !strings.Contains(strings.ToLower(result.Summary), "pressure cooker") {
		t.Errorf("Consolidation + high conviction summary should mention Pressure Cooker: %q", result.Summary)
	}
}

// ── Summary capitulation branching ──

func TestCompute_Summary_CapitulationStandard(t *testing.T) {
	result := Compute(Input{
		BTCDominancePct: 51.0, PriorDominancePct: ptr(52.0),
		PriorSnapshotAgeSec: ptr(3600), BreadthScore: 0.20, LPRatio: 0.20, BTCChange24h: ptr(-2.0),
	})
	if result.Regime != RegimeCapitulation || result.Modifier != ModifierNegativePressure {
		t.Fatal("expected capitulation with negative_pressure")
	}
	if strings.Contains(strings.ToLower(result.Summary), "stabilizing") {
		t.Error("standard capitulation should not mention stabilizing")
	}
}

func TestCompute_Summary_CapitulationStabilizing(t *testing.T) {
	result := Compute(Input{
		BTCDominancePct: 51.0, PriorDominancePct: ptr(52.0),
		PriorSnapshotAgeSec: ptr(3600), BreadthScore: 0.20, LPRatio: 0.20, BTCChange24h: ptr(0.0),
	})
	if !strings.Contains(strings.ToLower(result.Summary), "stabilizing") {
		t.Errorf("stabilizing capitulation should mention stabilizing: %q", result.Summary)
	}
}

func TestCompute_Summary_CapitulationAbnormal(t *testing.T) {
	result := Compute(Input{
		BTCDominancePct: 51.0, PriorDominancePct: ptr(52.0),
		PriorSnapshotAgeSec: ptr(3600), BreadthScore: 0.20, LPRatio: 0.20, BTCChange24h: ptr(3.0),
	})
	if !strings.Contains(strings.ToLower(result.Summary), "v-bottom") {
		t.Errorf("abnormal capitulation should mention V-bottom/short squeeze: %q", result.Summary)
	}
}

func ptr[T any](v T) *T { return &v }
