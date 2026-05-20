package v1

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/afshinator/cryptospect-cli/internal/api"
	"github.com/afshinator/cryptospect-cli/internal/api/binance"
	"github.com/afshinator/cryptospect-cli/internal/metrics"
	"github.com/afshinator/cryptospect-cli/internal/output"
)

func init() { metrics.MustRegister(&Provider{}) }

// Provider implements metrics.MetricProvider for volatility.
type Provider struct{}

// Def implements metrics.MetricProvider.
func (p *Provider) Def() metrics.MetricDef {
	return metrics.MetricDef{
		Name:        MetricName,
		Namespace:   metrics.CoreNamespace,
		Version:     MetricVersion,
		Aliases:     []string{"vol"},
		Endpoints:   []string{api.BinanceSpotVol_BTC_24h, api.BinanceSpotCVD_ETH_1h},
		Description: "Measures annualized realized volatility for BTC and ETH with ETH/BTC vol spread.",
	}
}

// Compute implements metrics.MetricProvider.
func (p *Provider) Compute(_ context.Context, data map[string]json.RawMessage) (output.MetricResult, error) {
	// Parse BTC klines
	btcRaw, ok := data[api.BinanceSpotVol_BTC_24h]
	if !ok || len(btcRaw) == 0 {
		return metrics.UnavailableResult(MetricName, MetricVersion, metrics.CoreNamespace, "missing BTC klines data")
	}
	btcSlice, err := binance.ParseMultiKlinesResponse(btcRaw)
	if err != nil {
		return metrics.UnavailableResult(MetricName, MetricVersion, metrics.CoreNamespace, fmt.Sprintf("parsing BTC klines: %v", err))
	}

	// Parse ETH klines
	ethRaw, ok := data[api.BinanceSpotCVD_ETH_1h]
	if !ok || len(ethRaw) == 0 {
		return metrics.UnavailableResult(MetricName, MetricVersion, metrics.CoreNamespace, "missing ETH klines data")
	}
	ethSlice, err := binance.ParseMultiKlinesResponse(ethRaw)
	if err != nil {
		return metrics.UnavailableResult(MetricName, MetricVersion, metrics.CoreNamespace, fmt.Sprintf("parsing ETH klines: %v", err))
	}

	// Extract closes
	btcCloses := make([]float64, len(btcSlice.Klines))
	for i, k := range btcSlice.Klines {
		btcCloses[i] = k.Close
	}
	ethCloses := make([]float64, len(ethSlice.Klines))
	for i, k := range ethSlice.Klines {
		ethCloses[i] = k.Close
	}

	input := Input{
		BTCCloses: btcCloses,
		ETHCloses: ethCloses,
	}

	dataResult, compMeta, err := Compute(input)
	if err != nil {
		return metrics.UnavailableResult(MetricName, MetricVersion, metrics.CoreNamespace, err.Error())
	}

	dataBytes, err := json.Marshal(dataResult)
	if err != nil {
		return metrics.UnavailableResult(MetricName, MetricVersion, metrics.CoreNamespace, fmt.Sprintf("marshaling data: %v", err))
	}

	meta := Meta{
		PrimarySource: "binance_us",
		Confidence:    compMeta.Confidence,
		BTCCandles:    len(btcSlice.Klines),
		ETHCandles:    len(ethSlice.Klines),
		Thresholds: map[string]float64{
			"subdued_max":  SpreadSubduedMax,
			"elevated_min": SpreadElevatedMin,
		},
		Description: metricDescription,
	}

	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return metrics.UnavailableResult(MetricName, MetricVersion, metrics.CoreNamespace, fmt.Sprintf("marshaling meta: %v", err))
	}

	conf := metrics.ConfidenceToFloat(compMeta.Confidence)
	status := metrics.DetectStatus(conf, false)

	return output.MetricResult{
		Metric:    MetricName,
		Version:   MetricVersion,
		Namespace: metrics.CoreNamespace,
		Status:    status,
		Data:      json.RawMessage(dataBytes),
		Meta:      json.RawMessage(metaBytes),
	}, nil
}
