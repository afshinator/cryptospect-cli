package v1

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/api"
	"github.com/afshinator/cryptospect-cli/internal/metrics"
)

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
	if len(def.Endpoints) != 2 {
		t.Errorf("Endpoints len = %d, want 2 (primary + validator)", len(def.Endpoints))
	}
}

func TestProvider_Compute_ReturnsUnavailable(t *testing.T) {
	p := &Provider{}
	result, err := p.Compute(context.Background(), nil)
	if err != nil {
		t.Fatalf("Compute returned unexpected error: %v", err)
	}
	if result.Status != "unavailable" {
		t.Errorf("Status = %q, want unavailable", result.Status)
	}
	if result.Metric != MetricName {
		t.Errorf("Metric = %q, want %q", result.Metric, MetricName)
	}
	if result.Version != MetricVersion {
		t.Errorf("Version = %q, want %q", result.Version, MetricVersion)
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

func TestClassification_Labels(t *testing.T) {
	if ClassificationHigh != "high" {
		t.Errorf("ClassificationHigh = %q, want %q", ClassificationHigh, "high")
	}
	if ClassificationNormal != "normal" {
		t.Errorf("ClassificationNormal = %q, want %q", ClassificationNormal, "normal")
	}
	if ClassificationLow != "low" {
		t.Errorf("ClassificationLow = %q, want %q", ClassificationLow, "low")
	}
}

func TestData_Fields(t *testing.T) {
	d := Data{
		VolumeToMcapRatio: 0.123,
		VolumeUSD:         1_000_000_000,
		MarketCapUSD:      8_000_000_000,
		Classification: Classification{
			Label:       ClassificationNormal,
			Description: "Healthy market",
		},
		Summary: "test summary",
	}
	if d.VolumeToMcapRatio != 0.123 {
		t.Errorf("VolumeToMcapRatio = %v, want 0.123", d.VolumeToMcapRatio)
	}
	if d.VolumeUSD != 1_000_000_000 {
		t.Errorf("VolumeUSD = %v, want 1_000_000_000", d.VolumeUSD)
	}
	if d.MarketCapUSD != 8_000_000_000 {
		t.Errorf("MarketCapUSD = %v, want 8_000_000_000", d.MarketCapUSD)
	}
	if d.Classification.Label != ClassificationNormal {
		t.Errorf("Classification.Label = %q, want %q", d.Classification.Label, ClassificationNormal)
	}
	if d.Summary != "test summary" {
		t.Errorf("Summary = %q, want %q", d.Summary, "test summary")
	}
}

func TestCompute_RatioAndClassification(t *testing.T) {
	coingeckoData := api.CoinGeckoGlobalMarket
	coinGeckoResp := json.RawMessage(`{
		"data": {
			"total_volume": {"usd": 1000000000},
			"total_market_cap": {"usd": 8000000000}
		}
	}`)

	dataMap := map[string]json.RawMessage{
		coingeckoData: coinGeckoResp,
	}

	p := &Provider{}
	result, err := p.Compute(context.Background(), dataMap)
	if err != nil {
		t.Fatalf("Compute returned unexpected error: %v", err)
	}
	if result.Status == "unavailable" {
		t.Error("Compute should return data, not unavailable")
	}

	var d Data
	if err := json.Unmarshal(result.Data, &d); err != nil {
		t.Fatalf("failed to unmarshal Data: %v", err)
	}

	expectedRatio := 0.125
	if d.VolumeToMcapRatio != expectedRatio {
		t.Errorf("VolumeToMcapRatio = %v, want %v", d.VolumeToMcapRatio, expectedRatio)
	}
	if d.VolumeUSD != 1_000_000_000 {
		t.Errorf("VolumeUSD = %v, want 1000000000", d.VolumeUSD)
	}
	if d.MarketCapUSD != 8_000_000_000 {
		t.Errorf("MarketCapUSD = %v, want 8000000000", d.MarketCapUSD)
	}
	if d.Classification.Label != ClassificationNormal {
		t.Errorf("Classification.Label = %q, want %q (0.125 is in normal range 0.05-0.15)",
			d.Classification.Label, ClassificationNormal)
	}
}

func TestCompute_ClassificationHigh(t *testing.T) {
	coingeckoData := api.CoinGeckoGlobalMarket
	highVolResp := json.RawMessage(`{
		"data": {
			"total_volume": {"usd": 1600000000},
			"total_market_cap": {"usd": 8000000000}
		}
	}`)

	dataMap := map[string]json.RawMessage{
		coingeckoData: highVolResp,
	}

	p := &Provider{}
	result, err := p.Compute(context.Background(), dataMap)
	if err != nil {
		t.Fatalf("Compute returned error: %v", err)
	}

	var d Data
	if err := json.Unmarshal(result.Data, &d); err != nil {
		t.Fatalf("failed to unmarshal Data: %v", err)
	}

	if d.Classification.Label != ClassificationHigh {
		t.Errorf("Classification.Label = %q, want %q (0.20 >= 0.15 is high)",
			d.Classification.Label, ClassificationHigh)
	}
}

func TestCompute_ClassificationLow(t *testing.T) {
	coingeckoData := api.CoinGeckoGlobalMarket
	lowVolResp := json.RawMessage(`{
		"data": {
			"total_volume": {"usd": 300000000},
			"total_market_cap": {"usd": 8000000000}
		}
	}`)

	dataMap := map[string]json.RawMessage{
		coingeckoData: lowVolResp,
	}

	p := &Provider{}
	result, err := p.Compute(context.Background(), dataMap)
	if err != nil {
		t.Fatalf("Compute returned error: %v", err)
	}

	var d Data
	if err := json.Unmarshal(result.Data, &d); err != nil {
		t.Fatalf("failed to unmarshal Data: %v", err)
	}

	if d.Classification.Label != ClassificationLow {
		t.Errorf("Classification.Label = %q, want %q (0.0375 < 0.05 is low)",
			d.Classification.Label, ClassificationLow)
	}
}

func TestCompute_WithValidator(t *testing.T) {
	coingeckoData := api.CoinGeckoGlobalMarket
	binanceData := api.BinanceSpotCVD_BTC_1h

	coinGeckoResp := json.RawMessage(`{
		"data": {
			"total_volume": {"usd": 1000000000},
			"total_market_cap": {"usd": 8000000000}
		}
	}`)

	binanceResp := json.RawMessage(`[[0, "0", "0", "0", "0", "900000000", "0", "0", "0", "450000000", "0", "0"]]`)

	dataMap := map[string]json.RawMessage{
		coingeckoData: coinGeckoResp,
		binanceData:  binanceResp,
	}

	p := &Provider{}
	result, err := p.Compute(context.Background(), dataMap)
	if err != nil {
		t.Fatalf("Compute returned error: %v", err)
	}

	if result.Meta == nil {
		t.Error("Meta should be present when validator data is provided")
	}
}

func TestCompute_StatusFromConfidence(t *testing.T) {
	coingeckoData := api.CoinGeckoGlobalMarket
	binanceData := api.BinanceSpotCVD_BTC_1h

	coinGeckoResp := json.RawMessage(`{
		"data": {
			"total_volume": {"usd": 1000000000},
			"total_market_cap": {"usd": 8000000000}
		}
	}`)

	largeDiffResp := json.RawMessage(`[[0, "0", "0", "0", "0", "100000000", "0", "0", "0", "50000", "0", "0"]]`) // 90% diff -> low confidence

	dataMap := map[string]json.RawMessage{
		coingeckoData: coinGeckoResp,
		binanceData:  largeDiffResp,
	}

	p := &Provider{}
	result, err := p.Compute(context.Background(), dataMap)
	if err != nil {
		t.Fatalf("Compute returned error: %v", err)
	}

	// With ~90% discrepancy, confidence should be "low" -> status should be "degraded"
	if result.Status == "ok" {
		t.Errorf("Status = %q, want degraded (90%% discrepancy -> low confidence)", result.Status)
	}
}

func TestCompute_FullDetailIncludesThresholds(t *testing.T) {
	coingeckoData := api.CoinGeckoGlobalMarket

	coinGeckoResp := json.RawMessage(`{
		"data": {
			"total_volume": {"usd": 1000000000},
			"total_market_cap": {"usd": 8000000000}
		}
	}`)

	dataMap := map[string]json.RawMessage{
		coingeckoData: coinGeckoResp,
	}

	p := &Provider{}
	result, err := p.Compute(context.Background(), dataMap)
	if err != nil {
		t.Fatalf("Compute returned error: %v", err)
	}

	if result.Meta == nil {
		t.Fatal("Meta should not be nil")
	}

	var metaData map[string]interface{}
	if err := json.Unmarshal(result.Meta, &metaData); err != nil {
		t.Fatalf("failed to unmarshal meta: %v", err)
	}

	if _, hasConfidence := metaData["confidence"]; !hasConfidence {
		t.Error("Meta should include confidence")
	}
}
