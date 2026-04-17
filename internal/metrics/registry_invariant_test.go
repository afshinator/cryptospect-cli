package metrics

import (
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/api"
)

// TestDefaultRegistryEndpointsAreKnown verifies that every endpoint key referenced
// in RegisterDefaultMetrics exists in api.AllEndpoints(). This catches renames in
// constants.go that are not propagated to the string literals in registry.go.
// After the refactor to use api constants directly, this test is redundant at runtime
// but remains as documentation and protection against future string-literal additions.
func TestDefaultRegistryEndpointsAreKnown(t *testing.T) {
	known := make(map[string]bool)
	for _, ep := range api.AllEndpoints() {
		known[ep] = true
	}

	reg := NewRegistry()
	RegisterDefaultMetrics(reg)

	for _, def := range reg.List() {
		for _, ep := range def.Endpoints {
			if !known[ep] {
				t.Errorf("metric %q references unknown endpoint %q (not in api.AllEndpoints)", def.Name, ep)
			}
		}
		for dp, ep := range def.Sources {
			if !known[ep] {
				t.Errorf("metric %q sources[%q] = %q is not in api.AllEndpoints", def.Name, dp, ep)
			}
		}
	}
}
