package metrics

import (
	"errors"
	"fmt"
	"slices"
)

var (
	ErrMetricNotFound  = errors.New("metric not found")
	ErrDuplicateMetric = errors.New("metric already registered")
	ErrInvalidProvider = errors.New("invalid provider")
)

// Registry maps versioned metric providers by full key ("namespace/name@version").
type Registry struct {
	providers  map[string]MetricProvider // "namespace/name@version" → provider
	aliasIndex map[string][]string       // alias → []full keys
	nameIndex  map[string][]string       // name  → []full keys
}

// globalRegistry is initialized at package load time so metric packages can
// call MustRegister from their init() functions before main() runs.
var globalRegistry = NewRegistry()

// GlobalRegistry returns the process-wide registry.
func GlobalRegistry() *Registry { return globalRegistry }

// MustRegister registers p in the global registry or panics.
func MustRegister(p MetricProvider) {
	if err := globalRegistry.Register(p); err != nil {
		panic(fmt.Sprintf("metrics.MustRegister: %v", err))
	}
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		providers:  make(map[string]MetricProvider),
		aliasIndex: make(map[string][]string),
		nameIndex:  make(map[string][]string),
	}
}

func fullKey(def MetricDef) string {
	return fmt.Sprintf("%s/%s@%s", def.Namespace, def.Name, def.Version)
}

// Register adds p to the registry.
// Returns ErrInvalidProvider if Def() has empty Name/Namespace/Version or an invalid SemVer.
// Returns ErrDuplicateMetric if the full key already exists.
func (r *Registry) Register(p MetricProvider) error {
	def := p.Def()

	if def.Name == "" {
		return fmt.Errorf("%w: empty Name", ErrInvalidProvider)
	}
	if def.Namespace == "" {
		return fmt.Errorf("%w: empty Namespace", ErrInvalidProvider)
	}
	if def.Version == "" {
		return fmt.Errorf("%w: empty Version", ErrInvalidProvider)
	}
	if _, err := ParseSemVer(def.Version); err != nil {
		return fmt.Errorf("%w: invalid Version: %v", ErrInvalidProvider, err)
	}

	key := fullKey(def)
	if _, exists := r.providers[key]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateMetric, key)
	}

	r.providers[key] = p
	r.nameIndex[def.Name] = append(r.nameIndex[def.Name], key)
	for _, alias := range def.Aliases {
		r.aliasIndex[alias] = append(r.aliasIndex[alias], key)
	}
	return nil
}

// bestProvider returns the highest-priority provider among the given full keys.
// Core-namespace providers beat all others; among equals, highest SemVer wins.
// Ties broken by namespace ascending (deterministic regardless of registration order).
func (r *Registry) bestProvider(keys []string) MetricProvider {
	if len(keys) == 0 {
		return nil
	}

	candidates := keys
	coreKeys := make([]string, 0, len(keys))
	for _, k := range keys {
		if p, ok := r.providers[k]; ok && p.Def().Namespace == CoreNamespace {
			coreKeys = append(coreKeys, k)
		}
	}
	if len(coreKeys) > 0 {
		candidates = coreKeys
	}

	sorted := make([]string, len(candidates))
	copy(sorted, candidates)
	slices.SortFunc(sorted, func(a, b string) int {
		pa, pb := r.providers[a], r.providers[b]
		va, _ := ParseSemVer(pa.Def().Version)
		vb, _ := ParseSemVer(pb.Def().Version)
		if c := CompareSemVer(vb, va); c != 0 { // descending
			return c
		}
		if pa.Def().Namespace < pb.Def().Namespace {
			return -1
		}
		if pa.Def().Namespace > pb.Def().Namespace {
			return 1
		}
		return 0
	})
	return r.providers[sorted[0]]
}

// Resolve returns the best provider for the given name or alias.
func (r *Registry) Resolve(nameOrAlias string) (MetricProvider, error) {
	if keys, ok := r.nameIndex[nameOrAlias]; ok {
		if p := r.bestProvider(keys); p != nil {
			return p, nil
		}
	}
	if keys, ok := r.aliasIndex[nameOrAlias]; ok {
		if p := r.bestProvider(keys); p != nil {
			return p, nil
		}
	}
	return nil, fmt.Errorf("%w: %q", ErrMetricNotFound, nameOrAlias)
}

// BestProviders returns one provider per unique metric name (highest version,
// core namespace preferred), sorted alphabetically by name.
func (r *Registry) BestProviders() []MetricProvider {
	seen := make(map[string]bool, len(r.providers))
	names := make([]string, 0, len(r.providers))
	for _, p := range r.providers {
		name := p.Def().Name
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	slices.Sort(names)

	out := make([]MetricProvider, 0, len(names))
	for _, name := range names {
		if p := r.bestProvider(r.nameIndex[name]); p != nil {
			out = append(out, p)
		}
	}
	return out
}

// List returns MetricDef for each best provider, sorted by name.
func (r *Registry) List() []MetricDef {
	providers := r.BestProviders()
	defs := make([]MetricDef, len(providers))
	for i, p := range providers {
		defs[i] = p.Def()
	}
	return defs
}
