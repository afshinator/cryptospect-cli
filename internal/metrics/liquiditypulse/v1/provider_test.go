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
	if def.Namespace != metrics.CoreNamespace {
		t.Errorf("Namespace = %q, want %q", def.Namespace, metrics.CoreNamespace)
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
	if result.Namespace != metrics.CoreNamespace {
		t.Errorf("Namespace = %q, want %q", result.Namespace, metrics.CoreNamespace)
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
