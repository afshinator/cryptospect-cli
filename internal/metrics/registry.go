package metrics

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

var (
	ErrMetricNotFound  = errors.New("metric not found")
	ErrDuplicateMetric = errors.New("metric already registered")
	ErrDuplicateAlias  = errors.New("alias already registered")
)

type MetricDef struct {
	Name        string            `json:"name"`
	Aliases     []string          `json:"aliases"`
	Endpoints   []string          `json:"endpoints"`
	Sources     map[string]string `json:"sources"`
	Description string            `json:"description,omitempty"`
}

type Registry struct {
	metrics     map[string]*MetricDef
	aliasToName map[string]string
}

func NewRegistry() *Registry {
	return &Registry{
		metrics:     make(map[string]*MetricDef),
		aliasToName: make(map[string]string),
	}
}

var (
	globalRegistry     *Registry
	globalRegistryOnce sync.Once
)

func GlobalRegistry() *Registry {
	globalRegistryOnce.Do(func() {
		globalRegistry = NewRegistry()
		RegisterDefaultMetrics(globalRegistry)
	})
	return globalRegistry
}

func RegisterDefaultMetrics(reg *Registry) {
	// Liquidity Pulse: ratio of 24h trading volume to market cap
	reg.Register("liquidity-pulse",
		[]string{"lp"},
		[]string{"coingecko.global_market"},
		map[string]string{"global_market": "coingecko.global_market"},
		"Measures the ratio of 24h trading volume to market cap.")

	// Stablecoin Power: stablecoin dominance and flow strength
	reg.Register("stablecoin-power",
		[]string{"sp"},
		[]string{"coingecko.global_market", "coingecko.spp_stables_markets"},
		map[string]string{
			"global_market":       "coingecko.global_market",
			"spp_stables_markets": "coingecko.spp_stables_markets",
		},
		"Measures stablecoin dominance and flow strength.")

	// Flow Tension: CVD-based market pressure indicator
	reg.Register("flow-tension",
		[]string{"ft"},
		[]string{"binance.spot_cvd_btc_1h", "coingecko.derivatives"},
		map[string]string{
			"spot_cvd":    "binance.spot_cvd_btc_1h",
			"derivatives": "coingecko.derivatives",
		},
		"CVD-based market pressure indicator.")

	// Market Breadth: participation across top assets
	reg.Register("market-breadth",
		[]string{"mb"},
		[]string{"coingecko.coin_markets_breadth"},
		map[string]string{"coin_markets_breadth": "coingecko.coin_markets_breadth"},
		"Measures participation across top assets.")

	// Momentum Divergence: RSI divergence patterns
	reg.Register("momentum-divergence",
		[]string{"md"},
		[]string{"coingecko.coin_markets_momentum"},
		map[string]string{"coin_markets_momentum": "coingecko.coin_markets_momentum"},
		"RSI divergence patterns across assets.")

	// Market Regime: composite regime classification
	reg.Register("market-regime",
		[]string{"mr"},
		[]string{"coingecko.global_market", "coingecko.coin_markets_breadth"},
		map[string]string{
			"global_market":        "coingecko.global_market",
			"coin_markets_breadth": "coingecko.coin_markets_breadth",
		},
		"Composite regime classification using multiple signals.")
}

func (r *Registry) Register(name string, aliases, endpoints []string, sources map[string]string, description string) error {
	if _, exists := r.metrics[name]; exists {
		return ErrDuplicateMetric
	}

	for _, alias := range aliases {
		if _, exists := r.aliasToName[alias]; exists {
			return ErrDuplicateAlias
		}
	}

	def := &MetricDef{
		Name:        name,
		Aliases:     aliases,
		Endpoints:   endpoints,
		Sources:     sources,
		Description: description,
	}

	r.metrics[name] = def
	for _, alias := range aliases {
		r.aliasToName[alias] = name
	}

	return nil
}

func (r *Registry) Get(name string) (*MetricDef, error) {
	m, ok := r.metrics[name]
	if !ok {
		return nil, ErrMetricNotFound
	}
	return m, nil
}

func (r *Registry) GetByAlias(alias string) (*MetricDef, error) {
	name, ok := r.aliasToName[alias]
	if !ok {
		return nil, ErrMetricNotFound
	}
	return r.Get(name)
}

func (r *Registry) List() []*MetricDef {
	list := make([]*MetricDef, 0, len(r.metrics))
	for _, m := range r.metrics {
		list = append(list, m)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
	return list
}

func (r *Registry) RequiredEndpoints(metricNames []string) []string {
	seen := make(map[string]bool)
	var endpoints []string

	for _, name := range metricNames {
		m, err := r.Get(name)
		if err != nil {
			continue
		}
		for _, e := range m.Endpoints {
			if !seen[e] {
				seen[e] = true
				endpoints = append(endpoints, e)
			}
		}
	}

	return endpoints
}

func (r *Registry) Validate(metricNames []string) []error {
	var errs []error
	for _, name := range metricNames {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			errs = append(errs, fmt.Errorf("empty metric name: '%s'", name))
			continue
		}
		if _, err := r.Get(name); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func (r *Registry) ValidateAlias(alias string) error {
	_, err := r.GetByAlias(alias)
	return err
}
