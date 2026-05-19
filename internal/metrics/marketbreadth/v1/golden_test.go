package v1

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/afshinator/cryptospect-cli/internal/metrics"
)

// TestGolden_Broad validates the weighted composite breadth score with no timeframe dropping.
//
// Timeframe pcts (green/total, all total=250 > StatisticalFloor=50 → no redistribution):
//
//	1h:  180/250 = 0.72, weight 0.10
//	24h: 160/250 = 0.64, weight 0.30
//	7d:  170/250 = 0.68, weight 0.40
//	30d: 140/250 = 0.56, weight 0.20
//
// Composite = 0.10×0.72 + 0.30×0.64 + 0.40×0.68 + 0.20×0.56
//
//	= 0.072 + 0.192 + 0.272 + 0.112 = 0.648
//
// Classification: 0.648 >= 0.60 (BroadThreshold) → "broad"
// Divergence: BTCChange24h=5.0 > 2.0 but composite=0.648 >= 0.40 (NarrowThreshold) → false
// Discrepancy: KlineClose=68500 > KlineOpen=68100 → dirBN=1, BTCChange=5.0 > 0 → dirCG=1 → match → no discrepancy
func TestGolden_Broad(t *testing.T) {
	input := &Input{
		TimeframeCounts: makeCounts(180, 250, 160, 250, 170, 250, 140, 250),
		CoinsCounted:    250,
		BTCChange24h:    5.0,
		BTCAvailable:    true,
		KlineClose:      68500,
		KlineOpen:       68100,
		KlineOpenTimeMs: time.Now().UnixMilli(),
		KlineAvailable:  true,
		TopN:            250,
		Now:             time.Now(),
	}
	result, err := Compute(input)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	b, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	metrics.AssertMatchesGolden(t, "../../testdata/golden/market-breadth/broad.golden", b)
}

// TestGolden_GhostRally validates divergence detection when BTC rises but breadth is narrow.
//
// Timeframe pcts (green/total, all total=250 > StatisticalFloor=50 → no redistribution):
//
//	1h:  50/250 = 0.20, weight 0.10
//	24h: 40/250 = 0.16, weight 0.30
//	7d:  30/250 = 0.12, weight 0.40
//	30d: 20/250 = 0.08, weight 0.20
//
// Composite = 0.10×0.20 + 0.30×0.16 + 0.40×0.12 + 0.20×0.08
//
//	= 0.020 + 0.048 + 0.048 + 0.016 = 0.132
//
// Classification: 0.132 < 0.40 (NarrowThreshold) → "narrow"
// Divergence: BTCChange24h=5.0 > 2.0 AND composite=0.132 < 0.40 → TRUE (Ghost Rally flagged)
// Summary: "Market breadth 13% (narrow): 7d at 12% green, 1h at 20%. BTC +5.0% but Ghost Rally flagged."
func TestGolden_GhostRally(t *testing.T) {
	input := &Input{
		TimeframeCounts: makeCounts(50, 250, 40, 250, 30, 250, 20, 250),
		CoinsCounted:    250,
		BTCChange24h:    5.0,
		BTCAvailable:    true,
		KlineClose:      68500,
		KlineOpen:       68100,
		KlineOpenTimeMs: time.Now().UnixMilli(),
		KlineAvailable:  true,
		TopN:            250,
		Now:             time.Now(),
	}
	result, err := Compute(input)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	b, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	metrics.AssertMatchesGolden(t, "../../testdata/golden/market-breadth/ghost-rally.golden", b)
}
