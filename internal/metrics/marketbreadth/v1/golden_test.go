package v1

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/afshinator/cryptospect-cli/internal/metrics"
)

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
