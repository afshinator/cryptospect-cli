package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/afshinator/cryptospect-cli/internal/api"
	"github.com/afshinator/cryptospect-cli/internal/api/binance"
	"github.com/afshinator/cryptospect-cli/internal/api/coingecko"
	"github.com/afshinator/cryptospect-cli/internal/metrics"
	"github.com/afshinator/cryptospect-cli/internal/output"
)

// MetricName is the canonical name for this metric.
const MetricName = "liquidity-pulse"

// MetricVersion is the SemVer version of this metric provider.
const MetricVersion = "v1.0.0"

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
	VolumeToMcapRatio metrics.MetricFloat `json:"volume_to_mcap_ratio"`
	VolumeUSD         metrics.MetricFloat `json:"volume_usd"`
	MarketCapUSD      metrics.MetricFloat `json:"market_cap_usd"`
	Classification    Classification      `json:"classification"`
	Summary           string              `json:"summary"`
}

const (
	thresholdHigh       = 0.15
	thresholdLow        = 0.05
	validationThreshold = 0.20
)

// Meta holds metadata about the metric computation.
type Meta struct {
	PrimarySource       string             `json:"primary_source"`
	ValidatorSource     string             `json:"validator_source,omitempty"`
	DiscrepancyDetected bool               `json:"discrepancy_detected,omitempty"`
	DiscrepancyNote     string             `json:"discrepancy_note,omitempty"`
	Confidence          string             `json:"confidence"`
	Thresholds          map[string]float64 `json:"thresholds,omitempty"`
	Description         string             `json:"description,omitempty"`
}

func classify(ratio float64) Classification {
	switch {
	case ratio >= thresholdHigh:
		return Classification{Label: ClassificationHigh, Description: "Strong short-term conviction"}
	case ratio >= thresholdLow:
		return Classification{Label: ClassificationNormal, Description: "Healthy market"}
	default:
		return Classification{Label: ClassificationLow, Description: "Low conviction"}
	}
}

func summary(ratio float64, label string) string {
	return fmt.Sprintf("Volume/MCap: %5.2f%% | Conviction: %s", ratio*100, label)
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
func (p *Provider) Compute(_ context.Context, data map[string]json.RawMessage) (output.MetricResult, error) {
	globalData, ok := data[api.CoinGeckoGlobalMarket]
	if !ok || len(globalData) == 0 {
		return metrics.UnavailableResult(MetricName, MetricVersion, metrics.CoreNamespace, "missing primary endpoint data")
	}

	parsed, err := coingecko.ParseGlobalResponse(globalData)
	if err != nil {
		return metrics.UnavailableResult(MetricName, MetricVersion, metrics.CoreNamespace, fmt.Sprintf("parsing primary data: %v", err))
	}

	volumeUSD, ok := parsed.GetVolumeUSD()
	if !ok {
		return metrics.UnavailableResult(MetricName, MetricVersion, metrics.CoreNamespace, "volume usd not in response")
	}
	marketCapUSD, ok := parsed.GetMarketCapUSD()
	if !ok || marketCapUSD == 0 {
		return metrics.UnavailableResult(MetricName, MetricVersion, metrics.CoreNamespace, "market_cap usd not in response or zero")
	}

	ratio := volumeUSD / marketCapUSD
	classification := classify(ratio)
	summaryStr := summary(ratio, classification.Label)

	d := Data{
		VolumeToMcapRatio: metrics.Ratio(ratio),
		VolumeUSD:         metrics.Currency(volumeUSD),
		MarketCapUSD:      metrics.Currency(marketCapUSD),
		Classification:    classification,
		Summary:           summaryStr,
	}
	dJSON, err := json.Marshal(d)
	if err != nil {
		return metrics.UnavailableResult(MetricName, MetricVersion, metrics.CoreNamespace, fmt.Sprintf("marshaling data: %v", err))
	}

	meta := Meta{
		PrimarySource: "coingecko",
		Confidence:    "high",
	}

	binanceData, hasValidator := data[api.BinanceSpotCVD_BTC_1h]
	if hasValidator && len(binanceData) > 0 {
		meta.ValidatorSource = "binance_us"
		binanceVol, err := binance.ParseKlinesResponse(binanceData)
		switch {
		case err != nil:
			meta.DiscrepancyDetected = true
			meta.DiscrepancyNote = fmt.Sprintf("failed to parse Binance response: %v", err)
			meta.Confidence = "medium"
		case binanceVol.TotalVolume == 0:
			meta.DiscrepancyNote = "Binance kline just opened, no volume yet"
			meta.Confidence = "high"
		case binanceVol.TotalVolume < 1.0:
			meta.DiscrepancyNote = "Binance kline just opened, no significant volume"
			meta.Confidence = "high"
		default:
			if volumeUSD > 0 {
				discrepancy := math.Abs(binanceVol.TotalVolume-volumeUSD) / volumeUSD
				switch {
				case discrepancy > validationThreshold:
					meta.DiscrepancyDetected = true
					meta.DiscrepancyNote = fmt.Sprintf("Binance-US reported %.0f vs CoinGecko %.0f (%.0f%% different)",
						binanceVol.TotalVolume, volumeUSD, discrepancy*100)
					meta.Confidence = "low"
				case discrepancy > validationThreshold/2:
					meta.Confidence = "medium"
				}
			}
		}
	}

	// Add thresholds and description for full detail level
	meta.Thresholds = map[string]float64{
		"high": thresholdHigh,
		"low":  thresholdLow,
	}
	meta.Description = "Liquidity pulse measures 24h trading volume as ratio of total market cap. " +
		"High values (>15%) indicate strong short-term conviction; " +
		"Low values (<5%) suggest accumulation phase or low conviction."

	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return metrics.UnavailableResult(MetricName, MetricVersion, metrics.CoreNamespace, fmt.Sprintf("marshaling meta: %v", err))
	}

	thinData := marketCapUSD < 1e12
	status := metrics.DetectStatus(metrics.ConfidenceToFloat(meta.Confidence), thinData)

	return output.MetricResult{
		Metric:    MetricName,
		Version:   MetricVersion,
		Namespace: metrics.CoreNamespace,
		Status:    status,
		Data:      json.RawMessage(dJSON),
		Meta:      json.RawMessage(metaJSON),
	}, nil
}
