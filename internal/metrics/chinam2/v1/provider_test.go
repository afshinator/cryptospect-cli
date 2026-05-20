package v1

import (
	"context"
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/metrics"
)

func TestProvider_Def(t *testing.T) {
	p := &Provider{}
	def := p.Def()
	if def.Name != MetricName {
		t.Errorf("Name = %q, want %q", def.Name, MetricName)
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

func TestCompute_Expanding(t *testing.T) {
	// 13 months: latest 34,922 (Feb 2026), 12mo ago 32,051.74 (Feb 2025)
	// YoY = (34922 - 32051.74) / 32051.74 * 100 ≈ 8.95%
	obs := []Observation{
		{Period: "2026-02", Value: 34922.0},
		{Period: "2026-01", Value: 34718.6},
		{Period: "2025-12", Value: 34029.48},
		{Period: "2025-11", Value: 33698.9},
		{Period: "2025-10", Value: 33513.12},
		{Period: "2025-09", Value: 33537.71},
		{Period: "2025-08", Value: 33198.31},
		{Period: "2025-07", Value: 32994.29},
		{Period: "2025-06", Value: 33028.68},
		{Period: "2025-05", Value: 32578.38},
		{Period: "2025-04", Value: 32517.39},
		{Period: "2025-03", Value: 32605.55},
		{Period: "2025-02", Value: 32051.74},
	}

	data, meta, err := Compute(Input{Observations: obs})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if data.Classification.Label != LabelExpanding {
		t.Errorf("Label = %q, want %q (YoY > 8%%)", data.Classification.Label, LabelExpanding)
	}
	if meta.Confidence != ConfidenceHigh {
		t.Errorf("Confidence = %q, want %q", meta.Confidence, ConfidenceHigh)
	}
	if data.YoYChangePct == nil {
		t.Fatal("YoYChangePct should not be nil with 13 data points")
	}
	if data.M2LevelCNYBillion.Value() != 3492.2 {
		t.Errorf("M2LevelCNYBillion = %v, want 3492.2", data.M2LevelCNYBillion.Value())
	}
}

func TestCompute_Normal(t *testing.T) {
	obs := []Observation{
		{Period: "2026-02", Value: 34000.0},
		{Period: "2026-01", Value: 33800.0},
		{Period: "2025-12", Value: 33600.0},
		{Period: "2025-11", Value: 33400.0},
		{Period: "2025-10", Value: 33200.0},
		{Period: "2025-09", Value: 33000.0},
		{Period: "2025-08", Value: 32800.0},
		{Period: "2025-07", Value: 32600.0},
		{Period: "2025-06", Value: 32400.0},
		{Period: "2025-05", Value: 32200.0},
		{Period: "2025-04", Value: 32000.0},
		{Period: "2025-03", Value: 31950.0},
		{Period: "2025-02", Value: 31924.88},
	}

	data, _, err := Compute(Input{Observations: obs})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if data.Classification.Label != LabelNormal {
		t.Errorf("Label = %q, want %q", data.Classification.Label, LabelNormal)
	}
}

func TestCompute_Slowing(t *testing.T) {
	obs := []Observation{
		{Period: "2026-02", Value: 32000.0},
		{Period: "2026-01", Value: 31900.0},
		{Period: "2025-12", Value: 31800.0},
		{Period: "2025-11", Value: 31700.0},
		{Period: "2025-10", Value: 31600.0},
		{Period: "2025-09", Value: 31500.0},
		{Period: "2025-08", Value: 31450.0},
		{Period: "2025-07", Value: 31400.0},
		{Period: "2025-06", Value: 31380.0},
		{Period: "2025-05", Value: 31370.0},
		{Period: "2025-04", Value: 31360.0},
		{Period: "2025-03", Value: 31370.0},
		{Period: "2025-02", Value: 31372.55},
	}

	data, _, err := Compute(Input{Observations: obs})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if data.Classification.Label != LabelSlowing {
		t.Errorf("Label = %q, want %q (YoY < 4%%)", data.Classification.Label, LabelSlowing)
	}
}

func TestCompute_InsufficientHistory(t *testing.T) {
	obs := []Observation{
		{Period: "2026-02", Value: 34922.0},
	}

	data, meta, err := Compute(Input{Observations: obs})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if data.Classification.Label != LabelNormal {
		t.Errorf("Label = %q, want %q (insufficient history)", data.Classification.Label, LabelNormal)
	}
	if meta.Confidence != ConfidenceMedium {
		t.Errorf("Confidence = %q, want %q", meta.Confidence, ConfidenceMedium)
	}
	if data.YoYChangePct != nil {
		t.Error("YoYChangePct should be nil with insufficient history")
	}
}

func TestCompute_EmptyInput(t *testing.T) {
	_, _, err := Compute(Input{})
	if err == nil {
		t.Error("Compute should return error for empty input")
	}
}
