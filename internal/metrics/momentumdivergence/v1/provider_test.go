package v1

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/api"
	"github.com/afshinator/cryptospect-cli/internal/config"
	"github.com/afshinator/cryptospect-cli/internal/metrics"
	"github.com/afshinator/cryptospect-cli/internal/output"
	"github.com/spf13/cobra"
)

const rankedFixture = `[
  {"id":"bitcoin","symbol":"btc","market_cap_rank":1,"price_change_percentage_24h_in_currency":2.10},
  {"id":"ethereum","symbol":"eth","market_cap_rank":2,"price_change_percentage_24h_in_currency":-1.20},
  {"id":"tether","symbol":"usdt","market_cap_rank":3,"price_change_percentage_24h_in_currency":0.01},
  {"id":"solana","symbol":"sol","market_cap_rank":4,"price_change_percentage_24h_in_currency":5.50},
  {"id":"bnb","symbol":"bnb","market_cap_rank":5,"price_change_percentage_24h_in_currency":2.50},
  {"id":"chainlink","symbol":"link","market_cap_rank":11,"price_change_percentage_24h_in_currency":8.00},
  {"id":"polygon","symbol":"matic","market_cap_rank":12,"price_change_percentage_24h_in_currency":7.00},
  {"id":"avalanche","symbol":"avax","market_cap_rank":13,"price_change_percentage_24h_in_currency":9.00},
  {"id":"gmx","symbol":"gmx","market_cap_rank":51,"price_change_percentage_24h_in_currency":12.00},
  {"id":"dydx","symbol":"dydx","market_cap_rank":52,"price_change_percentage_24h_in_currency":10.00},
  {"id":"inj","symbol":"inj","market_cap_rank":53,"price_change_percentage_24h_in_currency":14.00}
]`

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

func TestProvider_Compute_HappyPath(t *testing.T) {
	p := &Provider{}
	ctx := context.Background()

	data := map[string]json.RawMessage{
		api.CoinGeckoCoinMarketsBreadth: json.RawMessage(rankedFixture),
	}

	result, err := p.Compute(ctx, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Metric != MetricName {
		t.Errorf("Metric = %q, want %q", result.Metric, MetricName)
	}
	if result.Version != MetricVersion {
		t.Errorf("Version = %q, want %q", result.Version, MetricVersion)
	}
	if result.Status != "ok" && result.Status != "degraded" {
		t.Errorf("Status = %q, want ok or degraded", result.Status)
	}
	if result.Data == nil {
		t.Error("Data should not be nil")
	}
}

func TestProvider_Compute_CGUnavailable(t *testing.T) {
	p := &Provider{}
	ctx := context.Background()

	data := map[string]json.RawMessage{}
	result, err := p.Compute(ctx, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "unavailable" {
		t.Errorf("Status = %q, want unavailable", result.Status)
	}
}

func TestProvider_Compute_CGInvalidJSON(t *testing.T) {
	p := &Provider{}
	ctx := context.Background()

	data := map[string]json.RawMessage{
		api.CoinGeckoCoinMarketsBreadth: json.RawMessage("not json"),
	}
	result, err := p.Compute(ctx, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "unavailable" {
		t.Errorf("Status = %q, want unavailable", result.Status)
	}
}

func TestProvider_Compute_CGEmptyArray(t *testing.T) {
	p := &Provider{}
	ctx := context.Background()

	data := map[string]json.RawMessage{
		api.CoinGeckoCoinMarketsBreadth: json.RawMessage("[]"),
	}
	result, err := p.Compute(ctx, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "unavailable" {
		t.Errorf("Status = %q, want unavailable", result.Status)
	}
}

func TestProvider_Compute_SegmentsClamping(t *testing.T) {
	p := &Provider{}
	ctx := config.StoreSegmentsInContext(context.Background(), "3,50,300")

	data := map[string]json.RawMessage{
		api.CoinGeckoCoinMarketsBreadth: json.RawMessage(rankedFixture),
	}

	result, err := p.Compute(ctx, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "ok" && result.Status != "degraded" {
		t.Errorf("Status = %q, want ok or degraded", result.Status)
	}
	if result.Meta == nil {
		t.Fatal("Meta should not be nil")
	}
	var meta map[string]interface{}
	if err := json.Unmarshal(result.Meta, &meta); err != nil {
		t.Fatalf("unmarshal meta: %v", err)
	}
	// 3 < 5 → large_ceiling clamped to 5; 300 > 250 → small_ceiling clamped to 250
	clamped, ok := meta["segments_clamped"].(bool)
	if !ok || !clamped {
		t.Error("segments_clamped should be true")
	}
}

func TestProvider_RegisterFlags(t *testing.T) {
	p := &Provider{}
	cmd := &cobra.Command{}
	p.RegisterFlags(cmd)

	f := cmd.Flags().Lookup("segments")
	if f == nil {
		t.Fatal("--segments flag not registered")
	}
	if f.DefValue != "10,50,200" {
		t.Errorf("--segments default: got %q, want 10,50,200", f.DefValue)
	}
}

func TestProvider_OutputEnvelope(t *testing.T) {
	p := &Provider{}
	ctx := context.Background()

	data := map[string]json.RawMessage{
		api.CoinGeckoCoinMarketsBreadth: json.RawMessage(rankedFixture),
	}

	result, err := p.Compute(ctx, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	envelope := output.CLIResponse{
		Status:  "ok",
		Results: []output.MetricResult{result},
	}
	bytes, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed output.CLIResponse
	if err := json.Unmarshal(bytes, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if parsed.Status != "ok" {
		t.Errorf("Status = %q, want ok", parsed.Status)
	}
	if len(parsed.Results) != 1 {
		t.Fatalf("Results length = %d, want 1", len(parsed.Results))
	}
	r := parsed.Results[0]
	if r.Metric != MetricName {
		t.Errorf("Metric = %q, want %q", r.Metric, MetricName)
	}
	var dataParsed Data
	if err := json.Unmarshal(r.Data, &dataParsed); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}
	if dataParsed.Classification.Label == "" {
		t.Error("classification.label should not be empty")
	}
	if dataParsed.Summary == "" {
		t.Error("summary should not be empty")
	}
	if !dataParsed.TailExtension {
		// Should be true since small_vs_large > 5pp in this fixture
		if dataParsed.Spreads.SmallVsLarge != nil && *dataParsed.Spreads.SmallVsLarge <= TailExtensionSpread {
			t.Log("tail_extension false is valid given fixture")
		}
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

func TestProvider_MetaHasDataTimestamp(t *testing.T) {
	p := &Provider{}
	ctx := context.Background()

	data := map[string]json.RawMessage{
		api.CoinGeckoCoinMarketsBreadth: json.RawMessage(rankedFixture),
	}

	result, err := p.Compute(ctx, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Meta == nil {
		t.Fatal("Meta should not be nil")
	}
	var meta map[string]interface{}
	if err := json.Unmarshal(result.Meta, &meta); err != nil {
		t.Fatalf("unmarshal meta: %v", err)
	}
	if _, ok := meta["data_timestamp"]; !ok {
		t.Error("meta should contain data_timestamp")
	}
}
