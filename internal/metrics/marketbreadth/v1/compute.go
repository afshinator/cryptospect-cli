package v1

import (
	"fmt"

	"github.com/afshinator/cryptospect-cli/internal/metrics"
)

var nominalWeights = map[string]float64{
	"1h":  Weight1h,
	"24h": Weight24h,
	"7d":  Weight7d,
	"30d": Weight30d,
}

var timeframeOrder = []string{"1h", "24h", "7d", "30d"}

var classificationDescriptions = map[string]string{
	ClassificationBroad:  "Healthy Growth",
	ClassificationMixed:  "Selective Participation",
	ClassificationNarrow: "The Illusion",
}

// Compute calculates the market-breadth score from timeframe counts and validator data.
func Compute(input *Input) (ComputeResult, error) {
	// Step 1: Per-timeframe floor and weight redistribution
	effectiveWeights := make(map[string]float64, 4)
	for k, v := range nominalWeights {
		effectiveWeights[k] = v
	}

	var droppedTimeframes []string
	remainingWeight := 0.0

	for _, tf := range timeframeOrder {
		m := input.TimeframeCounts[tf]
		if m.TotalCount < StatisticalFloor {
			droppedTimeframes = append(droppedTimeframes, tf)
			effectiveWeights[tf] = 0.0
		} else {
			remainingWeight += nominalWeights[tf]
		}
	}

	if remainingWeight > 0 {
		for _, tf := range timeframeOrder {
			if effectiveWeights[tf] > 0 {
				effectiveWeights[tf] = nominalWeights[tf] / remainingWeight
			}
		}
	}

	// Step 2: Compute green_pct per timeframe and weighted composite
	timeframePct := make(map[string]float64, 4)
	var composite float64
	timeframeAllAbsent := true

	for _, tf := range timeframeOrder {
		m := input.TimeframeCounts[tf]
		if m.TotalCount > 0 {
			timeframeAllAbsent = false
			if effectiveWeights[tf] > 0 {
				pct := float64(m.GreenCount) / float64(m.TotalCount)
				timeframePct[tf] = pct
				composite += effectiveWeights[tf] * pct
			}
		}
	}

	// Step 3: CoinsCounted
	coinsCounted := input.CoinsCounted
	if coinsCounted == 0 {
		for _, tf := range timeframeOrder {
			if input.TimeframeCounts[tf].TotalCount > coinsCounted {
				coinsCounted = input.TimeframeCounts[tf].TotalCount
			}
		}
	}

	// Step 4: Determine metric status
	var status string
	switch {
	case timeframeAllAbsent:
		status = "unavailable"
	case coinsCounted < StatisticalFloor:
		status = "degraded"
	case len(droppedTimeframes) > 0:
		status = "degraded"
	default:
		status = "ok"
	}

	// Step 5: Classification
	var label string
	switch {
	case composite >= BroadThreshold:
		label = ClassificationBroad
	case composite >= NarrowThreshold:
		label = ClassificationMixed
	default:
		label = ClassificationNarrow
	}

	// Step 6: Divergence detection (Ghost Rally)
	divergence := input.BTCAvailable && input.BTCChange24h > DivergenceBTCChangeMin && composite < NarrowThreshold

	// Step 7: Discrepancy detection and validator confidence
	discrepancy := false
	discrepancyNote := ""
	confidence := "high"

	switch {
	case !input.KlineAvailable:
		discrepancy = true
		discrepancyNote = "Binance kline data unavailable — validator skipped"
		confidence = "low"
	case input.KlineClose == 0.0:
		discrepancy = true
		discrepancyNote = "Binance Close price is 0.0 — likely a parse failure, validator skipped"
		confidence = "low"
	default:
		openTimeSec := input.KlineOpenTimeMs / 1000
		if input.Now.Unix()-openTimeSec > StalenessThresholdSec {
			discrepancy = true
			discrepancyNote = "Binance candle stale (>90m) — validator skipped, directional consensus unavailable"
			confidence = "low"
		} else {
			dirCG := sign(input.BTCChange24h)
			dirBN := sign(input.KlineClose - input.KlineOpen)
			if dirCG != 0 && dirBN != 0 && dirCG != dirBN {
				discrepancy = true
				if dirCG > 0 {
					discrepancyNote = "BTC 24h trend positive (CoinGecko) but 1h candle negative (Binance) — intraday reversal signal"
				} else {
					discrepancyNote = "BTC 24h trend negative (CoinGecko) but 1h candle positive (Binance) — potential bottom forming"
				}
				confidence = "medium"
			}
		}
	}

	if !input.BTCAvailable {
		confidence = "medium"
	}

	// Step 8: Build summary
	divergenceText := ""
	if divergence {
		divergenceText = fmt.Sprintf(" BTC +%.1f%% but Ghost Rally flagged.", input.BTCChange24h)
	} else {
		divergenceText = " No divergence."
	}

	summary := fmt.Sprintf("Market breadth %.0f%% (%s): 7d at %.0f%% green, 1h at %.0f%%.%s",
		composite*100, label, timeframePct["7d"]*100, timeframePct["1h"]*100, divergenceText)

	// Step 9: Build result
	greenCounts := make(map[string]int, 4)
	totalCounts := make(map[string]int, 4)
	for _, tf := range timeframeOrder {
		m := input.TimeframeCounts[tf]
		greenCounts[tf] = m.GreenCount
		totalCounts[tf] = m.TotalCount
	}

	return ComputeResult{
		MarketBreadthScore: metrics.Ratio(composite),
		CoinsCounted:       coinsCounted,
		TimeframeBreadth: TimeframeFractions{
			OneHour:    metrics.Ratio(timeframePct["1h"]),
			TwentyFour: metrics.Ratio(timeframePct["24h"]),
			SevenDay:   metrics.Ratio(timeframePct["7d"]),
			ThirtyDay:  metrics.Ratio(timeframePct["30d"]),
		},
		DivergenceDetected: divergence,
		BTCChange24h:       metrics.Currency(input.BTCChange24h),
		Classification: Classification{
			Label:       label,
			Description: classificationDescriptions[label],
		},
		Summary: summary,

		DiscrepancyDetected: discrepancy,
		DiscrepancyNote:     discrepancyNote,
		ValidatorConfidence: confidence,
		MetricStatus:        status,
		WeightsUsed:         effectiveWeights,
		GreenCounts:         greenCounts,
		TotalCounts:         totalCounts,
	}, nil
}

func sign(x float64) int {
	if x > 0 {
		return 1
	}
	if x < 0 {
		return -1
	}
	return 0
}
