package v1

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/api"
	"github.com/afshinator/cryptospect-cli/internal/config"
	"github.com/afshinator/cryptospect-cli/internal/output"
	"github.com/spf13/cobra"
)

const breadthFixture = `[
  {"id":"bitcoin","symbol":"btc","price_change_percentage_1h_in_currency":2.5,"price_change_percentage_24h_in_currency":5.0,"price_change_percentage_7d_in_currency":-0.5,"price_change_percentage_30d_in_currency":5.0},
  {"id":"ethereum","symbol":"eth","price_change_percentage_1h_in_currency":-1.0,"price_change_percentage_24h_in_currency":3.0,"price_change_percentage_7d_in_currency":2.0,"price_change_percentage_30d_in_currency":-2.0},
  {"id":"tether","symbol":"usdt","price_change_percentage_1h_in_currency":0.01,"price_change_percentage_24h_in_currency":-0.02,"price_change_percentage_7d_in_currency":0.0,"price_change_percentage_30d_in_currency":0.01},
  {"id":"solana","symbol":"sol","price_change_percentage_1h_in_currency":5.0,"price_change_percentage_24h_in_currency":-3.0,"price_change_percentage_7d_in_currency":10.0,"price_change_percentage_30d_in_currency":20.0},
  {"id":"cardano","symbol":"ada","price_change_percentage_1h_in_currency":-0.5,"price_change_percentage_24h_in_currency":-2.0,"price_change_percentage_7d_in_currency":-5.0,"price_change_percentage_30d_in_currency":-10.0}
]`

const klinesFixture = `[[1775088000000,"68114.37000000","68304.58000000","68065.69000000","69000.00000000","0.04166000",1775091599999,"2838.36038640",48,"0.01687000","1149.81361760","0"]]`

func TestProvider_Compute_HappyPath(t *testing.T) {
	p := &Provider{}

	data := map[string]json.RawMessage{
		api.CoinGeckoCoinMarketsBreadth: json.RawMessage(breadthFixture),
		api.BinanceSpotCVD_BTC_1h:       json.RawMessage(klinesFixture),
	}

	ctx := context.Background()
	result, err := p.Compute(ctx, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "degraded" {
		t.Errorf("status: got %s, want degraded (5 coins < floor 50)", result.Status)
	}
	if result.Metric != MetricName {
		t.Errorf("metric name: got %s, want %s", result.Metric, MetricName)
	}
	if result.Version != MetricVersion {
		t.Errorf("version: got %s, want %s", result.Version, MetricVersion)
	}

	var dataOut ComputeResult
	if err := json.Unmarshal(result.Data, &dataOut); err != nil {
		t.Fatalf("failed to unmarshal data: %v", err)
	}

	if dataOut.CoinsCounted != 5 {
		t.Errorf("coins counted: got %d, want 5", dataOut.CoinsCounted)
	}
}

func TestProvider_Compute_CGUnavailable(t *testing.T) {
	p := &Provider{}

	data := map[string]json.RawMessage{
		api.CoinGeckoCoinMarketsBreadth: nil,
	}

	ctx := context.Background()
	result, err := p.Compute(ctx, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "unavailable" {
		t.Errorf("status: got %s, want unavailable", result.Status)
	}
}

func TestProvider_Compute_BinanceUnavailable(t *testing.T) {
	p := &Provider{}

	data := map[string]json.RawMessage{
		api.CoinGeckoCoinMarketsBreadth: json.RawMessage(breadthFixture),
	}

	ctx := context.Background()
	result, err := p.Compute(ctx, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "degraded" {
		t.Errorf("status: got %s, want degraded (5 coins < floor 50)", result.Status)
	}

	var meta Meta
	if err := json.Unmarshal(result.Meta, &meta); err != nil {
		t.Fatalf("failed to unmarshal meta: %v", err)
	}
	if meta.Confidence != "low" {
		t.Errorf("confidence: got %s, want low when Binance unavailable", meta.Confidence)
	}
	if !meta.DiscrepancyDetected {
		t.Error("discrepancy should be true when Binance unavailable")
	}
}

func TestProvider_Compute_TopClamping(t *testing.T) {
	p := &Provider{}

	data := map[string]json.RawMessage{
		api.CoinGeckoCoinMarketsBreadth: json.RawMessage(breadthFixture),
		api.BinanceSpotCVD_BTC_1h:       json.RawMessage(klinesFixture),
	}

	ctx := config.StoreTopNInContext(context.Background(), 3)
	result, err := p.Compute(ctx, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var meta Meta
	if err := json.Unmarshal(result.Meta, &meta); err != nil {
		t.Fatalf("failed to unmarshal meta: %v", err)
	}
	if !meta.TopClamped {
		t.Error("top should be clamped when --top 3 passed (below min 50)")
	}
}

func TestProvider_RegisterFlags(t *testing.T) {
	p := &Provider{}
	cmd := &cobra.Command{}
	p.RegisterFlags(cmd)

	f := cmd.Flags().Lookup("top")
	if f == nil {
		t.Fatal("--top flag not registered by RegisterFlags")
	}
	if f.DefValue != "250" {
		t.Errorf("--top default = %q, want 250", f.DefValue)
	}
}

func TestProvider_Compute_OutputEnvelope(t *testing.T) {
	p := &Provider{}

	data := map[string]json.RawMessage{
		api.CoinGeckoCoinMarketsBreadth: json.RawMessage(breadthFixture),
		api.BinanceSpotCVD_BTC_1h:       json.RawMessage(klinesFixture),
	}

	ctx := context.Background()
	result, err := p.Compute(ctx, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var envelope output.CLIResponse
	envelope.Status = "ok"
	envelope.Results = []output.MetricResult{result}

	raw, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("failed to marshal envelope: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("failed to unmarshal envelope: %v", err)
	}

	if parsed["status"] != "ok" {
		t.Errorf("envelope status: got %v", parsed["status"])
	}
	results, ok := parsed["results"].([]interface{})
	if !ok || len(results) != 1 {
		t.Fatal("expected 1 result in envelope")
	}
}
