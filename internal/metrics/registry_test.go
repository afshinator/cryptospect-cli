package metrics

import (
	"errors"
	"testing"
)

func TestMetricDef(t *testing.T) {
	def := MetricDef{
		Name:        "liquidity-pulse",
		Aliases:     []string{"lp"},
		Endpoints:   []string{"coingecko.global_market"},
		Sources:     map[string]string{"global_market": "coingecko.global_market"},
		Description: "Measures the ratio of 24h trading volume to market cap.",
	}
	if def.Name != "liquidity-pulse" {
		t.Errorf("Name = %v, want liquidity-pulse", def.Name)
	}
	if len(def.Aliases) != 1 || def.Aliases[0] != "lp" {
		t.Errorf("Aliases = %v, want [lp]", def.Aliases)
	}
	if len(def.Endpoints) != 1 || def.Endpoints[0] != "coingecko.global_market" {
		t.Errorf("Endpoints = %v, want [coingecko.global_market]", def.Endpoints)
	}
	if def.Sources["global_market"] != "coingecko.global_market" {
		t.Errorf("Sources mapping incorrect")
	}
}

func TestRegistry_Register(t *testing.T) {
	reg := NewRegistry()
	err := reg.Register("liquidity-pulse", []string{"lp"}, []string{"coingecko.global_market"},
		map[string]string{"global_market": "coingecko.global_market"},
		"Measures the ratio of 24h trading volume to market cap.")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Duplicate registration should fail
	err = reg.Register("liquidity-pulse", []string{"lp"}, []string{"coingecko.global_market"},
		map[string]string{"global_market": "coingecko.global_market"}, "")
	if !errors.Is(err, ErrDuplicateMetric) {
		t.Errorf("Duplicate registration error = %v, want ErrDuplicateMetric", err)
	}
}

func TestRegistry_Get(t *testing.T) {
	reg := NewRegistry()
	err := reg.Register("liquidity-pulse", []string{"lp"}, []string{"coingecko.global_market"},
		map[string]string{"global_market": "coingecko.global_market"}, "")
	if err != nil {
		t.Fatal(err)
	}

	def, err := reg.Get("liquidity-pulse")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if def.Name != "liquidity-pulse" {
		t.Errorf("Get returned wrong metric: %v", def.Name)
	}

	// Get by alias should also work
	def, err = reg.GetByAlias("lp")
	if err != nil {
		t.Fatalf("GetByAlias failed: %v", err)
	}
	if def.Name != "liquidity-pulse" {
		t.Errorf("GetByAlias returned wrong metric: %v", def.Name)
	}

	// Non-existent metric
	_, err = reg.Get("unknown")
	if !errors.Is(err, ErrMetricNotFound) {
		t.Errorf("Get unknown error = %v, want ErrMetricNotFound", err)
	}
}

func TestRegistry_List(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register("liquidity-pulse", []string{"lp"}, []string{"coingecko.global_market"},
		map[string]string{"global_market": "coingecko.global_market"}, "")
	_ = reg.Register("stablecoin-power", []string{"sp"}, []string{"coingecko.global_market", "coingecko.spp_stables_markets"},
		map[string]string{"global_market": "coingecko.global_market", "spp_stables_markets": "coingecko.spp_stables_markets"}, "")

	list := reg.List()
	if len(list) != 2 {
		t.Fatalf("List length = %v, want 2", len(list))
	}
	// Should be sorted by name
	if list[0].Name != "liquidity-pulse" {
		t.Errorf("First metric = %v, want liquidity-pulse", list[0].Name)
	}
	if list[1].Name != "stablecoin-power" {
		t.Errorf("Second metric = %v, want stablecoin-power", list[1].Name)
	}
}

func TestRegistry_RequiredEndpoints(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register("liquidity-pulse", []string{"lp"}, []string{"coingecko.global_market"},
		map[string]string{"global_market": "coingecko.global_market"}, "")
	_ = reg.Register("stablecoin-power", []string{"sp"}, []string{"coingecko.global_market", "coingecko.spp_stables_markets"},
		map[string]string{"global_market": "coingecko.global_market", "spp_stables_markets": "coingecko.spp_stables_markets"}, "")

	endpoints := reg.RequiredEndpoints([]string{"liquidity-pulse", "stablecoin-power"})
	want := []string{"coingecko.global_market", "coingecko.spp_stables_markets"}
	if len(endpoints) != len(want) {
		t.Fatalf("RequiredEndpoints length = %v, want %v", len(endpoints), len(want))
	}
	// order not guaranteed, use map
	seen := make(map[string]bool)
	for _, e := range endpoints {
		seen[e] = true
	}
	for _, w := range want {
		if !seen[w] {
			t.Errorf("Missing endpoint %v", w)
		}
	}
}

func TestRegistry_Validate(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register("liquidity-pulse", []string{"lp"}, []string{"coingecko.global_market"},
		map[string]string{"global_market": "coingecko.global_market"}, "")

	errs := reg.Validate([]string{"liquidity-pulse", "unknown"})
	if len(errs) != 1 {
		t.Fatalf("Validate errors length = %v, want 1", len(errs))
	}
	if !errors.Is(errs[0], ErrMetricNotFound) {
		t.Errorf("Validate error = %v, want ErrMetricNotFound", errs[0])
	}
}

func TestGlobalRegistry(t *testing.T) {
	reg := GlobalRegistry()
	// Should have default metrics registered
	def, err := reg.Get("liquidity-pulse")
	if err != nil {
		t.Fatalf("GlobalRegistry missing liquidity-pulse: %v", err)
	}
	if def.Name != "liquidity-pulse" {
		t.Errorf("GlobalRegistry metric name = %v, want liquidity-pulse", def.Name)
	}
	// Check alias
	def, err = reg.GetByAlias("lp")
	if err != nil {
		t.Fatalf("GetByAlias lp failed: %v", err)
	}
	if def.Name != "liquidity-pulse" {
		t.Errorf("GetByAlias lp returned %v", def.Name)
	}
}

func TestRegistry_UniqueAliases(t *testing.T) {
	reg := NewRegistry()
	err := reg.Register("liquidity-pulse", []string{"lp"}, []string{"coingecko.global_market"},
		map[string]string{"global_market": "coingecko.global_market"}, "")
	if err != nil {
		t.Fatal(err)
	}
	// Register another metric with same alias should fail
	err = reg.Register("other", []string{"lp"}, []string{"coingecko.global_market"},
		map[string]string{"global_market": "coingecko.global_market"}, "")
	if !errors.Is(err, ErrDuplicateAlias) {
		t.Errorf("Duplicate alias error = %v, want ErrDuplicateAlias", err)
	}
}
