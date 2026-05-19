package v1

import (
	"encoding/json"
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/api/coingecko"
	"github.com/afshinator/cryptospect-cli/internal/metrics"
)

// TestGolden_RiskOn validates the Risk-On rotation classification.
//
// All coins have equal market cap (100), so weighted avg = simple mean.
//
// Large (rank 1-10): btc=2.0, eth=3.0, usdt=0.01, sol=1.5, bnb=2.5
//
//	avg = (2.0+3.0+0.01+1.5+2.5)/5 = 9.01/5 = 1.802
//
// Mid (rank 11-50): link=8.0, matic=7.0, avax=9.0
//
//	avg = (8.0+7.0+9.0)/3 = 24/3 = 8.0
//
// Small (rank 51-200): gmx=12.0, dydx=10.0, inj=14.0
//
//	avg = (12.0+10.0+14.0)/3 = 36/3 = 12.0
//
// Spreads: mid_vs_large=8.0-1.802=6.198, small_vs_large=12.0-1.802=10.198, small_vs_mid=12.0-8.0=4.0
//
// Classification: mid_vs_large=6.198 > 5.0 (RiskOnSpread) && mid_avg=8.0 > 1.0 (MinPositivityGuard) → "risk_on"
// TailExtension: small_vs_large=10.198 > 5.0 (TailExtensionSpread) → true
func TestGolden_RiskOn(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "bitcoin", 2.0, 100),
		c(2, "ethereum", 3.0, 100),
		c(3, "tether", 0.01, 100),
		c(4, "solana", 1.5, 100),
		c(5, "bnb", 2.5, 100),
		c(11, "chainlink", 8.0, 100),
		c(12, "polygon", 7.0, 100),
		c(13, "avalanche", 9.0, 100),
		c(51, "gmx", 12.0, 100),
		c(52, "dydx", 10.0, 100),
		c(53, "inj", 14.0, 100),
	}
	input := Input{
		Coins:        coins,
		LargeCeiling: 10,
		MidCeiling:   50,
		SmallCeiling: 200,
	}
	data, _, err := Compute(input)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	metrics.AssertMatchesGolden(t, "../../testdata/golden/momentum-divergence/risk-on.golden", b)
}

// TestGolden_FlightToSafety validates the Flight-to-Safety classification.
//
// All coins have equal market cap (100), so weighted avg = simple mean.
//
// Large (rank 1-10): btc=-2.0, eth=-1.5, usdt=0.01, sol=-3.0, bnb=-1.0
//
//	avg = (-2.0 + -1.5 + 0.01 + -3.0 + -1.0)/5 = -7.49/5 = -1.498
//
// Mid (rank 11-50): link=-10.0, matic=-9.0, avax=-12.0
//
//	avg = (-10.0 + -9.0 + -12.0)/3 = -31/3 = -10.3333…
//
// Small (rank 51-200): gmx=-20.0, dydx=-18.0, inj=-22.0
//
//	avg = (-20.0 + -18.0 + -22.0)/3 = -60/3 = -20.0
//
// Spreads: mid_vs_large=-10.3333-(-1.498)=-8.8353, small_vs_large=-20.0-(-1.498)=-18.502, small_vs_mid=-20.0-(-10.3333)=-9.6667
//
// Classification: mid_vs_large=-8.8353 < -3.0 (TopHeavySpread), large_avg=-1.498 <= -0.5 (ConcentrationDeadBand) → "flight_to_safety"
// TailExtension: small_vs_large=-18.502 > 5.0? No → false
func TestGolden_FlightToSafety(t *testing.T) {
	coins := []coingecko.CoinMarketsRankedCoin{
		c(1, "bitcoin", -2.0, 100),
		c(2, "ethereum", -1.5, 100),
		c(3, "tether", 0.01, 100),
		c(4, "solana", -3.0, 100),
		c(5, "bnb", -1.0, 100),
		c(11, "chainlink", -10.0, 100),
		c(12, "polygon", -9.0, 100),
		c(13, "avalanche", -12.0, 100),
		c(51, "gmx", -20.0, 100),
		c(52, "dydx", -18.0, 100),
		c(53, "inj", -22.0, 100),
	}
	input := Input{
		Coins:        coins,
		LargeCeiling: 10,
		MidCeiling:   50,
		SmallCeiling: 200,
	}
	data, _, err := Compute(input)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	metrics.AssertMatchesGolden(t, "../../testdata/golden/momentum-divergence/flight-to-safety.golden", b)
}
