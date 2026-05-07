package main

import (
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/api"
	"github.com/afshinator/cryptospect-cli/internal/metrics"
)

var wantCatalogMetrics = []struct {
	name    string
	alias   string
	version string
}{
	{"flow-tension", "ft", "v1.0.0"},
	{"liquidity-pulse", "lp", "v1.0.0"},
	{"market-breadth", "mb", "v1.0.0"},
	{"market-regime", "mr", "v1.0.0"},
	{"momentum-divergence", "md", "v1.1.0"},
	{"stablecoin-power", "sp", "v1.0.0"},
}

func TestCatalog_AllProvidersRegistered(t *testing.T) {
	reg := metrics.GlobalRegistry()
	providers := reg.BestProviders()
	if len(providers) != len(wantCatalogMetrics) {
		t.Errorf("provider count = %d, want %d", len(providers), len(wantCatalogMetrics))
	}
	for _, w := range wantCatalogMetrics {
		p, err := reg.Resolve(w.name)
		if err != nil {
			t.Errorf("Resolve(%q): %v", w.name, err)
			continue
		}
		def := p.Def()
		if def.Version != w.version {
			t.Errorf("%s: Version = %q, want %q", w.name, def.Version, w.version)
		}
	}
}

func TestCatalog_AliasesResolvable(t *testing.T) {
	reg := metrics.GlobalRegistry()
	for _, w := range wantCatalogMetrics {
		p, err := reg.Resolve(w.alias)
		if err != nil {
			t.Errorf("Resolve alias %q: %v", w.alias, err)
			continue
		}
		if p.Def().Name != w.name {
			t.Errorf("alias %q resolved to %q, want %q", w.alias, p.Def().Name, w.name)
		}
	}
}

func TestCatalog_EndpointsAreKnown(t *testing.T) {
	known := make(map[string]bool)
	for _, ep := range api.AllEndpoints() {
		known[ep] = true
	}

	reg := metrics.GlobalRegistry()
	for _, p := range reg.BestProviders() {
		def := p.Def()
		for _, ep := range def.Endpoints {
			if !known[ep] {
				t.Errorf("metric %q references unknown endpoint %q", def.Name, ep)
			}
		}
	}
}
