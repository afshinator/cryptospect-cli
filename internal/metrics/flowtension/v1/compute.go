package v1

import (
	"fmt"

	"github.com/afshinator/cryptospect-cli/internal/metrics"
)

// Compute is the single entry point for the flow-tension metric calculation.
// It accepts an Input struct (rather than scalar args) because OI and funding
// values originate from a different source (CoinGecko) than CVD (Binance).
//
// Returns an error only for fundamentally invalid input (zero total volume).
// Missing OI/Funding data (zero/empty inputs) produces conservative hooks
// rather than an error.
func Compute(input Input) (Data, error) {
	if input.TotalVolume == 0 {
		return Data{}, fmt.Errorf("total volume must be non-zero")
	}

	// ── CVD signal ──
	cvdRatio := ComputeExchangeNetFlow(input.TakerBuyVolume, input.TakerSellVolume, input.TotalVolume)
	flowHook := ComputeFlowHook(cvdRatio)
	if input.NumTrades < MinTrades {
		flowHook = HookLowConfidence
	}

	// ── OI signal ──
	var oiChange *float64
	oiHook := HookOIStable
	if input.TotalOpenInterest > 0 && input.PrevOI != nil && *input.PrevOI > 0 {
		change := ComputeOIChange24h(input.TotalOpenInterest, *input.PrevOI)
		oiChange = &change
		oiHook = ComputeOIChangeHook(change)
	} else if input.TotalOpenInterest > 0 && input.PrevOI != nil {
		// PrevOI is 0 — clear division would give 0, default to stable
		oiHook = HookOIStable
	}

	// ── Funding signal ──
	fundingHook := ComputeFundingRateHook(input.FundingRate)

	// ── Summary ──
	summaryStr := ComputeSummary(fundingHook, oiHook, flowHook)

	// ── Build Data ──
	oiSignal := SignalOI{
		CurrentUSD:    metrics.Currency(input.TotalOpenInterest),
		Hook:          oiHook,
		ExchangeCount: input.ExchangeCount,
	}
	if oiChange != nil {
		ch := metrics.Ratio(*oiChange)
		oiSignal.ChangePct24h = &ch
	}

	return Data{
		Signals: Signals{
			CVD: SignalCVD{
				Ratio: metrics.Ratio(cvdRatio),
				Hook:  flowHook,
			},
			OpenInterest: oiSignal,
			FundingRate: SignalFR{
				Rate: metrics.Ratio(input.FundingRate),
				Hook: fundingHook,
			},
		},
		Summary: summaryStr,
	}, nil
}

// ComputeExchangeNetFlow derives the CVD proxy as a fraction of total volume:
// (takerBuyVol - takerSellVol) / totalVolume.
// Result is in [-1, 1]: positive = buyer aggression, negative = seller aggression.
// Returns 0 if totalVolume is zero.
func ComputeExchangeNetFlow(takerBuyVol, takerSellVol, totalVolume float64) float64 {
	if totalVolume == 0 {
		return 0
	}
	return (takerBuyVol - takerSellVol) / totalVolume
}

// ComputeFlowHook classifies the Exchange Net Flow value into a narrative label.
// Does NOT apply the thin-candle guard — that is handled in Compute().
func ComputeFlowHook(netFlow float64) string {
	if netFlow >= FlowAggressiveThreshold {
		return HookAggressiveBuy
	}
	if netFlow <= -FlowAggressiveThreshold {
		return HookAggressiveSell
	}
	return HookNeutral
}

// ComputeFundingRateHook classifies the perp funding rate into a narrative label.
// Calibrated for the 8-hour BTC perpetual cycle.
func ComputeFundingRateHook(rate float64) string {
	if rate <= FundingNegativeThreshold {
		return HookNegative
	}
	if rate > FundingOverheatedThreshold {
		return HookOverheated
	}
	if rate >= FundingPositiveThreshold {
		return HookPositive
	}
	return HookNeutralFR
}

// ComputeOIChange24h computes the 24h percentage change in Open Interest as a fraction.
// Returns 0 if prevOI is zero.
func ComputeOIChange24h(currentOI, prevOI float64) float64 {
	if prevOI == 0 {
		return 0
	}
	return (currentOI - prevOI) / prevOI
}

// ComputeOIChangeHook classifies the 24h OI change fraction into a narrative label.
func ComputeOIChangeHook(change float64) string {
	if change >= OIBuildingThreshold {
		return HookOIBuilding
	}
	if change <= OIUnwindingThreshold {
		return HookOIUnwinding
	}
	return HookOIStable
}

// ComputeSummary produces a one-line narrative from the three signal hooks.
func ComputeSummary(fundingHook, oiHook, flowHook string) string {
	// Low-confidence candle — not enough trades to trust the CVD signal
	if flowHook == HookLowConfidence {
		return "Thin candle — insufficient trades for reliable CVD signal. Flow direction unreliable."
	}

	// Named verdict combinations
	switch {
	case fundingHook == HookNegative && oiHook == HookOIBuilding:
		return "Shorts paying longs while leverage builds — early bull phase, sellers exhausted."
	case fundingHook == HookOverheated && oiHook == HookOIBuilding:
		return "Leverage building with overheated longs — elevated liquidation risk."
	case oiHook == HookOIBuilding && flowHook == HookAggressiveBuy:
		return "Leverage building with aggressive buying — tension coiling, breakout likely."
	case flowHook == HookAggressiveSell && oiHook == HookOIStable:
		return "Assets staged on exchanges with aggressive selling — supply shock / top warning."
	case oiHook == HookOIUnwinding:
		return "Leverage unwinding — deleveraging in progress, likely post-liquidation."
	case fundingHook == HookNeutralFR && oiHook == HookOIStable && flowHook == HookNeutral:
		return "Flow tension neutral — no directional conviction."
	default:
		return "Mixed signals — monitor for confirmation."
	}
}
