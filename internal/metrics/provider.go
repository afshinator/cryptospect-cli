package metrics

import (
	"context"
	"encoding/json"

	"github.com/afshinator/cryptospect-cli/internal/output"
)

// CoreNamespace is the namespace used by all built-in metric providers.
const CoreNamespace = "cryptospect"

// MetricDef describes a metric's identity and data requirements.
type MetricDef struct {
	Name        string   `json:"name"`
	Namespace   string   `json:"namespace"`
	Version     string   `json:"version"`
	Aliases     []string `json:"aliases"`
	Endpoints   []string `json:"endpoints"`
	Description string   `json:"description,omitempty"`
}

// MetricProvider is implemented by each metric package to supply identity and computation.
// Non-nil errors from Compute are reserved for catastrophic failures; unavailability is
// expressed via MetricResult.Status = "unavailable".
type MetricProvider interface {
	Def() MetricDef
	Compute(ctx context.Context, data map[string]json.RawMessage) (output.MetricResult, error)
}
