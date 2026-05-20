package v1

import (
	"context"
	"encoding/json"
	"math"
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/api"
	"github.com/afshinator/cryptospect-cli/internal/metrics"
)

// ── Provider identity ──

func TestProvider_Def(t *testing.T) {
	p := &Provider{}
	def := p.Def()

	if def.Name != MetricName {
		t.Errorf("Name = %q, want %q", def.Name, MetricName)
	}
	if def.Version != MetricVersion {
		t.Errorf("Version = %q, want %q", def.Version, MetricVersion)
	}
	if len(def.Aliases) == 0 {
		t.Error("Aliases must not be empty")
	}
	if len(def.Endpoints) == 0 {
		t.Error("Endpoints must not be empty")
	}
}

func TestProvider_RegisteredOnInit(t *testing.T) {
	reg := metrics.GlobalRegistry()
	p, err := reg.Resolve(MetricName)
	if err != nil {
		t.Fatalf("provider not in global registry: %v", err)
	}
	if p.Def().Name != MetricName {
		t.Errorf("Name = %q, want %q", p.Def().Name, MetricName)
	}
}

// ── Endpoint contract ──

func TestProvider_Compute_ReturnsUnavailable_WhenMissingData(t *testing.T) {
	p := &Provider{}
	result, err := p.Compute(context.Background(), nil)
	if err != nil {
		t.Fatalf("Compute returned unexpected error: %v", err)
	}
	if result.Status != "unavailable" {
		t.Errorf("Status = %q, want unavailable", result.Status)
	}
}

func TestProvider_Compute_ReturnsUnavailable_WhenEmptyResponse(t *testing.T) {
	p := &Provider{}
	result, err := p.Compute(context.Background(), map[string]json.RawMessage{
		api.CoinGeckoGlobalMarket: json.RawMessage(``),
	})
	if err != nil {
		t.Fatalf("Compute returned unexpected error: %v", err)
	}
	if result.Status != "unavailable" {
		t.Errorf("Status = %q, want unavailable", result.Status)
	}
}

// ── Compute: happy path (btc_rising) ──

func TestProvider_Compute_Happy_BTCRising(t *testing.T) {
	p := &Provider{}
	// Simulate a scenario where the state cache has prior values
	// BTC was 51.0%, now 52.3% → +1.3pp → "rising"
	// ETH was 18.7%, now 18.1% → -0.6pp → "falling"
	// Label: btc_rising
	result, err := p.Compute(context.Background(), map[string]json.RawMessage{
		api.CoinGeckoGlobalMarket: json.RawMessage(`{
			"data": {
				"market_cap_percentage": {
					"btc": 52.3,
					"eth": 18.1
				}
			}
		}`),
	})
	if err != nil {
		t.Fatalf("Compute returned unexpected error: %v", err)
	}

	if result.Status == "unavailable" {
		t.Fatal("Compute should return data, not unavailable (cold start is valid output)")
	}

	var d Data
	if err := json.Unmarshal(result.Data, &d); err != nil {
		t.Fatalf("failed to unmarshal Data: %v", err)
	}

	// On cold start, trends are neutral, deltas are null
	if d.BTC.Trend != TrendNeutral {
		t.Errorf("BTC.Trend = %q, want %q (cold start)", d.BTC.Trend, TrendNeutral)
	}
	if d.ETH.Trend != TrendNeutral {
		t.Errorf("ETH.Trend = %q, want %q (cold start)", d.ETH.Trend, TrendNeutral)
	}
	if d.BTC.DeltaPP != nil {
		t.Errorf("BTC.DeltaPP should be nil on cold start, got %v", *d.BTC.DeltaPP)
	}
	if d.Classification.Label != LabelNeutral {
		t.Errorf("Classification.Label = %q, want %q (cold start)", d.Classification.Label, LabelNeutral)
	}
}

// ── Compute: edge cases ──

func TestCompute_Happy_BTCRising(t *testing.T) {
	in := Input{
		BTCDominancePct:     52.3,
		ETHDominancePct:     18.1,
		PriorBTCDominance:   ptr(51.0),
		PriorETHDominance:   ptr(18.7),
		PriorSnapshotAgeSec: ptr(3600),
	}

	data, meta, err := Compute(in)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}

	if meta.ColdStart {
		t.Error("ColdStart should be false with valid prior data")
	}
	if data.BTC.Trend != TrendRising {
		t.Errorf("BTC.Trend = %q, want %q", data.BTC.Trend, TrendRising)
	}
	if data.ETH.Trend != TrendFalling {
		t.Errorf("ETH.Trend = %q, want %q (delta -0.6 < -0.3)", data.ETH.Trend, TrendFalling)
	}
	if data.Classification.Label != LabelBTCRising {
		t.Errorf("Classification.Label = %q, want %q", data.Classification.Label, LabelBTCRising)
	}
	if data.BTC.DeltaPP == nil {
		t.Fatal("BTC.DeltaPP should not be nil")
	}
	if math.Abs(data.BTC.DeltaPP.Value()-1.3) > 1e-9 {
		t.Errorf("BTC.DeltaPP = %v, want ~1.3", data.BTC.DeltaPP.Value())
	}
	if data.ETHBTCRatio.Value() < 0.34 || data.ETHBTCRatio.Value() > 0.35 {
		t.Errorf("ETHBTCRatio = %v, want ~0.346", data.ETHBTCRatio.Value())
	}
}

func TestCompute_Happy_ETHRising(t *testing.T) {
	in := Input{
		BTCDominancePct:     50.0,
		ETHDominancePct:     20.0,
		PriorBTCDominance:   ptr(51.0),
		PriorETHDominance:   ptr(18.5),
		PriorSnapshotAgeSec: ptr(7200),
	}

	data, _, err := Compute(in)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}

	if data.BTC.Trend != TrendFalling {
		t.Errorf("BTC.Trend = %q, want %q", data.BTC.Trend, TrendFalling)
	}
	if data.ETH.Trend != TrendRising {
		t.Errorf("ETH.Trend = %q, want %q", data.ETH.Trend, TrendRising)
	}
	if data.Classification.Label != LabelETHRising {
		t.Errorf("Classification.Label = %q, want %q", data.Classification.Label, LabelETHRising)
	}
}

func TestCompute_Neutral_WithinDeadBand(t *testing.T) {
	in := Input{
		BTCDominancePct:     52.1,
		ETHDominancePct:     18.0,
		PriorBTCDominance:   ptr(52.0),
		PriorETHDominance:   ptr(18.1),
		PriorSnapshotAgeSec: ptr(100),
	}

	data, _, err := Compute(in)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}

	if data.BTC.Trend != TrendNeutral {
		t.Errorf("BTC.Trend = %q, want %q (delta +0.1 < dead band 0.5)", data.BTC.Trend, TrendNeutral)
	}
	if data.ETH.Trend != TrendNeutral {
		t.Errorf("ETH.Trend = %q, want %q (delta -0.1 < dead band 0.3)", data.ETH.Trend, TrendNeutral)
	}
	if data.Classification.Label != LabelNeutral {
		t.Errorf("Classification.Label = %q, want %q", data.Classification.Label, LabelNeutral)
	}
}

func TestCompute_BothFalling(t *testing.T) {
	in := Input{
		BTCDominancePct:     47.0,
		ETHDominancePct:     15.0,
		PriorBTCDominance:   ptr(48.5),
		PriorETHDominance:   ptr(16.0),
		PriorSnapshotAgeSec: ptr(500),
	}

	data, _, err := Compute(in)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}

	if data.BTC.Trend != TrendFalling {
		t.Errorf("BTC.Trend = %q, want %q", data.BTC.Trend, TrendFalling)
	}
	if data.ETH.Trend != TrendFalling {
		t.Errorf("ETH.Trend = %q, want %q", data.ETH.Trend, TrendFalling)
	}
	if data.Classification.Label != LabelCapitalContracting {
		t.Errorf("Classification.Label = %q, want %q", data.Classification.Label, LabelCapitalContracting)
	}
}

func TestCompute_ColdStart(t *testing.T) {
	in := Input{
		BTCDominancePct: 52.3,
		ETHDominancePct: 18.1,
		// No prior data
	}

	data, meta, err := Compute(in)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}

	if !meta.ColdStart {
		t.Error("ColdStart should be true without prior data")
	}
	if data.BTC.DeltaPP != nil {
		t.Error("BTC.DeltaPP should be nil on cold start")
	}
	if data.BTC.Trend != TrendNeutral {
		t.Errorf("BTC.Trend = %q, want %q", data.BTC.Trend, TrendNeutral)
	}
}

func TestCompute_StaleSnapshot(t *testing.T) {
	// Prior snapshot older than MaxSnapshotAgeSec — should treat as cold start
	in := Input{
		BTCDominancePct:     52.3,
		ETHDominancePct:     18.1,
		PriorBTCDominance:   ptr(51.0),
		PriorETHDominance:   ptr(18.7),
		PriorSnapshotAgeSec: ptr(MaxSnapshotAgeSec + 1), // too old
	}

	_, meta, err := Compute(in)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}

	if !meta.ColdStart {
		t.Error("ColdStart should be true when prior snapshot is stale")
	}
}

func TestCompute_ETHBTCRatio(t *testing.T) {
	in := Input{
		BTCDominancePct:     60.0,
		ETHDominancePct:     18.0,
		PriorBTCDominance:   ptr(59.0),
		PriorETHDominance:   ptr(19.0),
		PriorSnapshotAgeSec: ptr(3600),
	}

	data, _, err := Compute(in)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}

	expected := 18.0 / 60.0 // 0.3
	if data.ETHBTCRatio.Value() != expected {
		t.Errorf("ETHBTCRatio = %v, want %v", data.ETHBTCRatio.Value(), expected)
	}
}

// ── Helpers ──

func ptr[T any](v T) *T { return &v }
