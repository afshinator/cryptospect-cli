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
