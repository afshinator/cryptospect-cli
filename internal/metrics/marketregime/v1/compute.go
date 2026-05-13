package v1

// Compute performs the pure market-regime classification.
func Compute(in Input) ComputeResult {
	// 1. Dominance trend
	var trend string
	coldStart := false
	if in.PriorDominancePct == nil {
		trend = TrendNeutral
		coldStart = true
	} else {
		delta := in.BTCDominancePct - *in.PriorDominancePct
		switch {
		case delta >= DomDeadBandPP:
			trend = TrendRising
		case delta <= -DomDeadBandPP:
			trend = TrendFalling
		default:
			trend = TrendNeutral
		}
	}

	// 2. Conviction (lp_ratio)
	var conviction string
	switch {
	case in.LPRatio > ConvictionHighThresh:
		conviction = ConvictionHigh
	case in.LPRatio >= ConvictionLowThresh:
		conviction = ConvictionNormal
	default:
		conviction = ConvictionLow
	}

	// 3. Modifier (BTC 24h change)
	var modifier string
	missingRef := false
	if in.BTCChange24h == nil {
		modifier = ModifierNeutral
		missingRef = true
	} else {
		switch {
		case *in.BTCChange24h >= ModifierDeadBandPP:
			modifier = ModifierPositiveMomentum
		case *in.BTCChange24h <= -ModifierDeadBandPP:
			modifier = ModifierNegativePressure
		default:
			modifier = ModifierNeutral
		}
	}

	// 4. Breadth band
	var band string
	switch {
	case in.BreadthScore >= BreadthBroadThresh:
		band = "broad"
	case in.BreadthScore >= BreadthNarrowThresh:
		band = "mixed"
	default:
		band = "narrow"
	}

	// 5. Matrix lookup
	regime := matrixLookup(trend, band, conviction)

	// 6. Capitulation sub-state notes
	capNote := ""
	if regime == RegimeCapitulation {
		switch modifier {
		case ModifierNegativePressure:
			// standard capitulation — no note
		case ModifierNeutral:
			capNote = "capitulation_price_stabilizing"
		case ModifierPositiveMomentum:
			capNote = "abnormal_capitulation"
		}
	}

	// 7. Notes
	notes := []string{}
	if coldStart {
		notes = append(notes, "cold_start")
	}
	if in.WeightRedistributed {
		notes = append(notes, "weight_redistribution")
	}
	if missingRef {
		notes = append(notes, "missing_reference_data")
	}
	if capNote != "" {
		notes = append(notes, capNote)
	}

	// 8. Confidence
	confidence := ConfidenceHigh
	if coldStart {
		confidence = minConfidence(confidence, ConfidenceMedium)
	}
	if in.WeightRedistributed {
		confidence = minConfidence(confidence, ConfidenceMedium)
	}
	if in.BreadthDegraded {
		confidence = minConfidence(confidence, ConfidenceLow)
	}
	if missingRef {
		confidence = minConfidence(confidence, ConfidenceLow)
	}
	if capNote != "" {
		confidence = minConfidence(confidence, ConfidenceMedium)
	}

	// 9. Summary
	summary := buildSummary(regime, modifier, trend, conviction, coldStart, missingRef, in.WeightRedistributed)

	return ComputeResult{
		Regime:               regime,
		Modifier:             modifier,
		DominanceTrend:       trend,
		Conviction:           conviction,
		BreadthScore:         in.BreadthScore,
		ColdStart:            coldStart,
		Confidence:           confidence,
		Notes:                notes,
		Summary:              summary,
		MissingReferenceData: missingRef,
		CapitulationNote:     capNote,
	}
}

func matrixLookup(trend, band, conviction string) string {
	switch trend {
	case TrendRising:
		switch band {
		case "broad":
			return RegimeBTCLedExpansion
		case "mixed":
			return RegimeInstitutionalBuild
		default:
			return RegimeFlightToSafety
		}
	case TrendFalling:
		switch band {
		case "broad":
			return RegimeAltSeasonMania
		case "mixed":
			return RegimeCapitalRotation
		default:
			if conviction == ConvictionHigh {
				return RegimeCapitulation
			}
			return RegimeStructuralDecay
		}
	default: // neutral
		switch band {
		case "broad":
			return RegimeSteadyAppreciation
		case "mixed":
			return RegimeConsolidation
		default:
			return RegimeStagnation
		}
	}
}

func minConfidence(a, b string) string {
	order := map[string]int{ConfidenceHigh: 0, ConfidenceMedium: 1, ConfidenceLow: 2}
	if order[b] > order[a] {
		return b
	}
	return a
}

func buildSummary(regime, modifier, _ /*trend*/, conviction string, coldStart, missingRef, weightRedist bool) string {
	var base string

	switch regime {
	case RegimeBTCLedExpansion:
		base = "BTC dominance rising, breadth broad — BTC-Led Expansion" +
			" with " + modifierText(modifier) + ". Capital flowing into BTC with broad alt participation."
	case RegimeInstitutionalBuild:
		base = "BTC dominance rising, breadth mixed — Institutional Build" +
			" with " + modifierText(modifier) + ". BTC outperforming a selective market; early accumulation or defensive concentration."
	case RegimeFlightToSafety:
		base = "BTC dominance rising, breadth narrow — Flight to Safety" +
			" with " + modifierText(modifier) + ". Capital retreating into BTC as the crypto safe harbor."
	case RegimeSteadyAppreciation:
		base = "Dominance neutral, breadth broad — Steady Appreciation" +
			" with " + modifierText(modifier) + ". Balanced market rising with broad participation and no dominance shift."
	case RegimeConsolidation:
		if conviction == ConvictionHigh {
			base = "Dominance neutral, breadth mixed, high conviction — Consolidation (Pressure Cooker)." +
				" High volume within a tightening range; imminent violent break likely in either direction."
		} else {
			base = "Dominance neutral, breadth mixed — Consolidation." +
				" Market seeking direction; do not increase exposure."
		}
	case RegimeStagnation:
		if conviction == ConvictionHigh {
			base = "Dominance neutral, breadth narrow, high conviction — Stagnation (Pressure Cooker)." +
				" High volume in a directionless market; breakout likely in either direction."
		} else {
			base = "Dominance neutral, breadth narrow, " + conviction + " conviction — Stagnation." +
				" Market ignored; alt exposure is high-risk."
		}
	case RegimeAltSeasonMania:
		base = "BTC dominance falling, breadth broad — Alt-Season / Mania" +
			" with " + modifierText(modifier) + ". Capital rotating down the risk curve with broad participation."
	case RegimeCapitalRotation:
		base = "BTC dominance falling, breadth mixed — Capital Rotation" +
			" with " + modifierText(modifier) + ". Selective rotation; some sectors capturing flow while others lag."
	case RegimeStructuralDecay:
		base = "BTC dominance falling, breadth narrow, " + conviction + " conviction — Structural Decay." +
			" Slow directional bleed on thin volume; no panic floor yet."
	case RegimeCapitulation:
		switch modifier {
		case ModifierNegativePressure:
			base = "BTC dominance falling, breadth narrow, high conviction — Capitulation." +
				" Panic selling confirmed by volume; alts collapsing, BTC relatively outperforming."
		case ModifierNeutral:
			base = "BTC dominance falling, breadth narrow, high conviction — Capitulation (Price Stabilizing)." +
				" Panic volume persists but BTC price has stopped falling; stabilization or passive absorption."
		case ModifierPositiveMomentum:
			base = "BTC dominance falling, breadth narrow, high conviction — Capitulation (Abnormal)." +
				" BTC price reversing upward against panic volume; V-bottom or short squeeze in progress."
		default:
			base = "BTC dominance falling, breadth narrow, high conviction — Capitulation."
		}
	}

	var tokens []string
	if coldStart {
		tokens = append(tokens, "[SIGNAL_UNVERIFIED] ")
	}
	if missingRef {
		tokens = append(tokens, " [MISSING_BTC_REF]")
	}
	if weightRedist {
		tokens = append(tokens, " [BREADTH_PARTIAL]")
	}

	if len(tokens) == 0 {
		return base
	}
	return tokens[0] + base + stringsJoin(tokens[1:])
}

func modifierText(m string) string {
	switch m {
	case ModifierPositiveMomentum:
		return "positive momentum"
	case ModifierNegativePressure:
		return "negative pressure"
	default:
		return "neutral BTC direction"
	}
}

func stringsJoin(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	s := ""
	for _, p := range parts {
		s += p
	}
	return s
}

func classificationDescription(regime, conviction string) string {
	switch regime {
	case RegimeBTCLedExpansion:
		return "BTC-Led Expansion — broad participation with rising BTC dominance"
	case RegimeInstitutionalBuild:
		return "Institutional Build — BTC outperforming a mixed market; early accumulation or defensive concentration"
	case RegimeFlightToSafety:
		return "Flight to Safety — capital retreating into BTC as the crypto safe harbor"
	case RegimeSteadyAppreciation:
		return "Steady Appreciation — balanced market rising with broad participation and no dominance shift"
	case RegimeConsolidation:
		if conviction == ConvictionHigh {
			return "Consolidation (Pressure Cooker) — high volume range-bound trading; imminent violent break likely"
		}
		return "Consolidation — market seeking direction, mixed participation"
	case RegimeStagnation:
		if conviction == ConvictionHigh {
			return "Stagnation (Pressure Cooker) — high volume, directionless market; explosive break likely in either direction"
		}
		return "Stagnation — flat dominance, narrow breadth, market ignored (Pressure Cooker if conviction is high)"
	case RegimeAltSeasonMania:
		return "Alt-Season / Mania — capital rotating down the risk curve with broad alt participation"
	case RegimeCapitalRotation:
		return "Capital Rotation — selective rotation; some sectors capturing flow while others lag"
	case RegimeCapitulation:
		return "Capitulation — panic selling, high volume, alts collapsing"
	case RegimeStructuralDecay:
		return "Structural Decay — slow bleed, falling dominance, thin volume"
	default:
		return regime
	}
}
