package v1

import (
	"context"
	"encoding/json"

	"github.com/afshinator/cryptospect-cli/internal/api"
	"github.com/afshinator/cryptospect-cli/internal/metrics"
	"github.com/afshinator/cryptospect-cli/internal/output"
)

// MetricName and MetricVersion identify this provider in the registry.
const (
	MetricName    = "stablecoin-power"
	MetricVersion = "v1.0.0"
)

func init() { metrics.MustRegister(&Provider{}) }

// Provider implements metrics.MetricProvider for stablecoin-power.
type Provider struct{}

// Def implements metrics.MetricProvider.
func (p *Provider) Def() metrics.MetricDef {
	return metrics.MetricDef{
		Name:      MetricName,
		Namespace: metrics.CoreNamespace,
		Version:   MetricVersion,
		Aliases:   []string{"sp"},
		Endpoints: []string{
			api.CoinGeckoGlobalMarket,
			api.CoinGeckoSPPStablesMarkets,
		},
		Description: "Measures stablecoin dominance and flow strength.",
	}
}

// Compute implements metrics.MetricProvider.
func (p *Provider) Compute(_ context.Context, _ map[string]json.RawMessage) (output.MetricResult, error) {
	msg, _ := json.Marshal(map[string]string{"error": "metric not yet implemented: " + MetricName})
	return output.MetricResult{
		Metric:  MetricName,
		Version: MetricVersion,
		Status: "unavailable",
		Data:   json.RawMessage(msg),
	}, nil
}
