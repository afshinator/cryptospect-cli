package v1

import (
	"fmt"
	"math"

	"github.com/afshinator/cryptospect-cli/internal/api/coingecko"
)

type tierCoin struct {
	coin      coingecko.CoinMarketsRankedCoin
	change24h float64
	marketCap float64
}

// Compute runs the 6-stage momentum-divergence pipeline: tier construction,
// statistical floor check, tier averages, spread matrix, classification, and
// summary string. It is pure (no I/O, no side effects).
func Compute(input Input) (Data, computedMeta, error) {
	if len(input.Coins) == 0 {
		return Data{}, computedMeta{}, fmt.Errorf("no coins provided")
	}

	var largeCoins, midCoins, smallCoins []tierCoin

	for _, coin := range input.Coins {
		if coin.Change24h == nil {
			continue
		}
		if coin.MarketCap <= 0 {
			continue
		}
		tc := tierCoin{coin: coin, change24h: *coin.Change24h, marketCap: coin.MarketCap}
		switch rank := coin.MarketCapRank; {
		case rank <= input.LargeCeiling:
			largeCoins = append(largeCoins, tc)
		case rank <= input.MidCeiling:
			midCoins = append(midCoins, tc)
		case rank <= input.SmallCeiling:
			smallCoins = append(smallCoins, tc)
		}
	}

	largeCount := len(largeCoins)
	midCount := len(midCoins)
	smallCount := len(smallCoins)

	largeAbsent := largeCount < TierFloorMinCoins
	midAbsent := midCount < TierFloorMinCoins
	smallAbsent := smallCount < TierFloorMinCoins

	if largeAbsent && midAbsent && smallAbsent {
		return Data{}, computedMeta{}, fmt.Errorf("all tiers absent: insufficient valid coins")
	}

	meta := computedMeta{
		TierCounts: TierCounts{
			Large: largeCount,
			Mid:   midCount,
			Small: smallCount,
		},
	}

	if largeAbsent || midAbsent || smallAbsent {
		meta.Confidence = "low"
	} else {
		meta.Confidence = "high"
	}

	var largeAvg, midAvg, smallAvg float64
	if !largeAbsent {
		largeAvg, meta.WeightingFallback = weightedMeanTier(largeCoins)
	}
	if !midAbsent {
		midAvg, _ = weightedMeanTier(midCoins)
	}
	if !smallAbsent {
		smallAvg, _ = weightedMeanTier(smallCoins)
	}

	averages := TierAverages{
		Large: largeAvg,
		Mid:   midAvg,
		Small: smallAvg,
	}

	var spreads Spreads
	if !largeAbsent && !midAbsent {
		v := round4(midAvg - largeAvg)
		spreads.MidVsLarge = &v
	}
	if !largeAbsent && !smallAbsent {
		v := round4(smallAvg - largeAvg)
		spreads.SmallVsLarge = &v
	}
	if !midAbsent && !smallAbsent {
		v := round4(smallAvg - midAvg)
		spreads.SmallVsMid = &v
	}

	var tailExtension bool
	if spreads.SmallVsLarge != nil && *spreads.SmallVsLarge > TailExtensionSpread {
		tailExtension = true
	}

	label := LabelNeutral
	desc := "Neutral"

	if spreads.MidVsLarge != nil {
		spread := *spreads.MidVsLarge
		if spread > RiskOnSpread && midAvg > MinPositivityGuard {
			label = LabelRiskOn
			desc = "Risk-On Rotation"
		} else if spread < TopHeavySpread {
			if largeAvg > ConcentrationDeadBand {
				label = LabelTopHeavy
				desc = "Top-Heavy — Narrow Rally"
			} else if largeAvg <= -ConcentrationDeadBand {
				label = LabelFlightToSafety
				desc = "Flight to Safety — Defensive Capital Concentration"
			}
			// dead band: stays neutral
		}
	}

	// Barbell modifier
	if tailExtension && label == LabelNeutral &&
		spreads.MidVsLarge != nil && *spreads.MidVsLarge > BarbellMidVsLargeMin {
		desc = "Barbell — Speculative Tail Extension"
	}

	meta.LabelDescription = desc

	classification := Classification{
		Label:       label,
		Description: desc,
	}

	summary := buildSummary(label, averages, spreads, tailExtension)

	td := TierDetail{}
	if !largeAbsent {
		td.Large = buildTierDetail(largeCoins)
	}
	if !midAbsent {
		td.Mid = buildTierDetail(midCoins)
	}
	if !smallAbsent {
		td.Small = buildTierDetail(smallCoins)
	}
	meta.TierDetail = &td

	meta.Thresholds = map[string]float64{
		"risk_on_spread":              RiskOnSpread,
		"top_heavy_spread":            TopHeavySpread,
		"min_positivity_guard":        MinPositivityGuard,
		"tail_extension_spread":       TailExtensionSpread,
		"tier_floor_min_coins":        TierFloorMinCoins,
		"segments_large_min":          float64(SegmentsLargeMin),
		"segments_small_max":          float64(SegmentsSmallMax),
		"concentration_dead_band_pct": ConcentrationDeadBand,
	}

	data := Data{
		TierAverages:   averages,
		Spreads:        spreads,
		TailExtension:  tailExtension,
		Classification: classification,
		Summary:        summary,
	}

	return data, meta, nil
}

func meanTier(coins []tierCoin) float64 {
	if len(coins) == 0 {
		return 0
	}
	sum := 0.0
	for _, tc := range coins {
		sum += tc.change24h
	}
	return sum / float64(len(coins))
}

func weightedMeanTier(coins []tierCoin) (avg float64, fallback bool) {
	weightedSum := 0.0
	totalWeight := 0.0
	for _, tc := range coins {
		if tc.marketCap > 0 {
			weightedSum += tc.change24h * tc.marketCap
			totalWeight += tc.marketCap
		}
	}
	if totalWeight == 0 {
		return meanTier(coins), true
	}
	return weightedSum / totalWeight, false
}

func buildTierDetail(coins []tierCoin) []TierCoinDetail {
	if len(coins) == 0 {
		return nil
	}
	totalCap := 0.0
	for _, tc := range coins {
		totalCap += tc.marketCap
	}
	details := make([]TierCoinDetail, len(coins))
	for i, tc := range coins {
		wp := 0.0
		if totalCap > 0 && tc.marketCap > 0 {
			wp = math.Round(tc.marketCap/totalCap*100*100) / 100
		}
		details[i] = TierCoinDetail{
			ID:        tc.coin.ID,
			Return24h: tc.coin.Change24h,
			MarketCap: tc.marketCap,
			WeightPct: wp,
		}
	}
	return details
}

func round4(v float64) float64 {
	return math.Round(v*10000) / 10000
}

func buildSummary(label string, averages TierAverages, spreads Spreads, tailExtension bool) string {
	prefix := fmt.Sprintf("[%s]", label)
	var body string

	switch label {
	case LabelRiskOn:
		if spreads.MidVsLarge != nil {
			body = fmt.Sprintf("Mid-caps outpacing mega-caps by %+.1fpp (mid avg %+.1f%%);",
				*spreads.MidVsLarge, averages.Mid)
		} else {
			body = fmt.Sprintf("Mid-caps outpacing mega-caps (mid avg %+.1f%%);",
				averages.Mid)
		}
		if tailExtension {
			if spreads.SmallVsLarge != nil {
				body += fmt.Sprintf(" full alt-season rotation detected (small_vs_large %+.1fpp).", *spreads.SmallVsLarge)
			} else {
				body += " full alt-season rotation detected."
			}
		} else {
			if spreads.SmallVsLarge != nil {
				if *spreads.SmallVsLarge < -5.0 {
					body += fmt.Sprintf(" but significant long-tail weakness detected (small_vs_large %+.1fpp).", *spreads.SmallVsLarge)
				} else {
					body += fmt.Sprintf(" long-tail not yet extending (small_vs_large %+.1fpp).", *spreads.SmallVsLarge)
				}
			} else {
				body += " long-tail data unavailable."
			}
		}
	case LabelTopHeavy:
		if spreads.MidVsLarge != nil {
			body = fmt.Sprintf("Mega-caps up %+.1f%%, mid-caps lagging (spread %+.1fpp) — narrow rally concentration.",
				averages.Large, *spreads.MidVsLarge)
		} else {
			body = fmt.Sprintf("Mega-caps up %+.1f%%, mid-caps lagging — narrow rally concentration.",
				averages.Large)
		}
	case LabelFlightToSafety:
		if spreads.MidVsLarge != nil {
			body = fmt.Sprintf("Mega-caps down %+.1f%%, mid-caps down harder (spread %+.1fpp) — defensive capital flight into large-caps.",
				averages.Large, *spreads.MidVsLarge)
		} else {
			body = fmt.Sprintf("Mega-caps down %+.1f%%, mid-caps down harder — defensive capital flight into large-caps.",
				averages.Large)
		}
	case LabelNeutral:
		if tailExtension {
			body = "Long-tail outperforming without mid-cap confirmation (speculative extension)."
		} else {
			body = "No high-conviction rotation signal detected."
		}
	}

	return prefix + " " + body
}
