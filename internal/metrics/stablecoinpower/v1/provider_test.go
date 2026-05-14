package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/afshinator/cryptospect-cli/internal/api"
	"github.com/afshinator/cryptospect-cli/internal/config"
	"github.com/afshinator/cryptospect-cli/internal/metrics"
)

// ----- JSON fixtures -----

func cgGlobalJSON(totalMcap float64) json.RawMessage {
	type inner struct {
		TotalMarketCap map[string]float64 `json:"total_market_cap"`
		TotalVolume    map[string]float64 `json:"total_volume"`
	}
	type outer struct {
		Data inner `json:"data"`
	}
	b, _ := json.Marshal(outer{Data: inner{
		TotalMarketCap: map[string]float64{"usd": totalMcap},
		TotalVolume:    map[string]float64{"usd": 1e11},
	}})
	return b
}

func cgStablesJSON(n int, mcapEach float64) json.RawMessage {
	type entry struct {
		ID        string  `json:"id"`
		Symbol    string  `json:"symbol"`
		MarketCap float64 `json:"market_cap"`
	}
	entries := make([]entry, n)
	for i := range entries {
		entries[i] = entry{
			ID:        fmt.Sprintf("stable-%d", i),
			Symbol:    fmt.Sprintf("st%d", i),
			MarketCap: mcapEach,
		}
	}
	b, _ := json.Marshal(entries)
	return b
}

func dlJSON(current, prevWeek float64) json.RawMessage {
	type circ struct {
		PeggedUSD float64 `json:"peggedUSD"`
	}
	type asset struct {
		Circulating         circ `json:"circulating"`
		CirculatingPrevWeek circ `json:"circulatingPrevWeek"`
	}
	type resp struct {
		PeggedAssets []asset `json:"peggedAssets"`
	}
	b, _ := json.Marshal(resp{PeggedAssets: []asset{{
		Circulating:         circ{PeggedUSD: current},
		CirculatingPrevWeek: circ{PeggedUSD: prevWeek},
	}}})
	return b
}

func dataMap3(global, stables, dl json.RawMessage) map[string]json.RawMessage {
	return map[string]json.RawMessage{
		api.CoinGeckoGlobalMarket:      global,
		api.CoinGeckoSPPStablesMarkets: stables,
		api.DefiLlamaStablecoins:       dl,
	}
}

// ----- Def / registry -----

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
	if len(def.Endpoints) != 3 {
		t.Errorf("Endpoints len = %d, want 3 (global + stables + defillama)", len(def.Endpoints))
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

func TestProvider_RegisterFlags(t *testing.T) {
	p := &Provider{}
	cmd := &cobra.Command{}
	p.RegisterFlags(cmd)

	f := cmd.Flags().Lookup("top")
	if f == nil {
		t.Fatal("--top flag not registered by RegisterFlags")
	}
	if f.DefValue != "8" {
		t.Errorf("--top default = %q, want 8", f.DefValue)
	}
}

// ----- classify -----

func TestClassify(t *testing.T) {
	tests := []struct {
		name      string
		ratio     float64
		trend     string
		wantLabel string
		wantDesc  string
	}{
		{"high", 0.20, "stable", ClassificationHigh, "Dry Powder Alert"},
		{"high boundary", 0.15, "stable", ClassificationHigh, "Dry Powder Alert"},
		{"normal", 0.10, "stable", ClassificationNormal, "Healthy Balance"},
		{"normal lower boundary", 0.07, "stable", ClassificationNormal, "Healthy Balance"},
		{"low + stable trend → overextended", 0.05, "stable", ClassificationLow, "Overextended"},
		{"low + expanding trend → overextended", 0.05, "expanding", ClassificationLow, "Overextended"},
		{"low + contracting → capital flight", 0.05, "contracting", ClassificationLow, "Capital Flight"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			cl := classify(tc.ratio, tc.trend)
			if cl.Label != tc.wantLabel {
				t.Errorf("Label = %q, want %q", cl.Label, tc.wantLabel)
			}
			if cl.Description != tc.wantDesc {
				t.Errorf("Description = %q, want %q", cl.Description, tc.wantDesc)
			}
		})
	}
}

// ----- Compute: failure paths -----

func TestCompute_GlobalMissing_Unavailable(t *testing.T) {
	p := &Provider{}
	dm := map[string]json.RawMessage{
		api.CoinGeckoSPPStablesMarkets: cgStablesJSON(8, 27e9),
	}
	result, err := p.Compute(context.Background(), dm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "unavailable" {
		t.Errorf("Status = %q, want unavailable", result.Status)
	}
}

func TestCompute_StablesMissing_Unavailable(t *testing.T) {
	p := &Provider{}
	dm := map[string]json.RawMessage{
		api.CoinGeckoGlobalMarket: cgGlobalJSON(3e12),
	}
	result, err := p.Compute(context.Background(), dm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "unavailable" {
		t.Errorf("Status = %q, want unavailable", result.Status)
	}
}

func TestCompute_NilData_Unavailable(t *testing.T) {
	p := &Provider{}
	result, err := p.Compute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "unavailable" {
		t.Errorf("Status = %q, want unavailable", result.Status)
	}
}

// ----- Compute: happy paths -----

func TestCompute_HappyPath_Normal(t *testing.T) {
	// 8 stables @ 27e9 → stable = 216e9, volatile = 2.784e12, ratio ≈ 0.0776 → normal
	dm := dataMap3(
		cgGlobalJSON(3e12),
		cgStablesJSON(8, 27e9),
		dlJSON(216e9, 215e9), // 0% discrepancy, stable trend
	)
	p := &Provider{}
	result, err := p.Compute(context.Background(), dm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "ok" {
		t.Errorf("Status = %q, want ok", result.Status)
	}

	var d Data
	if err := json.Unmarshal(result.Data, &d); err != nil {
		t.Fatalf("unmarshal Data: %v", err)
	}
	if d.Classification.Label != ClassificationNormal {
		t.Errorf("Label = %q, want normal (ratio ≈ 0.0776)", d.Classification.Label)
	}
	if d.StablePowerRatio.Value() <= 0 {
		t.Error("StablePowerRatio must be positive")
	}
	if d.StablecoinsCounted != 8 {
		t.Errorf("StablecoinsCounted = %d, want 8", d.StablecoinsCounted)
	}
	if d.SupplyTrend7d == "" {
		t.Error("SupplyTrend7d must not be empty")
	}
	if d.Summary == "" {
		t.Error("Summary must not be empty")
	}

	if result.Meta == nil {
		t.Fatal("Meta must not be nil")
	}
	var m Meta
	if err := json.Unmarshal(result.Meta, &m); err != nil {
		t.Fatalf("unmarshal Meta: %v", err)
	}
	if m.Confidence != "high" {
		t.Errorf("Confidence = %q, want high", m.Confidence)
	}
	if m.DiscrepancyDetected {
		t.Error("DiscrepancyDetected should be false for 0% discrepancy")
	}
	if m.PrimarySource != "coingecko" {
		t.Errorf("PrimarySource = %q, want coingecko", m.PrimarySource)
	}
	if m.ValidatorSource != "defillama" {
		t.Errorf("ValidatorSource = %q, want defillama", m.ValidatorSource)
	}
	if m.SupplyTrendSource != "defillama" {
		t.Errorf("SupplyTrendSource = %q, want defillama", m.SupplyTrendSource)
	}
	if m.StablecoinScope != "all_usd_equivalent" {
		t.Errorf("StablecoinScope = %q, want all_usd_equivalent", m.StablecoinScope)
	}
	// full-detail fields always computed internally
	if m.Thresholds == nil {
		t.Error("Thresholds must not be nil in Compute output")
	}
	if len(m.TopNStablecoins) != 8 {
		t.Errorf("TopNStablecoins len = %d, want 8", len(m.TopNStablecoins))
	}
}

func TestCompute_HighClassification(t *testing.T) {
	// 8 stables @ 80e9 → stable = 640e9, volatile = 2.36e12, ratio ≈ 0.271 → high
	dm := dataMap3(
		cgGlobalJSON(3e12),
		cgStablesJSON(8, 80e9),
		dlJSON(640e9, 638e9),
	)
	p := &Provider{}
	result, err := p.Compute(context.Background(), dm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var d Data
	if err := json.Unmarshal(result.Data, &d); err != nil {
		t.Fatalf("unmarshal Data: %v", err)
	}
	if d.Classification.Label != ClassificationHigh {
		t.Errorf("Label = %q, want high", d.Classification.Label)
	}
	if d.Classification.Description != "Dry Powder Alert" {
		t.Errorf("Description = %q, want Dry Powder Alert", d.Classification.Description)
	}
}

func TestCompute_LowOverextended(t *testing.T) {
	// 8 stables @ 18e9 → stable = 144e9, volatile = 2.856e12, ratio ≈ 0.0504 → low
	// DL: stable trend (0.5% change)
	dm := dataMap3(
		cgGlobalJSON(3e12),
		cgStablesJSON(8, 18e9),
		dlJSON(144e9, 143.3e9), // +0.49% → stable
	)
	p := &Provider{}
	result, err := p.Compute(context.Background(), dm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var d Data
	if err := json.Unmarshal(result.Data, &d); err != nil {
		t.Fatalf("unmarshal Data: %v", err)
	}
	if d.Classification.Label != ClassificationLow {
		t.Errorf("Label = %q, want low", d.Classification.Label)
	}
	if d.Classification.Description != "Overextended" {
		t.Errorf("Description = %q, want Overextended (stable trend)", d.Classification.Description)
	}
	if d.SupplyTrend7d != "stable" {
		t.Errorf("SupplyTrend7d = %q, want stable", d.SupplyTrend7d)
	}
	if !strings.Contains(d.Summary, "Overextended") {
		t.Errorf("Summary %q should mention Overextended", d.Summary)
	}
}

func TestCompute_LowCapitalFlight(t *testing.T) {
	// DL: contracting trend (-13%)
	dm := dataMap3(
		cgGlobalJSON(3e12),
		cgStablesJSON(8, 18e9),
		dlJSON(130e9, 150e9), // -13.3% → contracting
	)
	p := &Provider{}
	result, err := p.Compute(context.Background(), dm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var d Data
	if err := json.Unmarshal(result.Data, &d); err != nil {
		t.Fatalf("unmarshal Data: %v", err)
	}
	if d.Classification.Label != ClassificationLow {
		t.Errorf("Label = %q, want low", d.Classification.Label)
	}
	if d.Classification.Description != "Capital Flight" {
		t.Errorf("Description = %q, want Capital Flight (contracting trend)", d.Classification.Description)
	}
	if d.SupplyTrend7d != "contracting" {
		t.Errorf("SupplyTrend7d = %q, want contracting", d.SupplyTrend7d)
	}
}

// ----- Compute: top-N behaviour -----

func TestCompute_TopClamped(t *testing.T) {
	// context has topN=3 (below minimum 8) — expect clamped to 8
	ctx := config.StoreTopNInContext(context.Background(), 3)
	dm := dataMap3(
		cgGlobalJSON(3e12),
		cgStablesJSON(12, 27e9),
		dlJSON(216e9, 215e9),
	)
	p := &Provider{}
	result, err := p.Compute(ctx, dm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var d Data
	if err := json.Unmarshal(result.Data, &d); err != nil {
		t.Fatalf("unmarshal Data: %v", err)
	}
	if d.StablecoinsCounted != 8 {
		t.Errorf("StablecoinsCounted = %d, want 8 (clamped from 3)", d.StablecoinsCounted)
	}

	var m Meta
	if err := json.Unmarshal(result.Meta, &m); err != nil {
		t.Fatalf("unmarshal Meta: %v", err)
	}
	if !m.TopClamped {
		t.Error("TopClamped should be true when --top 3 was requested")
	}
	if !strings.Contains(m.TopClampedReason, "3") {
		t.Errorf("TopClampedReason %q should mention the original value 3", m.TopClampedReason)
	}
}

func TestCompute_TopNSelected(t *testing.T) {
	// context has topN=10; 12 stables available → counted = 10
	ctx := config.StoreTopNInContext(context.Background(), 10)
	dm := dataMap3(
		cgGlobalJSON(3e12),
		cgStablesJSON(12, 27e9),
		dlJSON(270e9, 268e9),
	)
	p := &Provider{}
	result, err := p.Compute(ctx, dm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var d Data
	if err := json.Unmarshal(result.Data, &d); err != nil {
		t.Fatalf("unmarshal Data: %v", err)
	}
	if d.StablecoinsCounted != 10 {
		t.Errorf("StablecoinsCounted = %d, want 10", d.StablecoinsCounted)
	}
	// stable_mcap = 10 * 27e9 = 270e9
	if d.StableMcapUSD.Value() != 270e9 {
		t.Errorf("StableMcapUSD = %g, want 270e9", d.StableMcapUSD.Value())
	}
}

// ----- Compute: discrepancy -----

func TestCompute_DefiLlamaMissing_StillComputes(t *testing.T) {
	// No DL data — metric still computes; trend defaults to "stable", confidence "high"
	dm := map[string]json.RawMessage{
		api.CoinGeckoGlobalMarket:      cgGlobalJSON(3e12),
		api.CoinGeckoSPPStablesMarkets: cgStablesJSON(8, 27e9),
	}
	p := &Provider{}
	result, err := p.Compute(context.Background(), dm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "ok" {
		t.Errorf("Status = %q, want ok (DL missing should not block)", result.Status)
	}
	var d Data
	if err := json.Unmarshal(result.Data, &d); err != nil {
		t.Fatalf("unmarshal Data: %v", err)
	}
	if d.SupplyTrend7d != "stable" {
		t.Errorf("SupplyTrend7d = %q, want stable (default when DL absent)", d.SupplyTrend7d)
	}
	var m Meta
	if err := json.Unmarshal(result.Meta, &m); err != nil {
		t.Fatalf("unmarshal Meta: %v", err)
	}
	if m.Confidence != "high" {
		t.Errorf("Confidence = %q, want high when validator absent", m.Confidence)
	}
}

func TestCompute_DiscrepancyMedium(t *testing.T) {
	// stable_mcap = 8*27e9 = 216e9; DL reports 260e9 → |216-260|/260 ≈ 16.9% → medium
	dm := dataMap3(
		cgGlobalJSON(3e12),
		cgStablesJSON(8, 27e9),
		dlJSON(260e9, 258e9),
	)
	p := &Provider{}
	result, err := p.Compute(context.Background(), dm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "degraded" {
		t.Errorf("Status = %q, want degraded (medium confidence)", result.Status)
	}
	var m Meta
	if err := json.Unmarshal(result.Meta, &m); err != nil {
		t.Fatalf("unmarshal Meta: %v", err)
	}
	if m.Confidence != "medium" {
		t.Errorf("Confidence = %q, want medium", m.Confidence)
	}
	if !m.DiscrepancyDetected {
		t.Error("DiscrepancyDetected should be true")
	}
	if m.DiscrepancyNote == "" {
		t.Error("DiscrepancyNote should be set when discrepancy detected")
	}
}

func TestCompute_DiscrepancyLow(t *testing.T) {
	// stable_mcap = 216e9; DL reports 360e9 → |216-360|/360 = 40% → low confidence
	dm := dataMap3(
		cgGlobalJSON(3e12),
		cgStablesJSON(8, 27e9),
		dlJSON(360e9, 358e9),
	)
	p := &Provider{}
	result, err := p.Compute(context.Background(), dm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// low confidence (0.3) → DetectStatus → "unavailable"
	if result.Status != "unavailable" {
		t.Errorf("Status = %q, want unavailable (low confidence)", result.Status)
	}
	// data should still be present (not the error sentinel)
	var d Data
	if err := json.Unmarshal(result.Data, &d); err != nil {
		t.Fatalf("data should be present even when status=unavailable via low confidence: %v", err)
	}
	var m Meta
	if err := json.Unmarshal(result.Meta, &m); err != nil {
		t.Fatalf("unmarshal Meta: %v", err)
	}
	if m.Confidence != "low" {
		t.Errorf("Confidence = %q, want low", m.Confidence)
	}
	if !m.DiscrepancyDetected {
		t.Error("DiscrepancyDetected should be true")
	}
}

// ----- Compute: data/meta field completeness -----

func TestCompute_ThresholdKeys_NoLowDuplicate(t *testing.T) {
	dm := dataMap3(cgGlobalJSON(3e12), cgStablesJSON(8, 27e9), dlJSON(216e9, 215e9))
	p := &Provider{}
	result, err := p.Compute(context.Background(), dm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m Meta
	if err := json.Unmarshal(result.Meta, &m); err != nil {
		t.Fatalf("unmarshal Meta: %v", err)
	}
	if _, ok := m.Thresholds["low"]; ok {
		t.Error(`thresholds["low"] is a duplicate of "normal_min" and must not be present`)
	}
	if _, ok := m.Thresholds["high"]; !ok {
		t.Error(`thresholds["high"] must be present`)
	}
	if _, ok := m.Thresholds["normal_min"]; !ok {
		t.Error(`thresholds["normal_min"] must be present`)
	}
}

func TestCompute_DataFields(t *testing.T) {
	dm := dataMap3(cgGlobalJSON(3e12), cgStablesJSON(8, 27e9), dlJSON(216e9, 215e9))
	p := &Provider{}
	result, _ := p.Compute(context.Background(), dm)

	var d Data
	if err := json.Unmarshal(result.Data, &d); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if d.StableMcapUSD.Value() <= 0 {
		t.Error("StableMcapUSD must be positive")
	}
	if d.VolatileMcapUSD.Value() <= 0 {
		t.Error("VolatileMcapUSD must be positive")
	}
	wantVol := 3e12 - 8*27e9
	if d.VolatileMcapUSD.Value() != wantVol {
		t.Errorf("VolatileMcapUSD = %g, want %g", d.VolatileMcapUSD.Value(), wantVol)
	}
}
