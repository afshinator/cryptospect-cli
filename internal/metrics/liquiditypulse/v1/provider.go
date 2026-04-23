package v1

import (
	"context"
	"encoding/json"

	"github.com/afshinator/cryptospect-cli/internal/api"
	"github.com/afshinator/cryptospect-cli/internal/metrics"
	"github.com/afshinator/cryptospect-cli/internal/output"
)

const (
	MetricName    = "liquidity-pulse"
	MetricVersion = "v1.0.0"
)

func init() { metrics.MustRegister(&Provider{}) }

// Provider implements metrics.MetricProvider for liquidity-pulse.
type Provider struct{}

func (p *Provider) Def() metrics.MetricDef {
	return metrics.MetricDef{
		Name:        MetricName,
		Namespace:   metrics.CoreNamespace,
		Version:     MetricVersion,
		Aliases:     []string{"lp"},
		Endpoints:   []string{api.CoinGeckoGlobalMarket},
		Description: "Measures the ratio of 24h trading volume to market cap.",
	}
}

func (p *Provider) Compute(_ context.Context, _ map[string]json.RawMessage) (output.MetricResult, error) {
	msg, _ := json.Marshal(map[string]string{"error": "metric not yet implemented: " + MetricName})
	return output.MetricResult{
		Metric:    MetricName,
		Namespace: metrics.CoreNamespace,
		Version:   MetricVersion,
		Status:    "unavailable",
		Data:      json.RawMessage(msg),
	}, nil
}
