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

const (
	MetricName    = "liquidity-pulse"
	MetricVersion = "v1.0.0"
)

const (
	ClassificationHigh   = "high"
	ClassificationNormal = "normal"
	ClassificationLow    = "low"
)

type Classification struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

type Data struct {
	VolumeToMcapRatio float64       `json:"volume_to_mcap_ratio"`
	VolumeUSD         float64       `json:"volume_usd"`
	MarketCapUSD     float64       `json:"market_cap_usd"`
	Classification   Classification `json:"classification"`
	Summary          string        `json:"summary"`
}

const (
	thresholdHigh        = 0.15
	thresholdLow          = 0.05
	validationThreshold  = 0.20
)

func confidenceToFloat(conf string) float64 {
	switch conf {
	case "high":
		return 0.9
	case "medium":
		return 0.6
	case "low":
		return 0.3
	default:
		return 0.0
	}
}

type Meta struct {
	PrimarySource       string `json:"primary_source"`
	ValidatorSource  string `json:"validator_source,omitempty"`
	DiscrepancyDetected bool   `json:"discrepancy_detected,omitempty"`
	DiscrepancyNote  string `json:"discrepancy_note,omitempty"`
	Confidence       string `json:"confidence"`
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
	return fmt.Sprintf("Volume/MCap: %05.2f%% | Conviction: %s", ratio*100, label)
}

func init() { metrics.MustRegister(&Provider{}) }

type Provider struct{}

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

func (p *Provider) Compute(_ context.Context, data map[string]json.RawMessage) (output.MetricResult, error) {
	globalData, ok := data[api.CoinGeckoGlobalMarket]
	if !ok || len(globalData) == 0 {
		return p.unavailable("missing primary endpoint data")
	}

	parsed, err := coingecko.ParseGlobalResponse(globalData)
	if err != nil {
		return p.unavailable(fmt.Sprintf("parsing primary data: %v", err))
	}

	volumeUSD, ok := parsed.GetVolumeUSD()
	if !ok {
		return p.unavailable("volume usd not in response")
	}
	marketCapUSD, ok := parsed.GetMarketCapUSD()
	if !ok || marketCapUSD == 0 {
		return p.unavailable("market_cap usd not in response or zero")
	}

	ratio := volumeUSD / marketCapUSD
	classification := classify(ratio)
	summaryStr := summary(ratio, classification.Label)

	d := Data{
		VolumeToMcapRatio: ratio,
		VolumeUSD:         volumeUSD,
		MarketCapUSD:      marketCapUSD,
		Classification:   classification,
		Summary:          summaryStr,
	}
	dJSON, err := json.Marshal(d)
	if err != nil {
		return p.unavailable(fmt.Sprintf("marshaling data: %v", err))
	}

	meta := Meta{
		PrimarySource: "coingecko",
		Confidence:   "high",
	}

	binanceData, hasValidator := data[api.BinanceSpotCVD_BTC_1h]
	if hasValidator && len(binanceData) > 0 {
		meta.ValidatorSource = "binance_us"
		binanceVol, err := binance.ParseKlinesResponse(binanceData)
		if err == nil && binanceVol.TotalVolume > 0 && volumeUSD > 0 {
			discrepancy := math.Abs(binanceVol.TotalVolume-volumeUSD) / volumeUSD
			if discrepancy > validationThreshold {
				meta.DiscrepancyDetected = true
				meta.DiscrepancyNote = fmt.Sprintf("Binance-US reported %.0f vs CoinGecko %.0f (%.0f%% different)",
					binanceVol.TotalVolume, volumeUSD, discrepancy*100)
				meta.Confidence = "low"
			} else if discrepancy > validationThreshold/2 {
				meta.Confidence = "medium"
			}
		}
	}

	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return p.unavailable(fmt.Sprintf("marshaling meta: %v", err))
	}

	thinData := marketCapUSD < 1e12
	status := metrics.DetectStatus(confidenceToFloat(meta.Confidence), thinData)

	return output.MetricResult{
		Metric:  MetricName,
		Version: MetricVersion,
		Status: status,
		Data:   json.RawMessage(dJSON),
		Meta:   json.RawMessage(metaJSON),
	}, nil
}

func (p *Provider) unavailable(msg string) (output.MetricResult, error) {
	errMsg, _ := json.Marshal(map[string]string{"error": msg})
	return output.MetricResult{
		Metric:  MetricName,
		Version: MetricVersion,
		Status: "unavailable",
		Data:   json.RawMessage(errMsg),
	}, nil
}