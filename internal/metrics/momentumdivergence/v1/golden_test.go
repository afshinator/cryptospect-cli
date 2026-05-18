package v1

import (
	"encoding/json"
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/api/coingecko"
	"github.com/afshinator/cryptospect-cli/internal/metrics"
)

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
