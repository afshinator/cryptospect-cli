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
	MetricName    = "liquidity-pulse"
	MetricVersion = "v1.0.0"
)

// Classification labels for the liquidity pulse metric.
const (
	ClassificationHigh   = "high"
	ClassificationNormal = "normal"
	ClassificationLow    = "low"
)

// Classification holds the categorical classification of the liquidity pulse metric.
type Classification struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// Data holds the computed liquidity pulse data.
type Data struct {
	VolumeToMcapRatio float64        `json:"volume_to_mcap_ratio"`
	VolumeUSD         float64        `json:"volume_usd"`
	MarketCapUSD      float64        `json:"market_cap_usd"`
	Classification    Classification `json:"classification"`
	Summary           string         `json:"summary"`
}

func init() { metrics.MustRegister(&Provider{}) }

// Provider implements metrics.MetricProvider for liquidity-pulse.
type Provider struct{}

// Def implements metrics.MetricProvider.
func (p *Provider) Def() metrics.MetricDef {
	return metrics.MetricDef{
		Name:        MetricName,
		Namespace:   metrics.CoreNamespace,
		Version:     MetricVersion,
		Aliases:     []string{"lp"},
		Endpoints:   []string{api.CoinGeckoGlobalMarket, api.BinanceSpotCVD_BTC_1h},
		Description: "Measures the ratio of 24h trading volume to market cap.",
	}
}

// Compute implements metrics.MetricProvider.
func (p *Provider) Compute(_ context.Context, _ map[string]json.RawMessage) (output.MetricResult, error) {
	msg, _ := json.Marshal(map[string]string{"error": "metric not yet implemented: " + MetricName})
	return output.MetricResult{
		Metric:  MetricName,
		Version: MetricVersion,
		Status:  "unavailable",
		Data:    json.RawMessage(msg),
	}, nil
}
