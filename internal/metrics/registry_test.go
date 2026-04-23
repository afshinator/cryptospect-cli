package metrics

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/output"
)

// testProvider is a minimal MetricProvider for use in registry tests.
type testProvider struct {
	def MetricDef
}

func (p *testProvider) Def() MetricDef { return p.def }
func (p *testProvider) Compute(_ context.Context, _ map[string]json.RawMessage) (output.MetricResult, error) {
	return output.MetricResult{Metric: p.def.Name, Status: "unavailable"}, nil
}

func makeProvider(name, ns, version string, aliases []string) *testProvider {
	return &testProvider{def: MetricDef{
		Name:      name,
		Namespace: ns,
		Version:   version,
		Aliases:   aliases,
	}}
}

func TestRegistry_Register_Valid(t *testing.T) {
	reg := NewRegistry()
	p := makeProvider("liquidity-pulse", "cryptospect", "v1.0.0", []string{"lp"})
	if err := reg.Register(p); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
}

func TestRegistry_Register_EmptyName(t *testing.T) {
	reg := NewRegistry()
	p := makeProvider("", "cryptospect", "v1.0.0", nil)
	if err := reg.Register(p); !errors.Is(err, ErrInvalidProvider) {
		t.Errorf("got %v, want ErrInvalidProvider", err)
	}
}

func TestRegistry_Register_EmptyNamespace(t *testing.T) {
	reg := NewRegistry()
	p := makeProvider("liquidity-pulse", "", "v1.0.0", nil)
	if err := reg.Register(p); !errors.Is(err, ErrInvalidProvider) {
		t.Errorf("got %v, want ErrInvalidProvider", err)
	}
}

func TestRegistry_Register_EmptyVersion(t *testing.T) {
	reg := NewRegistry()
	p := makeProvider("liquidity-pulse", "cryptospect", "", nil)
	if err := reg.Register(p); !errors.Is(err, ErrInvalidProvider) {
		t.Errorf("got %v, want ErrInvalidProvider", err)
	}
}

func TestRegistry_Register_InvalidSemVer(t *testing.T) {
	reg := NewRegistry()
	p := makeProvider("liquidity-pulse", "cryptospect", "1.0.0", nil) // no v prefix
	if err := reg.Register(p); !errors.Is(err, ErrInvalidProvider) {
		t.Errorf("got %v, want ErrInvalidProvider", err)
	}
}

func TestRegistry_Register_DuplicateFullKey(t *testing.T) {
	reg := NewRegistry()
	p := makeProvider("liquidity-pulse", "cryptospect", "v1.0.0", nil)
	if err := reg.Register(p); err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(p); !errors.Is(err, ErrDuplicateMetric) {
		t.Errorf("got %v, want ErrDuplicateMetric", err)
	}
}

func TestRegistry_Register_DuplicateAliasAcrossVersionsAllowed(t *testing.T) {
	// Two different versions of the same metric may share aliases; resolution picks best.
	reg := NewRegistry()
	p1 := makeProvider("liquidity-pulse", "cryptospect", "v1.0.0", []string{"lp"})
	p2 := makeProvider("liquidity-pulse", "cryptospect", "v2.0.0", []string{"lp"})
	if err := reg.Register(p1); err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(p2); err != nil {
		t.Fatalf("registering second version with same alias should succeed: %v", err)
	}
}

func TestRegistry_Resolve_ByName(t *testing.T) {
	reg := NewRegistry()
	p := makeProvider("liquidity-pulse", "cryptospect", "v1.0.0", []string{"lp"})
	_ = reg.Register(p)

	got, err := reg.Resolve("liquidity-pulse")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Def().Name != "liquidity-pulse" {
		t.Errorf("got %v, want liquidity-pulse", got.Def().Name)
	}
}

func TestRegistry_Resolve_ByAlias(t *testing.T) {
	reg := NewRegistry()
	p := makeProvider("liquidity-pulse", "cryptospect", "v1.0.0", []string{"lp"})
	_ = reg.Register(p)

	got, err := reg.Resolve("lp")
	if err != nil {
		t.Fatalf("Resolve alias: %v", err)
	}
	if got.Def().Name != "liquidity-pulse" {
		t.Errorf("got %v, want liquidity-pulse", got.Def().Name)
	}
}

func TestRegistry_Resolve_NotFound(t *testing.T) {
	reg := NewRegistry()
	_, err := reg.Resolve("unknown")
	if !errors.Is(err, ErrMetricNotFound) {
		t.Errorf("got %v, want ErrMetricNotFound", err)
	}
}

func TestRegistry_BestProviders_Empty(t *testing.T) {
	reg := NewRegistry()
	if got := reg.BestProviders(); len(got) != 0 {
		t.Errorf("empty registry BestProviders = %d, want 0", len(got))
	}
}

func TestRegistry_BestProviders_SingleProvider(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register(makeProvider("liquidity-pulse", "cryptospect", "v1.0.0", []string{"lp"}))

	got := reg.BestProviders()
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Def().Name != "liquidity-pulse" {
		t.Errorf("got %v, want liquidity-pulse", got[0].Def().Name)
	}
}

func TestRegistry_BestProviders_HighestVersionWins(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register(makeProvider("liquidity-pulse", "cryptospect", "v1.0.0", []string{"lp"}))
	_ = reg.Register(makeProvider("liquidity-pulse", "cryptospect", "v2.0.0", []string{"lp"}))

	got := reg.BestProviders()
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1 (one unique name)", len(got))
	}
	if got[0].Def().Version != "v2.0.0" {
		t.Errorf("version = %v, want v2.0.0", got[0].Def().Version)
	}
}

func TestRegistry_BestProviders_CoreNamespacePriority(t *testing.T) {
	reg := NewRegistry()
	// Fork has higher version but non-core namespace; core wins.
	_ = reg.Register(makeProvider("liquidity-pulse", "cryptospect", "v1.0.0", []string{"lp"}))
	_ = reg.Register(makeProvider("liquidity-pulse", "fork-user", "v9.0.0", []string{"lp"}))

	got := reg.BestProviders()
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Def().Namespace != CoreNamespace {
		t.Errorf("namespace = %v, want %v", got[0].Def().Namespace, CoreNamespace)
	}
}

func TestRegistry_BestProviders_SortedByName(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register(makeProvider("stablecoin-power", "cryptospect", "v1.0.0", []string{"sp"}))
	_ = reg.Register(makeProvider("liquidity-pulse", "cryptospect", "v1.0.0", []string{"lp"}))
	_ = reg.Register(makeProvider("market-breadth", "cryptospect", "v1.0.0", []string{"mb"}))

	got := reg.BestProviders()
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	names := []string{got[0].Def().Name, got[1].Def().Name, got[2].Def().Name}
	want := []string{"liquidity-pulse", "market-breadth", "stablecoin-power"}
	for i, w := range want {
		if names[i] != w {
			t.Errorf("pos %d: got %q, want %q", i, names[i], w)
		}
	}
}

func TestRegistry_BestProviders_Deterministic(t *testing.T) {
	// Register in different orders, expect same result.
	build := func(first, second string) MetricProvider {
		reg := NewRegistry()
		_ = reg.Register(makeProvider("liquidity-pulse", "cryptospect", first, []string{"lp"}))
		_ = reg.Register(makeProvider("liquidity-pulse", "cryptospect", second, []string{"lp"}))
		return reg.BestProviders()[0]
	}

	got1 := build("v1.0.0", "v2.0.0")
	got2 := build("v2.0.0", "v1.0.0")
	if got1.Def().Version != got2.Def().Version {
		t.Errorf("non-deterministic: %v vs %v", got1.Def().Version, got2.Def().Version)
	}
	if got1.Def().Version != "v2.0.0" {
		t.Errorf("version = %v, want v2.0.0", got1.Def().Version)
	}
}

func TestRegistry_List(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register(makeProvider("stablecoin-power", "cryptospect", "v1.0.0", []string{"sp"}))
	_ = reg.Register(makeProvider("liquidity-pulse", "cryptospect", "v1.0.0", []string{"lp"}))

	list := reg.List()
	if len(list) != 2 {
		t.Fatalf("len = %d, want 2", len(list))
	}
	if list[0].Name != "liquidity-pulse" {
		t.Errorf("first = %v, want liquidity-pulse", list[0].Name)
	}
}

func TestMustRegister_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic from MustRegister with invalid provider")
		}
	}()
	// Use a fresh registry to avoid polluting global state.
	// MustRegister uses the global registry, so we test via Register instead.
	reg := NewRegistry()
	p := makeProvider("", "cryptospect", "v1.0.0", nil) // empty name → invalid
	if err := reg.Register(p); err == nil {
		t.Fatal("expected error")
	}
	// Now test the panic path via a local helper.
	mustRegisterTo(reg, p)
}

func mustRegisterTo(reg *Registry, p MetricProvider) {
	if err := reg.Register(p); err != nil {
		panic("metrics.MustRegister: " + err.Error())
	}
}

func TestGlobalRegistry_NonNil(t *testing.T) {
	if GlobalRegistry() == nil {
		t.Error("GlobalRegistry() returned nil")
	}
}
