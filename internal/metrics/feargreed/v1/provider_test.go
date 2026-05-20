package v1

import (
	"context"
	"encoding/json"
	"testing"

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

// ── Pure Compute tests ──

func TestCompute_ExtremeFear(t *testing.T) {
	data, _, err := Compute(Input{Values: []int{10, 15, 20, 12, 18, 14, 22}})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if data.Value != 10 {
		t.Errorf("Value = %d, want 10", data.Value)
	}
	if data.Classification.Label != LabelExtremeFear {
		t.Errorf("Label = %q, want %q", data.Classification.Label, LabelExtremeFear)
	}
}

func TestCompute_Fear_WithMA(t *testing.T) {
	// 7 values: 35, 28, 30, 32, 25, 27, 31
	// MA = (35+28+30+32+25+27+31)/7 = 208/7 ≈ 29.714
	// 35 > 29.7+2 → improving
	data, meta, err := Compute(Input{Values: []int{35, 28, 30, 32, 25, 27, 31}})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if data.Classification.Label != LabelFear {
		t.Errorf("Label = %q, want %q", data.Classification.Label, LabelFear)
	}
	if meta.Confidence != ConfidenceHigh {
		t.Errorf("Confidence = %q, want %q", meta.Confidence, ConfidenceHigh)
	}
	if meta.Trend != TrendImproving {
		t.Errorf("Trend = %q, want %q", meta.Trend, TrendImproving)
	}
	if meta.SevdMA == nil {
		t.Fatal("SevdMA should not be nil with 7 data points")
	}
}

func TestCompute_Neutral(t *testing.T) {
	data, _, err := Compute(Input{Values: []int{50}})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if data.Classification.Label != LabelNeutral {
		t.Errorf("Label = %q, want %q", data.Classification.Label, LabelNeutral)
	}
}

func TestCompute_Greed(t *testing.T) {
	data, _, err := Compute(Input{Values: []int{65}})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if data.Classification.Label != LabelGreed {
		t.Errorf("Label = %q, want %q", data.Classification.Label, LabelGreed)
	}
}

func TestCompute_ExtremeGreed(t *testing.T) {
	data, _, err := Compute(Input{Values: []int{85}})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if data.Classification.Label != LabelExtremeGreed {
		t.Errorf("Label = %q, want %q", data.Classification.Label, LabelExtremeGreed)
	}
}

func TestCompute_InsufficientHistory(t *testing.T) {
	_, meta, err := Compute(Input{Values: []int{25, 30, 35}})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if meta.Confidence != ConfidenceMedium {
		t.Errorf("Confidence = %q, want %q (< 7 data points)", meta.Confidence, ConfidenceMedium)
	}
	if meta.SevdMA != nil {
		t.Error("SevdMA should be nil with < 7 data points")
	}
}

func TestCompute_StableTrend(t *testing.T) {
	// 7 values all close to 45
	// MA = (45+46+44+45+47+43+45)/7 = 315/7 = 45
	// 45 - 45 = 0 < 2 → stable
	_, meta, err := Compute(Input{Values: []int{45, 46, 44, 45, 47, 43, 45}})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if meta.Trend != TrendStable {
		t.Errorf("Trend = %q, want %q", meta.Trend, TrendStable)
	}
}

func TestCompute_Deteriorating(t *testing.T) {
	// Current: 20, MA of 7: [40, 42, 38, 45, 41, 39, 43] = 288/7 ≈ 41.14
	// 20 < 41.14 - 2 → deteriorating
	_, meta, err := Compute(Input{Values: []int{20, 40, 42, 38, 45, 41, 39}})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if meta.Trend != TrendDeteriorating {
		t.Errorf("Trend = %q, want %q", meta.Trend, TrendDeteriorating)
	}
}

func TestCompute_EmptyInput(t *testing.T) {
	_, _, err := Compute(Input{Values: []int{}})
	if err == nil {
		t.Error("Compute should return error for empty input")
	}
}

func TestProvider_Compute_Happy_WithMockAPI(t *testing.T) {
	mockJSON := json.RawMessage(`{
		"name": "Fear and Greed Index",
		"data": [
			{"value": "25", "value_classification": "Extreme Fear", "timestamp": "1779148800", "time_until_update": "5325"},
			{"value": "24", "value_classification": "Extreme Fear", "timestamp": "1779062400", "time_until_update": "0"},
			{"value": "26", "value_classification": "Fear", "timestamp": "1778976000", "time_until_update": "0"},
			{"value": "30", "value_classification": "Fear", "timestamp": "1778889600", "time_until_update": "0"},
			{"value": "28", "value_classification": "Fear", "timestamp": "1778803200", "time_until_update": "0"},
			{"value": "32", "value_classification": "Fear", "timestamp": "1778716800", "time_until_update": "0"},
			{"value": "35", "value_classification": "Fear", "timestamp": "1778630400", "time_until_update": "0"}
		],
		"metadata": {"error": null}
	}`)

	p := &Provider{}
	result, err := p.Compute(context.Background(), map[string]json.RawMessage{
		"alternativeme.fng": mockJSON,
	})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if result.Status == "unavailable" {
		t.Fatal("Compute should return data, not unavailable")
	}

	var d Data
	if err := json.Unmarshal(result.Data, &d); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}
	if d.Value != 25 {
		t.Errorf("Value = %d, want 25", d.Value)
	}
}
